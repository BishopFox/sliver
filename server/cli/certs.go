package cli

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/certs"
	"github.com/spf13/cobra"
)

var (
	// CATypes - CA types
	CATypes = map[string]string{
		"operator": certs.OperatorCA,
		"mtls":     certs.MtlsImplantCA,
		"https":    certs.HTTPSCA,
	}
)

// CA - Exported CA format
type CA struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

func validCATypes() []string {
	types := []string{}
	for caType := range CATypes {
		types = append(types, caType)
	}
	return types
}

var cmdImportCA = &cobra.Command{
	Use:   "import-ca",
	Short: "Import certificate authority",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		caType, err := cmd.Flags().GetString(caTypeFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s", caTypeFlagStr, err)
			os.Exit(1)
		}
		ca, ok := CATypes[caType]
		if !ok {
			CAs := strings.Join(validCATypes(), ", ")
			fmt.Printf("Invalid ca type '%s' must be one of %s", caType, CAs)
			os.Exit(1)
		}

		load, err := cmd.Flags().GetString(loadFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s\n", loadFlagStr, err)
			os.Exit(1)
		}
		fi, err := os.Stat(load)
		if os.IsNotExist(err) || fi.IsDir() {
			fmt.Printf("Cannot load file %s\n", load)
			os.Exit(1)
		}

		data, err := os.ReadFile(load)
		if err != nil {
			fmt.Printf("Cannot read file %s", err)
			os.Exit(1)
		}

		importCA := &CA{}
		err = json.Unmarshal(data, importCA)
		if err != nil {
			fmt.Printf("Failed to parse file %s", err)
			os.Exit(1)
		}
		cert := []byte(importCA.Certificate)
		key := []byte(importCA.PrivateKey)
		certs.SaveCertificateAuthority(ca, cert, key)
	},
}

var cmdExportCA = &cobra.Command{
	Use:   "export-ca",
	Short: "Export certificate authority",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		caType, err := cmd.Flags().GetString(caTypeFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s", caTypeFlagStr, err)
			os.Exit(1)
		}
		ca, ok := CATypes[caType]
		if !ok {
			CAs := strings.Join(validCATypes(), ", ")
			fmt.Printf("Invalid ca type '%s' must be one of %s", caType, CAs)
			os.Exit(1)
		}

		save, err := cmd.Flags().GetString(saveFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s\n", saveFlagStr, err)
			os.Exit(1)
		}
		if save == "" {
			save, _ = os.Getwd()
		}

		certs.SetupCAs()
		certificateData, privateKeyData, err := certs.GetCertificateAuthorityPEM(ca)
		if err != nil {
			fmt.Printf("Error reading CA %s\n", err)
			os.Exit(1)
		}
		exportedCA := &CA{
			Certificate: string(certificateData),
			PrivateKey:  string(privateKeyData),
		}

		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if !os.IsNotExist(err) && !fi.IsDir() {
			fmt.Printf("File already exists: %s\n", err)
			os.Exit(1)
		}
		if !os.IsNotExist(err) && fi.IsDir() {
			filename := fmt.Sprintf("%s.ca", filepath.Base(caType))
			saveTo = filepath.Join(saveTo, filename)
		}
		data, _ := json.Marshal(exportedCA)
		err = os.WriteFile(saveTo, data, 0600)
		if err != nil {
			fmt.Printf("Write failed: %s (%s)\n", saveTo, err)
			os.Exit(1)
		}
	},
}
