package configs

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
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestCoerceHeaderProbability(t *testing.T) {
	headers := []*clientpb.HTTPC2Header{
		{Name: "no-probability", Value: "a"},
		{Name: "explicit-probability", Value: "b", Probability: 25},
		{Name: "full-probability", Value: "c", Probability: 100},
	}

	coerceHeaderProbability(headers)

	for _, header := range headers {
		if header.Probability == 0 {
			t.Errorf("expected header %q to have a non-zero probability, got 0", header.Name)
		}
	}
	if headers[0].Probability != 100 {
		t.Errorf("expected missing probability to default to 100, got %d", headers[0].Probability)
	}
	if headers[1].Probability != 25 {
		t.Errorf("expected explicit probability 25 to be preserved, got %d", headers[1].Probability)
	}
	if headers[2].Probability != 100 {
		t.Errorf("expected explicit probability 100 to be preserved, got %d", headers[2].Probability)
	}
}

func TestCheckHTTPC2ConfigDefaultsHeaderProbability(t *testing.T) {
	config := &clientpb.HTTPC2Config{
		ServerConfig: &clientpb.HTTPC2ServerConfig{
			Headers: []*clientpb.HTTPC2Header{
				{Name: "server-header", Value: "a"},
			},
			Cookies: []*clientpb.HTTPC2Cookie{
				{Name: "jsessionid"},
			},
		},
		ImplantConfig: &clientpb.HTTPC2ImplantConfig{
			Headers: []*clientpb.HTTPC2Header{
				{Name: "implant-header", Value: "b"},
				{Name: "complete-header", Value: "c", Probability: 50},
			},
			Extensions: []string{"js", "php"},
		},
	}

	err := checkHTTPC2Config(config)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if config.ServerConfig.Headers[0].Probability != 100 {
		t.Errorf("expected server header probability to default to 100, got %d", config.ServerConfig.Headers[0].Probability)
	}
	if config.ImplantConfig.Headers[0].Probability != 100 {
		t.Errorf("expected implant header probability to default to 100, got %d", config.ImplantConfig.Headers[0].Probability)
	}
	if config.ImplantConfig.Headers[1].Probability != 50 {
		t.Errorf("expected explicit probability 50 to be preserved, got %d", config.ImplantConfig.Headers[1].Probability)
	}
}
