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

	"github.com/bishopfox/sliver/server/codenames"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// Finally found a good use for Github Co-Pilot!
	// Country -> State -> Localities -> Street Addresses
	subjects = map[string]map[string]map[string][]string{
		"US": {
			"": {
				"": {
					"",
				},
			},
			"Arizona": {
				"Phoenix":    {""},
				"Mesa":       {""},
				"Scottsdale": {""},
				"Chandler":   {""},
			},
			"California": {
				"San Francisco": {"", "", "Golden Gate"},
				"Oakland":       {""},
				"Berkeley":      {""},
				"Palo Alto":     {""},
				"Los Angeles":   {""},
				"San Diego":     {""},
				"San Jose":      {""},
				"Sunnyvale":     {""},
				"Santa Clara":   {""},
				"Mountain View": {""},
				"San Mateo":     {""},
				"Redwood City":  {""},
				"Menlo Park":    {""},
				"San Bruno":     {""},
				"San Carlos":    {""},
				"San Leandro":   {""},
				"San Rafael":    {""},
				"San Ramon":     {""},
				"Santa Monica":  {""},
				"Santa Rosa":    {""},
				"South San Francisco": {
					"",
				},
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
			"Indiana": {
				"Indianapolis": {""},
				"Fort Wayne":   {""},
				"Evansville":   {""},
				"South Bend":   {""},
			},
			"Massachusetts": {
				"Boston": {""},
				"Worcester": {
					"",
					"Polytechnic Institute",
				},
				"Springfield": {""},
				"Lowell":      {""},
			},
			"Michigan": {
				"Detroit": {""},
				"Grand Rapids": {
					"",
					"State University",
				},
				"Warren": {""},
				"Sterling Heights": {
					"",
					"Community College",
				},
			},
			"Minnesota": {
				"Minneapolis": {""},
				"Saint Paul":  {""},
				"Bloomington": {""},
				"Plymouth":    {""},
			},
			"New Jersey": {
				"Newark": {""},
				"Jersey City": {
					"",
					"Institute of Technology",
				},
				"Paterson": {""},
				"Elizabeth": {
					"",
					"Princeton",
				},
			},
			"New York": {
				"New York": {""},
				"Buffalo":  {""},
				"Rochester": {
					"",
					"University",
				},
				"Yonkers": {""},
			},
			"North Carolina": {
				"Charlotte": {""},
				"Raleigh":   {""},
				"Greensboro": {
					"",
					"University",
				},
				"Winston-Salem": {""},
			},
			"Ohio": {
				"Columbus": {""},
				"Cleveland": {
					"",
					"State University",
				},
				"Cincinnati": {""},
				"Toledo":     {""},
			},
		},
		"CA": {
			"": {
				"": {
					"",
				},
			},
			"Alberta": {
				"Calgary": {""},
				"Edmonton": {
					"",
					"University",
				},
				"Red Deer": {""},
				"Fort McMurray": {
					"",
					"University",
				},
			},
			"British Columbia": {
				"Vancouver": {""},
				"Victoria":  {""},
				"Kelowna":   {""},
				"Richmond":  {""},
			},
			"Manitoba": {
				"Winnipeg": {""},
				"Brandon":  {""},
				"Thompson": {""},
				"Portage la Prairie": {
					"",
					"University",
				},
			},
			"New Brunswick": {
				"Fredericton": {""},
				"Moncton":     {""},
				"Saint John":  {""},
				"Dieppe":      {""},
			},
			"Newfoundland and Labrador": {
				"St. John's": {""},
				"Mount Pearl": {
					"",
					"College",
				},
				"Conception Bay South": {""},
				"Paradise": {
					"",
					"College",
				},
			},
		},
		"JP": {
			"": {
				"": {
					"",
				},
			},
			"Aichi": {
				"Nagoya": {""},
				"Kasugai": {
					"",
					"University",
				},
				"Okazaki": {""},
				"Handa":   {""},
			},
			"Chiba": {
				"Chiba": {""},
				"Kashiwa": {
					"",
					"University",
				},
				"Funabashi": {""},
				"Kimitsu":   {""},
			},
		},
	}
)

func randomSubject(commonName string) *pkix.Name {
	country, province, locale, street := randomProvinceLocalityStreetAddress()
	return &pkix.Name{
		Organization:  randomOrganization(),
		Country:       country,
		Province:      province,
		Locality:      locale,
		StreetAddress: street,
		PostalCode:    randomPostalCode(country),
		CommonName:    commonName,
	}
}

