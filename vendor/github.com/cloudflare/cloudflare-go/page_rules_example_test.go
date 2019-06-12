package cloudflare_test

import (
	"fmt"
	"log"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

var exampleNewPageRule = cloudflare.PageRule{
	Actions: []cloudflare.PageRuleAction{
		{
			ID:    "always_online",
			Value: "on",
		},
		{
			ID:    "ssl",
			Value: "flexible",
		},
	},
	Targets: []cloudflare.PageRuleTarget{
		{
			Target: "url",
			Constraint: struct {
				Operator string "json:\"operator\""
				Value    string "json:\"value\""
			}{Operator: "matches", Value: fmt.Sprintf("example.%s", domain)},
		},
	},
	Priority: 1,
	Status:   "active",
}

func ExampleAPI_CreatePageRule() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	pageRule, err := api.CreatePageRule(zoneID, exampleNewPageRule)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", pageRule)
}

func ExampleAPI_ListPageRules() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	pageRules, err := api.ListPageRules(zoneID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", pageRules)
	for _, r := range pageRules {
		fmt.Printf("%+v\n", r)
	}
}

func ExampleAPI_PageRule() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	pageRules, err := api.PageRule(zoneID, "my_page_rule_id")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", pageRules)
}

func ExampleAPI_DeletePageRule() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	err = api.DeletePageRule(zoneID, "my_page_rule_id")
	if err != nil {
		log.Fatal(err)
	}
}
