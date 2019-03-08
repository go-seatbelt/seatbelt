package rand

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

// NewString generates a cryptographically secure random string. Because
// key generation is critical, it will panic if it fails.
func NewString(n int) string {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		panic("source of randomness unavailable: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}
