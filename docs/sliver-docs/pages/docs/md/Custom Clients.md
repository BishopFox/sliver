Although the `sliver-client` is the default way to interact with a `sliver-server` and with implant sessions, there might be a time where you would want to automate some tasks upon reception of certain events.

To do so, one case use [sliver-script](https://github.com/moloch--/sliver-script), [sliver-py](https://github.com/moloch--/sliver-py) or write a custom client in another language. As all the communications between the client and the server are based on gRPC, any language with gRPC support should in theory be used to create a custom client.

## Writing a Go client

In this example, we will focus on writing a custom Go client that executes a new system command on every new implant that connects to the sliver server.

Create a new Go project somewhere on your file system:

```
mkdir sliver-custom-client
cd sliver-custom-client
touch main.go
go mod init github.com/<your-username>/<your-project-name>
go get github.com/bishopfox/sliver
```

The module path (`github.com/<your-username>/<your-project-name>`) can be anything, as long as it respects the [requirements](https://golang.org/ref/mod#go-mod-init).

The next step is to write our client code (`main.go`):

```go
package main

import (
	"context"
	"flag"
	"io"
	"log"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func makeRequest(session *clientpb.Session) *commonpb.Request {
	if session == nil {
		return nil
	}
	timeout := int64(60)
	return &commonpb.Request{
		SessionID: session.ID,
		Timeout:   timeout,
	}
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to sliver client config file")
	flag.Parse()

	// load the client configuration from the filesystem
	config, err := assets.ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	// connect to the server
	rpc, ln, err := transport.MTLSConnect(config)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[*] Connected to sliver server")
	defer ln.Close()

	// Open the event stream to be able to collect all events sent by  the server
	eventStream, err := rpc.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		log.Fatal(err)
	}

	// infinite loop
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}
		// Trigger event based on type
		switch event.EventType {

		// a new session just came in
		case consts.SessionOpenedEvent:
			session := event.Session
			// call any RPC you want, for the full list, see
			// https://github.com/BishopFox/sliver/blob/master/protobuf/rpcpb/services.proto
			resp, err := rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
				Path:    `c:\windows\system32\calc.exe`,
				Output:  false,
				Request: makeRequest(session),
			})
			if err != nil {
				log.Fatal(err)
			}
			// Don't forget to check for errors in the Response object
			if resp.Response != nil && resp.Response.Err != "" {
				log.Fatal(resp.Response.Err)
			}
		}
	}
}
```

Finally, run `go mod tidy` to make sure to have all the external dependencies, and run `go build .` to build the code.

That's it, you wrote your first custom client in Go for sliver.
