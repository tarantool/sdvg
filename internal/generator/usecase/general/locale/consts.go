package locale

// Common string charsets.

var (
	Numbers      = []rune("0123456789")
	SpecialChars = []rune("!#$%&()*+,-.:;<=>?@_{|}")
)

// Genders declaration.

type Gender int

const (
	FemaleGender Gender = iota
	MaleGender
	AnyGender
	WordsDelimiter = " "
)
