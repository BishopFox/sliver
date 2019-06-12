package cloudflare_test

import (
	"fmt"
	"log"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func ExampleAPI_ListUserAgentRules_all() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch all Zone Lockdown rules for a zone, by page.
	rules, err := api.ListUserAgentRules(zoneID, 1)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range rules.Result {
		fmt.Printf("%s: %s\n", r.Configuration.Target, r.Configuration.Value)
	}
}
