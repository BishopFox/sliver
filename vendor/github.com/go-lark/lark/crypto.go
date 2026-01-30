package lark

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// EncryptKey .
func EncryptKey(key string) []byte {
	sha256key := sha256.Sum256([]byte(key))
	return sha256key[:sha256.Size]
}

// Decrypt with AES Cipher
func Decrypt(encryptedKey []byte, data string) ([]byte, error) {
	block, err := aes.NewCipher(encryptedKey)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(data)
	iv := encryptedKey[:aes.BlockSize]
	blockMode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(data))
	blockMode.CryptBlocks(decryptedData, ciphertext)
	msg := unpad(decryptedData)
	if len(msg) < block.BlockSize() {
		return nil, errors.New("msg length is less than blocksize")
	}
	return msg[block.BlockSize():], err
}

func unpad(data []byte) []byte {
	length := len(data)
	var unpadding, unpaddingIdx int
	for i := length - 1; i > 0; i-- {
		if data[i] != 0 {
			unpadding = int(data[i])
			unpaddingIdx = length - 1 - i
			break
		}
	}
	return data[:(length - unpaddingIdx - unpadding)]
}

// GenSign generate sign for notification bot
func GenSign(secret string, timestamp int64) (string, error) {
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret

	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}

	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}
