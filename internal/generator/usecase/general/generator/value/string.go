package value

import (
	"bytes"
	"math"
	"math/big"
	"slices"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale/en"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale/ru"
)

// Verify interface compliance in compile time.
var _ Generator = (*StringGenerator)(nil)

// StringGenerator type is used to describe generator for strings.
type StringGenerator struct {
	*models.ColumnStringParams
	totalValuesCount uint64
	template         *template.Template
	bufPool          *sync.Pool
	localeModule     locale.LocalModule
	charset          []rune
	countByPrefix    []float64
	sumByPrefix      []float64
	completions      []int64 // completions[i] stores the number of ways to form a text of length i
}

//nolint:cyclop
func (g *StringGenerator) Prepare() error {
	if g.Template != "" {
		tmpl, err := template.New("template").
			Option("missingkey=error").
			Funcs(template.FuncMap{
				"upper": strings.ToUpper,
				"lower": strings.ToLower,
			}).
			Parse(g.Template)
		if err != nil {
			return errors.Errorf("failed to parse template: %s", err.Error())
		}

		g.template = tmpl
		g.bufPool = &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		}
	}

	switch g.Locale {
	case "ru":
		g.localeModule = ru.NewLocaleModule(g.LogicalType, g.MinLength, g.MaxLength)
	case "en":
		g.localeModule = en.NewLocaleModule(g.LogicalType, g.MinLength, g.MaxLength)
	default:
		return errors.Errorf("unknown locale: %q", g.Locale)
	}

	switch g.LogicalType {
	case models.FirstNameType:
		if len(g.localeModule.GetFirstNames(locale.MaleGender)) == 0 {
			return errors.Errorf("no male first names with length between %v and %v", g.MinLength, g.MaxLength)
		}

		if len(g.localeModule.GetFirstNames(locale.FemaleGender)) == 0 {
			return errors.Errorf("no female first names with length between %v and %v", g.MinLength, g.MaxLength)
		}
	case models.LastNameType:
		if len(g.localeModule.GetLastNames(locale.MaleGender)) == 0 {
			return errors.Errorf("no male last names with length between %v and %v", g.MinLength, g.MaxLength)
		}

		if len(g.localeModule.GetLastNames(locale.FemaleGender)) == 0 {
			return errors.Errorf("no female last names with length between %v and %v", g.MinLength, g.MaxLength)
		}
	case models.PhoneType:
		if len(g.localeModule.GetPhonePatterns()) == 0 {
			return errors.Errorf("no phone patterns with length between %v and %v", g.MinLength, g.MaxLength)
		}
	}

	g.charset = make([]rune, 0)

	if !g.WithoutLargeLetters {
		g.charset = append(g.charset, g.localeModule.LargeLetters()...)
	}

	if !g.WithoutSmallLetters {
		g.charset = append(g.charset, g.localeModule.SmallLetters()...)
	}

	if !g.WithoutNumbers {
		g.charset = append(g.charset, locale.Numbers...)
	}

	if !g.WithoutSpecialChars {
		g.charset = append(g.charset, locale.SpecialChars...)
	}

	slices.Sort(g.charset)

	if g.LogicalType == models.TextType {
		g.completions = g.calculateCompletions(g.MaxLength + 1)
	}

	return nil
}

func (g *StringGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount

	if g.LogicalType == "" && g.Template == "" {
		countByLength := make([]float64, g.MaxLength+1)
		avgRangeCount := math.Ceil(float64(totalValuesCount) / float64(g.MaxLength-g.MinLength+1))

		for length := g.MinLength; length <= g.MaxLength; length++ {
			rangeCount := math.Pow(float64(len(g.charset)), float64(length))

			var currentLenCount float64
			if avgRangeCount > rangeCount {
				currentLenCount = rangeCount
				avgRangeCount += (avgRangeCount - rangeCount) / float64(g.MaxLength-length)
			} else {
				currentLenCount = math.Ceil(avgRangeCount)
			}

			countByLength[length] = currentLenCount
		}

		g.countByPrefix = make([]float64, g.MaxLength+1)
		g.sumByPrefix = make([]float64, g.MaxLength+1)

		for prefix := 0; prefix <= g.MaxLength; prefix++ {
			prefixDivider := math.Pow(float64(len(g.charset)), float64(prefix))
			g.countByPrefix[prefix] = countByLength[prefix] / prefixDivider

			for length := 0; length <= g.MaxLength-prefix; length++ {
				g.sumByPrefix[prefix] += countByLength[length+prefix] / prefixDivider
			}
		}
	}

	return nil
}

