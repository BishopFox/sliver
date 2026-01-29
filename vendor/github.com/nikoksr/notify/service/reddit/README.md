# Reddit Usage

Ensure that you have already navigated to your GOPATH and installed the following packages:

* `go get -u github.com/nikoksr/notify`

## Steps for Reddit notifications

These are general and very high level instructions

1. Log into Reddit create a new "script" type by visiting [here](https://www.reddit.com/prefs/apps/)
2. The "redirect uri" parameter doesn't matter in this case and can just be set to `http://localhost:8080`
2. Copy the *client id* and *client secret* for usage below
4. Now you should be good to use the code detailed in [doc.go](doc.go)

**NOTE**: You may have difficulties using your user's password if you have 2FA enabled. You can disable it by going [here](https://www.reddit.com/prefs/update/) but be aware of the security implications and ensure you have a strong password set.