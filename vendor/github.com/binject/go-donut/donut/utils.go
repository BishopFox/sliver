package donut

import (
	"bytes"
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
)

// RandomString - generates random string of given length
func RandomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		r, _ := rand.Int(rand.Reader, big.NewInt(25))
		bytes[i] = 97 + byte(r.Int64()) //a=97
	}
	return string(bytes)
}

// DownloadFile will download an URL to a byte buffer
func DownloadFile(url string) (*bytes.Buffer, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// GenerateRandomBytes : Generates as many random bytes as you ask for, returns them as []byte
func GenerateRandomBytes(count int) ([]byte, error) {
	b := make([]byte, count)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}
