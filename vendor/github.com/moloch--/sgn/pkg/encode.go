package sgn

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// OPERANDS string array containing logical & arithmatic operands
// for encoding the decoder stub
var OPERANDS = []string{"XOR", "SUB", "ADD", "ROL", "ROR", "NOT"}

// SCHEMA contains the operand and keys to apply single step encoding
type SCHEMA []struct {
	OP  string
	Key []byte
}

// Encoder struct for keeping encoder specs
type Encoder struct {
	architecture     int
	ObfuscationLimit int
	PlainDecoder     bool
	Seed             byte
	EncodingCount    int
	SaveRegisters    bool
}

// NewEncoder for creating new encoder structures
func NewEncoder(arch int) (*Encoder, error) {
	seed, err := secureRandomByte()
	if err != nil {
		return nil, err
	}

	// Create with default settings
	encoder := Encoder{
		ObfuscationLimit: 50,
		PlainDecoder:     false,
		Seed:             seed,
		EncodingCount:    1,
		SaveRegisters:    false,
	}

	switch arch {
	case 32:
		encoder.architecture = 32
	case 64:
		encoder.architecture = 64
	default:
		return nil, errors.New("invalid architecture")
	}

	return &encoder, nil
}

// SetArchitecture sets the encoder architecture
func (encoder *Encoder) SetArchitecture(arch int) error {
	switch arch {
	case 32:
		encoder.architecture = 32
	case 64:
		encoder.architecture = 64
	default:
		return errors.New("invalid architecture")
	}
	return nil
}

// GetArchitecture returns the encoder architecture
func (encoder *Encoder) GetArchitecture() int {
	return encoder.architecture
}

// Encode function is the primary encode method for SGN
// all nessary options and parameters are contained inside the encodder struct
func (encoder *Encoder) Encode(payload []byte) ([]byte, error) {
	var randomSeed RandomSeed
	if _, err := cryptorand.Read(randomSeed[:]); err != nil {
		return nil, fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return encoder.EncodeWithSeed(payload, randomSeed)
}

// CipherADFL (Additive Feedback Loop) performs a additive feedback xor operation
// similar to LFSR (Linear-feedback shift register) IN REVERSE ORDER !! with the supplied seed
func CipherADFL(data []byte, seed byte) []byte {
	for i := 1; i < len(data)+1; i++ {
		current := data[len(data)-i]
		data[len(data)-i] ^= seed
		seed = byte((int(current) + int(seed)) % 256)
		//seed = byte(byte(current+seed) % 255)
	}
	return data
}

// SchemaCipher encodes a part of the given binary starting from the given index.
// Encoding done without using any loop conditions based on the schema values.
// Function performs logical/arithmetic operations given in the schema array.
// If invalid operand supplied function returns nil
func (encoder *Encoder) SchemaCipher(data []byte, index int, schema SCHEMA) []byte {

	for _, cursor := range schema {

		switch cursor.OP {
		case "XOR":
			binary.BigEndian.PutUint32(data[index:index+4], (binary.BigEndian.Uint32(data[index:index+4]) ^ binary.LittleEndian.Uint32(cursor.Key)))
		case "ADD":
			binary.LittleEndian.PutUint32(data[index:index+4], binary.LittleEndian.Uint32(data[index:index+4])-binary.BigEndian.Uint32(cursor.Key))
		case "SUB":
			binary.LittleEndian.PutUint32(data[index:index+4], binary.LittleEndian.Uint32(data[index:index+4])+binary.BigEndian.Uint32(cursor.Key))
		case "ROL":
			binary.LittleEndian.PutUint32(data[index:index+4], bits.RotateLeft32(binary.LittleEndian.Uint32(data[index:index+4]), -int(binary.BigEndian.Uint32(cursor.Key))))
		case "ROR":
			binary.LittleEndian.PutUint32(data[index:index+4], bits.RotateLeft32(binary.LittleEndian.Uint32(data[index:index+4]), int(binary.BigEndian.Uint32(cursor.Key))))
		case "NOT":
			binary.BigEndian.PutUint32(data[index:index+4], (^binary.BigEndian.Uint32(data[index : index+4])))
		}
		index += 4
	}
	return data
}

// RandomOperand generates a random operand string
func RandomOperand() string {
	return OPERANDS[secureIntn(len(OPERANDS))]
}

// GetRandomByte generates a random single byte
func GetRandomByte() byte {
	value, err := secureRandomByte()
	if err != nil {
		panic(err)
	}
	return value
}

// GetRandomBytes generates a random byte slice with given size
func GetRandomBytes(num int) []byte {
	slice := make([]byte, num)
	for i := range slice {
		slice[i] = GetRandomByte()
	}
	return slice
}

// CoinFlip implements a coin flip witch returns true/false
func CoinFlip() bool {
	return secureIntn(2) == 0
}

// NewCipherSchema generates random schema for
// using int the SchemaCipher function.
// Generated schema contains random operands and keys.
func (encoder *Encoder) NewCipherSchema(num int) SCHEMA {
	schema := make(SCHEMA, num)

	for i, cursor := range schema {
		cursor.OP = RandomOperand()
		// cursor.OP = "XOR"
		if cursor.OP == "NOT" {
			cursor.Key = nil
		} else if cursor.OP == "ROL" || cursor.OP == "ROR" {

			cursor.Key = []byte{0, 0, 0, GetRandomByte()}
		} else {
			// 4 byte blocks used because of the x64 xor qword ptr instruction boundaries
			cursor.Key = GetRandomBytes(4)
		}
		schema[i] = cursor
	}
	//PrintSchema(schema)
	return schema
}

// GetSchemaTable returns the printable encoder schema table
func GetSchemaTable(schema SCHEMA) string {

	data := &strings.Builder{}
	table := tablewriter.NewWriter(data)
	table.SetHeader([]string{"OPERAND", "KEY"})
	for _, cursor := range schema {
		if cursor.Key == nil {
			table.Append([]string{cursor.OP, "0x00000000"})
		} else {
			table.Append([]string{cursor.OP, fmt.Sprintf("0x%x", cursor.Key)})
		}

	}
	table.Render()

	return data.String()
}
