package libolmpickle

import (
	"crypto/aes"
	"encoding/json"
	"fmt"

	"maunium.net/go/mautrix/crypto/olm"
)

// PickleAsJSON returns an object as a base64 string encrypted using the supplied key. The unencrypted representation of the object is in JSON format.
func PickleAsJSON(object any, pickleVersion byte, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("pickle: %w", olm.ErrNoKeyProvided)
	}
	marshaled, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("pickle marshal: %w", err)
	}
	marshaled = append([]byte{pickleVersion}, marshaled...)
	toEncrypt := make([]byte, len(marshaled))
	copy(toEncrypt, marshaled)
	//pad marshaled to get block size
	if len(marshaled)%aes.BlockSize != 0 {
		padding := aes.BlockSize - len(marshaled)%aes.BlockSize
		toEncrypt = make([]byte, len(marshaled)+padding)
		copy(toEncrypt, marshaled)
	}
	encrypted, err := Pickle(key, toEncrypt)
	if err != nil {
		return nil, fmt.Errorf("pickle encrypt: %w", err)
	}
	return encrypted, nil
}

// UnpickleAsJSON updates the object by a base64 encrypted string using the supplied key. The unencrypted representation has to be in JSON format.
func UnpickleAsJSON(object any, pickled, key []byte, pickleVersion byte) error {
	if len(key) == 0 {
		return fmt.Errorf("unpickle: %w", olm.ErrNoKeyProvided)
	}
	decrypted, err := Unpickle(key, pickled)
	if err != nil {
		return fmt.Errorf("unpickle decrypt: %w", err)
	}
	//unpad decrypted so unmarshal works
	for i := len(decrypted) - 1; i >= 0; i-- {
		if decrypted[i] != 0 {
			decrypted = decrypted[:i+1]
			break
		}
	}
	if decrypted[0] != pickleVersion {
		return fmt.Errorf("unpickle: %w", olm.ErrWrongPickleVersion)
	}
	err = json.Unmarshal(decrypted[1:], object)
	if err != nil {
		return fmt.Errorf("unpickle unmarshal: %w", err)
	}
	return nil
}