// calculateCompletions precomputes completions.
func (g *StringGenerator) calculateCompletions(length int) []int64 {
	words := g.localeModule.GetWords()
	bytesPerChar := g.localeModule.GetBytesPerChar()
	delimiterLen := len(locale.WordsDelimiter)

	completionsBig := make([]*big.Int, length+1)
	for i := range completionsBig {
		completionsBig[i] = big.NewInt(0)
	}

	// Base case: one way to form a text of length 0 (the empty text).
	completionsBig[0].SetInt64(1)

	// Base case: all one-letter words.
	for _, w := range words {
		if len(w) == 1 {
			completionsBig[1].Add(completionsBig[1], big.NewInt(1))
		}
	}

	// For every target length, add ways by choosing each word that fits.
	for l := 2; l <= length; l++ {
		for _, w := range words {
			wLen := len(w)/bytesPerChar + delimiterLen
			if wLen <= l {
				completionsBig[l].Add(completionsBig[l], completionsBig[l-wLen])
			}
		}
	}

	// convert from big.Int to int64
	completions := make([]int64, 0, length+1)

	for _, blockCount := range completionsBig {
		if !blockCount.IsInt64() {
			break
		}

		completions = append(completions, blockCount.Int64())
	}

	return completions
}

// templateString returns n-th string by template.
//
//nolint:forcetypeassert
func (g *StringGenerator) templateString(rowValues map[string]any) (string, error) {
	buf := g.bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	err := g.template.Execute(buf, rowValues)
	if err != nil {
		g.bufPool.Put(buf)

		return "", errors.New(err.Error())
	}

	val := buf.String()
	g.bufPool.Put(buf)

	return val, nil
}

// patternString returns n-th string by pattern.
func (g *StringGenerator) patternString(number float64) string {
	val := []rune(g.Pattern)
	index := number / float64(g.totalValuesCount)

	for i := range val {
		var letters []rune

		switch val[i] {
		case 'A':
			letters = g.localeModule.LargeLetters()
		case 'a':
			letters = g.localeModule.SmallLetters()
		case '0':
			letters = locale.Numbers
		case '#':
			letters = locale.SpecialChars
		default:
			continue
		}

		var pos int
		pos, index = orderedPos(len(letters), index)
		val[i] = letters[pos]
	}

	return string(val)
}

// firstName returns n-th first name from range.
func (g *StringGenerator) firstName(number float64) string {
	firstNames := g.localeModule.GetFirstNames(locale.AnyGender)

	pos := orderedInt64(0, int64(len(firstNames)-1), number, g.totalValuesCount)

	return firstNames[pos]
}

// lastName returns n-th last name from range.
func (g *StringGenerator) lastName(number float64) string {
	lastNames := g.localeModule.GetLastNames(locale.AnyGender)

	pos := orderedInt64(0, int64(len(lastNames)-1), number, g.totalValuesCount)

	return lastNames[pos]
}

// phone returns n-th phone number from range.
func (g *StringGenerator) phone(number float64) string {
	patterns := g.localeModule.GetPhonePatterns()

	pos := orderedInt64(0, int64(len(patterns)-1), number, g.totalValuesCount)

	pattern := patterns[pos]
	maxPhone := int64(math.Pow(10, float64(strings.Count(pattern, "#")))) - 1 //nolint:mnd

	phone := orderedInt64(0, maxPhone, number, g.totalValuesCount)

	return replaceWithNumber(pattern, '#', phone)
}