func randomPostalCode(country []string) []string {
	// 1 in `n` will include a postal code
	// From my cursory view of a few TLS certs it seems uncommon to include this
	// in the distinguished name so right now it's set to 1/20
	const postalProbability = 20

	if len(country) == 0 {
		return []string{}
	}
	switch country[0] {

	case "US":
		// American postal codes are 5 digits
		switch insecureRand.Intn(postalProbability) {
		case 0:
			return []string{fmt.Sprintf("%05d", insecureRand.Intn(90000)+1000)}
		default:
			return []string{}
		}

	case "CA":
		// Canadian postal codes are weird and include letter/number combo's
		letters := "ABHLMNKGJPRSTVYX"
		switch insecureRand.Intn(postalProbability) {
		case 0:
			letter1 := string(letters[insecureRand.Intn(len(letters))])
			letter2 := string(letters[insecureRand.Intn(len(letters))])
			if insecureRand.Intn(2) == 0 {
				letter1 = strings.ToLower(letter1)
				letter2 = strings.ToLower(letter2)
			}
			return []string{
				fmt.Sprintf("%s%d%s", letter1, insecureRand.Intn(9), letter2),
			}
		default:
			return []string{}
		}
	}
	return []string{}
}

func randomProvinceLocalityStreetAddress() ([]string, []string, []string, []string) {
	country := randomCountry()
	state := randomState(country)
	locality := randomLocality(country, state)
	streetAddress := randomStreetAddress(country, state, locality)
	return []string{country}, []string{state}, []string{locality}, []string{streetAddress}
}

func randomCountry() string {
	keys := make([]string, 0, len(subjects))
	for k := range subjects {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomState(country string) string {
	keys := make([]string, 0, len(subjects[country]))
	for k := range subjects[country] {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomLocality(country string, state string) string {
	locales := subjects[country][state]
	keys := make([]string, 0, len(locales))
	for k := range locales {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomStreetAddress(country string, state string, locality string) string {
	addresses := subjects[country][state][locality]
	return addresses[insecureRand.Intn(len(addresses))]
}

var (
	orgSuffixes = []string{
		"",
		"",
		"co",
		"llc",
		"inc",
		"corp",
		"ltd",
		"plc",
		"inc.",
		"corp.",
		"ltd.",
		"plc.",
		"co.",
		"llc.",
		"incorporated",
		"limited",
		"corporation",
		"company",
		"incorporated",
		"limited",
		"corporation",
		"company",
		"GmbH",
		"AG",
		"S.A.",
		"B.V.",
		"LLP",
		"Pte Ltd",
		"Sdn Bhd",
		"KG",
		"Limited",
		"Partnership",
		"Associates",
		"Group",
		"Holdings",
		"Enterprises",
		"Industries",
		"Ventures",
		"International",
		"Systems",
		"Technologies",
		"Incorporated",
		"Services",
		"Solutions",
		"Enterprises",
		"Global",
		"Trading",
		"Manufacturing",
		"Development",
		"Management",
		"Consulting",
		"Logistics",
		"Communications",
		"Finance",
		"Electronics",
		"Pharmaceuticals",
		"Automotive",
		"Energy",
		"Healthcare",
		"Technology",
		"Biotech",
		"Media",
		"Software",
		"Hardware",
		"Entertainment",
		"Construction",
	}
)

func randomOrganization() []string {
	adjective, _ := codenames.RandomAdjective()
	noun, _ := codenames.RandomNoun()
	suffix := orgSuffixes[insecureRand.Intn(len(orgSuffixes))]

	// Not exactly sure this even matters much, but hey its fun to add more randomness
	caseTitles := []cases.Caser{
		cases.Title(language.English),
		cases.Title(language.AmericanEnglish),
		cases.Title(language.BritishEnglish),
	}
	caseTitle := caseTitles[insecureRand.Intn(len(caseTitles))]

	var orgName string
	switch insecureRand.Intn(8) {
	case 0:
		orgName = strings.TrimSpace(fmt.Sprintf("%s %s, %s", adjective, noun, suffix))
	case 1:
		orgName = strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s, %s", adjective, noun, suffix)))
	case 2:
		orgName = strings.TrimSpace(fmt.Sprintf("%s, %s", noun, suffix))
	case 3:
		orgName = strings.TrimSpace(caseTitle.String(fmt.Sprintf("%s %s, %s", adjective, noun, suffix)))
	case 4:
		orgName = strings.TrimSpace(caseTitle.String(fmt.Sprintf("%s %s", adjective, noun)))
	case 5:
		orgName = strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s", adjective, noun)))
	case 6:
		orgName = strings.TrimSpace(caseTitle.String(fmt.Sprintf("%s", noun)))
	case 7:
		noun2, _ := codenames.RandomNoun()
		orgName = strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s-%s", noun, noun2)))
	default:
		orgName = ""
	}

	return []string{orgName}
}
