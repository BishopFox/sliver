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

const (
	// SliverCA - Directory containing sliver certificates
	SliverCA = "sliver"
)

// SliverGenerateECCCertificate - Generate a certificate signed with a given CA
func SliverGenerateECCCertificate(sliverName string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(SliverCA, sliverName, false, true)
	err := SaveCertificate(SliverCA, ECCKey, sliverName, cert, key)
	return cert, key, err
}

// SliverGenerateRSACertificate - Generate a certificate signed with a given CA
func SliverGenerateRSACertificate(sliverName string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(SliverCA, sliverName, false, true)
	err := SaveCertificate(SliverCA, RSAKey, sliverName, cert, key)
	return cert, key, err
}
