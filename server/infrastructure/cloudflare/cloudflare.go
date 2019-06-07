package cloudflare

import (
	"fmt"
	"os"

	"github.com/bishopfox/sliver/server/config"
	"github.com/bishopfox/sliver/server/log"
	"github.com/cloudflare/cloudflare-go"
)

var (
	cloudflareLog = log.NamedLogger("infrastruture", "cloudflare")
)

func getCFCredentials() (string, string) {
	apiKey, err := config.GetConfig("CF_API_KEY")
	if err != nil || apiKey == "" {
		apiKey = os.Getenv("CF_API_KEY")
	}
	apiEmail, err := config.GetConfig("CF_API_EMAIL")
	if err != nil || apiEmail == "" {
		apiEmail = os.Getenv("CF_API_EMAIL")
	}
	return apiKey, apiEmail
}

// DNSConfigureForC2 - Confiture a Cloudflare domain for DNS C2
func DNSConfigureForC2(parentDomain string) error {
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
	fmt.Println(u)

	// Fetch the zone ID
	id, err := api.ZoneIDByName(parentDomain) // Assuming example.com exists in your Cloudflare account already
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
