package value

// Generator interface is used to describe generator methods.
type Generator interface {
	// Prepare method should parse input params for generation
	Prepare() error
	// SetTotalCount method should remember count of rows to generate
	SetTotalCount(totalValuesCount uint64) error
	// Value method should return ordered unique value by number
	Value(number float64, rowValues map[string]any) (any, error)
	// ValuesCount method should return the number of possible values to generate
	ValuesCount() float64
}
