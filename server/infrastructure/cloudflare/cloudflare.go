package cloudflare

import (
	"os"

	"github.com/bishopfox/sliver/server/config"
	"github.com/bishopfox/sliver/server/log"
	"github.com/cloudflare/cloudflare-go"
)

const (
	// APIKeyEnvVar - Name of env variable to pull the cf api key
	APIKeyEnvVar = "CF_API_KEY"
	// APIEmailEnvVar - Name of env variable to pull the cf api email
	APIEmailEnvVar = "CF_API_EMAIL"
)

var (
	cloudflareLog = log.NamedLogger("infrastruture", "cloudflare")
)

// Pulls the CF creds from either the local configuration database
// or from the environment variables.
func getCFCredentials() (string, string) {
	apiKey, err := config.GetConfig(APIKeyEnvVar)
	if err != nil || apiKey == "" {
		cloudflareLog.Warnf("Local config returned %s", err)
		apiKey = os.Getenv(APIKeyEnvVar)
	}
	apiEmail, err := config.GetConfig(APIEmailEnvVar)
	if err != nil || apiEmail == "" {
		cloudflareLog.Warnf("Local config returned %s", err)
		apiEmail = os.Getenv(APIEmailEnvVar)
	}
	if apiKey == "" || apiEmail == "" {
		cloudflareLog.Warn("Failed to find crednetials")
	}
	return apiKey, apiEmail
}

// DNSConfigureForC2 - Confiture a Cloudflare domain for DNS C2
func DNSConfigureForC2(parentDomain string, force bool) error {
	// Construct a new API object
	api, err := cloudflare.New(getCFCredentials())
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}

	// Fetch user details on the account
	u, err := api.UserDetails()
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}
	// Print user details
	cloudflareLog.Info(u)

	// Fetch the zone ID
	id, err := api.ZoneIDByName(parentDomain)
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}

	// Fetch zone details
	zone, err := api.ZoneDetails(id)
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}

	// Print zone details
	cloudflareLog.Info(zone)
	return nil
}
