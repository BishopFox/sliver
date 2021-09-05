package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var EncryptionKey *string

//Encrypt the results
func Encrypt(bites []byte) []byte {
	plainText, err := pad(bites, aes.BlockSize)
	if err != nil {
		return make([]byte, 0)
	}
	block, _ := aes.NewCipher([]byte(*EncryptionKey))
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return make([]byte, 0)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[aes.BlockSize:], plainText)
	return []byte(fmt.Sprintf("%x", cipherText))
}

//Decrypt a command
func Decrypt(text string) string {
	cipherText, _ := hex.DecodeString(text)
	block, err := aes.NewCipher([]byte(*EncryptionKey))
	if err != nil {
		return ""
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(cipherText, cipherText)
	cipherText, _ = unpad(cipherText, aes.BlockSize)
	return string(cipherText)
}

func pad(buf []byte, size int) ([]byte, error) {
	bufLen := len(buf)
	padLen := size - bufLen%size
	padded := make([]byte, bufLen+padLen)
	copy(padded, buf)
	for i := 0; i < padLen; i++ {
		padded[bufLen+i] = byte(padLen)
	}
	return padded, nil
}

func unpad(padded []byte, size int) ([]byte, error) {
	if len(padded)%size != 0 {
		return nil, errors.New("pkcs7: Padded value wasn't in correct size")
	}
	bufLen := len(padded) - int(padded[len(padded)-1])
	buf := make([]byte, bufLen)
	copy(buf, padded[:bufLen])
	return buf, nil
}
