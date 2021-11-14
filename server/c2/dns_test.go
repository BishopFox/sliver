package c2

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"crypto/rand"
	insecureRand "math/rand"
	"testing"

	"github.com/bishopfox/sliver/util/encoders"
)

var (
	example1  = "1.example.com."
	c2Domains = []string{example1}
)

func randomDataRandomSize(maxSize int) []byte {
	buf := make([]byte, insecureRand.Intn(maxSize))
	rand.Read(buf)
	return buf
}

func TestIsC2Domain(t *testing.T) {
	listener := StartDNSListener("", uint16(9999), c2Domains, false)
	isC2, domain := listener.isC2SubDomain(c2Domains, "asdf.1.example.com.")
	if !isC2 {
		t.Error("IsC2Domain expected true, got false")
	}
	if domain != example1 {
		t.Error("IsC2Domain expected example1, got", domain)
	}
	isC2, _ = listener.isC2SubDomain(c2Domains, "asdf.1.foobar.com.")
	if isC2 {
		t.Error("IsC2Domain expected false, got true (1)")
	}
	isC2, _ = listener.isC2SubDomain(c2Domains, "asdf.2.example.com.")
	if isC2 {
		t.Error("IsC2Domain expected false, got true (2)")
	}
}

func TestDetermineLikelyEncoders(t *testing.T) {
	listener := StartDNSListener("", uint16(9999), c2Domains, false)
	sample := randomDataRandomSize(2048)
	b58Sample := string(encoders.Base58{}.Encode(sample))
	likelyEncoders := listener.determineLikelyEncoders(b58Sample)
	_, err := likelyEncoders[0].Decode([]byte(b58Sample))
	if err != nil {
		t.Error("DetermineLikelyEncoders failed to decode sample")
	}
}
