package value

import (
	"math"
)

func orderedInt64(from, to int64, number float64, total uint64) int64 {
	step := (float64(to) - float64(from) + 1) / float64(total)

	// We should cast float64 to uint64 firstly, because int64(float64(math.MaxUint64)) == math.MaxInt64
	return from + int64(uint64(step*number))
}

func orderedFloat64(from, to float64, number float64, total uint64) float64 {
	if from == to {
		return from
	}

	scale := number / float64(total)

	return from*(1-scale) + to*scale
}

// orderedPos selects a position from a given list length based on a fractional index value.
// The function ensures that each position is chosen in a distributed manner, maintaining order while
// allowing precise selection even for fractional index values.
//
// Parameters:
//   - length - the length of list to choose from;
//   - index - a fractional value in [0, 1) that determines the position of the selected rune.
//
// Returns:
//   - int - the selected position from list;
//   - float64 - the updated index, adjusted for the next selection.
//
// Step-by-step explanation:
//   - scale index (which is between 0 and 1) to a length of list;
//   - extract the integer part of floatPos, which represents the index of the selected position;
//   - compute the new index, representing the fractional part that remains after selecting res.
func orderedPos(length int, index float64) (int, float64) {
	floatPos := float64(length) * index
	intPos := math.Floor(floatPos)

	res := int(intPos)

	// there is a bug with float64 calculations on Apple ARM cpu's, so this fix this
	index = floatPos - intPos
	if index < 0 {
		index = 0
	}

	return res, index
}

func replaceWithNumber(str string, char rune, number int64) string {
	if str == "" {
		return str
	}

	runes := []rune(str)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == char {
			runes[i] = '0' + rune(number%10) //nolint:mnd
			number /= 10
		}
	}

	return string(runes)
}
