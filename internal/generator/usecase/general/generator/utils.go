package generator

import (
	"crypto/sha1"
	"math"
	"math/bits"
	"time"
)

//nolint:mnd
var primitivePolynomials = map[int]uint64{
	1:  0x1,
	2:  0x3,
	3:  0x6,
	4:  0xC,
	5:  0x14,
	6:  0x30,
	7:  0x60,
	8:  0xB8,
	9:  0x110,
	10: 0x240,
	11: 0x500,
	12: 0x829,
	13: 0x100D,
	14: 0x2015,
	15: 0x6000,
	16: 0xD008,
	17: 0x12000,
	18: 0x20400,
	19: 0x40023,
	20: 0x90000,
	21: 0x140000,
	22: 0x300000,
	23: 0x420000,
	24: 0xE10000,
	25: 0x1200000,
	26: 0x2000023,
	27: 0x4000013,
	28: 0x9000000,
	29: 0x14000000,
	30: 0x20000029,
	31: 0x48000000,
	32: 0x80200003,
	33: 0x100080000,
	34: 0x204000003,
	35: 0x500000000,
	36: 0x801000000,
	37: 0x100000001F,
	38: 0x2000000031,
	39: 0x4400000000,
	40: 0xA000140000,
	41: 0x12000000000,
	42: 0x300000C0000,
	43: 0x63000000000,
	44: 0xC0000030000,
	45: 0x1B0000000000,
	46: 0x300003000000,
	47: 0x420000000000,
	48: 0xC00000180000,
	49: 0x1008000000000,
	50: 0x3000000C00000,
	51: 0x6000C00000000,
	52: 0x9000000000000,
	53: 0x18003000000000,
	54: 0x30000000030000,
	55: 0x40000040000000,
	56: 0xC0000600000000,
	57: 0x102000000000000,
	58: 0x200004000000000,
	59: 0x600003000000000,
	60: 0xC00000000000000,
	61: 0x1800300000000000,
	62: 0x3000000000000030,
	63: 0x6000000000000000,
	64: 0xD800000000000000,
}

type sequencer func() uint64

func getSeed(intSeed uint64, strSeed string) uint64 {
	if intSeed == 0 {
		intSeed = uint64(time.Now().UnixNano())
	}

	for _, b := range sha1.Sum([]byte(strSeed)) {
		intSeed += uint64(b)
	}

	return intSeed
}

func newOrderedSequencer(distinctValuesCount, rowsCycleSize uint64) sequencer {
	number := uint64(0)

	return func() uint64 {
		res := float64(number) * float64(distinctValuesCount) / float64(rowsCycleSize)

		number++
		number %= rowsCycleSize

		return uint64(res)
	}
}

// newLFSRSequencer creates a new Linear Feedback Shift Register (LFSR) sequencer function.
// The LFSR is a type of shift register that forms a binary sequence using a linear feedback function.
// The LFSR used here is a Galois LFSR (Galois Linear Feedback Shift Register).
// This function generates a sequence of unique numbers up to distinctValuesCount based on the initial seed.
//
// The logic and idea behind the LFSR sequencer are as follows:
//   - Initialize seed.
//     Apply the mask to the input seed to limit its size. If the resulting seed is 0,
//     set it to a default non-zero value to ensure the LFSR starts correctly.
//   - Select primitive polynomial.
//     Retrieve the primitive polynomial for the calculated bit size.
//     Primitive polynomials are used to ensure the LFSR sequence has good statistical properties and a long period.
//     If no polynomial is defined for the given bit size, the function panics (it's impossible for 64-bits integers).
//   - Generate Sequence.
//     Extract the least significant bit of the seed and right shift the seed by one bit.
//     If the extracted bit is 1, XOR the seed with the primitive polynomial to apply the feedback function.
//     Apply the mask to the seed to ensure it remains within the required bit size.
//     If the seed is less than or equal to distinctValuesCount, it is a valid number in the sequence.
//     Otherwise, continue the loop to generate the next seed.
func newLFSRSequencer(distinctValuesCount, rowsCycleSize, seed uint64) sequencer {
	bitsCount := bits.Len64(distinctValuesCount)
	mask := (uint64(1) << bitsCount) - 1

	seed &= mask
	if seed == 0 {
		seed = (uint64(1) << (bitsCount - 1)) | 1
	}

	poly, exists := primitivePolynomials[bitsCount]
	if !exists {
		panic("feedback polynomial not defined for given bit size")
	}

	number := uint64(0)
	initialSeed := seed

	return func() uint64 {
		for {
			bit := seed & 1
			seed >>= 1

			if bit != 0 {
				seed ^= poly
			}

			seed &= mask

			// skip numbers > distinctValuesCount
			if seed <= distinctValuesCount {
				break
			}
		}

		res := seed - 1

		number++
		if number == rowsCycleSize {
			seed = initialSeed
		}

		return res
	}
}

// fastRandomFloat generates a pseudo-random float64 in the range [0,1) from a given uint64 seed.
// This function is deterministic: the same seed always produces the same output.
//
// The logic and idea behind the function are as follows:
//   - It applies the SplitMix64 algorithm to the seed to ensure a well-distributed state.
//   - It then performs a XorShift64 transformation for better bit scrambling, enhancing the uniformity of the output.
//   - The final result is normalized by dividing by 2^64 to map the value to [0,1).
//
// Benchmarks:
//   - rand.New(rand.NewSource(seed)).Float64() - 85885 RPS (13574 ns/op)
//   - fastRandomFloat(seed) - 1000000000 RPS (0.4422 ns/op)
func fastRandomFloat(seed uint64) float64 {
	// SplitMix64 algorithm
	seed += 0x9e3779b97f4a7c15
	seed ^= seed >> 30 //nolint:mnd
	seed *= 0xbf58476d1ce4e5b9
	seed ^= seed >> 27 //nolint:mnd
	seed *= 0x94d049bb133111eb
	seed ^= seed >> 31 //nolint:mnd

	// XorShift64 transformation
	seed ^= seed >> 12 //nolint:mnd
	seed ^= seed << 25 //nolint:mnd
	seed ^= seed >> 27 //nolint:mnd

	// Normalizing to map the value to [0,1)
	return float64(seed) / float64(math.MaxUint64)
}
