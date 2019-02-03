package console

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sliver/server/assets"
	"sliver/server/certs"

	"github.com/desertbit/grumble"
)

// ClientConfig - Client JSON config
type ClientConfig struct {
	Operator   string `json:"operator"`
	LHost      string `json:"lhost"`
	LPort      int    `json:"lport"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func newPlayerCmd(ctx *grumble.Context) {
	operator := ctx.Flags.String("operator")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	save := ctx.Flags.String("save")

	if operator == "" {
		fmt.Printf("\n" + Warn + "Operator name required (--operator) \n\n")
		return
	}

	if lhost == "" {
		fmt.Printf("\n"+Warn+"Invalid lhost '%s' \n\n", lhost)
		return
	}

	fmt.Printf("\n" + Info + "Generating new client certificate, please wait ... \n")
	rootDir := assets.GetRootAppDir()
	publicKey, privateKey := certs.GenerateClientCertificate(rootDir, operator, true)
	if save == "" {
		fmt.Printf("\n%s\n", privateKey)
	} else {
		config := ClientConfig{
			Operator:   operator,
			LHost:      lhost,
			LPort:      lport,
			PrivateKey: string(privateKey),
			PublicKey:  string(publicKey),
		}
		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if err != nil {
			fmt.Printf(Warn+"Failed to generate sliver %v\n\n", err)
			return
		}
		if fi.IsDir() {
			filename := fmt.Sprintf("%s.cfg", filepath.Base(operator))
			saveTo = filepath.Join(saveTo, filename)
		}
		configJSON, _ := json.Marshal(config)
		err = ioutil.WriteFile(saveTo, configJSON, 0644)
		if err != nil {
			fmt.Printf("\n"+Warn+"Failed to write config to: %s (%v) \n\n", saveTo, err)
			return
		}
		fmt.Printf("\n"+Info+"Saved new client config to: %s \n\n", saveTo)
	}

}

func kickPlayerCmd(ctx *grumble.Context) {

}

func listPlayersCmd(ctx *grumble.Context) {

}
