package sgn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// STUB will contain the decoder stub for the selected architecture
// Values will be set on init
var STUB map[int]string

// x86DecoderStub is base decoder assembly for 32 bit binaries
const X86_DECODER_STUB = `
	CALL getip
getip:
	POP {R}
	MOV ECX,{S}
	MOV {RL},{K}
decode:
	XOR BYTE PTR [{R}+ECX+data-6],{RL}
	ADD {RL},BYTE PTR [{R}+ECX+data-6]
	LOOP decode
data:
`

// x64DecoderStub  is base decoder assembly for 64 bit binaries
const X64_DECODER_STUB = `
	MOV {RL},{K}
	MOV RCX,{S}
	LEA {R},[RIP+data-1]
decode:
	XOR BYTE PTR [{R}+RCX],{RL}
	ADD {RL},BYTE PTR [{R}+RCX]
	LOOP decode
data:
`

// NewDecoderAssembly creates a unobfuscated decoder stub to the given encoded payload
// with the given architecture and seed value
func (encoder *Encoder) NewDecoderAssembly(payloadSize int) (string, error) {

	decoder := STUB[encoder.architecture]
	reg, err := encoder.GetSafeRandomRegister(encoder.architecture, "ECX")
	if err != nil {
		return "", err
	}

	regL, err := encoder.GetSafeRandomRegister(8, reg, "CL")
	if err != nil {
		return "", err
	}

	decoder = strings.ReplaceAll(decoder, "{R}", reg)
	decoder = strings.ReplaceAll(decoder, "{RL}", regL)
	decoder = strings.ReplaceAll(decoder, "{K}", fmt.Sprintf("0x%x", encoder.Seed))
	decoder = strings.ReplaceAll(decoder, "{S}", fmt.Sprintf("0x%x", payloadSize))
	// fmt.Println(decoder)
	return decoder, nil
}

// AddADFLDecoder creates decoder stub for binaries that are ciphered with CipherADFL function.
func (encoder *Encoder) AddADFLDecoder(payload []byte) ([]byte, error) {
	decoderAssembly, err := encoder.NewDecoderAssembly(len(payload))
	if err != nil {
		return nil, err
	}
	decoder, ok := encoder.Assemble(decoderAssembly)
	if !ok {
		return nil, errors.New("decoder assembly failed")
	}
	return append(decoder, payload...), nil
}

// AddSchemaDecoder creates decoder stub for binaries that are ciphered with SchemaCipher function.
// The schema array that is used on the given payload, architecture of the payload and obfuscation level is required.
func (encoder *Encoder) AddSchemaDecoder(payload []byte, schema SCHEMA) ([]byte, error) {

	index := 0
	// Add garbage instrctions before the ciphered decoder stub
	garbage, err := encoder.GenerateGarbageInstructions()
	if err != nil {
		return nil, err
	}

	payload = append(garbage, payload...)
	index += len(garbage)

	// Add call instruction over the ciphered payload
	payload, err = encoder.AddCallOver(payload)
	if err != nil {
		return nil, err
	}

	// Add garbage instrctions after the ciphered decoder stub
	garbage, err = encoder.GenerateGarbageInstructions()
	if err != nil {
		return nil, err
	}
	payload = append(payload, garbage...)

	reg, err := encoder.GetSafeRandomRegister(encoder.architecture, encoder.GetStackPointer())
	if err != nil {
		return nil, err
	}

	pop, ok := encoder.Assemble(fmt.Sprintf("POP %s;", reg)) // !!
	if !ok {
		return nil, errors.New("schema decoder assembly failed")
	}
	payload = append(payload, pop...)

	for _, cursor := range schema {

		// Mandatory obfuscation with coin flip for true polimorphism
		garbage, err = encoder.GenerateGarbageInstructions()
		if err != nil {
			return nil, err
		}
		payload = append(payload, garbage...)

		stepAssembly := ""
		if cursor.Key == nil {
			stepAssembly += fmt.Sprintf("\t%s DWORD PTR [%s+0x%x];\n", cursor.OP, reg, index)
		} else {
			stepAssembly += fmt.Sprintf("\t%s DWORD PTR [%s+0x%x],0x%x;\n", cursor.OP, reg, index, binary.BigEndian.Uint32(cursor.Key))
		}
		//fmt.Println(stepAssembly)
		decipherStep, ok := encoder.Assemble(stepAssembly)
		if !ok {
			//fmt.Println(stepAssembly)
			return nil, errors.New("schema decoder step assembly failed")
		}
		payload = append(payload, decipherStep...)
		index += 4
	}

	returnInstruction, ok := encoder.Assemble(fmt.Sprintf("jmp %s;", reg))
	if !ok {
		return nil, errors.New("schema decoder return assembly failed")
	}

	return append(payload, returnInstruction...), nil
}
