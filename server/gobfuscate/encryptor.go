package gobfuscate

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const hashedSymbolSize = 5

// An Encrypter encrypts textual tokens.
type Encrypter struct {
	Key string
}

// Encrypt encrypts the token.
// The case of the first letter of the token is preserved.
func (e *Encrypter) Encrypt(token string) string {
	hashArray := sha256.Sum256([]byte(e.Key + token))
	hexStr := strings.ToLower(hex.EncodeToString(hashArray[:hashedSymbolSize]))
	for i, x := range hexStr {
		if x >= '0' && x <= '9' {
			x = 'g' + (x - '0')
			hexStr = hexStr[:i] + string(x) + hexStr[i+1:]
		}
	}
	if strings.ToUpper(token[:1]) == token[:1] {
		hexStr = strings.ToUpper(hexStr[:1]) + hexStr[1:]
	}
	return hexStr
}
