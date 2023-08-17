package systemd

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"bytes"
	// Embed our example teamserver.service file.
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"text/template"

	"github.com/reeflective/team/internal/version"
)

// Config is a stub to generate systemd configuration files.
type Config struct {
	User    string   // User to configure systemd for, default is current user.
	Binpath string   // Path to binary
	Args    []string // The command is the position of the daemon command in the application command tree.
}

//go:embed teamserver.service
var systemdServiceTemplate string

// NewFrom returns a new templated systemd configuration file.
func NewFrom(name string, userCfg *Config) string {
	cfg := NewDefaultConfig()

	if userCfg != nil {
		cfg.User = userCfg.User
		cfg.Binpath = userCfg.Binpath
		cfg.Args = userCfg.Args
	}

	// Prepare all values before running templates
	ver := version.Semantic()
	version := fmt.Sprintf("%d.%d.%d", ver[0], ver[1], ver[2])
	desc := fmt.Sprintf("%s Teamserver daemon (v%s)", name, version)

	systemdUser := cfg.User
	if systemdUser == "" {
		systemdUser = "root"
	}

	// Command
	command := strings.Join(cfg.Args, " ")

	TemplateValues := struct {
		Application string
		Description string
		User        string
		Command     string
	}{
		Application: name,
		Description: desc,
		User:        systemdUser,
		Command:     command,
	}

	var config bytes.Buffer

	templ := template.New(name)
	parsed, err := templ.Parse(systemdServiceTemplate)
	if err != nil {
		log.Fatalf("Failed to parse: %s", err)
	}

	parsed.Execute(&config, TemplateValues)

	systemdFile := config.String()

	return systemdFile
}

// NewDefaultConfig returns a default Systemd service file configuration.
func NewDefaultConfig() *Config {
	c := &Config{}

	user, _ := user.Current()
	if user != nil {
		c.User = user.Username
	}

	currentPath, err := os.Executable()
	if err != nil {
		return c
	}

	c.Binpath = currentPath

	return c
}
