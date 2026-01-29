[![Build Status](https://github.com/firebase/firebase-admin-go/workflows/Continuous%20Integration/badge.svg?branch=dev)](https://github.com/firebase/firebase-admin-go/actions)
[![GoDoc](https://godoc.org/firebase.google.com/go?status.svg)](https://godoc.org/firebase.google.com/go)
[![Go Report Card](https://goreportcard.com/badge/github.com/firebase/firebase-admin-go)](https://goreportcard.com/report/github.com/firebase/firebase-admin-go)

# Firebase Admin Go SDK

## Table of Contents

 * [Overview](#overview)
 * [Installation](#installation)
 * [Contributing](#contributing)
 * [Documentation](#documentation)
 * [License and Terms](#license-and-terms)

## Overview

[Firebase](https://firebase.google.com) provides the tools and infrastructure
you need to develop apps, grow your user base, and earn money. The Firebase
Admin Go SDK enables access to Firebase services from privileged environments
(such as servers or cloud) in Go. Currently this SDK provides
Firebase custom authentication support.

For more information, visit the
[Firebase Admin SDK setup guide](https://firebase.google.com/docs/admin/setup/).


## Installation

The Firebase Admin Go SDK can be installed using the `go get` utility:

```
# Install the latest version:
go get firebase.google.com/go/v4@latest

# Or install a specific version:
go get firebase.google.com/go/v4@4.x.x
```

## Contributing

Please refer to the [CONTRIBUTING page](./CONTRIBUTING.md) for more information
about how you can contribute to this project. We welcome bug reports, feature
requests, code review feedback, and also pull requests.

## Supported Go Versions

The Admin Go SDK is compatible with the two most-recent major Go releases.
We currently support Go v1.23 and 1.24.
[Continuous integration](https://github.com/firebase/firebase-admin-go/actions) system
tests the code on Go v1.23 and v1.24.

## Documentation

* [Setup Guide](https://firebase.google.com/docs/admin/setup/)
* [Authentication Guide](https://firebase.google.com/docs/auth/admin/)
* [Cloud Firestore](https://firebase.google.com/docs/firestore/)
* [Cloud Messaging Guide](https://firebase.google.com/docs/cloud-messaging/admin/)
* [Storage Guide](https://firebase.google.com/docs/storage/admin/start)
* [API Reference](https://godoc.org/firebase.google.com/go)
* [Release Notes](https://firebase.google.com/support/release-notes/admin/go)


## License and Terms

Firebase Admin Go SDK is licensed under the
[Apache License, version 2.0](http://www.apache.org/licenses/LICENSE-2.0).

Your use of Firebase is governed by the
[Terms of Service for Firebase Services](https://firebase.google.com/terms/).
