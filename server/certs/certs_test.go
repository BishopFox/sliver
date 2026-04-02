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

func TestOperatorServerGenerateCertificateReplacesExistingCertificate(t *testing.T) {
	GenerateCertificateAuthority(OperatorCA, "")

	cert1, key1, err := OperatorServerGenerateCertificate("multiplayer")
	if err != nil {
		t.Fatalf("generate first operator server certificate: %v", err)
	}

	cert2, key2, err := OperatorServerGenerateCertificate("multiplayer")
	if err != nil {
		t.Fatalf("generate replacement operator server certificate: %v", err)
	}

	storedCert, storedKey, err := OperatorServerGetCertificate("multiplayer")
	if err != nil {
		t.Fatalf("get operator server certificate: %v", err)
	}
	if !bytes.Equal(storedCert, cert2) || !bytes.Equal(storedKey, key2) {
		t.Fatalf("expected latest operator server certificate to be stored")
	}
	if bytes.Equal(cert1, cert2) && bytes.Equal(key1, key2) {
		t.Fatalf("expected regenerated operator server certificate to differ from original")
	}
}
