package cloudflare_test

import (
	"fmt"
	"log"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

var (
	createdAndModifiedTimestamp, _ = time.Parse(time.RFC3339, "2018-08-28T17:26:26Z")
	expiresAtTimestamp, _          = time.Parse(time.RFC3339, "2019-08-28T23:59:59Z")
	expectedRegistrarTransferIn    = cloudflare.RegistrarTransferIn{
		UnlockDomain:      "ok",
		DisablePrivacy:    "ok",
		EnterAuthCode:     "needed",
		ApproveTransfer:   "unknown",
		AcceptFoa:         "needed",
		CanCancelTransfer: true,
	}
	expectedRegistrarContact = cloudflare.RegistrantContact{
		ID:           "ea95132c15732412d22c1476fa83f27a",
		FirstName:    "John",
		LastName:     "Appleseed",
		Organization: "Cloudflare, Inc.",
		Address:      "123 Sesame St.",
		Address2:     "Suite 430",
		City:         "Austin",
		State:        "TX",
		Zip:          "12345",
		Country:      "US",
		Phone:        "+1 123-123-1234",
		Email:        "user@example.com",
		Fax:          "123-867-5309",
	}
	expectedRegistrarDomain = cloudflare.RegistrarDomain{
		ID:                "ea95132c15732412d22c1476fa83f27a",
		Available:         false,
		SupportedTLD:      true,
		CanRegister:       false,
		TransferIn:        expectedRegistrarTransferIn,
		CurrentRegistrar:  "Cloudflare",
		ExpiresAt:         expiresAtTimestamp,
		RegistryStatuses:  "ok,serverTransferProhibited",
		Locked:            false,
		CreatedAt:         createdAndModifiedTimestamp,
		UpdatedAt:         createdAndModifiedTimestamp,
		RegistrantContact: expectedRegistrarContact,
	}
)

func ExampleAPI_RegistrarDomain() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	domain, err := api.RegistrarDomain("01a7362d577a6c3019a474fd6f485823", "cloudflare.com")

	fmt.Printf("%+v\n", domain)
}

func ExampleAPI_RegistrarDomains() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	domains, err := api.RegistrarDomains("01a7362d577a6c3019a474fd6f485823")

	fmt.Printf("%+v\n", domains)
}

func ExampleAPI_TransferRegistrarDomain() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	domain, err := api.TransferRegistrarDomain("01a7362d577a6c3019a474fd6f485823", "cloudflare.com")

	fmt.Printf("%+v\n", domain)
}

func ExampleAPI_CancelRegistrarDomainTransfer() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	domains, err := api.CancelRegistrarDomainTransfer("01a7362d577a6c3019a474fd6f485823", "cloudflare.com")

	fmt.Printf("%+v\n", domains)
}

func ExampleAPI_UpdateRegistrarDomain() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	domain, err := api.UpdateRegistrarDomain(
		"01a7362d577a6c3019a474fd6f485823",
		"cloudflare.com",
		cloudflare.RegistrarDomainConfiguration{
			NameServers: []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
			Locked:      false,
		},
	)

	fmt.Printf("%+v\n", domain)
}
