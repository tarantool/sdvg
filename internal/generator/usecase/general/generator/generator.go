package generator

import (
	"math"
	"sync"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/generator/value"
)

type rangeGenerator struct {
	numFrom          uint64
	numTo            uint64
	sequencer        sequencer
	dataRandomFactor float64
	generator        value.Generator
	nullPercentage   float64
}

type ColumnGenerator struct {
	dataColumnSeed  uint64
	batchNumber     uint64
	batchMutex      *sync.Mutex
	sequencer       func() uint64
	rangeGenerators []*rangeGenerator
}

func NewColumnGenerator(
	baseSeed uint64, distinctValuesCountByColumn map[string]uint64,
	modelName string, model *models.Model, column *models.Column,
	dataModelName string, dataModel *models.Model, dataColumn *models.Column,
) (*ColumnGenerator, error) {
	columnSeed := getSeed(baseSeed, common.GetKey(modelName, column.Name))
	dataColumnSeed := getSeed(baseSeed, common.GetKey(dataModelName, dataColumn.Name))

	rowsCount := dataModel.RowsCount

	if column.ForeignKey != "" && !column.ForeignKeyOrder {
		rowsCount = model.RowsCount
	}

	columnSequencer := newLFSRSequencer(rowsCount, rowsCount, dataColumnSeed)

	// TODO: add availability to generate ordered ranges
	//nolint:godox
	// columnSequencer := newOrderedSequencer(model.RowsCount, model.RowsCount)

	rangeGenerators := make([]*rangeGenerator, 0, len(dataColumn.Ranges))
	rangeRowsOffset := uint64(0)

	for i, dataRange := range dataColumn.Ranges {
		rangeRowsCount := uint64(math.Ceil(float64(rowsCount) * dataRange.RangePercentage))

		gen, err := newRangeGenerator(
			column, columnSeed, distinctValuesCountByColumn,
			dataModel, dataColumn, dataColumnSeed,
			dataRange, rangeRowsOffset, rangeRowsCount,
		)
		if err != nil {
			return nil, errors.WithMessagef(
				err,
				"failed to create generator for models[%s].columns[%s].ranges[%d]",
				dataModelName, dataColumn.Name, i,
			)
		}

		rangeGenerators = append(rangeGenerators, gen)
		rangeRowsOffset += rangeRowsCount
	}

	return &ColumnGenerator{
		dataColumnSeed:  dataColumnSeed,
		batchMutex:      &sync.Mutex{},
		sequencer:       columnSequencer,
		rangeGenerators: rangeGenerators,
	}, nil
}

func (cg *ColumnGenerator) SkipRows(count uint64) {
	for range count {
		generatorNumber := cg.sequencer()

		_, rangeGen, ok := findRangeGenerator(cg.rangeGenerators, generatorNumber)
		if !ok {
			panic("needed generator not found in rangeGenerator")
		}

		rangeGen.sequencer()
	}
}

