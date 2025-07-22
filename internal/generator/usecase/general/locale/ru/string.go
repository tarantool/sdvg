package ru

import (
	_ "embed"
	"log"
	"slices"
	"strings"

	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/usecase/general/locale"
	"gopkg.in/yaml.v3"
)

var _ locale.LocalModule = LocaleModule{}

//go:embed strings.yml
var stringsFile []byte
var defaultModule LocaleModule

type LocaleModule struct {
	MaleFirstNames   []string `yaml:"male_first_names"`
	FemaleFirstNames []string `yaml:"female_first_names"`
	allFirstNames    []string
	MaleLastNames    []string `yaml:"male_last_names"`
	femaleLastNames  []string
	allLastNames     []string
	PhonePatterns    []string `yaml:"phone_patterns"`
	Words            []string `yaml:"words"`
	oneLetterWords   []string
}

func toFemaleLastName(surname string) string {
	switch {
	case strings.HasSuffix(surname, "ий"):
		return strings.TrimSuffix(surname, "ий") + "ая"
	case strings.HasSuffix(surname, "ый"):
		return strings.TrimSuffix(surname, "ый") + "ая"
	case strings.HasSuffix(surname, "ой"):
		return strings.TrimSuffix(surname, "ой") + "ая"
	case strings.HasSuffix(surname, "ов"), strings.HasSuffix(surname, "ев"), strings.HasSuffix(surname, "ин"):
		return surname + "а"
	default:
		return surname
	}
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
			MaleLastNames:   common.Filter(defaultModule.MaleLastNames, isLenGood),
			femaleLastNames: common.Filter(defaultModule.femaleLastNames, isLenGood),
			allLastNames:    common.Filter(defaultModule.allLastNames, isLenGood),
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
		'а', 'б', 'в', 'г', 'д', 'е', 'ж', 'з', 'и', 'й', 'к',
		'л', 'м', 'н', 'о', 'п', 'р', 'с', 'т', 'у', 'ф', 'х',
		'ц', 'ч', 'ш', 'щ', 'ъ', 'ы', 'ь', 'э', 'ю', 'я', 'ё',
	}
}

func (lm LocaleModule) LargeLetters() []rune {
	return []rune{
		'Ё', 'А', 'Б', 'В', 'Г', 'Д', 'Е', 'Ж', 'З', 'И', 'Й',
		'К', 'Л', 'М', 'Н', 'О', 'П', 'Р', 'С', 'Т', 'У', 'Ф',
		'Х', 'Ц', 'Ч', 'Ш', 'Щ', 'Ъ', 'Ы', 'Ь', 'Э', 'Ю', 'Я',
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

func (lm LocaleModule) GetLastNames(gender locale.Gender) []string {
	switch gender {
	case locale.FemaleGender:
		return lm.femaleLastNames
	case locale.MaleGender:
		return lm.MaleLastNames
	default:
		return lm.allLastNames
	}
}

func (lm LocaleModule) GetPhonePatterns() []string {
	return lm.PhonePatterns
}

func (lm LocaleModule) GetBytesPerChar() int {
	return 2 //nolint:mnd
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

	defaultModule.femaleLastNames = make([]string, len(defaultModule.MaleLastNames))
	for i, lastName := range defaultModule.MaleLastNames {
		defaultModule.femaleLastNames[i] = toFemaleLastName(lastName)
	}

	defaultModule.allFirstNames = []string{}
	defaultModule.allFirstNames = append(defaultModule.allFirstNames, defaultModule.MaleFirstNames...)
	defaultModule.allFirstNames = append(defaultModule.allFirstNames, defaultModule.FemaleFirstNames...)

	defaultModule.allLastNames = []string{}
	defaultModule.allLastNames = append(defaultModule.allLastNames, defaultModule.MaleLastNames...)
	defaultModule.allLastNames = append(defaultModule.allLastNames, defaultModule.femaleLastNames...)

	slices.Sort(defaultModule.MaleFirstNames)
	slices.Sort(defaultModule.FemaleFirstNames)
	slices.Sort(defaultModule.allFirstNames)
	slices.Sort(defaultModule.MaleLastNames)
	slices.Sort(defaultModule.femaleLastNames)
	slices.Sort(defaultModule.allLastNames)
	locale.SortPhones(defaultModule.PhonePatterns)
	slices.Sort(defaultModule.Words)

	defaultModule.oneLetterWords = common.Filter(defaultModule.Words, func(w string) bool { return len([]rune(w)) == 1 })
}
