package value

import "github.com/tarantool/sdvg/internal/generator/models"

// Verify interface compliance in compile time.
var _ Generator = (*IntegerGenerator)(nil)

// IntegerGenerator type is used to describe generator for integer numbers.
type IntegerGenerator struct {
	*models.ColumnIntegerParams
	totalValuesCount uint64
}

func (g *IntegerGenerator) Prepare() error {
	return nil
}

func (g *IntegerGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount

	return nil
}

// Value returns n-th integer number from range.
func (g *IntegerGenerator) Value(number float64, _ map[string]any) (any, error) {
	value := orderedInt64(g.From, g.To, number, g.totalValuesCount)

	switch g.BitWidth {
	case 8: //nolint:mnd
		return int8(value), nil
	case 16: //nolint:mnd
		return int16(value), nil
	case 32: //nolint:mnd
		return int32(value), nil
	default:
		return value, nil
	}
}

func (g *IntegerGenerator) ValuesCount(_ map[string]uint64) float64 {
	return float64(uint64(g.To-g.From)) + 1
}
