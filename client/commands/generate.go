package commands

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/connection"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"gopkg.in/AlecAivazis/survey.v1"
)

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

// Generate - Configure and compile an implant
type Generate struct {
	StageOptions // Command makes use of full stage options
}

// StageOptions - All these options, regrouped by area, are used by any command that needs full
// configuration information for a stage Sliver implant.
type StageOptions struct {
	// CoreOptions - All options about OS/arch, files to save, debugs, etc.
	CoreOptions struct {
		OS      string `long:"os" description:"Target host operating system (default: windows)" default:"windows"`
		Arch    string `long:"arch" description:"Target host CPU architecture (default: amd64)" default:"amd64"`
		Format  string `long:"format" description:"Specifies output formats ('exe', 'shared' (DLL), 'service' (see 'psexec' for more info), 'shellcode' (Windows only)" default:"exe"`
		Profile string `long:"profile-name" description:"Implant profile name to use"`
		Save    string `long:"save" description:"Directory/file where to save binary"`
		Debug   bool   `long:"debug" description:"Enable debug features (incompatible with obfuscation, and prevailing)"`
	} `group:"Core options"`

	// TransportOptions - All options pertaining to transport/RPC matters
	TransportOptions struct {
		MTLS      []string `long:"mtls" description:"mTLS C2 domain(s), comma-separated (ex: mtls://host:port)" env-delim:","`
		DNS       []string `long:"dns" description:"DNS C2 domain(s), comma-separated (ex: dns://mydomain.com)" env-delim:","`
		HTTP      []string `long:"http" description:"HTTP(S) C2 domain(s)" env-delim:","`
		NamedPipe []string `long:"named-pipe" description:"Named pipe connection strings, comma-separated" env-delim:","`
		TCPPivot  []string `long:"tcp-pivot" description:"TCP pivot connection strings, comma-separated" env-delim:","`
		Reconnect int      `long:"reconnect" description:"Attempt to reconnect every n second(s)"`
		MaxErrors int      `long:"max-errors" description:"Max number of connection errors"`
		Timeout   int      `long:"timeout" description:"Command timeout in seconds"`
	} `group:"Transport options"`

	// SecurityOptions - All security-oriented options like restrictions.
	SecurityOptions struct {
		LimitDatetime  string `long:"limit-datetime" description:"Limit execution to before datetime"`
		LimitDomain    bool   `long:"limit-domain-joined" description:"Limit execution to domain joined machines"`
		LimitUsername  string `long:"limit-username" description:"Limit execution to specified username"`
		LimitHosname   string `long:"limit-hostname" description:"Limit execution to specified hostname"`
		LimitFileExits string `long:"limit-file-exists" description:"Limit execution to hosts with this file in the filesystem"`
	} `group:"Security options"`

	// EvasionOptions - All proactive security options (obfuscation, evasion, etc)
	EvasionOptions struct {
		Canary      []string `long:"canary" description:"DNS canary domain strings, comma-separated" env-delim:","`
		SkipSymbols bool     `long:"skip-obfuscation" description:"Skip binary/symbol obfuscation"`
		Evasion     bool     `long:"evasion" description:"Enable evasion features"`
	} `group:"Evasion options"`
}

// Execute - Configure and compile an implant
func (g *Generate) Execute(args []string) (err error) {
	config := parseCompileFlags(g.StageOptions)
	if config == nil {
		return
	}
	save := g.CoreOptions.Save
	if save == "" {
		save, _ = os.Getwd()
	}
	compile(config, save)

	return
}

