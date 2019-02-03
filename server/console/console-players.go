package console

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sliver/server/assets"
	"sliver/server/certs"

	"github.com/desertbit/grumble"
)

// ClientConfig - Client JSON config
type ClientConfig struct {
	Operator      string `json:"operator"`
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

func newPlayerCmd(ctx *grumble.Context) {
	operator := ctx.Flags.String("operator")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	save := ctx.Flags.String("save")

	regex, _ := regexp.Compile("[^A-Za-z0-9]+") // Only allow alphanumeric chars
	operator = regex.ReplaceAllString(operator, "")

	if operator == "" {
		fmt.Printf("\n" + Warn + "Operator name required (--operator) \n\n")
		return
	}

	if lhost == "" {
		fmt.Printf("\n" + Warn + "Missing lhost (--lhost) \n\n")
		return
	}

	if save == "" {
		fmt.Printf("\n" + Warn + "Save file required (--save)\n\n")
		return
	}

	fmt.Printf("\n" + Info + "Generating new client certificate, please wait ... \n")
	rootDir := assets.GetRootAppDir()
	publicKey, privateKey := certs.GenerateClientCertificate(rootDir, operator, true)
	caCertPEM, _, _ := certs.GetCertificateAuthorityPEM(rootDir, certs.ClientsCertDir)
	config := ClientConfig{
		Operator:      operator,
		LHost:         lhost,
		LPort:         lport,
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}
	configJSON, _ := json.Marshal(config)
	saveTo, _ := filepath.Abs(save)
	fi, err := os.Stat(saveTo)
	if err != nil {
		fmt.Printf(Warn+"Failed to generate sliver %v\n\n", err)
		return
	}
	if fi.IsDir() {
		filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(operator), filepath.Base(lhost))
		saveTo = filepath.Join(saveTo, filename)
	}
	err = ioutil.WriteFile(saveTo, configJSON, 0644)
	if err != nil {
		fmt.Printf("\n"+Warn+"Failed to write config to: %s (%v) \n\n", saveTo, err)
		return
	}
	fmt.Printf("\n"+Info+"Saved new client config to: %s \n\n", saveTo)
}

func kickPlayerCmd(ctx *grumble.Context) {

}

func listPlayersCmd(ctx *grumble.Context) {

}
