# plivo-go

[![Build, Unit Tests, Linters Status](https://github.com/plivo/plivo-go/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/plivo/plivo-go/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/plivo/plivo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/plivo/plivo-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/plivo/plivo-go)](https://goreportcard.com/report/github.com/plivo/plivo-go)
[![GoDoc](https://godoc.org/github.com/plivo/plivo-go?status.svg)](https://godoc.org/github.com/plivo/plivo-go)

The Plivo Go SDK makes it simpler to integrate communications into your Go applications using the Plivo REST API. Using the SDK, you will be able to make voice calls, send messages and generate Plivo XML to control your call flows.

## Prerequisites

- Go >= 1.13.x

## Getting started

The steps described below uses go modules.

###### Create a new project (optional)
```sh
$ mkdir ~/helloplivo
$ cd ~/helloplivo
$ go mod init helloplivo
```

This will generate a `go.mod` and `go.sum` file.

###### Add plivo-go as a dependency to your project
```sh
$ go get github.com/plivo/plivo-go/v7
```

### Authentication
To make the API requests, you need to create a `Client` and provide it with authentication credentials (which can be found at [https://manage.plivo.com/dashboard/](https://manage.plivo.com/dashboard/)).

We recommend that you store your credentials in the `PLIVO_AUTH_ID` and the `PLIVO_AUTH_TOKEN` environment variables, so as to avoid the possibility of accidentally committing them to source control. If you do this, you can initialise the client with no arguments and it will automatically fetch them from the environment variables:

```go
package main

import "github.com/plivo/plivo-go/v7"

func main()  {
	client, err := plivo.NewClient("", "", &plivo.ClientOptions{})
	if err != nil {
		panic(err)
	}
}
```
Alternatively, you can specifiy the authentication credentials while initializing the `Client`.

```go
package main

import "github.com/plivo/plivo-go/v7"

func main()  {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		panic(err)
	}
}
```

## The Basics
The SDK uses consistent interfaces to create, retrieve, update, delete and list resources. The pattern followed is as follows:

```go
client.Resources.Create(Params{}) // Create
client.Resources.Get(Id) // Get
client.Resources.Update(Id, Params{}) // Update
client.Resources.Delete(Id) // Delete
client.Resources.List() // List all resources, max 20 at a time
```

Using `client.Resources.List()` would list the first 20 resources by default (which is the first page, with `limit` as 20, and `offset` as 0). To get more, you will have to use `limit` and `offset` to get the second page of resources.

## Examples

### Send a message

```go
package main

import "github.com/plivo/plivo-go/v7"

func main() {
	client, err := plivo.NewClient("", "", &plivo.ClientOptions{})
	if err != nil {
		panic(err)
	}
	client.Messages.Create(plivo.MessageCreateParams{
		Src:  "the_source_number",
		Dst:  "the_destination_number",
		Text: "Hello, world!",
	})
}
```

### Make a call

```go
package main

import "github.com/plivo/plivo-go/v7"

func main() {
	client, err := plivo.NewClient("", "", &plivo.ClientOptions{})
	if err != nil {
		panic(err)
	}
	client.Calls.Create(plivo.CallCreateParams{
		From:      "the_source_number",
		To:        "the_destination_number",
		AnswerURL: "http://answer.url",
	})
}

```

### Lookup a number

```go
package main

import (
	"fmt"
	"log"

	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		log.Fatalf("plivo.NewClient() failed: %s", err.Error())
	}

	resp, err := client.Lookup.Get("<insert-number-here>", plivo.LookupParams{})
	if err != nil {
		if respErr, ok := err.(*plivo.LookupError); ok {
			fmt.Printf("API ID: %s\nError Code: %d\nMessage: %s\n",
				respErr.ApiID, respErr.ErrorCode, respErr.Message)
			return
		}
		log.Fatalf("client.Lookup.Get() failed: %s", err.Error())
	}

	fmt.Printf("%+v\n", resp)
}
```

### Generate Plivo XML

```go
package main

import "github.com/plivo/plivo-go/v7/xml"

func main() {
	println(xml.ResponseElement{
		Contents: []interface{}{
			new(xml.SpeakElement).SetContents("Hello, world!"),
		},
	}.String())
}
```

This generates the following XML:

```xml
<Response>
  <Speak>Hello, world!</Speak>
</Response>
```

### Run a PHLO

```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

// Initialize the following params with corresponding values to trigger resources

const authId = "auth_id"
const authToken = "auth_token"
const phloId = "phlo_id"

// with payload in request

func main() {
	testPhloRunWithParams()
}

func testPhloRunWithParams() {
	phloClient, err := plivo.NewPhloClient(authId, authToken, &plivo.ClientOptions{})
	if err != nil {
		panic(err)
	}
	phloGet, err := phloClient.Phlos.Get(phloId)
	if err != nil {
		panic(err)
	}
	//pass corresponding from and to values
	type params map[string]interface{}
	response, err := phloGet.Run(params{
		"from": "111111111",
		"to":   "2222222222",
	})

	if err != nil {
		println(err)
	}
	fmt.Printf("Response: %#v\n", response)
}
```


## WhatsApp Messaging
Plivo's WhatsApp API allows you to send different types of messages over WhatsApp, including templated messages, free form messages and interactive messages. Below are some examples on how to use the Plivo Go SDK to send these types of messages.

### Templated Messages
Templated messages are a crucial to your WhatsApp messaging experience, as businesses can only initiate WhatsApp conversation with their customers using templated messages.

WhatsApp templates support 4 components:  `header` ,  `body`,  `footer`  and `button`. At the point of sending messages, the template object you see in the code acts as a way to pass the dynamic values within these components.  `header`  can accomodate `text` or `media` (images, video, documents) content.  `body`  can accomodate text content.  `button`  can support dynamic values in a `url` button or to specify a developer-defined payload which will be returned when the WhatsApp user clicks on the `quick_reply` button. `footer`  cannot have any dynamic variables.

Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create a WhatsApp template
	template, err := plivo.CreateWhatsappTemplate(`{ 
            "name": "sample_purchase_feedback",
            "language": "en_US",
            "components": [
                {
                    "type": "header",
                    "parameters": [
                        {
                            "type": "media",
                            "media": "https://xyz.com/img.jpg"
                        }
                    ]
                },
                {
                    "type": "body",
                    "parameters": [
                        {
                            "type": "text",
                            "text": "Water Purifier"
                        }
                    ]
                }
            ]
          }`)
	if err != nil {
		fmt.Println("Error creating template:", err)
		return
	}

	// Send a templated message
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:     "source_number",
		Dst:     "destination_number",
		Type:    "whatsapp",
		Template: &template,
	})
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

### Free Form Messages
Non-templated or Free Form WhatsApp messages can be sent as a reply to a user-initiated conversation (Service conversation) or if there is an existing ongoing conversation created previously by sending a templated WhatsApp message.

#### Free Form Text Message
Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Send a free form message
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:  "source_number",
		Dst:  "destination_number",
		Text: "Hello! How can I help you today?",
		Type: "whatsapp",
	})
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

#### Free Form Media Message
Example:
```go
package main

import (
        "fmt"
        "github.com/plivo/plivo-go/v7"
)

func main() {
        client, err := plivo.NewClient("<auth_id>", "<auth_token>", &plivo.ClientOptions{})
        if err != nil {
			fmt.Print("Error", err.Error())
			return
        }
        response, err := client.Messages.Create(plivo.MessageCreateParams{
			Src:"source_number",
			Dst:"destination_number",
			Type:"whatsapp", 
			Text:"Hello, this is sample text",
			MediaUrls:[]string{"https://sample-videos.com/img/Sample-png-image-1mb.png"},
			URL: "https://foo.com/whatsapp_status/",
		})
         if err != nil {
			fmt.Print("Error sending message:", err.Error())
			return
        }
        fmt.Printf("Response: %#v\n", response)
}
```

### Interactive Messages
This guide shows how to send non-templated interactive messages to recipients using Plivo’s APIs.

#### Quick Reply Buttons
Quick reply buttons allow customers to quickly respond to your message with predefined options.

Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create quick reply buttons
	interactive, err := plivo.CreateWhatsappInteractive(`{
		"type": "button",
		"body": {
			"text": "Would you like to proceed?"
		},
		"action": {
			"buttons": [
				{
					"title": "Yes",
					"id": "yes"
				},
				{
					"title": "No",
					"id": "no"
				}
			]
		}
	}`)
	if err != nil {
		fmt.Println("Error creating interactive buttons:", err)
		return
	}

	// Send interactive message with quick reply buttons
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:       "source_number",
		Dst:       "destination_number",
		Type:      "whatsapp",
		Interactive: &interactive,
	})
	if err != nil {
		fmt.Println("Error sending interactive message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

#### Interactive Lists
Interactive lists allow you to present customers with a list of options.

Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create an interactive list
	interactive, err := plivo.CreateWhatsappInteractive(`{
		"type": "list",
		"header": {
			"type": "text",
			"text": "Select an option"
		},
		"body": {
			"text": "Choose from the following options:"
		},
		"action": {
			"sections": [
				{
					"title": "Options",
					"rows": [
						{
							"id": "option1",
							"title": "Option 1",
							"description": "Description of option 1"
						},
						{
							"id": "option2",
							"title": "Option 2",
							"description": "Description of option 2"
						}
					]
				}
			]
		}
	}`)
	if err != nil {
		fmt.Println("Error creating interactive list:", err)
		return
	}

	// Send interactive message with list
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:       "source_number",
		Dst:       "destination_number",
		Type:      "whatsapp",
		Interactive: &interactive,
	})
	if err != nil {
		fmt.Println("Error sending interactive message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

#### Interactive CTA URLs
CTA URL messages allow you to send links and call-to-action buttons.

Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create a CTA URL message
	interactive, err := plivo.CreateWhatsappInteractive(`{
		"type": "cta_url",
		"header": {
			"type": "media",
			"media": "https://example.com/image.jpg"
		},
		"body": {
			"text": "Check out this link!"
		},
		"footer": {
			"text": "Footer text"
		},
		"action": {
			"buttons": [
				{
					"title": "Visit Website",
					"url": "https://example.com"
				}
			]
		}
	}`)
	if err != nil {
		fmt.Println("Error creating CTA URL:", err)
		return
	}

	// Send interactive message with CTA URL
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:       "source_number",
		Dst:       "destination_number",
		Type:      "whatsapp",
		Interactive: &interactive,
	})
	if err != nil {
		fmt.Println("Error sending interactive message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

### Location Messages
This guide shows how to send templated and non-templated location messages to recipients using Plivo’s APIs.

#### Templated Location Messages
Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create a WhatsApp template
	template, err := plivo.CreateWhatsappTemplate(`{
        "name": "plivo_order_pickup",
        "language": "en_US",
        "components": [
            {
                "type": "header",
                "parameters": [
                    {
                        "type": "location",
                        "location": {
                            "latitude": "37.483307",
                            "longitude": "122.148981",
                            "name": "Pablo Morales",
                            "address": "1 Hacker Way, Menlo Park, CA 94025"
                        }
                    }
                ]
            }
        ]
    }`)
	if err != nil {
		fmt.Println("Error creating template:", err)
		return
	}

	// Send a templated message
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:     "source_number",
		Dst:     "destination_number",
		Type:    "whatsapp",
		Template: &template,
	})
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

