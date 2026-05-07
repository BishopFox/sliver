package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

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
	"context"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

const (
	timeFormat = "2006-01-02 15:04:05 UTC-0700"
)

func convertDatabaseRecordToProtobuf(record *models.Certificate) *clientpb.CertificateData {
	if record == nil {
		return nil
	}

	certData := &clientpb.CertificateData{}
	// These values are default until we parse the cert
	certData.ValidityStart = "Unknown (could not parse certificate)"
	certData.ValidityExpiry = "Unknown (could not parse certificate)"

	certData.CN = record.CommonName
	certData.ID = record.ID.String()
	certData.CreationTime = record.CreatedAt.Format(timeFormat)
	certData.Type = certificateTypeLabel(record.CAType)

	switch record.KeyType {
	case certs.ECCKey:
		certData.KeyAlgorithm = "ECC"
	case certs.RSAKey:
		certData.KeyAlgorithm = "RSA"
	default:
		certData.KeyAlgorithm = strings.ToUpper(record.KeyType)
	}

	// To get the validity period, we need to parse the certificate information
	pemBlock, _ := pem.Decode([]byte(record.CertificatePEM))
	if pemBlock == nil {
		return certData
	}

	certificate, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return certData
	}

	certData.ValidityStart = certificate.NotBefore.Format(timeFormat)
	certData.ValidityExpiry = certificate.NotAfter.Format(timeFormat)

	return certData
}

func convertAuthorityRecordToProtobuf(record *models.CertificateAuthority) *clientpb.CertificateAuthorityData {
	if record == nil {
		return nil
	}

	authData := &clientpb.CertificateAuthorityData{}
	authData.ValidityStart = "Unknown (could not parse certificate)"
	authData.ValidityExpiry = "Unknown (could not parse certificate)"

	authData.CN = record.CommonName
	authData.ID = record.ID.String()
	authData.CreationTime = record.CreatedAt.Format(timeFormat)
	authData.Type = certificateTypeLabel(record.CAType)
	authData.KeyAlgorithm = "Unknown"

	pemBlock, _ := pem.Decode([]byte(record.CertificatePEM))
	if pemBlock == nil {
		return authData
	}

	certificate, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return authData
	}

	if authData.CN == "" {
		authData.CN = certificate.Subject.CommonName
	}

	authData.ValidityStart = certificate.NotBefore.Format(timeFormat)
	authData.ValidityExpiry = certificate.NotAfter.Format(timeFormat)

	switch certificate.PublicKeyAlgorithm {
	case x509.ECDSA:
		authData.KeyAlgorithm = "ECC"
	case x509.RSA:
		authData.KeyAlgorithm = "RSA"
	case x509.Ed25519:
		authData.KeyAlgorithm = "Ed25519"
	default:
		authData.KeyAlgorithm = strings.ToUpper(certificate.PublicKeyAlgorithm.String())
	}

	return authData
}

func certificateTypeLabel(caType string) string {
	switch caType {
	case certs.MtlsImplantCA:
		return "MTLS (Implant)"
	case certs.MtlsServerCA:
		return "MTLS (Server)"
	case certs.HTTPSCA:
		return "HTTPS"
	case certs.OperatorCA:
		return "Operator"
	default:
		return caType
	}
}

func (rpc *Server) GetCertificateInfo(ctx context.Context, req *clientpb.CertificatesReq) (*clientpb.CertificateInfo, error) {
	certInfo := clientpb.CertificateInfo{}

	certInfoDB, err := db.GetCertificateInfo(req.CategoryFilters, req.CN)
	if err != nil {
		return nil, rpcError(err)
	}

	certInfo.Info = make([]*clientpb.CertificateData, len(certInfoDB))

	for idx, record := range certInfoDB {
		certInfo.Info[idx] = convertDatabaseRecordToProtobuf(record)
	}

	return &certInfo, nil
}

func (rpc *Server) GetCertificateAuthorityInfo(ctx context.Context, _ *commonpb.Empty) (*clientpb.CertificateAuthorityInfo, error) {
	authInfo := clientpb.CertificateAuthorityInfo{}

	caInfoDB, err := db.CertificateAuthorities()
	if err != nil {
		return nil, rpcError(err)
	}

	authInfo.Info = make([]*clientpb.CertificateAuthorityData, len(caInfoDB))
	for idx, record := range caInfoDB {
		authInfo.Info[idx] = convertAuthorityRecordToProtobuf(record)
	}

	return &authInfo, nil
}
