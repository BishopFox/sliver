package certs

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
	"testing"
)

func TestOperatorGenerateCertificate(t *testing.T) {
	GenerateCertificateAuthority(OperatorCA, "")
	cert1, key1, err := OperatorClientGenerateCertificate("test3")
	if err != nil {
		t.Errorf("Failed to store ecc certificate %v", err)
		return
	}

	cert2, key2, err := OperatorClientGetCertificate("test3")
	if err != nil {
		t.Errorf("Failed to get ecc certificate %v", err)
		return
	}

	if !bytes.Equal(cert1, cert2) || !bytes.Equal(key1, key2) {
		t.Errorf("Stored ecc cert/key does match generated cert/key: %v != %v", cert1, cert2)
		return
	}
}
