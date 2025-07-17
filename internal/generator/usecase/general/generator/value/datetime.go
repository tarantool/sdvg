package value

import (
	"time"

	"sdvg/internal/generator/models"
)

// Verify interface compliance in compile time.
var _ Generator = (*DateTimeGenerator)(nil)

// DateTimeGenerator type is used to describe params for DateTime fields.
type DateTimeGenerator struct {
	*models.ColumnDateTimeParams
	totalValuesCount uint64
}

func (g *DateTimeGenerator) Prepare() error {
	return nil
}

func (g *DateTimeGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount

	return nil
}

// Value returns n-th date from range.
func (g *DateTimeGenerator) Value(number float64) (any, error) {
	fromSec := g.From.Unix()
	toSec := g.To.Unix()

	fromNSec := g.From.Nanosecond()
	toNSec := g.To.Nanosecond()

	if toNSec < fromNSec {
		toNSec += int(time.Second)
	}

	valueSec := orderedInt64(fromSec, toSec, number, g.totalValuesCount)

	valueNSec := orderedInt64(int64(fromNSec), int64(toNSec), number, g.totalValuesCount)
	if valueNSec > int64(time.Second) {
		valueNSec -= int64(time.Second)
	}

	value := time.Unix(valueSec, valueNSec).UTC()

	return value, nil
}

func (g *DateTimeGenerator) ValuesCount() float64 {
	fromSec := g.From.Unix()
	toSec := g.To.Unix()

	fromNSec := g.From.Nanosecond()
	toNSec := g.To.Nanosecond()

	if toNSec < fromNSec {
		toNSec += int(time.Second)
	}

	secCount := float64(uint64(toSec-fromSec)) + 1
	nSecCount := float64(uint64(toNSec-fromNSec)) + 1

	return secCount * nSecCount
}
