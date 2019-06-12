package cloudflare_test

import (
	"fmt"
	"log"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func ExampleAPI_DNSRecords_all() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch all records for a zone
	recs, err := api.DNSRecords(zoneID, cloudflare.DNSRecord{})
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range recs {
		fmt.Printf("%s: %s\n", r.Name, r.Content)
	}
}

func ExampleAPI_DNSRecords_filterByContent() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch only records whose content is 127.0.0.1
	localhost := cloudflare.DNSRecord{Content: "127.0.0.1"}
	recs, err := api.DNSRecords(zoneID, localhost)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range recs {
		fmt.Printf("%s: %s\n", r.Name, r.Content)
	}
}

func ExampleAPI_DNSRecords_filterByName() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch records of any type with name "foo.example.com"
	// The name must be fully-qualified
	foo := cloudflare.DNSRecord{Name: "foo.example.com"}
	recs, err := api.DNSRecords(zoneID, foo)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range recs {
		fmt.Printf("%s: %s\n", r.Name, r.Content)
	}
}

func ExampleAPI_DNSRecords_filterByType() {
	api, err := cloudflare.New("deadbeef", "test@example.org")
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName("example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch only AAAA type records
	aaaa := cloudflare.DNSRecord{Type: "AAAA"}
	recs, err := api.DNSRecords(zoneID, aaaa)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range recs {
		fmt.Printf("%s: %s\n", r.Name, r.Content)
	}
}
