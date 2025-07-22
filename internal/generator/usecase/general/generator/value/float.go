package value

import (
	"math"

	"github.com/tarantool/sdvg/internal/generator/models"
)

// Verify interface compliance in compile time.
var _ Generator = (*FloatGenerator)(nil)

// FloatGenerator type is used to describe generator for float numbers.
type FloatGenerator struct {
	*models.ColumnFloatParams
	totalValuesCount uint64
}

func (g *FloatGenerator) Prepare() error {
	return nil
}

func (g *FloatGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount

	return nil
}

// Value returns n-th float number from range.
func (g *FloatGenerator) Value(number float64) (any, error) {
	value := orderedFloat64(g.From, g.To, number, g.totalValuesCount)

	if g.BitWidth == 32 { //nolint:mnd
		return float32(value), nil
	}

	return value, nil
}

func (g *FloatGenerator) ValuesCount() float64 {
	return math.Inf(1)
}
