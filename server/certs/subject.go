package certs

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"crypto/x509/pkix"
	"fmt"
	insecureRand "math/rand"
	"strings"
)

var (
	// State -> Localities -> Street Addresses
	states = map[string]map[string][]string{
		"": {
			"": {""},
		},
		"Arizona": {
			"Phoenix":    {""},
			"Mesa":       {""},
			"Scottsdale": {""},
			"Chandler":   {""},
		},
		"California": {
			"San Francisco": {"", "Golden Gate Bridge"},
			"Oakland":       {""},
			"Berkeley":      {""},
			"Palo Alto":     {""},
			"Los Angeles":   {""},
			"San Diego":     {""},
			"San Jose":      {""},
		},
		"Colorado": {
			"Denver":       {""},
			"Boulder":      {""},
			"Aurora":       {""},
			"Fort Collins": {""},
		},
		"Connecticut": {
			"New Haven":  {""},
			"Bridgeport": {""},
			"Stamford":   {""},
			"Norwalk":    {""},
		},
		"Washington": {
			"Seattle": {""},
			"Tacoma":  {""},
			"Olympia": {""},
			"Spokane": {""},
		},
		"Florida": {
			"Miami":        {""},
			"Orlando":      {""},
			"Tampa":        {""},
			"Jacksonville": {""},
		},
		"Illinois": {
			"Chicago":    {""},
			"Aurora":     {""},
			"Naperville": {""},
			"Peoria":     {""},
		},
	}
)

func randomSubject(commonName string) *pkix.Name {
	province, locale, street := randomProvinceLocalityStreetAddress()
	return &pkix.Name{
		Organization:  randomOrganization(),
		Country:       []string{"US"},
		Province:      province,
		Locality:      locale,
		StreetAddress: street,
		PostalCode:    randomPostalCode(),
		CommonName:    commonName,
	}
}

func randomPostalCode() []string {
	switch insecureRand.Intn(1) {
	case 0:
		return []string{fmt.Sprintf("%d", insecureRand.Intn(8000)+1000)}
	default:
		return []string{}
	}
}

func randomProvinceLocalityStreetAddress() ([]string, []string, []string) {
	state := randomState()
	locality := randomLocality(state)
	streetAddress := randomStreetAddress(state, locality)
	return []string{state}, []string{locality}, []string{streetAddress}
}

func randomState() string {
	keys := make([]string, 0, len(states))
	for k := range states {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomLocality(state string) string {
	locales := states[state]
	keys := make([]string, 0, len(locales))
	for k := range locales {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomStreetAddress(state string, locality string) string {
	addresses := states[state][locality]
	return addresses[insecureRand.Intn(len(addresses))]
}

var (
	orgNames = []string{
		"",
		"ACME",
		"Partners",
		"Tech",
		"Cloud",
		"Synergy",
		"Test",
		"Debug",
	}
	orgSuffixes = []string{
		"",
		"co",
		"llc",
		"inc",
		"corp",
		"ltd",
	}
)

func randomOrganization() []string {
	name := orgNames[insecureRand.Intn(len(orgNames))]
	suffix := orgSuffixes[insecureRand.Intn(len(orgSuffixes))]
	switch insecureRand.Intn(4) {
	case 0:
		return []string{strings.TrimSpace(strings.ToLower(name + " " + suffix))}
	case 1:
		return []string{strings.TrimSpace(strings.ToUpper(name + " " + suffix))}
	case 2:
		return []string{strings.TrimSpace(strings.Title(fmt.Sprintf("%s %s", name, suffix)))}

	default:
		return []string{}
	}
}
