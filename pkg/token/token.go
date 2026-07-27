package token

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
