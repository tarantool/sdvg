package value

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale/en"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale/ru"
)

type prepareFunc func() error

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
	completions      []int64  // completions[i] stores the number of ways to form a text of length i
	lexOrderedOctets []string // precomputed lexicographically ordered IPv4 octets
	powersOfTen      []uint64 // precomputed powers of ten for ISBN generation
	base64Endings    []string // precomputed Base64 endings
}

//nolint:cyclop
func (g *StringGenerator) Prepare() error {
	prepareFuncs := []prepareFunc{
		g.prepareTemplate,
		g.prepareLocaleModule,
		g.prepareCharset,
		g.prepareLogicalType,
	}

	for _, fn := range prepareFuncs {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (g *StringGenerator) prepareTemplate() error {
	if g.Template == "" {
		return nil
	}

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

	return nil
}

func (g *StringGenerator) prepareLocaleModule() error {
	switch g.Locale {
	case "ru":
		g.localeModule = ru.NewLocaleModule(g.LogicalType, g.MinLength, g.MaxLength)
	case "en":
		g.localeModule = en.NewLocaleModule(g.LogicalType, g.MinLength, g.MaxLength)
	default:
		return errors.Errorf("unknown locale: %q", g.Locale)
	}

	return nil
}

func (g *StringGenerator) prepareCharset() error {
	switch g.LogicalType {
	case models.Base64Type:
		g.charset = locale.Base64Charset

	case models.Base64URLType, models.Base64RawURLType:
		g.charset = locale.Base64URLCharset

	case models.HexType:
		g.charset = locale.HexCharset

	default:
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
	}

	slices.Sort(g.charset)

	return nil
}

func (g *StringGenerator) prepareLogicalType() error {
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

	case models.TextType:
		g.calculateCompletions(g.MaxLength + 1)

	case models.Ipv4Type:
		g.generateSortedOctets()

	case models.IsbnType:
		g.calculatePowersOfTen()

	case models.Base64Type, models.Base64URLType:
		g.generateBase64SortedEndings()
	}

	return nil
}

func (g *StringGenerator) SetTotalCount(totalValuesCount uint64) error {
	g.totalValuesCount = totalValuesCount

	if g.LogicalType == models.SimpleStringType || g.LogicalType == models.Base64Type ||
		g.LogicalType == models.Base64URLType || g.LogicalType == models.Base64RawURLType ||
		g.LogicalType == models.HexType {
		tailCount, allowedLength, prefixLength := g.lexicographicRules()
		charsetLength := float64(len(g.charset))

		var allowedCount int

		for length := g.MinLength; length <= g.MaxLength; length++ {
			if allowedLength(length) {
				allowedCount++
			}
		}

		countByLength := make([]float64, g.MaxLength+1)
		avgRangeCount := math.Ceil(float64(totalValuesCount) / float64(allowedCount))

		for length := g.MinLength; length <= g.MaxLength; length++ {
			if !allowedLength(length) {
				continue
			}

			rangeCount := float64(tailCount) * math.Pow(charsetLength, float64(prefixLength(length)))

			var currentLengthCount float64
			if avgRangeCount > rangeCount {
				currentLengthCount = rangeCount

				remainAllowed := 0
				for candidateLength := length + 1; candidateLength <= g.MaxLength; candidateLength++ {
					if allowedLength(candidateLength) {
						remainAllowed++
					}
				}

				if remainAllowed > 0 {
					avgRangeCount += (avgRangeCount - rangeCount) / float64(remainAllowed)
				}
			} else {
				currentLengthCount = math.Ceil(avgRangeCount)
			}
			countByLength[length] = currentLengthCount
		}

		g.countByPrefix = make([]float64, g.MaxLength+1)
		g.sumByPrefix = make([]float64, g.MaxLength+2)

		for prefix := 0; prefix <= g.MaxLength; prefix++ {
			prefixDivider := math.Pow(charsetLength, float64(prefix))
			nextPrefixDivider := prefixDivider * charsetLength

			var endNow float64
			for length := g.MinLength; length <= g.MaxLength; length++ {
				if allowedLength(length) && prefixLength(length) == prefix {
					endNow += countByLength[length] / prefixDivider
				}
			}
			g.countByPrefix[prefix] = endNow

			var sumNext float64
			for length := g.MinLength; length <= g.MaxLength; length++ {
				if allowedLength(length) && prefixLength(length) >= prefix+1 {
					sumNext += countByLength[length] / nextPrefixDivider
				}
			}
			g.sumByPrefix[prefix+1] = sumNext
		}
	}

	return nil
}

func (g *StringGenerator) lexicographicRules() (int, func(int) bool, func(int) int) {
	var (
		tailCount     = 1
		allowedLength = func(length int) bool {
			return true
		}
		prefixLength = func(length int) int {
			return length
		}
	)

	switch g.LogicalType {
	case models.Base64Type, models.Base64URLType:
		tailCount = 4161
		allowedLength = func(length int) bool {
			return length >= 4 && length%4 == 0
		}

		prefixLength = func(length int) int {
			if length < 2 {
				return 0
			}

			return length - 2
		}

	case models.HexType:
		allowedLength = func(length int) bool {
			return length >= 2 && length%2 == 0
		}
	}

	return tailCount, allowedLength, prefixLength
}

// calculateCompletions precomputes completions.
func (g *StringGenerator) calculateCompletions(length int) {
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
	g.completions = make([]int64, 0, length+1)

	for _, blockCount := range completionsBig {
		if !blockCount.IsInt64() {
			break
		}

		g.completions = append(g.completions, blockCount.Int64())
	}
}

func (g *StringGenerator) generateSortedOctets() {
	g.lexOrderedOctets = make([]string, 256)

	for val := 0; val < 256; val++ {
		g.lexOrderedOctets[val] = strconv.Itoa(val)
	}

	sort.Strings(g.lexOrderedOctets)
}

func (g *StringGenerator) calculatePowersOfTen() {
	g.powersOfTen = make([]uint64, 11)
	g.powersOfTen[0] = 1

	for i := 1; i <= 10; i++ {
		g.powersOfTen[i] = g.powersOfTen[i-1] * 10
	}
}

func (g *StringGenerator) generateBase64SortedEndings() {
	alphabet := make([]rune, len(g.charset)+1)
	copy(alphabet, g.charset)
	alphabet[len(g.charset)] = '='
	slices.Sort(alphabet)

	eqIndex := -1
	for i, r := range alphabet {
		if r == '=' {
			eqIndex = i
			break
		}
	}
	if eqIndex == -1 {
		panic("'=' not found in base64 alphabet")
	}

	charsetLength := len(alphabet)
	g.base64Endings = make([]string, 0, 4161)

	for i := 0; i < charsetLength; i++ {
		for j := 0; j < charsetLength; j++ {
			if i < eqIndex || (alphabet[i] == '=' && alphabet[j] == '=') || i > eqIndex {
				g.base64Endings = append(g.base64Endings, string([]rune{alphabet[i], alphabet[j]}))
			}
		}
	}

	sort.Strings(g.base64Endings)
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

func (g *StringGenerator) ipv4(number float64) string {
	index := uint32(orderedInt64(0, math.MaxUint32, number, g.totalValuesCount))

	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, index)

	return fmt.Sprintf(
		"%s.%s.%s.%s",
		g.lexOrderedOctets[int(indexBytes[0])],
		g.lexOrderedOctets[int(indexBytes[1])],
		g.lexOrderedOctets[int(indexBytes[2])],
		g.lexOrderedOctets[int(indexBytes[3])],
	)
}

// isbn generates lexicographically ordered ISBN-13 strings (with prefix "978" or "979").
// The ordering is determined by the input `number` which maps proportionally to the total ISBN space.
//
// ISBN structure used:
//   - Prefix: fixed 978 or 979 (determined first, all 978 go before all 979 in order);
//   - Country group: 1–5 digits;
//   - Publisher: 1–7 digits (limited so that Country+Publisher ≤ 8 digits total);
//   - Item number: fills remaining digits so that Country+Publisher+Item = 9 digits;
//   - Check digit: 1 digit (0–9, does not follow real ISBN rules).
//
// Precomputed values (from Prepare method):
//   - powersOfTen — [10^0, 10^1, ..., 10^10], used to quickly calculate weights without math.Pow.
//
// Steps in generation:
//
//  1. Scale number to total space
//     step = totalValues / totalValuesCount, index = floor(step * number).
//     This gives the lexicographic index in the full ISBN list.
//
//  2. Determine prefix ("978" or "979")
//     If index >= totalValuesPerPrefix, choose "979" and subtract totalValuesPerPrefix from index.
//     Otherwise, — "978". This ensures all "978..." go before all "979..." lexicographically.
//
// 3. Generate Country Group (1–5 digits)
//   - Each position can either be a hyphen (end of group) or a digit 0–9.
//   - First position must be a digit (cannot be a hyphen).
//   - The number of possible publisherBlockLengths after a given countryBlockLength is precomputed using the formula:
//     ((5-countryBlockLength)(10-countryBlockLength))/2
//     This avoids looping for each length.
//   - Multiply by 10^(remaining digits) to get digitWeight — number of ISBNs for each digit choice.
//   - digit = index / digitWeight, then index %= digitWeight.
//   - Append the digit and update hyphenWeight — number of ISBNs if we put a hyphen next.
//
// 4. Generate Publisher (1–maxPublisherBlockLength digits)
//   - Similar logic as Country group: first digit is mandatory, subsequent positions can be a hyphen or digit.
//   - maxPublisherLength = min(7, 8 - countryLen).
//   - digitWeight here = (remaining publisher digits) × 10^(remaining total digits after current position).
//   - Append digits until either max length reached or hyphen chosen.
//
// 5. Generate Item Number & Check Digit
//
//   - itemBlockLength = 9 - countryBlockLength - publisherBlockLength.
//
//   - itemBlock = index / 10 — because last digit is reserved for check digit.
//
//   - checkDigit = index % 10.
//
//     6. Format output
//     Combine all blocks: "prefix-countryBlock-publisherBlock-itemBlock-checkDigit".
//     Item number is zero-padded to always match itemBlockLength.
//
// This approach ensures:
//   - Full lexicographic ordering across all possible ISBNs.
//   - Even distribution when scaling from `number`.
//   - No pre-generation of all ISBNs — computed on demand in O(1) time.
func (g *StringGenerator) isbn(number float64) string {
	totalValuesPerPrefix := 25 * g.powersOfTen[10]
	totalValues := 2 * totalValuesPerPrefix

	step := float64(totalValues) / float64(g.totalValuesCount)
	index := uint64(step * number)

	prefix := "978"
	if index >= totalValuesPerPrefix {
		prefix = "979"
		index -= totalValuesPerPrefix
	}

	var (
		countryBlock       = make([]byte, 0, 5)
		countryBlockLength int
		hyphenWeight       uint64
	)

	for countryBlockLength < 5 && (countryBlockLength == 0 || index >= hyphenWeight) {
		if countryBlockLength > 0 {
			index -= hyphenWeight
		}

		totalPossiblePublisherLengths := uint64((5 - countryBlockLength) * (10 - countryBlockLength) / 2)

		digitWeight := totalPossiblePublisherLengths * g.powersOfTen[9-countryBlockLength]
		digit := index / digitWeight
		index %= digitWeight

		countryBlock = append(countryBlock, '0'+byte(digit))
		countryBlockLength++

		hyphenWeight = uint64(8-countryBlockLength) * g.powersOfTen[10-countryBlockLength]
	}

	var (
		maxPublisherBlockLength = 8 - countryBlockLength
		publisherBlock          = make([]byte, 0, maxPublisherBlockLength)
		publisherBlockLength    int
	)

	for publisherBlockLength < maxPublisherBlockLength && (publisherBlockLength == 0 || index >= hyphenWeight) {
		if publisherBlockLength > 0 {
			index -= hyphenWeight
		}

		remaining := maxPublisherBlockLength - publisherBlockLength
		digitWeight := uint64(remaining) * g.powersOfTen[9-countryBlockLength-publisherBlockLength]
		digit := index / digitWeight
		index %= digitWeight

		publisherBlock = append(publisherBlock, '0'+byte(digit))
		publisherBlockLength++

		hyphenWeight = g.powersOfTen[10-countryBlockLength-publisherBlockLength]
	}

	var (
		itemBlockLength = 9 - countryBlockLength - publisherBlockLength
		itemBlock       = index / 10
		checkDigit      = index % 10
	)

	return fmt.Sprintf(
		"%s-%s-%s-%0*d-%d",
		prefix,
		string(countryBlock),
		string(publisherBlock),
		itemBlockLength, itemBlock,
		checkDigit,
	)
}

func (g *StringGenerator) base64(number float64) string {
	prefix := make([]rune, 0, g.MaxLength)

	var (
		remain    float64
		prefixLen int
	)

	for remain = number; ; {
		prefixLen = len(prefix)

		remain -= g.countByPrefix[prefixLen]
		if remain < 0 || g.sumByPrefix[prefixLen+1] == 0 {
			break
		}

		i := int(remain / g.sumByPrefix[prefixLen+1])
		remain -= g.sumByPrefix[prefixLen+1] * float64(i)
		prefix = append(prefix, g.charset[i])
	}

	pos := remain + g.countByPrefix[prefixLen]
	idx := int(pos / g.countByPrefix[prefixLen] * float64(len(g.base64Endings)))

	return string(prefix) + g.base64Endings[idx]
}

func (g *StringGenerator) base64URL(number float64) string {
	return g.base64(number)
}

func (g *StringGenerator) base64RawURL(number float64) string {
	return g.simpleString(number)
}

func (g *StringGenerator) hex(number float64) string {
	return g.simpleString(number)
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
	case models.HexType:
		return g.hex(number), nil
	case models.Ipv4Type:
		return g.ipv4(number), nil
	case models.IsbnType:
		return g.isbn(number), nil
	case models.Base64Type:
		return g.base64(number), nil
	case models.Base64URLType:
		return g.base64URL(number), nil
	case models.Base64RawURLType:
		return g.base64RawURL(number), nil
	case models.SimpleStringType:
		return g.simpleString(number), nil
	default:
		return nil, errors.Errorf("unknown logical type: %s", g.LogicalType)
	}
}