// Shared function that extracts the compile flags from a StageOptions struct above, and returns a configuration.
func parseCompileFlags(g StageOptions) *clientpb.ImplantConfig {
	var name string
	targetOS := strings.ToLower(g.CoreOptions.OS)
	arch := strings.ToLower(g.CoreOptions.Arch)

	// if ctx.Flags["name"] != nil {
	//         name = strings.ToLower(ctx.Flags.String("name"))
	//
	//         if name != "" {
	//                 isAlphanumeric := regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
	//                 if !isAlphanumeric(name) {
	//                         fmt.Printf(Warn + "Agent's name must be in alphanumeric only\n")
	//                         return nil
	//                 }
	//         }
	// }

	c2s := []*clientpb.ImplantC2{}

	mtlsC2 := parseMTLSc2(g.TransportOptions.MTLS)
	c2s = append(c2s, mtlsC2...)

	httpC2 := parseHTTPc2(g.TransportOptions.HTTP)
	c2s = append(c2s, httpC2...)

	dnsC2 := parseDNSc2(g.TransportOptions.DNS)
	c2s = append(c2s, dnsC2...)

	namedPipeC2 := parseNamedPipec2(g.TransportOptions.NamedPipe)
	c2s = append(c2s, namedPipeC2...)

	tcpPivotC2 := parseTCPPivotc2(g.TransportOptions.TCPPivot)
	c2s = append(c2s, tcpPivotC2...)

	var symbolObfuscation bool
	if g.CoreOptions.Debug {
		symbolObfuscation = false
	} else {
		symbolObfuscation = !g.EvasionOptions.SkipSymbols
	}

	if len(mtlsC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 && len(namedPipeC2) == 0 && len(tcpPivotC2) == 0 {
		fmt.Printf(util.Error + "Must specify at least one of --mtls, --http, --dns, --named-pipe, or --tcp-pivot\n")
		return nil
	}

	var canaryDomains []string
	if 0 < len(g.EvasionOptions.Canary) {
		for _, canaryDomain := range g.EvasionOptions.Canary {
			if !strings.HasSuffix(canaryDomain, ".") {
				canaryDomain += "." // Ensure we have the FQDN
			}
			canaryDomains = append(canaryDomains, canaryDomain)
		}
	}

	reconnectInterval := g.TransportOptions.Reconnect
	maxConnectionErrors := g.TransportOptions.MaxErrors

	limitDomainJoined := g.SecurityOptions.LimitDomain
	limitHostname := g.SecurityOptions.LimitHosname
	limitUsername := g.SecurityOptions.LimitUsername
	limitDatetime := g.SecurityOptions.LimitDatetime
	limitFileExists := g.SecurityOptions.LimitFileExits

	isSharedLib := false
	isService := false
	isShellcode := false

	format := g.CoreOptions.Format
	var configFormat clientpb.ImplantConfig_OutputFormat
	switch format {
	case "exe":
		configFormat = clientpb.ImplantConfig_EXECUTABLE
	case "shared":
		configFormat = clientpb.ImplantConfig_SHARED_LIB
		isSharedLib = true
	case "shellcode":
		configFormat = clientpb.ImplantConfig_SHELLCODE
		isShellcode = true
	case "service":
		configFormat = clientpb.ImplantConfig_SERVICE
		isService = true
	default:
		// default to exe
		configFormat = clientpb.ImplantConfig_EXECUTABLE
	}
	/* For UX we convert some synonymous terms */
	if targetOS == "darwin" || targetOS == "mac" || targetOS == "macos" || targetOS == "m" || targetOS == "osx" {
		targetOS = "darwin"
	}
	if targetOS == "windows" || targetOS == "win" || targetOS == "w" || targetOS == "shit" {
		targetOS = "windows"
	}
	if targetOS == "linux" || targetOS == "unix" || targetOS == "l" {
		targetOS = "linux"
	}
	if arch == "x64" || strings.HasPrefix(arch, "64") {
		arch = "amd64"
	}
	if arch == "x86" || strings.HasPrefix(arch, "32") {
		arch = "386"
	}

	if len(namedPipeC2) > 0 && targetOS != "windows" {
		fmt.Printf(util.Error + "Named pipe pivoting can only be used in Windows.")
		return nil
	}

	config := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           arch,
		Name:             name,
		Debug:            g.CoreOptions.Debug,
		Evasion:          g.EvasionOptions.Evasion,
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,

		ReconnectInterval:   uint32(reconnectInterval),
		MaxConnectionErrors: uint32(maxConnectionErrors),

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,
		LimitFileExists:   limitFileExists,

		Format:      configFormat,
		IsSharedLib: isSharedLib,
		IsService:   isService,
		IsShellcode: isShellcode,
	}

	return config
}

