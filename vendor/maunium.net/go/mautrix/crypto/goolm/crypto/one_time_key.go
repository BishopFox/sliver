package crypto

import (
	"encoding/base64"
	"encoding/binary"

	"maunium.net/go/mautrix/crypto/goolm/libolmpickle"
)

// OneTimeKey stores the information about a one time key.
type OneTimeKey struct {
	ID        uint32            `json:"id"`
	Published bool              `json:"published"`
	Key       Curve25519KeyPair `json:"key,omitempty"`
}

// Equal compares the one time key to the given one.
func (otk OneTimeKey) Equal(other OneTimeKey) bool {
	return otk.ID == other.ID &&
		otk.Published == other.Published &&
		otk.Key.PrivateKey.Equal(other.Key.PrivateKey) &&
		otk.Key.PublicKey.Equal(other.Key.PublicKey)
}

// PickleLibOlm pickles the key pair into the encoder.
func (c OneTimeKey) PickleLibOlm(encoder *libolmpickle.Encoder) {
	encoder.WriteUInt32(c.ID)
	encoder.WriteBool(c.Published)
	c.Key.PickleLibOlm(encoder)
}

// UnpickleLibOlm unpickles the unencryted value and populates the [OneTimeKey]
// accordingly.
func (c *OneTimeKey) UnpickleLibOlm(decoder *libolmpickle.Decoder) (err error) {
	if c.ID, err = decoder.ReadUInt32(); err != nil {
		return
	} else if c.Published, err = decoder.ReadBool(); err != nil {
		return
	}
	return c.Key.UnpickleLibOlm(decoder)
}

// KeyIDEncoded returns the base64 encoded key ID.
func (c OneTimeKey) KeyIDEncoded() string {
	return base64.RawStdEncoding.EncodeToString(binary.BigEndian.AppendUint32(nil, c.ID))
}
