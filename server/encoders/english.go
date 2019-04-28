package encoders

// English Encoder - An ASCIIEncoder for binary to english text
type English struct{}

// Encode - Base64 Encode
func (e English) Encode(data []byte) string {

	return ""
}

// Decode - Base64 Decode
func (e English) Decode(data string) ([]byte, error) {

	return nil, nil
}
