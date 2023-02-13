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
	SmallRandom  = "SmallRandom"
	MediumRandom = "MediumRandom"
	LargeRandom  = "LargeRandom"
)

var (
	TrafficEncoderTesters = map[string]TrafficEncoderTestFunc{
		SmallRandom:  SmallRandomTester,
		MediumRandom: MediumRandomTester,
		LargeRandom:  LargeRandomTester,
	}
)

// SmallRandomTester - 64 byte sample
func SmallRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	test := &clientpb.TrafficEncoderTest{Name: SmallRandom, Success: false}
	started := time.Now()
	defer func() {
		test.Duration = int64(time.Since(started))
		test.Completed = true
	}()
	sample := randomDataOfSize(64)
	encodedSample, err := encoder.Encode(sample)
	if err != nil {
		test.Err = err.Error()
		return test
	}
	decodedSample, err := encoder.Decode(encodedSample)
	if err != nil {
		test.Err = err.Error()
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

// MediumRandomTester - 8KB random sample
func MediumRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	test := &clientpb.TrafficEncoderTest{Name: MediumRandom, Success: false}
	started := time.Now()
	defer func() {
		test.Duration = int64(time.Since(started))
		test.Completed = true
	}()
	sample := randomDataOfSize(8 * 1024)
	encodedSample, err := encoder.Encode(sample)
	if err != nil {
		test.Err = err.Error()
		return test
	}
	decodedSample, err := encoder.Decode(encodedSample)
	if err != nil {
		test.Err = err.Error()
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

// LargeRandomTester - 2MB random sample
func LargeRandomTester(encoder *TrafficEncoder) *clientpb.TrafficEncoderTest {
	test := &clientpb.TrafficEncoderTest{Name: LargeRandom, Success: false}
	started := time.Now()
	defer func() {
		test.Duration = int64(time.Since(started))
		test.Completed = true
	}()
	sample := randomDataOfSize(2 * 1024 * 1024)
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