//nolint:cyclop
func newRangeGenerator(
	column *models.Column, columnSeed uint64, distinctValuesCountByColumn map[string]uint64,
	dataModel *models.Model, dataColumn *models.Column, dataColumnSeed uint64,
	dataRange *models.Params, rangeRowsOffset, rangeRowsCount uint64,
) (*rangeGenerator, error) {
	var valueGenerator value.Generator

	if dataRange.Values != nil {
		valueGenerator = &value.EnumGenerator{Values: dataRange.Values}
	} else {
		switch dataColumn.Type {
		case "integer":
			valueGenerator = &value.IntegerGenerator{ColumnIntegerParams: dataRange.IntegerParams}
		case "float":
			valueGenerator = &value.FloatGenerator{ColumnFloatParams: dataRange.FloatParams}
		case "string":
			valueGenerator = &value.StringGenerator{ColumnStringParams: dataRange.StringParams}
		case "uuid":
			valueGenerator = &value.UUIDGenerator{}
		case "datetime":
			valueGenerator = &value.DateTimeGenerator{ColumnDateTimeParams: dataRange.DateTimeParams}
		default:
			return nil, errors.Errorf("unsupported type: %q", dataColumn.Type)
		}
	}

	if err := valueGenerator.Prepare(); err != nil {
		return nil, err
	}

	distinctValuesCount := uint64(math.Ceil(float64(dataModel.RowsCount) * dataRange.RangePercentage))

	if dataRange.DistinctPercentage != 0 {
		distinctValuesCount = uint64(math.Ceil(float64(distinctValuesCount) * dataRange.DistinctPercentage))
	}

	if dataRange.DistinctCount != 0 {
		if dataRange.DistinctCount > distinctValuesCount {
			return nil, errors.Errorf(
				"impossible to generate %d distinct values in %d rows",
				dataRange.DistinctCount, distinctValuesCount,
			)
		}

		distinctValuesCount = dataRange.DistinctCount
	}

	generatorValuesCount := valueGenerator.ValuesCount(distinctValuesCountByColumn)

	if float64(distinctValuesCount) > generatorValuesCount {
		if dataRange.DistinctPercentage != 0 || dataRange.DistinctCount != 0 {
			return nil, errors.Errorf("impossible to generate %d distinct values", distinctValuesCount)
		}

		distinctValuesCount = uint64(generatorValuesCount)
	}

	distinctValuesCountByColumn[column.Name] += distinctValuesCount

	rangeOrdered := dataRange.Ordered
	orderSeed := dataColumnSeed

	if column.ForeignKey != "" && !column.ForeignKeyOrder {
		rangeOrdered = column.Params.Ordered
		orderSeed = columnSeed
	}

	var rangeSequencer sequencer

	if rangeOrdered {
		rangeSequencer = newOrderedSequencer(distinctValuesCount, rangeRowsCount)
	} else {
		rangeSequencer = newLFSRSequencer(distinctValuesCount, rangeRowsCount, orderSeed)
	}

	if err := valueGenerator.SetTotalCount(distinctValuesCount); err != nil {
		return nil, err
	}

	dataRandomFactor := 1 - float64(distinctValuesCount)/generatorValuesCount

	return &rangeGenerator{
		numFrom:          rangeRowsOffset,
		numTo:            rangeRowsOffset + rangeRowsCount,
		dataRandomFactor: dataRandomFactor,
		generator:        valueGenerator,
		sequencer:        rangeSequencer,
		nullPercentage:   dataRange.NullPercentage,
	}, nil
}

func findRangeGenerator(rangeGenerators []*rangeGenerator, generatorNumber uint64) (int, *rangeGenerator, bool) {
	for i, rangeGen := range rangeGenerators {
		if rangeGen.numFrom <= generatorNumber && generatorNumber < rangeGen.numTo {
			return i, rangeGen, true
		}
	}

	return 0, nil, false
}

type valueID struct {
	generatorIndex int
	number         float64
}

type BatchGenerator struct {
	numbers    []valueID
	nextNumber int
	valuer     func(number valueID, generatedValues map[string]any) (any, error)
}

func (cg *ColumnGenerator) NewBatchGenerator(batchSize uint64) *BatchGenerator {
	cg.batchMutex.Lock()
	defer cg.batchMutex.Unlock()

	cg.batchNumber++

	numbers := make([]valueID, batchSize)

	for i := range batchSize {
		generatorNumber := cg.sequencer()

		genIdx, rangeGen, ok := findRangeGenerator(cg.rangeGenerators, generatorNumber)
		if !ok {
			panic("needed generator not found in rangeGenerator")
		}

		num := rangeGen.sequencer()
		numbers[i] = valueID{
			generatorIndex: genIdx,
			number:         float64(num) + fastRandomFloat(cg.dataColumnSeed+num)*rangeGen.dataRandomFactor,
		}
	}

	valuer := func(id valueID, generatedValues map[string]any) (any, error) {
		vg := cg.rangeGenerators[id.generatorIndex]

		if vg.nullPercentage > 0 && fastRandomFloat(cg.dataColumnSeed+uint64(id.number)) < vg.nullPercentage {
			return nil, nil //nolint:nilnil
		}

		return vg.generator.Value(id.number, generatedValues)
	}

	return &BatchGenerator{
		numbers: numbers,
		valuer:  valuer,
	}
}

// Value returns random value for described column.
func (g *BatchGenerator) Value(generatedValues map[string]any) (any, error) {
	res, err := g.valuer(g.numbers[g.nextNumber], generatedValues)
	g.nextNumber++
	g.nextNumber %= len(g.numbers)

	return res, err
}
