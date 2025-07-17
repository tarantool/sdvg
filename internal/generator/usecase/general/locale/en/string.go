package en

import (
	_ "embed"
	"log"
	"slices"

	"gopkg.in/yaml.v3"

	"sdvg/internal/generator/common"
	"sdvg/internal/generator/models"
	"sdvg/internal/generator/usecase/general/locale"
)

// Verify interface compliance in compile time.
var _ locale.LocalModule = LocaleModule{}

//go:embed strings.yml
var stringsFile []byte
var defaultModule LocaleModule

type LocaleModule struct {
	MaleFirstNames   []string `yaml:"male_first_names"`
	FemaleFirstNames []string `yaml:"female_first_names"`
	allFirstNames    []string
	LastNames        []string `yaml:"last_names"`
	PhonePatterns    []string `yaml:"phone_patterns"`
	Words            []string `yaml:"words"`
	oneLetterWords   []string
}

func NewLocaleModule(logicalType string, minLen, maxLen int) *LocaleModule {
	isLenGood := func(v string) bool {
		return minLen <= len(v) && len(v) <= maxLen
	}

	switch logicalType {
	case models.FirstNameType:
		return &LocaleModule{
			MaleFirstNames:   common.Filter(defaultModule.MaleFirstNames, isLenGood),
			FemaleFirstNames: common.Filter(defaultModule.FemaleFirstNames, isLenGood),
			allFirstNames:    common.Filter(defaultModule.allFirstNames, isLenGood),
		}
	case models.LastNameType:
		return &LocaleModule{
			LastNames: common.Filter(defaultModule.LastNames, isLenGood),
		}
	case models.PhoneType:
		return &LocaleModule{
			PhonePatterns: common.Filter(defaultModule.PhonePatterns, isLenGood),
		}
	case models.TextType:
		return &LocaleModule{
			Words:          defaultModule.Words,
			oneLetterWords: defaultModule.oneLetterWords,
		}
	default:
		return &LocaleModule{}
	}
}

func (lm LocaleModule) SmallLetters() []rune {
	return []rune{
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
		'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	}
}

func (lm LocaleModule) LargeLetters() []rune {
	return []rune{
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
		'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	}
}

func (lm LocaleModule) GetFirstNames(gender locale.Gender) []string {
	switch gender {
	case locale.FemaleGender:
		return lm.FemaleFirstNames
	case locale.MaleGender:
		return lm.MaleFirstNames
	default:
		return lm.allFirstNames
	}
}

func (lm LocaleModule) GetLastNames(_ locale.Gender) []string {
	return lm.LastNames
}

func (lm LocaleModule) GetPhonePatterns() []string {
	return lm.PhonePatterns
}

func (lm LocaleModule) GetBytesPerChar() int {
	return 1
}

func (lm LocaleModule) GetWords() []string {
	return lm.Words
}

func (lm LocaleModule) GetOneLetterWords() []string {
	return lm.oneLetterWords
}

func init() {
	err := yaml.Unmarshal(stringsFile, &defaultModule)
	if err != nil {
		log.Fatalf("parse locale constants: %s", err)
	}

	defaultModule.allFirstNames = []string{}
	defaultModule.allFirstNames = append(defaultModule.allFirstNames, defaultModule.MaleFirstNames...)
	defaultModule.allFirstNames = append(defaultModule.allFirstNames, defaultModule.FemaleFirstNames...)

	slices.Sort(defaultModule.MaleFirstNames)
	slices.Sort(defaultModule.FemaleFirstNames)
	slices.Sort(defaultModule.allFirstNames)
	slices.Sort(defaultModule.LastNames)
	locale.SortPhones(defaultModule.PhonePatterns)
	slices.Sort(defaultModule.Words)

	defaultModule.oneLetterWords = common.Filter(defaultModule.Words, func(w string) bool { return len([]rune(w)) == 1 })
}
