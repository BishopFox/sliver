package encryptor

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
)

// Sign calculates the aispeech request signature.
func Sign(token string, timestamp int64, nonce string, body []byte) string {
	bodyMD5 := md5.Sum(body)
	bodySign := hex.EncodeToString(bodyMD5[:])

	h := md5.New()
	h.Write([]byte(token))
	h.Write([]byte(strconv.FormatInt(timestamp, 10)))
	h.Write([]byte(nonce))
	h.Write([]byte(bodySign))
	return hex.EncodeToString(h.Sum(nil))
}
