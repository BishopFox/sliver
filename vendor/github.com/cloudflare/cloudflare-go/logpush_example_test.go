package cloudflare_test

import (
	"fmt"
	"log"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

var exampleNewLogpushJob = cloudflare.LogpushJob{
	Enabled:         false,
	Name:            "example.com",
	LogpullOptions:  "fields=RayID,ClientIP,EdgeStartTimestamp&timestamps=rfc3339",
	DestinationConf: "s3://mybucket/logs?region=us-west-2",
}

var exampleUpdatedLogpushJob = cloudflare.LogpushJob{
	Enabled: true, 
	Name: "updated.com", 
	LogpullOptions:  "fields=RayID,ClientIP,EdgeStartTimestamp",
	DestinationConf: "gs://mybucket/logs",
}

func ExampleAPI_CreateLogpushJob() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	job, err := api.CreateLogpushJob(zoneID, exampleNewLogpushJob)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", job)
}

func ExampleAPI_UpdateLogpushJob() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	err = api.UpdateLogpushJob(zoneID, 1, exampleUpdatedLogpushJob)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleAPI_LogpushJobs() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	jobs, err := api.LogpushJobs(zoneID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", jobs)
	for _, r := range jobs {
		fmt.Printf("%+v\n", r)
	}
}

func ExampleAPI_LogpushJob() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	job, err := api.LogpushJob(zoneID, 1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", job)
}

func ExampleAPI_DeleteLogpushJob() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	err = api.DeleteLogpushJob(zoneID, 1)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleAPI_GetLogpushOwnershipChallenge() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	ownershipChallenge, err := api.GetLogpushOwnershipChallenge(zoneID, "destination_conf")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", ownershipChallenge)
}

func ExampleAPI_ValidateLogpushOwnershipChallenge() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	isValid, err := api.ValidateLogpushOwnershipChallenge(zoneID, "destination_conf", "ownership_challenge")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", isValid)
}

func ExampleAPI_CheckLogpushDestinationExists() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	exists, err := api.CheckLogpushDestinationExists(zoneID, "destination_conf")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", exists)
}
