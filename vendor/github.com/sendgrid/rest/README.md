![SendGrid Logo](twilio_sendgrid_logo.png)

[![BuildStatus](https://github.com/sendgrid/rest/actions/workflows/test.yml/badge.svg)](https://github.com/sendgrid/rest/actions/workflows/test.yml)
[![GoDoc](https://godoc.org/github.com/sendgrid/rest?status.png)](http://godoc.org/github.com/sendgrid/rest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sendgrid/rest)](https://goreportcard.com/report/github.com/sendgrid/rest)
[![Twitter Follow](https://img.shields.io/twitter/follow/sendgrid.svg?style=social&label=Follow)](https://twitter.com/sendgrid)
[![GitHub contributors](https://img.shields.io/github/contributors/sendgrid/rest.svg)](https://github.com/sendgrid/rest/graphs/contributors)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Quickly and easily access any RESTful or RESTful-like API.**

If you are looking for the SendGrid API client library, please see [this repo](https://github.com/sendgrid/sendgrid-go).

# Announcements
**The default branch name for this repository has been changed to `main` as of 07/27/2020.**

All updates to this library is documented in our [CHANGELOG](CHANGELOG.md).

# Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [How to Contribute](#contribute)
- [About](#about)
- [License](#license)

<a name="installation"></a>
# Installation

## Supported Versions

This library supports the following Go implementations:

* Go 1.14
* Go 1.15
* Go 1.16
* Go 1.17

## Install Package

```bash
go get github.com/sendgrid/rest
```

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

## With Docker

A Docker image has been created to allow you to get started with `rest` right away.

```bash
docker-compose up -d --build

# Ensure the container is running with 'docker ps'
docker ps
CONTAINER ID        IMAGE               COMMAND               CREATED              STATUS              PORTS               NAMES
40c8d984a620        rest_go             "tail -f /dev/null"   About a minute ago   Up About a minute                       rest_go_1
```

With the container running, you can execute your local `go` scripts using the following:

```bash
# docker exec <container_name> <go command>
docker exec rest_go_1 go run docker/example.go
200
{
  "args": {},
  "headers": {
    "Accept-Encoding": "gzip",
    "Connection": "close",
    "Host": "httpbin.org",
    "User-Agent": "Go-http-client/1.1"
  },
  "origin": "86.180.177.202",
  "url": "https://httpbin.org/get"
}

map[Access-Control-Allow-Origin:[*] Access-Control-Allow-Credentials:[true] Via:[1.1 vegur] Connection:[keep-alive] Server:[gunicorn/19.9.0] Date:[Tue, 02 Oct 2018 18:20:43 GMT] Content-Type:[application/json] Content-Length:[233]]

# You can install libraries too, using the same command
# NOTE: Any libraries installed will be removed when the container is stopped.
docker exec rest_go_1 go get github.com/uniplaces/carbon
```

Your go files will be executed relative to the root of this directory. So in the example above, to execute the `example.go` file within the `docker` directory, we run `docker exec rest_go_1 go run docker/example.go`. If this file was in the root of this repository (next to README.exe, rest.go etc.), you would run `docker exec rest_go_1 go run my_go_script.go`

<a name="quick-start"></a>
# Quick Start

`GET /your/api/{param}/call`

```go
package main

import "github.com/sendgrid/rest"
import "fmt"

func main() {
	const host = "https://api.example.com"
	param := "myparam"
	endpoint := "/your/api/" + param + "/call"
	baseURL := host + endpoint
	method := rest.Get
	request := rest.Request{
		Method:  method,
		BaseURL: baseURL,
	}
	response, err := rest.Send(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```

`POST /your/api/{param}/call` with headers, query parameters and a request body.

```go
package main

import "github.com/sendgrid/rest"
import "fmt"

func main() {
	const host = "https://api.example.com"
	param := "myparam"
	endpoint := "/your/api/" + param + "/call"
	baseURL := host + endpoint
	Headers := make(map[string]string)
	key := os.Getenv("API_KEY")
	Headers["Authorization"] = "Bearer " + key
	Headers["X-Test"] = "Test"
	var Body = []byte(`{"some": 0, "awesome": 1, "data": 3}`)
	queryParams := make(map[string]string)
	queryParams["hello"] = "0"
	queryParams["world"] = "1"
	method := rest.Post
	request = rest.Request{
		Method:      method,
		BaseURL:     baseURL,
		Headers:     Headers,
		QueryParams: queryParams,
		Body:        Body,
	}
	response, err := rest.Send(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```

<a name="usage"></a>
# Usage

- [Usage Examples](USAGE.md)

<a name="contribute"></a>
# How to Contribute

We encourage contribution to our projects, please see our [CONTRIBUTING](CONTRIBUTING.md) guide for details.

Quick links:

- [Feature Request](CONTRIBUTING.md#feature-request)
- [Bug Reports](CONTRIBUTING.md#submit-a-bug-report)
- [Improvements to the Codebase](CONTRIBUTING.md#improvements-to-the-codebase)
- [Code Reviews](CONTRIBUTING.md#code-reviews)

<a name="about"></a>
# About

rest is maintained and funded by Twilio SendGrid, Inc. The names and logos for rest are trademarks of Twilio SendGrid, Inc.

If you need help installing or using the library, please check the [Twilio SendGrid Support Help Center](https://support.sendgrid.com).

If you've instead found a bug in the library or would like new features added, go ahead and open issues or pull requests against this repo!

<a name="license"></a>
# License
[The MIT License (MIT)](LICENSE)