// text sorts texts only within their respective length groups.
// Texts of the same length will be ordered, but ordering
// between texts of different lengths is not guaranteed.
//
//nolint:cyclop
func (g *StringGenerator) text(num float64) (string, error) {
	words := g.localeModule.GetWords()
	oneLetterWords := g.localeModule.GetOneLetterWords()
	oneLetterWordsLen := int64(len(oneLetterWords))

	delimiter := locale.WordsDelimiter
	delimiterLen := len(delimiter)

	bytesPerChar := g.localeModule.GetBytesPerChar()

	maxPreComputedLength := len(g.completions) - 1

	wantedLen := g.MinLength + delimiterLen + int(num)%(g.MaxLength-g.MinLength+1)

	number := int64(math.Floor(float64(g.completions[maxPreComputedLength]-1) * (num / float64(g.totalValuesCount))))

	result := make([]byte, 0, wantedLen*bytesPerChar)

	var textLen int

	remaining := maxPreComputedLength
	// Process until we've built the full text.
	for remaining > 0 {
		found := false
		// Iterate over words in lexicographical order.
		if remaining == 1 {
			if number > oneLetterWordsLen-1 {
				return "", errors.Errorf("remaining length is 1 but k: %v overflows: %v", number, oneLetterWordsLen)
			}

			result = append(result, oneLetterWords[number]...)

			textLen++

			break
		}

		for _, w := range words {
			wLen := len(w)/bytesPerChar + delimiterLen
			if wLen > remaining {
				continue
			}
			// count = number of completions if we choose word w at this step.
			count := g.completions[remaining-wLen]
			// If k is within the block for word w, choose it.
			if number < count {
				result = append(result, w...)
				result = append(result, delimiter...)

				textLen += wLen

				remaining -= wLen
				found = true

				break
			}
			// Otherwise, skip this block.
			number -= count
		}

		if !found {
			return "", errors.Errorf("index %v out of range for remaining length %d, %v", number, remaining, wantedLen)
		}
	}

	for textLen < wantedLen {
		w := words[number%int64(len(words)-1)]

		result = append(result, w...)
		result = append(result, delimiter...)

		textLen += len(w)/bytesPerChar + delimiterLen
	}

	text := string(result)

	if textLen > wantedLen {
		if bytesPerChar == 1 {
			text = text[:wantedLen]
		} else {
			text = string([]rune(text)[:wantedLen])
		}
	}

	return text, nil
}

// simpleString generates a lexicographically ordered string based on the given number.
// The function ensures that strings of different lengths are evenly distributed.
//
// Prepared variables (from Prepare method):
//   - countByLength - determines how many strings of each length should be generated; aims for an even distribution
//     but adjusts when the number of possible strings at a given length is limited;
//   - countByPrefix - determines how many times a given prefix should be repeated across generated strings;
//   - sumByPrefix - keeps total number of strings that should be generated with a specific prefix of a certain length.
//
// Each iteration of loop follows these steps:
//   - Subtracting the Current Prefix Group.
//     countByPrefix[prefixLen] represents how many times the current prefix is repeated.
//     We subtract this value from remain to determine if the target string falls within this group.
//     If remain is negative, it means the desired index falls within the current prefix group, so we stop.
//     If sumByPrefix[prefixLen+1] == 0, it means no further characters can be added, so we also stop.
//   - Determining the Next Character.
//     sumByPrefix[prefixLen+1] tells us how many strings exist for the next character choices.
//     remain / sumByPrefix[prefixLen+1] determines how many prefixes we need to skip before choosing next character.
//     We update remain according to reflect the choice. The selected character charset[i] is added to prefix.
//
// This approach ensures precision up to 217 characters in prefix length due to float64 limitations.
// Any additional characters required beyond the ordered prefix are filled in using a pattern based on `number`.
//
// Let's assume that:
//   - charset = ['a', 'b']
//   - min length = 2, max length = 3
//   - total strings = 10
//
// Generated strings and counts:
//   - a   → 0 times
//   - aa  → 1 time
//   - aaa → 0.75 times
//   - aab → 0.75 times
//   - ab  → 1 time
//   - ...
//
// Precomputed values:
//   - countByLength = [0, 4, 6]
//   - countByPrefix = [0, 0, 1, 0.75]
//   - sumByPrefix   = [10, 5, 2.5, 0.75]
//
// Suppose we want to generate simpleString(7), let's trace the loop:
//   - remain -= countByPrefix[0] = 7 - 0 = 7
//     i = remain / sumByPrefix[1] = 7 / 5 = 1 (selects 'b')
//     remain -= sumByPrefix[1] * i = 7 - (5 * 1) = 2
//     prefix = ['b']
//   - remain -= countByPrefix[1] = 2 - 0 = 2
//     i = remain / sumByPrefix[2] = 2 / 2.5 = 0 (selects 'a')
//     remain -= sumByPrefix[2] * i = 2 - (2.5 * 0) = 2
//     prefix = ['b', 'a']
//   - remain -= countByPrefix[2] = 2 - 1 = 1
//     i = remain / sumByPrefix[3] = 1 / 0.75 = 1 (selects 'b')
//     remain -= sumByPrefix[3] * i = 1 - (0.75 * 1) = 0.25
//     prefix = ['b', 'a', 'b']
//   - remain -= countByPrefix[3] = 0.25 - 0.75 = -0.5
//     remain < 0 → break with result "bab"
func (g *StringGenerator) simpleString(number float64) string {
	prefix := make([]rune, 0, g.MaxLength)

	var prefixLen int

	for remain := number; ; {
		prefixLen = len(prefix)

		remain -= g.countByPrefix[prefixLen]
		if remain < 0 || g.sumByPrefix[prefixLen+1] == 0 {
			break
		}

		i := int(remain / g.sumByPrefix[prefixLen+1])
		remain -= g.sumByPrefix[prefixLen+1] * float64(i)
		prefix = append(prefix, g.charset[i])
	}

	// The precision of float64 allows us to generate only 217 prefix characters (which is enough for us).
	// Within the ordered prefix, we can supplement with random characters.
	if prefixLen < g.MinLength {
		destLen := g.MinLength + int(number)%(g.MaxLength-g.MinLength+1)
		for i := range destLen - prefixLen {
			prefix = append(prefix, g.charset[(int(number)+i*i)%len(g.charset)])
		}
	}

	return string(prefix)
}