//nolint:cyclop
func (g *StringGenerator) ValuesCount() float64 {
	if g.Template != "" {
		// Using `distinct` or `ordered` parameters with templates
		// is not possible, we cannot guarantee that these parameters
		// will be met, so we just need to return something other than 0.
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

	case models.Ipv4Type:
		// IPv4: 32-bit address space, total unique addresses = 2^32.
		// +1 because MaxUint32 is 2^32 - 1.
		return float64(math.MaxUint32 + 1)

	case models.IsbnType:
		// ISBN-13: we support prefixes 978 and 979 -> 2 variants.
		// For each prefix: 25 possible group partitioning schemes × 10^10 digit combinations.
		// Total unique ISBNs = 2 * 25 * 10^10.
		return 2 * 25 * math.Pow(10, 10)

	case models.Base64Type, models.Base64URLType:
		// Lengths are always multiples of 4. For each length L:
		//   - The first L-2 characters can be any of the 64 Base64 symbols (no '=' allowed there).
		//   - The last 2 characters can form 3 types of endings:
		//     1) Both are Base64 symbols -> 64 * 64 = 4096 combinations
		//     2) Base64 symbol + '=' -> 64 * 1 = 64 combinations
		//     3) '=' + '=' -> 1 * 1 = 1 combination
		//   Total endings per prefix = 4096 + 64 + 1 = 4161.
		total := float64(0)
		for length := g.MinLength; length <= g.MaxLength; length += 4 {
			total += math.Pow(float64(len(g.charset)), float64(length-2)) * 4161
		}

		return total

	case models.HexType:
		total := float64(0)
		for length := g.MinLength; length <= g.MaxLength; length += 2 {
			total += math.Pow(float64(len(g.charset)), float64(length))
		}

		return total

	case models.SimpleStringType, models.Base64RawURLType:
		total := float64(0)
		for length := g.MinLength; length <= g.MaxLength; length++ {
			total += math.Pow(float64(len(g.charset)), float64(length))
		}

		return total

	default:
		return 0
	}
}
