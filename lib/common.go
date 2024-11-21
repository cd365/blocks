package lib

import (
	"math/rand/v2"
)

const (
	Number        = "0123456789"
	EnglishLetter = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz"
)

func EnglishLetterLower() []byte {
	letter := make([]byte, 0, 26)
	for i := byte('a'); i < 'z'; i++ {
		letter = append(letter, i)
	}
	return letter
}

func EnglishLetterUpper() []byte {
	letter := make([]byte, 0, 26)
	for i := byte('A'); i <= 'Z'; i++ {
		letter = append(letter, i)
	}
	return letter
}

func EnglishSymbol() []byte {
	letter := make([]byte, 0, 32)
	for i := byte('!'); i <= '/'; i++ {
		letter = append(letter, i)
	}
	for i := byte(':'); i <= '@'; i++ {
		letter = append(letter, i)
	}
	for i := byte('['); i <= '`'; i++ {
		letter = append(letter, i)
	}
	for i := byte('{'); i <= '~'; i++ {
		letter = append(letter, i)
	}
	return letter
}

// RandomString Generates a random string of specified length.
func RandomString(length int, chars ...byte) string {
	count := len(chars)
	if count == 0 {
		chars = append(chars, Number...)
		count = len(chars)
	}
	if length < 1 {
		length = 1
	}
	randoms := make([]byte, 0, length)
	for i := 0; i < length; i++ {
		randoms = append(randoms, chars[rand.IntN(count)])
	}
	return string(randoms)
}