func parseMTLSc2(args []string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if len(args) == 0 {
		return c2s
	}
	for index, arg := range args {
		uri := url.URL{Scheme: "mtls"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultMTLSLPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseHTTPc2(args []string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if len(args) == 0 {
		return c2s
	}
	for index, arg := range args {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			uri, err = url.Parse(arg)
			if err != nil {
				log.Printf("Failed to parse c2 URL %v", err)
				continue
			}
		} else {
			uri = &url.URL{Scheme: "https"} // HTTPS is the default, will fallback to HTTP
			uri.Host = arg
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseDNSc2(args []string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if len(args) == 0 {
		return c2s
	}
	for index, arg := range args {
		uri := url.URL{Scheme: "dns"}
		if len(arg) < 1 {
			continue
		}
		// Make sure we have the FQDN
		if !strings.HasSuffix(arg, ".") {
			arg += "."
		}
		if strings.HasPrefix(arg, ".") {
			arg = arg[1:]
		}

		uri.Host = arg
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseNamedPipec2(args []string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if len(args) == 0 {
		return c2s
	}
	for index, arg := range args {
		uri, err := url.Parse("namedpipe://" + arg)
		if len(arg) < 1 {
			continue
		}
		if err != nil {
			return c2s
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseTCPPivotc2(args []string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if len(args) == 0 {
		return c2s
	}
	for index, arg := range args {

		uri := url.URL{Scheme: "tcppivot"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultTCPPivotPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func compile(config *clientpb.ImplantConfig, save string) (*commonpb.File, error) {

	fmt.Printf(util.Info+"Generating new %s/%s implant binary\n", config.GOOS, config.GOARCH)

	if config.ObfuscateSymbols {
		fmt.Printf(util.Info+"%sSymbol obfuscation is enabled.%s\n", bold, normal)
		fmt.Printf(util.Info + "This process can take awhile, and consumes significant amounts of CPU/Memory\n")
	} else if !config.Debug {
		fmt.Printf(util.Warn+"Symbol obfuscation is %sdisabled%s\n", bold, normal)
	}

	start := time.Now()
	ctrl := make(chan bool)
	go spin.Until("Compiling, please wait ...", ctrl)

	generated, err := connection.RPC.Generate(context.Background(), &clientpb.GenerateReq{
		Config: config,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil, err
	}

	end := time.Now()
	elapsed := time.Time{}.Add(end.Sub(start))
	fmt.Printf(clearln+util.Info+"Build completed in %s\n", elapsed.Format("15:04:05"))
	if len(generated.File.Data) == 0 {
		fmt.Printf(util.Warn + "Build failed, no file data\n")
		return nil, errors.New("No file data")
	}

	saveTo, err := saveLocation(save, generated.File.Name)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(saveTo, generated.File.Data, 0700)
	if err != nil {
		fmt.Printf(util.Warn+"Failed to write to: %s\n", saveTo)
		return nil, err
	}
	fmt.Printf(util.Info+"Implant saved to %s\n", saveTo)
	return generated.File, err
}

func saveLocation(save, defaultName string) (string, error) {
	var saveTo string
	if save == "" {
		save, _ = os.Getwd()
	}
	fi, err := os.Stat(save)
	if os.IsNotExist(err) {
		log.Printf("%s does not exist\n", save)
		if strings.HasSuffix(save, "/") {
			log.Printf("%s is dir\n", save)
			os.MkdirAll(save, 0700)
			saveTo, _ = filepath.Abs(path.Join(saveTo, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			saveDir := filepath.Dir(save)
			_, err := os.Stat(saveTo)
			if os.IsNotExist(err) {
				os.MkdirAll(saveDir, 0700)
			}
			saveTo, _ = filepath.Abs(save)
		}
	} else {
		log.Printf("%s does exist\n", save)
		if fi.IsDir() {
			log.Printf("%s is dir\n", save)
			saveTo, _ = filepath.Abs(path.Join(save, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			prompt := &survey.Confirm{Message: "Overwrite existing file?"}
			var confirm bool
			survey.AskOne(prompt, &confirm, nil)
			if !confirm {
				return "", errors.New("File already exists")
			}
			saveTo, _ = filepath.Abs(save)
		}
	}
	return saveTo, nil
}
