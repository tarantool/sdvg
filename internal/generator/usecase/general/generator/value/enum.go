package value

import (
	"math"

	"github.com/pkg/errors"
)

// Verify interface compliance in compile time.
var _ Generator = (*EnumGenerator)(nil)

// EnumGenerator type is used to describe generator for enumerated values.
type EnumGenerator struct {
	Values           []any
	totalValuesCount uint64
	rowsPerValue     int
}

func (g *EnumGenerator) Prepare() error {
	if len(g.Values) == 0 {
		return errors.New("empty values params fields")
	}

	return nil
}

func (g *EnumGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount
	g.rowsPerValue = int(math.Ceil((float64(totalValuesCount)) / float64(len(g.Values))))

	return nil
}

func (g *EnumGenerator) Value(number float64, _ map[string]any) (any, error) {
	idx := int(math.Floor(number)) / g.rowsPerValue

	return g.Values[idx], nil
}

func (g *EnumGenerator) ValuesCount(_ map[string]uint64) float64 {
	return float64(len(g.Values))
}
