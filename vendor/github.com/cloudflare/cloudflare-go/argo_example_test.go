package cloudflare_test

import (
	"fmt"
	"log"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func ExampleAPI_ArgoSmartRouting() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	smartRoutingSettings, err := api.ArgoSmartRouting("01a7362d577a6c3019a474fd6f485823")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("smart routing is %s", smartRoutingSettings.Value)
}

func ExampleAPI_ArgoTieredCaching() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	tieredCachingSettings, err := api.ArgoTieredCaching("01a7362d577a6c3019a474fd6f485823")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tiered caching is %s", tieredCachingSettings.Value)
}

func ExampleAPI_UpdateArgoSmartRouting() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	smartRoutingSettings, err := api.UpdateArgoSmartRouting("01a7362d577a6c3019a474fd6f485823", "on")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("smart routing is %s", smartRoutingSettings.Value)
}

func ExampleAPI_UpdateArgoTieredCaching() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	tieredCachingSettings, err := api.UpdateArgoTieredCaching("01a7362d577a6c3019a474fd6f485823", "on")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tiered caching is %s", tieredCachingSettings.Value)
}
