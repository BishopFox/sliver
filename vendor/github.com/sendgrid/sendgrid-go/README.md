![Twilio SendGrid Logo](twilio_sendgrid_logo.png)

[![Test and Deploy](https://github.com/sendgrid/sendgrid-go/actions/workflows/test-and-deploy.yml/badge.svg)](https://github.com/sendgrid/sendgrid-go/actions/workflows/test-and-deploy.yml)
[![GoDoc](https://godoc.org/github.com/sendgrid/sendgrid-go?status.svg)](https://godoc.org/github.com/sendgrid/sendgrid-go)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Twitter Follow](https://img.shields.io/twitter/follow/sendgrid.svg?style=social&label=Follow)](https://twitter.com/sendgrid)
[![GitHub contributors](https://img.shields.io/github/contributors/sendgrid/sendgrid-go.svg)](https://github.com/sendgrid/sendgrid-go/graphs/contributors)
[![Open Source Helpers](https://www.codetriage.com/sendgrid/sendgrid-go/badges/users.svg)](https://www.codetriage.com/sendgrid/sendgrid-go)

**This library allows you to quickly and easily use the Twilio SendGrid Web API v3 via Go.**

Version 3.X.X of this library provides full support for all Twilio SendGrid [Web API v3](https://sendgrid.com/docs/API_Reference/Web_API_v3/index.html) endpoints, including the new [v3 /mail/send](https://sendgrid.com/blog/introducing-v3mailsend-sendgrids-new-mail-endpoint).

This library represents the beginning of a new path for Twilio SendGrid. We want this library to be community driven and Twilio SendGrid led. We need your help to realize this goal. To help make sure we are building the right things in the right order, we ask that you create [issues](https://github.com/sendgrid/sendgrid-go/issues) and [pull requests](CONTRIBUTING.md) or simply upvote or comment on existing issues or pull requests.

**If you need help using SendGrid, please check the [Twilio SendGrid Support Help Center](https://support.sendgrid.com).**

# Table of Contents

* [Installation](#installation)
* [Quick Start](#quick-start)
* [Processing Inbound Email](#inbound)
* [Usage](#usage)
* [Use Cases](#use-cases)
* [Announcements](#announcements)
* [How to Contribute](#contribute)
* [Troubleshooting](#troubleshooting)
* [About](#about)
* [License](#license)

<a name="installation"></a>
# Installation

## Supported Versions

This library supports the following Go implementations:

* Go 1.14
* Go 1.15
* Go 1.16
* Go 1.17
* Go 1.18
* Go 1.19

## Prerequisites

- The Twilio SendGrid service, starting at the [free level](https://sendgrid.com/free?source=sendgrid-go), to send up to 40,000 emails for the first 30 days, then send 100 emails/day free forever or check out [our pricing](https://sendgrid.com/pricing?source=sendgrid-go).

## Setup Environment Variables

Update the development environment with your [SENDGRID_API_KEY](https://app.sendgrid.com/settings/api_keys), for example:

```bash
echo "export SENDGRID_API_KEY='YOUR_API_KEY'" > sendgrid.env
echo "sendgrid.env" >> .gitignore
source ./sendgrid.env
```

## Install Package

`go get github.com/sendgrid/sendgrid-go`

## Dependencies

- [rest](https://github.com/sendgrid/rest)

## Setup Environment Variables

### Initial Setup

```bash
cp .env_sample .env
```

### Environment Variable

Update the development environment with your [SENDGRID_API_KEY](https://app.sendgrid.com/settings/api_keys), for example:

```bash
echo "export SENDGRID_API_KEY='YOUR_API_KEY'" > sendgrid.env
echo "sendgrid.env" >> .gitignore
source ./sendgrid.env
```

<a name="quick-start"></a>
# Quick Start

## Hello Email

The following is the minimum needed code to send an email with the [/mail/send Helper](helpers/mail) ([here](examples/helpers/mail/example.go#L32) is a full example):

### With Mail Helper Class

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func main() {
	from := mail.NewEmail("Example User", "test@example.com")
	subject := "Sending with Twilio SendGrid is Fun"
	to := mail.NewEmail("Example User", "test@example.com")
	plainTextContent := "and easy to do anywhere, even with Go"
	htmlContent := "<strong>and easy to do anywhere, even with Go</strong>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```

The `NewEmail` constructor creates a [personalization object](https://sendgrid.com/docs/Classroom/Send/v3_Mail_Send/personalizations.html) for you. [Here](examples/helpers/mail/example.go#L28) is an example of how to add to it.

### Without Mail Helper Class

The following is the minimum needed code to send an email without the /mail/send Helper ([here](examples/mail/mail.go#L47) is a full example):

```go
package main

import (
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"log"
	"os"
)

func main() {
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = []byte(` {
	"personalizations": [
		{
			"to": [
				{
					"email": "test@example.com"
				}
			],
			"subject": "Sending with Twilio SendGrid is Fun"
		}
	],
	"from": {
		"email": "test@example.com"
	},
	"content": [
		{
			"type": "text/plain",
			"value": "and easy to do anywhere, even with Go"
		}
	]
}`)
	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```

## General v3 Web API Usage

```go
package main

import (
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"log"
	"os"
)

func main() {
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/api_keys", "https://api.sendgrid.com")
	request.Method = "GET"

	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```


<a name="inbound"></a>
# Processing Inbound Email

Please see [our helper](helpers/inbound) for utilizing our Inbound Parse webhook.

<a name="usage"></a>
# Usage

- [Twilio SendGrid Docs](https://sendgrid.com/docs/API_Reference/index.html)
- [Library Usage Docs](USAGE.md)
- [Example Code](examples)
- [How-to: Migration from v2 to v3](https://sendgrid.com/docs/Classroom/Send/v3_Mail_Send/how_to_migrate_from_v2_to_v3_mail_send.html)
- [v3 Web API Mail Send Helper](helpers/mail/README.md)

<a name="use-cases"></a>
# Use Cases

[Examples of common API use cases](use-cases/README.md), such as how to send an email with a transactional template.

<a name="announcements"></a>
# Announcements

All updates to this library are documented in our [CHANGELOG](CHANGELOG.md) and [releases](https://github.com/sendgrid/sendgrid-go/releases).

<a name="contribute"></a>
# How to Contribute

We encourage contribution to our libraries (you might even score some nifty swag), please see our [CONTRIBUTING](CONTRIBUTING.md) guide for details.

Quick links:

- [Feature Request](CONTRIBUTING.md#feature-request)
- [Bug Reports](CONTRIBUTING.md#submit-a-bug-report)
- [Improvements to the Codebase](CONTRIBUTING.md#improvements-to-the-codebase)
- [Review Pull Requests](CONTRIBUTING.md#code-reviews)

<a name="troubleshooting"></a>
# Troubleshooting

Please see our [troubleshooting guide](TROUBLESHOOTING.md) for common library issues.

<a name="about"></a>
# About

sendgrid-go is maintained and funded by Twilio SendGrid, Inc. The names and logos for sendgrid-go are trademarks of Twilio SendGrid, Inc.

<a name="support"><a>
# Support

If you need help using SendGrid, please check the [Twilio SendGrid Support Help Center](https://support.sendgrid.com).

# License
[The MIT License (MIT)](LICENSE)
