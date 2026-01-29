// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"regexp"
	"strconv"
)

type ParsedServerNameType int

const (
	ServerNameDNS ParsedServerNameType = iota
	ServerNameIPv4
	ServerNameIPv6
)

type ParsedServerName struct {
	Type ParsedServerNameType
	Host string
	Port int
}

var ServerNameRegex = regexp.MustCompile(`^(?:\[([0-9A-Fa-f:.]{2,45})]|(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})|([0-9A-Za-z.-]{1,255}))(?::(\d{1,5}))?$`)

func ValidateServerName(serverName string) bool {
	return len(serverName) <= 255 && len(serverName) > 0 && ServerNameRegex.MatchString(serverName)
}

func ParseServerName(serverName string) *ParsedServerName {
	if len(serverName) > 255 || len(serverName) < 1 {
		return nil
	}
	match := ServerNameRegex.FindStringSubmatch(serverName)
	if len(match) != 5 {
		return nil
	}
	port, _ := strconv.Atoi(match[4])
	parsed := &ParsedServerName{
		Port: port,
	}
	switch {
	case match[1] != "":
		parsed.Type = ServerNameIPv6
		parsed.Host = match[1]
	case match[2] != "":
		parsed.Type = ServerNameIPv4
		parsed.Host = match[2]
	case match[3] != "":
		parsed.Type = ServerNameDNS
		parsed.Host = match[3]
	}
	return parsed
}
