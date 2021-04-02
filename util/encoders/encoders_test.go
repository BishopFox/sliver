package encoders

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"crypto/rand"
	insecureRand "math/rand"
	"testing"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(1024))
	rand.Read(buf)
	return buf
}

func TestNopNonce(t *testing.T) {
	nop := NopNonce()
	_, _, err := EncoderFromNonce(nop)
	if err != nil {
		t.Errorf("Nop nonce returned error %s", err)
	}

	nop2 := implantEncoders.NopNonce()
	_, _, err = EncoderFromNonce(nop2)
	if err != nil {
		t.Errorf("Nop nonce returned error %s", err)
	}
}

func TestRandomEncoder(t *testing.T) {
	for index := 0; index < 20; index++ {
		sample := randomData()

		nonce, encoder := RandomEncoder()
		_, encoder2, err := EncoderFromNonce(nonce)
		if err != nil {
			t.Errorf("RandomEncoder() nonce returned error %s", err)
		}
		output := encoder.Encode(sample)
		data, err := encoder2.Decode(output)
		if err != nil {
			t.Errorf("RandomEncoder() encoder2 returned error %s", err)
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("RandomEncoder() encoder2 failed to decode encoder data %s", err)
		}

		nonce, encoder = implantEncoders.RandomEncoder()
		_, encoder2, err = implantEncoders.EncoderFromNonce(nonce)
		if err != nil {
			t.Errorf("RandomEncoder() nonce returned error %s", err)
		}
		output = encoder.Encode(sample)
		data, err = encoder2.Decode(output)
		if err != nil {
			t.Errorf("RandomEncoder() encoder2 returned error %s", err)
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("RandomEncoder() encoder2 failed to decode encoder data %s", err)
		}

		nonce, encoder = RandomEncoder()
		_, encoder2, err = implantEncoders.EncoderFromNonce(nonce)
		if err != nil {
			t.Errorf("RandomEncoder() nonce returned error %s", err)
		}
		output = encoder.Encode(sample)
		data, err = encoder2.Decode(output)
		if err != nil {
			t.Errorf("RandomEncoder() encoder2 returned error %s", err)
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("RandomEncoder() encoder2 failed to decode encoder data %s", err)
		}

		nonce, encoder = implantEncoders.RandomEncoder()
		_, encoder2, err = EncoderFromNonce(nonce)
		if err != nil {
			t.Errorf("RandomEncoder() nonce returned error %s", err)
		}
		output = encoder.Encode(sample)
		data, err = encoder2.Decode(output)
		if err != nil {
			t.Errorf("RandomEncoder() encoder2 returned error %s", err)
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("RandomEncoder() encoder2 failed to decode encoder data %s", err)
		}

	}
}
