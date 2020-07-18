package configs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
)

const (
	// APIKeyEnvVar - Name of env variable to pull the cf api key
	APIKeyEnvVar = "CF_API_KEY"
	// APIEmailEnvVar - Name of env variable to pull the cf api email
	APIEmailEnvVar = "CF_API_EMAIL"
)

var (
	cfConfigFileName = "cloudflare.json"
	cfConfigLog      = log.NamedLogger("config", "cloudflare")
)

type cfConfigFile struct {
	APIKey   string `json:"api_key"`
	APIEmail string `json:"api_email"`
}

// GetCloudflareConfigPath - File path to config.json
func GetCloudflareConfigPath() string {
	appDir := assets.GetRootAppDir()
	cfConfigPath := path.Join(appDir, "configs", cfConfigFileName)
	cfConfigLog.Infof("Loading cloudflare config from %s", cfConfigPath)
	return cfConfigPath
}

// CloudflareConfig - Cloudflare configuration values
type CloudflareConfig struct {
	configFile *cfConfigFile
}

// Credentials - Pulls the CF creds from either the local configuration database
//               or from the environment variables.
func (c *CloudflareConfig) Credentials() (string, string) {
	apiKey := c.configFile.APIKey
	if apiKey == "" {
		apiKey = os.Getenv(APIKeyEnvVar)
	}
	apiEmail := c.configFile.APIEmail
	if apiEmail == "" {
		apiEmail = os.Getenv(APIEmailEnvVar)
	}
	if apiKey == "" || apiEmail == "" {
		cfConfigLog.Warn("Failed to find cloudflare credentials in file/env")
	}
	return apiKey, apiEmail
}

// GetCloudflareConfig - Get the cloudflare config
func GetCloudflareConfig() *CloudflareConfig {
	cfConfig := &CloudflareConfig{
		configFile: &cfConfigFile{},
	}
	cfConfigPath := GetCloudflareConfigPath()
	if _, err := os.Stat(cfConfigPath); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(cfConfigPath)
		if err != nil {
			cfConfigLog.Errorf("Failed to read cloudflare config %s", err)
		}
		err = json.Unmarshal(data, cfConfig.configFile)
		if err != nil {
			cfConfigLog.Errorf("Failed to parse cloudflare config %s", err)

		}
	} else {
		cfConfigLog.Warnf("Cloudflare config file does not exist")
	}
	return cfConfig
}
