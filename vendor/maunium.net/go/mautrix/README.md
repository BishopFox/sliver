# mautrix-go
[![GoDoc](https://pkg.go.dev/badge/maunium.net/go/mautrix)](https://pkg.go.dev/maunium.net/go/mautrix)

A Golang Matrix framework. Used by [gomuks](https://gomuks.app),
[go-neb](https://github.com/matrix-org/go-neb),
[mautrix-whatsapp](https://github.com/mautrix/whatsapp)
and others.

Matrix room: [`#go:maunium.net`](https://matrix.to/#/#go:maunium.net)

This project is based on [matrix-org/gomatrix](https://github.com/matrix-org/gomatrix).
The original project is licensed under [Apache 2.0](https://github.com/matrix-org/gomatrix/blob/master/LICENSE).

In addition to the basic client API features the original project has, this framework also has:

* Appservice support (Intent API like mautrix-python, room state storage, etc)
* End-to-end encryption support (incl. key backup, cross-signing, interactive verification, etc)
* High-level module for building puppeting bridges
* Partial federation module (making requests, PDU processing and event authorization)
* A media proxy server which can be used to expose anything as a Matrix media repo
* Wrapper functions for the Synapse admin API
* Structs for parsing event content
* Helpers for parsing and generating Matrix HTML
* Helpers for handling push rules
