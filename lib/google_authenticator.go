package lib

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
)

// GoogleAuthenticator Google Authenticator
func GoogleAuthenticator(secret string) (verifyCode string, remainSeconds int64, err error) {
	secret = strings.Replace(secret, " ", "", -1)
	secret = strings.ToUpper(secret)
	base32Secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", 0, err
	}
	timestamp := time.Now().Unix()
	timestampInterval30 := timestamp / 30
	remainSeconds = 30 - (timestamp % 30)
	// sign the value using HMAC-SHA1
	hmacSha1 := hmac.New(sha1.New, base32Secret)
	result := make([]byte, 0, 8)
	{
		mask := int64(0xFF)
		shifts := [8]uint16{56, 48, 40, 32, 24, 16, 8, 0}
		for _, shift := range shifts {
			result = append(result, byte((timestampInterval30>>shift)&mask))
		}
	}
	hmacSha1.Write(result)
	hash := hmacSha1.Sum(nil)
	// We're going to use a subset of the generated hash.
	// Using the last nibble (half-byte) to choose the index to start from.
	// This number is always appropriate as it's maximum decimal 15, the hash will
	// have the maximum index 19 (20 bytes of SHA1) and we need 4 bytes.
	offset := hash[len(hash)-1] & 0x0F
	// get a 32-bit (4-byte) chunk from the hash starting at offset
	hashParts := hash[offset : offset+4]
	// ignore the most significant bit as per RFC 4226
	hashParts[0] = hashParts[0] & 0x7F
	number := (uint32(hashParts[0]) << 24) + (uint32(hashParts[1]) << 16) + (uint32(hashParts[2]) << 8) + uint32(hashParts[3])
	// size to 6 digits
	// one million is the first number with 7 digits so the remainder
	// of the division will always return < 7 digits
	pwd := number % 1000000
	return fmt.Sprintf("%06d", pwd), remainSeconds, nil
}

// NewGoogleAuthenticatorSecret Create a Google authentication key from a random string(base32 characters).
func NewGoogleAuthenticatorSecret(length int) string {
	return RandomString(length, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567")...)
}
