package httpclient

import (
	"log"
	"net/url"

	"github.com/bishopfox/sliver/implant/sliver/proxy"
)

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
			log.Printf("Found proxy %#v\n", p)
			// {{end}}

			proxyURL := p.URL()
			proxies = addProxyURLs(proxies, proxyURL)
		}
	default:
		// {{if .Config.Debug}}
		log.Printf("Force proxy %#v\n", opts.ProxyConfig)
		// {{end}}

		proxyURL, err := url.Parse(opts.ProxyConfig)
		if err != nil {
			break
		}

		proxies = addProxyURLs(proxies, proxyURL)
	}

	// {{if .Config.Debug}}
	log.Printf("Proxy list: %+v\n", proxies)
	// {{end}}

	return proxies
}

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
