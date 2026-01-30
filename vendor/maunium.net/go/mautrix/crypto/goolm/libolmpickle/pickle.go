package libolmpickle

import (
	"crypto/aes"
	"fmt"

	"maunium.net/go/mautrix/crypto/goolm/aessha2"
	"maunium.net/go/mautrix/crypto/goolm/goolmbase64"
	"maunium.net/go/mautrix/crypto/olm"
)

const pickleMACLength = 8

var kdfPickle = []byte("Pickle") //used to derive the keys for encryption

// Pickle encrypts the input with the key and the cipher AESSHA256. The result is then encoded in base64.
func Pickle(key, plaintext []byte) ([]byte, error) {
	if c, err := aessha2.NewAESSHA2(key, kdfPickle); err != nil {
		return nil, err
	} else if ciphertext, err := c.Encrypt(plaintext); err != nil {
		return nil, err
	} else if mac, err := c.MAC(ciphertext); err != nil {
		return nil, err
	} else {
		return goolmbase64.Encode(append(ciphertext, mac[:pickleMACLength]...)), nil
	}
}

// Unpickle decodes the input from base64 and decrypts the decoded input with the key and the cipher AESSHA256.
func Unpickle(key, input []byte) ([]byte, error) {
	ciphertext, err := goolmbase64.Decode(input)
	if err != nil {
		return nil, err
	}
	ciphertext, mac := ciphertext[:len(ciphertext)-pickleMACLength], ciphertext[len(ciphertext)-pickleMACLength:]
	if c, err := aessha2.NewAESSHA2(key, kdfPickle); err != nil {
		return nil, err
	} else if verified, err := c.VerifyMAC(ciphertext, mac); err != nil {
		return nil, err
	} else if !verified {
		return nil, fmt.Errorf("decrypt pickle: %w", olm.ErrBadMAC)
	} else {
		// Set to next block size
		targetCipherText := make([]byte, int(len(ciphertext)/aes.BlockSize)*aes.BlockSize)
		copy(targetCipherText, ciphertext)
		return c.Decrypt(targetCipherText)
	}
}
