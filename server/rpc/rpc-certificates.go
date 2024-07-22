package rpc

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

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
	switch record.CAType {
	case certs.MtlsImplantCA:
		certData.Type = "MTLS (Implant)"
	case certs.MtlsServerCA:
		certData.Type = "MTLS (Server)"
	case certs.HTTPSCA:
		certData.Type = "HTTPS"
	default:
		certData.Type = record.CAType
	}

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

func (rpc *Server) GetCertificateInfo(ctx context.Context, req *clientpb.CertificatesReq) (*clientpb.CertificateInfo, error) {
	certInfo := clientpb.CertificateInfo{}

	certInfoDB, err := db.GetCertificateInfo(req.CategoryFilters, req.CN)
	if err != nil {
		return nil, err
	}

	certInfo.Info = make([]*clientpb.CertificateData, len(certInfoDB))

	for idx, record := range certInfoDB {
		certInfo.Info[idx] = convertDatabaseRecordToProtobuf(record)
	}

	return &certInfo, nil
}
