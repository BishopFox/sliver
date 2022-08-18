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
	"net/url"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/proxy"
)

// getHTTPClientProxyOptions returns a list of candidate proxy URLs for SliverHTTPClient generation
func getHTTPClientProxyOptions(origin string, driver HTTPDriverType, opts *HTTPOptions) []*url.URL {
	var proxies []*url.URL

	if driver != GoHTTPDriverType {
		return proxies
	}

	switch opts.ProxyConfig {
	case "never":
		break
	case "", "auto":
		p := proxy.NewProvider("").GetHTTPSProxy(origin)
		if p != nil {
			// {{if .Config.Debug}}
			log.Printf("[http] Found proxy %#v\n", p)
			// {{end}}

			proxyURL := p.URL()
			proxies = addProxyURLs(proxies, proxyURL)
		}
	default:
		// {{if .Config.Debug}}
		log.Printf("[http] Force proxy %#v\n", opts.ProxyConfig)
		// {{end}}

		proxyURL, err := url.Parse(opts.ProxyConfig)
		if err != nil {
			break
		}

		proxies = addProxyURLs(proxies, proxyURL)
	}

	// {{if .Config.Debug}}
	log.Printf("[http] Proxy list: %+v\n", proxies)
	// {{end}}

	return proxies
}

// addProxyURLs appends a proxy URL to a list of proxy URLs, or adds an HTTP- as well as HTTPS-based URL if the proxy URL scheme is not specified
func addProxyURLs(proxies []*url.URL, proxyURL *url.URL) []*url.URL {
	if proxyURL.Scheme != "" {
		proxies = append(proxies, proxyURL)
	} else {
		proxyURL.Scheme = "https"
		proxies = append(proxies, proxyURL)

		proxyURLCopy := *proxyURL
		proxyURL.Scheme = "http"
		proxies = append(proxies, &proxyURLCopy)
	}

	return proxies
}
