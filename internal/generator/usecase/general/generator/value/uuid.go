package value

import (
	"math"

	"github.com/google/uuid"
)

// Verify interface compliance in compile time.
var _ Generator = (*UUIDGenerator)(nil)

// UUIDGenerator type is used to describe generator for UUID.
type UUIDGenerator struct {
	totalValuesCount uint64
}

func (g *UUIDGenerator) Prepare() error {
	return nil
}

func (g *UUIDGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount

	return nil
}

// Value returns n-th UUID from range.
func (g *UUIDGenerator) Value(number float64, _ map[string]any) (any, error) {
	res := uuid.UUID{}
	index := number / float64(g.totalValuesCount)

	for i := range res {
		var val int
		val, index = orderedPos(math.MaxUint8, index)
		res[i] = byte(val)
	}

	// Version 4
	res[6] = (res[6] & 0x0f) | 0x40 //nolint:mnd
	// Variant is 10
	res[8] = (res[8] & 0x3f) | 0x80 //nolint:mnd

	return res, nil
}

func (g *UUIDGenerator) ValuesCount() float64 {
	return float64(1<<(128-10) - 1) //nolint:mnd
}