#### Non-Templated Location Messages
Example:
```go
package main

import (
	"fmt"
	"github.com/plivo/plivo-go/v7"
)

func main() {
	client, err := plivo.NewClient("<auth-id>", "<auth-token>", &plivo.ClientOptions{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create a WhatsApp location object
	location, err := plivo.CreateWhatsappLocation(`{
        "latitude": "37.483307",
        "longitude": "122.148981",
        "name": "Pablo Morales",
        "address": "1 Hacker Way, Menlo Park, CA 94025"
    }`)
	if err != nil {
		fmt.Println("Error creating location:", err)
		return
	}

	// Send a templated message
	response, err := client.Messages.Create(plivo.MessageCreateParams{
		Src:     "source_number",
		Dst:     "destination_number",
		Type:    "whatsapp",
		Location: &location,
	})
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Response: %#v\n", response)
}
```

### More examples
Refer to the [Plivo API Reference](https://www.plivo.com/docs/sms/api/overview/) for more examples.

## Local Development
> Note: Requires latest versions of Docker & Docker-Compose. If you're on MacOS, ensure Docker Desktop is running.
1. Export the following environment variables in your host machine:
```bash
export PLIVO_AUTH_ID=<your_auth_id>
export PLIVO_AUTH_TOKEN=<your_auth_token>
export PLIVO_API_DEV_HOST=<plivoapi_dev_endpoint>
export PLIVO_API_PROD_HOST=<plivoapi_public_endpoint>
```
2. Run `make build`. This will create a docker container in which the sdk will be setup and dependencies will be installed.
> The entrypoint of the docker container will be the `setup_sdk.sh` script. The script will handle all the necessary changes required for local development.
3. The above command will print the docker container id (and instructions to connect to it) to stdout.
4. The testing code can be added to `<sdk_dir_path>/go-sdk-test/test.go` in host  
 (or `/usr/src/app/go-sdk-test/test.go` in container)
5. The sdk directory will be mounted as a volume in the container. So any changes in the sdk code will also be reflected inside the container.
6. To run test code, run `make run CONTAINER=<cont_id>` in host.
7. To run unit tests, run `make test CONTAINER=<cont_id>` in host.
> `<cont_id>` is the docker container id created in 2.   
(The docker container should be running)

> Test code and unit tests can also be run within the container using
`make run` and `make test` respectively. (`CONTAINER` argument should be omitted when running from the container)