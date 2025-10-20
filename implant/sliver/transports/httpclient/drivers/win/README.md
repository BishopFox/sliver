# Forked from https://gitlab.com/mjwhitta/win/

#### Changes

 * Removed winhttp code
 * Removed custom errors package
 * Removed custom pathname package
 * Refactored names to be consistent with the rest of the Sliver code base
 * Merged client `http` sub-packages with `winhttp` and `wininet`
 * Added custom cookie jar implementation to wininet client

# Win

<a href="https://www.buymeacoffee.com/mjwhitta">🍪 Buy me a cookie</a>

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/mjwhitta/win)](https://goreportcard.com/report/gitlab.com/mjwhitta/win)

## What is this?

This Go module wraps WinHTTP and WinINet functions and includes HTTP
clients that use those functions to make HTTP requests. Hopefully,
this makes it easier to make HTTP requests in Go on Windows. WinHTTP
(and theoretically WinINet) can even handle NTLM authentication
automatically for you.

Microsoft recommends [WinINet over WinHTTP] unless you're writing a
Windows service.

**Note:** This is probably beta quality at best.

[WinINet over WinHTTP]: https://docs.microsoft.com/en-us/windows/win32/wininet/wininet-vs-winhttp

## How to install

Open a terminal and run the following:

```
$ go get --ldflags="-s -w" --trimpath -u gitlab.com/mjwhitta/win
```

## Usage

Minimal example:

```
package main

import (
    "fmt"
    "io/ioutil"

    // "gitlab.com/mjwhitta/win/winhttp/http"
    "gitlab.com/mjwhitta/win/wininet/http"
)

func main() {
    var b []byte
    var dst = "http://127.0.0.1:8080/asdf"
    var e error
    var headers = map[string]string{
        "User-Agent": "testing, testing, 1, 2, 3...",
    }
    var req *http.Request
    var res *http.Response

    http.DefaultClient.TLSClientConfig.InsecureSkipVerify = true

    if _, e = http.Get(dst); e != nil {
        panic(e)
    }

    req = http.NewRequest(http.MethodPost, dst, []byte("test"))
    req.AddCookie(&http.Cookie{Name: "chocolatechip", Value: "tasty"})
    req.AddCookie(&http.Cookie{Name: "oatmealraisin", Value: "gross"})
    req.AddCookie(&http.Cookie{Name: "snickerdoodle", Value: "yummy"})
    req.Headers = headers

    if res, e = http.DefaultClient.Do(req); e != nil {
        panic(e)
    }

    if res.Body != nil {
        if b, e = ioutil.ReadAll(res.Body); e != nil {
            panic(e)
        }
    }

    fmt.Println(res.Status)
    for k, vs := range res.Header {
        for _, v := range vs {
            fmt.Printf("%s: %s\n", k, v)
        }
    }
    if len(b) > 0 {
        fmt.Println(string(b))
    }

    if len(res.Cookies()) > 0 {
        fmt.Println()
        fmt.Println("# COOKIEJAR")
    }

    for _, cookie := range res.Cookies() {
        fmt.Printf("%s = %s\n", cookie.Name, cookie.Value)
    }
}
```

## Links

- [Source](https://gitlab.com/mjwhitta/win)

## TODO

- Mirror `net/http` as close as possible
    - CookieJar for the Client
    - etc...
- WinINet
    - FTP client
