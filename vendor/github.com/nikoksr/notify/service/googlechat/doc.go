/*
Package googlechat provides message notification integration sent to multiple
spaces within a Google Chat Application.

Disclaimer:
In order to integrate `notify` with a Google Chat Application, an "Application Default
Credentials" file must be supplied.

For more information on Google Application credential JSON files see:
https://cloud.google.com/docs/authentication/application-default-credentials

Usage:

			package main

			import (
			    "context"
			    "fmt"
			    "log"
			    "strings"

			    "github.com/nikoksr/notify"
			    "github.com/nikoksr/notify/service/googlechat"
			    "google.golang.org/api/chat/v1"
			    "google.golang.org/api/option"
			)

			func main() {
			    ctx := context.Background()

			    withCred := option.WithCredentialsFile("credentials.json")
			    withSpacesScope := option.WithScopes("https://www.googleapis.com/auth/chat.spaces")

		        // In this example, we'll send a message to all spaces within the google workspace
		        // Start by using the google chat API to find the spaces within a workspace.

			    listSvc, err := chat.NewService(ctx, withCred, withSpacesScope)
			    spaces, err := listSvc.Spaces.List().Do()

			    if err != nil {
			        log.Fatalf("svc.Spaces.List().Do() failed: %s", err.Error())
			    }

		        // With the the list of spaces, loop over each space creating a receivers slice
		        // of all the space.Name's.
		        receivers := make([]string, 0)

			    for _, space := range spaces.Spaces {
		            fmt.Printf("space %s\n", space.DisplayName)

		             // The googlechat service handles prepending "spaces/" to the name.
		             // Make sure the space.Name does not prepend "spaces/".
			        name := strings.Replace(space.Name, "spaces/", "", 1)

			        sps = append(sps, name)
			    }

			    msgSvc, err := googlechat.New(withCred)

	            // alternatively, if you would like to pass a context
	            // use googlechat.NewWithContext(ctx, ...options)

			    if err != nil {
			        log.Fatalf("googlechat.New() failed: %s", err.Error())
			    }

			    msgSvc.AddReceivers(receivers...)

			    notifier := notify.NewWithServices(msgSvc)

			    fmt.Printf("sending message to %d spaces\n", len(receivers))
			    err = notifier.Send(ctx, "subject", "message")
			    if err != nil {
			        log.Fatalf("notifier.Send() failed: %s", err.Error())
			    }

			    log.Println("notification sent")
			}
*/
package googlechat
