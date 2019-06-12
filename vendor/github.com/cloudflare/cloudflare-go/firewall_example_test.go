package cloudflare_test

import (
	"fmt"
	"log"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func ExampleAPI_ListZoneAccessRules_all() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch all access rules for a zone
	response, err := api.ListZoneAccessRules(zoneID, cloudflare.AccessRule{}, 1)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range response.Result {
		fmt.Printf("%s: %s\n", r.Configuration.Value, r.Mode)
	}
}

func ExampleAPI_ListZoneAccessRules_filterByIP() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch only access rules whose target is 127.0.0.1
	localhost := cloudflare.AccessRule{
		Configuration: cloudflare.AccessRuleConfiguration{Target: "127.0.0.1"},
	}
	response, err := api.ListZoneAccessRules(zoneID, localhost, 1)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range response.Result {
		fmt.Printf("%s: %s\n", r.Configuration.Value, r.Mode)
	}
}

func ExampleAPI_ListZoneAccessRules_filterByMode() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch access rules with an action of "block"
	foo := cloudflare.AccessRule{
		Mode: "block",
	}
	response, err := api.ListZoneAccessRules(zoneID, foo, 1)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range response.Result {
		fmt.Printf("%s: %s\n", r.Configuration.Value, r.Mode)
	}
}

func ExampleAPI_ListZoneAccessRules_filterByNote() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch only access rules with notes containing "example"
	foo := cloudflare.AccessRule{
		Notes: "example",
	}
	response, err := api.ListZoneAccessRules(zoneID, foo, 1)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range response.Result {
		fmt.Printf("%s: %s\n", r.Configuration.Value, r.Mode)
	}
}
