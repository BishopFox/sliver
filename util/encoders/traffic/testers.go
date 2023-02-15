package traffic

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

type TrafficEncoderTestFunc func(*TrafficEncoder) *clientpb.TrafficEncoderTest

const (
	SmallRandom     = "Small Random (64 bytes)"
	MediumRandom    = "Medium Random (8KB)"
	LargeRandom     = "Large Random (8MB)"
	VeryLargeRandom = "Very Large Random (128MB)"
)

var (
	TrafficEncoderTesters = map[string]TrafficEncoderTestFunc{
		SmallRandom:     SmallRandomTester,
		MediumRandom:    MediumRandomTester,
		LargeRandom:     LargeRandomTester,
		VeryLargeRandom: VeryLargeRandomTester,
	}
)

// SmallRandomTester - 64 byte sample
func SmallRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	sample := randomDataOfSize(64)
	return randomTester(SmallRandom, encoder, sample)
}

// MediumRandomTester - 8KB random sample
func MediumRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	sample := randomDataOfSize(8 * 1024)
	return randomTester(MediumRandom, encoder, sample)
}

// LargeRandomTester - 8MB random sample
func LargeRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	sample := randomDataOfSize(8 * 1024 * 1024)
	return randomTester(LargeRandom, encoder, sample)
}

// VeryLargeRandomTester - 128MB random sample
func VeryLargeRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	sample := randomDataOfSize(128 * 1024 * 1024)
	return randomTester(VeryLargeRandom, encoder, sample)
}

func randomTester(name string, encoder *TrafficEncoder, sample []byte) *clientpb.TrafficEncoderTest {
	test := &clientpb.TrafficEncoderTest{Name: name, Success: false}
	started := time.Now()
	defer func() {
		test.Completed = true
		test.Duration = int64(time.Since(started))
		if int64(30*time.Second) < test.Duration {
			test.Success = false
			test.Err = "Test exceeded 30 second limit"
		}
	}()
	encodedSample, err := encoder.Encode(sample)
	if err != nil {
		test.Err = err.Error()
		test.Sample = sample
		return test
	}
	decodedSample, err := encoder.Decode(encodedSample)
	if err != nil {
		test.Err = err.Error()
		test.Sample = sample
		return test
	}
	if !bytes.Equal(sample, decodedSample) {
		test.Err = "Encoded and decoded samples do not match"
		test.Sample = sample
		return test
	}
	test.Success = true
	return test
}
func randomDataOfSize(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}
