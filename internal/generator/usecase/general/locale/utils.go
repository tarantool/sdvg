package locale

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

func SortPhones(phonePatterns []string) {
	slices.SortFunc(phonePatterns, func(a, b string) int {
		a0Replaced := strings.ReplaceAll(a, "#", "0")
		a9Replaced := strings.ReplaceAll(a, "#", "9")
		b0Replaced := strings.ReplaceAll(b, "#", "0")
		b9Replaced := strings.ReplaceAll(b, "#", "9")

		cmp0 := cmp.Compare(a0Replaced, b0Replaced)
		cmp9 := cmp.Compare(a9Replaced, b9Replaced)

		if cmp0 != cmp9 {
			panic(fmt.Sprintf("impossible to order phone patterns: %s and %s", a, b))
		}

		return cmp0
	})
}
