package locale

// LocalModule interface implementation should have strings for selected locale.
type LocalModule interface {
	// SmallLetters should return small letters of selected language
	SmallLetters() []rune
	// LargeLetters should return large letters of selected language
	LargeLetters() []rune
	// GetFirstNames should return first names
	GetFirstNames(gender Gender) []string
	// GetLastNames should return last names
	GetLastNames(gender Gender) []string
	// GetPhonePatterns should return country phone patterns
	GetPhonePatterns() []string
	// GetBytesPerChar should return how many bytes per char of selected language
	GetBytesPerChar() int
	// GetWords should return words of selected language
	GetWords() []string
	// GetOneLetterWords should return one-letter words of selected language
	GetOneLetterWords() []string
}
