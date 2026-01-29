/*
Package http provides an HTTP service. It is used to send notifications to HTTP endpoints. The service is configured
with a list of webhooks. Each webhook contains the URL of the endpoint, the HTTP method to use, the content type of the
HTTP request and a function that builds the payload of the request.

The service also allows to register pre and post send hooks. These hooks are called before and after the request is sent
to the receiver. The pre send hook can be used to modify the request before it is sent. The post send hook can be used
to modify the response after it is received. The hooks are called in the order they are registered.

Usage:

	    package main

	    import (
		    "context"
		    "log"
		    stdhttp "net/http"

		    "github.com/nikoksr/notify"
		    "github.com/nikoksr/notify/service/http"
	    )

	    func main() {
			// Create http service
			httpService := http.New()

			//
			// In the following example, we will send two requests to the same HTTP endpoint. This is meant to be used
			// with Notify's test http server: https://github.com/nikoksr/notify-http-test. It supports multiple
			// content-types on the same endpoint, to simplify testing. All you have to do is run `go run main.go` in
			// the test server's directory and
			// this example will work.
			// So the following example should send two requests to the same endpoint, one with content-type
			// application/json and one with content-type text/plain. The requests should be logged differently by the
			// test server since we provide
			// a custom payload builder func for the second webhook.

			// Add a default webhook; this uses application/json as content type and POST as request method.
			httpService.AddReceiversURLs("http://localhost:8080")

			// Add a custom webhook; the build payload function is used to build the payload that will be sent to the
			// receiver
			// from the given subject and message.
			httpService.AddReceivers(&http.Webhook{
				URL:         "http://localhost:8080",
				Header:      stdhttp.Header{},
				ContentType: "text/plain",
				Method:      stdhttp.MethodPost,
				BuildPayload: func(subject, message string) (payload any) {
					return "[text/plain]: " + subject + " - " + message
				},
			})

			//
			// NOTE: In case of an unsupported content type, we could provide a custom marshaller here.
			// See http.Service.Serializer and http.defaultMarshaller.Marshal for details.
			//

			// Add pre-send hook to log the request before it is sent.
			httpService.PreSend(func(ctx context.Context, req *stdhttp.Request) error {
				log.Printf("Sending request to %s", req.URL)
				return nil
			})

			// Add post-send hook to log the response after it is received.
			httpService.PostSend(func(ctx context.Context, req *stdhttp.Request, resp *stdhttp.Response) error {
				log.Printf("Received response from %s", resp.Request.URL)
				return nil
			})

			// Create the notifier and use the HTTP service
			n := notify.NewWithServices(httpService)

			// Send a test message.
			if err := n.Send(context.Background(), "Testing new features", "Notify's HTTP service is here."); err != nil {
				log.Fatal(err)
			}
		}
*/
package http
