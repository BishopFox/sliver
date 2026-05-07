package aka

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/assets"
)

type AkaAlias struct {
	Name        string   `json:"name"`         // alias Name
	Command     string   `json:"command"`      // command being aliased
	DefaultArgs []string `json:"default_args"` // args passed by the user
	Description string   `json:"description"`
}

var akaAliases = make(map[string]*AkaAlias)

const (
	akaAliasFileName = "aka-aliases.json"
)

func GetAkaAliasesFilePath() string {
	return filepath.Join(assets.GetRootAppDir(), akaAliasFileName)
}

func LoadAkaAliases() error {
	filepath := GetAkaAliasesFilePath()
	data, err := os.ReadFile(filepath)
	if err != nil {
		// if file hasn't been created yet, we fail gracefully
		if os.IsNotExist(err) {
			return nil
		}
		// otherwise, return the actual error; something else went wrong
		return err
	}

	aliases := []AkaAlias{}
	err = json.Unmarshal(data, &aliases)
	if err != nil {
		return err
	}

	akaAliases = make(map[string]*AkaAlias)

	for _, alias := range aliases {
		akaAliases[alias.Name] = &alias
	}

	return nil
}

func SaveAkaAliases() error {
	filepath := GetAkaAliasesFilePath()
	aliases := make([]*AkaAlias, 0, len(akaAliases))
	for _, alias := range akaAliases {
		aliases = append(aliases, alias)
	}

	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0o600)
}
