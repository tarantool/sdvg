package locale

// Common string charsets.

var (
	Numbers          = []rune("0123456789")
	SpecialChars     = []rune("!#$%&()*+,-.:;<=>?@_{|}")
	Base64Charset    = []rune("+/0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	Base64URLCharset = []rune("-_0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	HexCharset       = []rune("0123456789abcdefABCDEF")
)

// Genders declaration.

type Gender int

const (
	FemaleGender Gender = iota
	MaleGender
	AnyGender
	WordsDelimiter = " "
)
