package cloudflare_test

import (
	"fmt"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"log"
)

var (
	workerScript = "addEventListener('fetch', event => {\n    event.passThroughOnException()\nevent.respondWith(handleRequest(event.request))\n})\n\nasync function handleRequest(request) {\n    return fetch(request)\n}"
)

func ExampleAPI_UploadWorker() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	res, err := api.UploadWorker(&cloudflare.WorkerRequestParams{ZoneID: zoneID}, workerScript)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)

	UploadWorkerWithName()
}

func UploadWorkerWithName() {
	api, err := cloudflare.New(apiKey, user, cloudflare.UsingOrganization("foo"))
	if err != nil {
		log.Fatal(err)
	}

	res, err := api.UploadWorker(&cloudflare.WorkerRequestParams{ScriptName: "baz"}, workerScript)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}

func ExampleAPI_DownloadWorker() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}

	res, err := api.DownloadWorker(&cloudflare.WorkerRequestParams{ZoneID: zoneID})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)

	DownloadWorkerWithName()
}

func DownloadWorkerWithName() {
	api, err := cloudflare.New(apiKey, user, cloudflare.UsingOrganization("foo"))
	if err != nil {
		log.Fatal(err)
	}

	res, err := api.DownloadWorker(&cloudflare.WorkerRequestParams{ScriptName: "baz"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}

func ExampleAPI_DeleteWorker() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}
	res, err := api.DeleteWorker(&cloudflare.WorkerRequestParams{ZoneID: zoneID})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)

	DeleteWorkerWithName()
}

func DeleteWorkerWithName() {
	api, err := cloudflare.New(apiKey, user, cloudflare.UsingOrganization("foo"))
	if err != nil {
		log.Fatal(err)
	}

	res, err := api.DeleteWorker(&cloudflare.WorkerRequestParams{ScriptName: "baz"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}

func ExampleAPI_ListWorkerScripts() {
	api, err := cloudflare.New(apiKey, user, cloudflare.UsingOrganization("foo"))
	if err != nil {
		log.Fatal(err)
	}

	res, err := api.ListWorkerScripts()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res.WorkerList)
}

func ExampleAPI_CreateWorkerRoute() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}
	route := cloudflare.WorkerRoute{Pattern: "app1.example.com/*", Enabled: true}
	res, err := api.CreateWorkerRoute(zoneID, route)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}

func ExampleAPI_UpdateWorkerRoute() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}
	// pull from existing list of routes to perform update on
	routesResponse, err := api.ListWorkerRoutes(zoneID)
	if err != nil {
		log.Fatal(err)
	}
	route := cloudflare.WorkerRoute{Pattern: "app2.example.com/*", Enabled: true}
	// update first route retrieved from the listWorkerRoutes call with details above
	res, err := api.UpdateWorkerRoute(zoneID, routesResponse.Routes[0].ID, route)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}

func ExampleAPI_ListWorkerRoutes() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}
	res, err := api.ListWorkerRoutes(zoneID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}

func ExampleAPI_DeleteWorkerRoute() {
	api, err := cloudflare.New(apiKey, user)
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(domain)
	if err != nil {
		log.Fatal(err)
	}
	// pull from existing list of routes to perform delete on
	routesResponse, err := api.ListWorkerRoutes(zoneID)
	if err != nil {
		log.Fatal(err)
	}
	// delete first route retrieved from the listWorkerRoutes call
	res, err := api.DeleteWorkerRoute(zoneID, routesResponse.Routes[0].ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v", res)
}