// Value returns n-th string from range.
func (g *StringGenerator) Value(number float64, rowValues map[string]any) (any, error) {
	if g.Template != "" {
		val, err := g.templateString(rowValues)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to render template string")
		}

		return val, nil
	}

	if g.Pattern != "" {
		return g.patternString(number), nil
	}

	switch g.LogicalType {
	case models.FirstNameType:
		return g.firstName(number), nil
	case models.LastNameType:
		return g.lastName(number), nil
	case models.PhoneType:
		return g.phone(number), nil
	case models.TextType:
		return g.text(number)
	}

	return g.simpleString(number), nil
}

//nolint:cyclop
func (g *StringGenerator) ValuesCount() float64 {
	if g.Template != "" {
		return 1.0
	}

	if g.Pattern != "" {
		total := 1.0

		if count := strings.Count(g.Pattern, "A"); count > 0 {
			total *= math.Pow(float64(len(g.localeModule.LargeLetters())), float64(count))
		}

		if count := strings.Count(g.Pattern, "a"); count > 0 {
			total *= math.Pow(float64(len(g.localeModule.SmallLetters())), float64(count))
		}

		if count := strings.Count(g.Pattern, "0"); count > 0 {
			total *= math.Pow(float64(len(locale.Numbers)), float64(count))
		}

		if count := strings.Count(g.Pattern, "#"); count > 0 {
			total *= math.Pow(float64(len(locale.SpecialChars)), float64(count))
		}

		return total
	}

	switch g.LogicalType {
	case models.FirstNameType:
		return float64(len(g.localeModule.GetFirstNames(locale.AnyGender)))

	case models.LastNameType:
		return float64(len(g.localeModule.GetLastNames(locale.AnyGender)))

	case models.PhoneType:
		totalCount := float64(0)
		for _, pattern := range g.localeModule.GetPhonePatterns() {
			totalCount += math.Pow(float64(10), float64(strings.Count(pattern, "#"))) //nolint:mnd
		}

		return totalCount

	case models.TextType:
		if g.MinLength > len(g.completions) {
			return math.Inf(1)
		}

		totalCount := float64(0)
		for length := g.MinLength; length <= g.MaxLength && length+1 < len(g.completions); length++ {
			totalCount += float64(g.completions[length+1])
		}

		return totalCount
	}

	totalCount := float64(0)
	for length := g.MinLength; length <= g.MaxLength; length++ {
		totalCount += math.Pow(float64(len(g.charset)), float64(length))
	}

	return totalCount
}
