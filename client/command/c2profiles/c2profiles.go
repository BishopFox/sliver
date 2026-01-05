package c2profiles

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var (
	ErrNoSelection = errors.New("no selection")
)

// C2ProfileCmd list available http profiles
func C2ProfileCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	profileName, _ := cmd.Flags().GetString("name")

	if profileName == constants.DefaultC2Profile {
		httpC2Profiles, err := con.Rpc.GetHTTPC2Profiles(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("failed to fetch HTTP C2 profiles: %s", err.Error())
			return
		}
		if len(httpC2Profiles.Configs) != 1 {
			profileName, err = selectC2Profile(httpC2Profiles.Configs)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
		}
	}

	profile, err := con.Rpc.GetHTTPC2ProfileByName(context.Background(), &clientpb.C2ProfileReq{Name: profileName})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintC2Profiles(profile, con)
}

func ImportC2ProfileCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	profileName, _ := cmd.Flags().GetString("name")
	if profileName == "" {
		con.PrintErrorf("Invalid c2 profile name\n")
		return
	}

	filepath, _ := cmd.Flags().GetString("file")
	if filepath == "" {
		con.PrintErrorf("Missing file path\n")
		return
	}

	overwrite, _ := cmd.Flags().GetBool("overwrite")

	// retrieve and unmarshal profile config
	jsonFile, err := os.Open(filepath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	byteFile, _ := io.ReadAll(jsonFile)
	var config *assets.HTTPC2Config = &assets.HTTPC2Config{}
	err = json.Unmarshal(byteFile, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	httpC2ConfigReq := clientpb.HTTPC2ConfigReq{Overwrite: overwrite, C2Config: C2ConfigToProtobuf(profileName, config)}

	_, err = con.Rpc.SaveHTTPC2Profile(context.Background(), &httpC2ConfigReq)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
}

func ExportC2ProfileCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {

	filepath, _ := cmd.Flags().GetString("file")
	if filepath == "" {
		con.PrintErrorf("Missing file path\n")
		return
	}

	profileName, _ := cmd.Flags().GetString("name")
	if profileName == "" {
		con.PrintErrorf("Invalid c2 profile name\n")
		return
	}

	if profileName == constants.DefaultC2Profile {
		httpC2Profiles, err := con.Rpc.GetHTTPC2Profiles(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("failed to fetch HTTP C2 profiles: %s", err.Error())
			return
		}
		if len(httpC2Profiles.Configs) != 1 {
			profileName, err = selectC2Profile(httpC2Profiles.Configs)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
		}
	}

	profile, err := con.Rpc.GetHTTPC2ProfileByName(context.Background(), &clientpb.C2ProfileReq{Name: profileName})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	config, err := C2ConfigToJSON(profileName, profile)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	jsonProfile, err := json.Marshal(config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	err = os.WriteFile(filepath, jsonProfile, 0644)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.Println(profileName, "C2 profile exported to ", filepath)
}

func GenerateC2ProfileCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {

	// load template to use as starting point
	template, err := cmd.Flags().GetString("template")
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	profileName, _ := cmd.Flags().GetString("name")
	if profileName == "" {
		con.PrintErrorf("Invalid c2 profile name\n")
		return
	}

	profile, err := con.Rpc.GetHTTPC2ProfileByName(context.Background(), &clientpb.C2ProfileReq{Name: template})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	config, err := C2ConfigToJSON(profileName, profile)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// read urls files and replace segments
	filepath, err := cmd.Flags().GetString("file")
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	urlsFile, err := os.Open(filepath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	fileContent, err := io.ReadAll(urlsFile)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	urls := strings.Split(string(fileContent), "\n")

	jsonProfile, err := updateC2Profile(config, urls)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// save or display config
	importC2Profile, err := cmd.Flags().GetBool("import")
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if importC2Profile {
		httpC2ConfigReq := clientpb.HTTPC2ConfigReq{C2Config: C2ConfigToProtobuf(profileName, jsonProfile)}
		_, err = con.Rpc.SaveHTTPC2Profile(context.Background(), &httpC2ConfigReq)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Println("C2 profile generated and saved as ", profileName)
	} else {
		PrintC2Profiles(C2ConfigToProtobuf(profileName, jsonProfile), con)
	}
}

// convert protobuf to json
func C2ConfigToJSON(profileName string, profile *clientpb.HTTPC2Config) (*assets.HTTPC2Config, error) {
	implantConfig := assets.HTTPC2ImplantConfig{
		UserAgent:          profile.ImplantConfig.UserAgent,
		ChromeBaseVersion:  int(profile.ImplantConfig.ChromeBaseVersion),
		MacOSVersion:       profile.ImplantConfig.MacOSVersion,
		NonceQueryArgChars: profile.ImplantConfig.NonceQueryArgChars,
		NonceQueryLength:   int(profile.ImplantConfig.NonceQueryLength),
		NonceMode:          profile.ImplantConfig.NonceMode,
		MaxFileGen:         int(profile.ImplantConfig.MaxFileGen),
		MinFileGen:         int(profile.ImplantConfig.MinFileGen),
		MaxPathGen:         int(profile.ImplantConfig.MaxPathGen),
		MinPathGen:         int(profile.ImplantConfig.MinFileGen),
		MaxPathLength:      int(profile.ImplantConfig.MaxPathLength),
		MinPathLength:      int(profile.ImplantConfig.MinPathLength),
		Extensions:         profile.ImplantConfig.Extensions,
	}

	var headers []assets.NameValueProbability
	for _, header := range profile.ImplantConfig.Headers {
		headers = append(headers, assets.NameValueProbability{
			Name:        header.Name,
			Value:       header.Value,
			Probability: int(header.Probability),
			Method:      header.Method,
		})
	}
	implantConfig.Headers = headers

	var urlParameters []assets.NameValueProbability
	for _, urlParameter := range profile.ImplantConfig.ExtraURLParameters {
		urlParameters = append(urlParameters, assets.NameValueProbability{
			Name:        urlParameter.Name,
			Value:       urlParameter.Value,
			Probability: int(urlParameter.Probability),
		})
	}
	implantConfig.URLParameters = urlParameters

	var (
		files []string
		paths []string
	)

	for _, pathSegment := range profile.ImplantConfig.PathSegments {
		if pathSegment.IsFile {
			files = append(files, pathSegment.Value)
		} else {
			paths = append(paths, pathSegment.Value)
		}
	}
	implantConfig.Files = files
	implantConfig.Paths = paths

	var serverHeaders []assets.NameValueProbability
	for _, header := range profile.ServerConfig.Headers {
		serverHeaders = append(serverHeaders, assets.NameValueProbability{
			Name:        header.Name,
			Value:       header.Value,
			Probability: int(header.Probability),
			Method:      header.Method,
		})
	}

	var serverCookies []string
	for _, cookie := range profile.ServerConfig.Cookies {
		serverCookies = append(serverCookies, cookie.Name)
	}

	serverConfig := assets.HTTPC2ServerConfig{
		RandomVersionHeaders: profile.ServerConfig.RandomVersionHeaders,
		Headers:              serverHeaders,
		Cookies:              serverCookies,
	}

	config := assets.HTTPC2Config{
		ImplantConfig: implantConfig,
		ServerConfig:  serverConfig,
	}

	return &config, nil
}

// convert json to protobuf
func C2ConfigToProtobuf(profileName string, config *assets.HTTPC2Config) *clientpb.HTTPC2Config {

	httpC2UrlParameters := []*clientpb.HTTPC2URLParameter{}
	httpC2Headers := []*clientpb.HTTPC2Header{}
	pathSegments := []*clientpb.HTTPC2PathSegment{}

	// files
	for _, file := range config.ImplantConfig.Files {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile: true,
			Value:  file,
		})
	}

	for _, path := range config.ImplantConfig.Paths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile: false,
			Value:  path,
		})
	}

	for _, clientHeader := range config.ImplantConfig.Headers {
		httpC2Headers = append(httpC2Headers, &clientpb.HTTPC2Header{
			Method:      clientHeader.Method,
			Name:        clientHeader.Name,
			Value:       clientHeader.Value,
			Probability: int32(clientHeader.Probability),
		})
	}

	for _, urlParameter := range config.ImplantConfig.URLParameters {
		httpC2UrlParameters = append(httpC2UrlParameters, &clientpb.HTTPC2URLParameter{
			Method:      urlParameter.Method,
			Name:        urlParameter.Name,
			Value:       urlParameter.Value,
			Probability: int32(urlParameter.Probability),
		})
	}

	implantConfig := &clientpb.HTTPC2ImplantConfig{
		UserAgent:          config.ImplantConfig.UserAgent,
		ChromeBaseVersion:  int32(config.ImplantConfig.ChromeBaseVersion),
		MacOSVersion:       config.ImplantConfig.MacOSVersion,
		NonceQueryArgChars: config.ImplantConfig.NonceQueryArgChars,
		NonceQueryLength:   int32(config.ImplantConfig.NonceQueryLength),
		NonceMode:          config.ImplantConfig.NonceMode,
		ExtraURLParameters: httpC2UrlParameters,
		Headers:            httpC2Headers,
		MaxFileGen:         int32(config.ImplantConfig.MaxFileGen),
		MinFileGen:         int32(config.ImplantConfig.MinFileGen),
		MaxPathGen:         int32(config.ImplantConfig.MaxPathGen),
		MinPathGen:         int32(config.ImplantConfig.MinPathGen),
		MaxPathLength:      int32(config.ImplantConfig.MaxPathLength),
		MinPathLength:      int32(config.ImplantConfig.MinPathLength),
		Extensions:         config.ImplantConfig.Extensions,
		PathSegments:       pathSegments,
	}

	// Server Config
	serverHeaders := []*clientpb.HTTPC2Header{}
	for _, serverHeader := range config.ServerConfig.Headers {
		serverHeaders = append(serverHeaders, &clientpb.HTTPC2Header{
			Method:      serverHeader.Method,
			Name:        serverHeader.Name,
			Value:       serverHeader.Value,
			Probability: int32(serverHeader.Probability),
		})
	}

	serverCookies := []*clientpb.HTTPC2Cookie{}
	for _, cookie := range config.ServerConfig.Cookies {
		serverCookies = append(serverCookies, &clientpb.HTTPC2Cookie{
			Name: cookie,
		})
	}
	serverConfig := &clientpb.HTTPC2ServerConfig{
		RandomVersionHeaders: config.ServerConfig.RandomVersionHeaders,
		Headers:              serverHeaders,
		Cookies:              serverCookies,
	}

	return &clientpb.HTTPC2Config{
		Name:          profileName,
		ImplantConfig: implantConfig,
		ServerConfig:  serverConfig,
	}
}

// PrintImplantBuilds - Print the implant builds on the server
func PrintC2Profiles(profile *clientpb.HTTPC2Config, con *console.SliverClient) {

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Parameter",
		"Value",
	})

	// Profile metadata

	tw.AppendRow(table.Row{
		"Profile Name",
		profile.Name,
	})

	// Server side configuration

	var serverHeaders []string
	for _, header := range profile.ServerConfig.Headers {
		serverHeaders = append(serverHeaders, header.Name)
	}
	tw.AppendRow(table.Row{
		"Server Headers",
		strings.Join(serverHeaders[:], ","),
	})

	var serverCookies []string
	for _, cookie := range profile.ServerConfig.Cookies {
		serverCookies = append(serverCookies, cookie.Name)
	}
	tw.AppendRow(table.Row{
		"Server Cookies",
		strings.Join(serverCookies[:], ","),
	})

	tw.AppendRow(table.Row{
		"Randomize Server Headers",
		profile.ServerConfig.RandomVersionHeaders,
	})

	// Client side configuration

	var clientHeaders []string
	for _, header := range profile.ImplantConfig.Headers {
		clientHeaders = append(clientHeaders, header.Name)
	}
	tw.AppendRow(table.Row{
		"Client Headers",
		strings.Join(clientHeaders[:], ","),
	})

	var clientUrlParams []string
	for _, clientUrlParam := range profile.ImplantConfig.ExtraURLParameters {
		clientUrlParams = append(clientUrlParams, clientUrlParam.Name)
	}
	tw.AppendRow(table.Row{
		"Extra URL Parameters",
		strings.Join(clientUrlParams[:], ","),
	})
	tw.AppendRow(table.Row{
		"User agent",
		profile.ImplantConfig.UserAgent,
	})
	tw.AppendRow(table.Row{
		"Chrome base version",
		profile.ImplantConfig.ChromeBaseVersion,
	})
	tw.AppendRow(table.Row{
		"MacOS version",
		profile.ImplantConfig.MacOSVersion,
	})
	tw.AppendRow(table.Row{
		"Nonce query arg chars",
		profile.ImplantConfig.NonceQueryArgChars,
	})
	tw.AppendRow(table.Row{
		"Nonce query length",
		profile.ImplantConfig.NonceQueryLength,
	})
	tw.AppendRow(table.Row{
		"Nonce mode",
		profile.ImplantConfig.NonceMode,
	})
	tw.AppendRow(table.Row{
		"Max files",
		profile.ImplantConfig.MaxFileGen,
	})
	tw.AppendRow(table.Row{
		"Min files",
		profile.ImplantConfig.MinFileGen,
	})
	tw.AppendRow(table.Row{
		"Max paths",
		profile.ImplantConfig.MaxPathGen,
	})
	tw.AppendRow(table.Row{
		"Min paths",
		profile.ImplantConfig.MinPathGen,
	})

	tw.AppendRow(table.Row{
		"Max path length",
		profile.ImplantConfig.MaxPathLength,
	})

	tw.AppendRow(table.Row{
		"Min path length",
		profile.ImplantConfig.MinPathLength,
	})

	tw.AppendRow(table.Row{
		"File extensions",
		strings.Join(profile.ImplantConfig.Extensions, ","),
	})

	var (
		paths []string
		files []string
	)
	for _, segment := range profile.ImplantConfig.PathSegments {
		if segment.IsFile {
			files = append(files, segment.Value)
		} else {
			paths = append(paths, segment.Value)
		}
	}

	tw.AppendRow(table.Row{
		"Paths",
		strings.Join(paths[:], ",")[:50] + "...",
	})
	tw.AppendRow(table.Row{
		"Files",
		strings.Join(files[:], ",")[:50] + "...",
	})

	con.Println(tw.Render())
	con.Println("\n")
}

func selectC2Profile(c2profiles []*clientpb.HTTPC2Config) (string, error) {
	c2profile := ""
	var choices []string
	for _, c2profile := range c2profiles {
		choices = append(choices, c2profile.Name)
	}

	_ = forms.Select("Select a c2 profile", choices, &c2profile)

	if c2profile == "" {
		return "", ErrNoSelection
	}

	return c2profile, nil
}

func updateC2Profile(template *assets.HTTPC2Config, urls []string) (*assets.HTTPC2Config, error) {
	// update the template with the urls

	var (
		paths      []string
		filenames  []string
		extensions []string
	)

	for _, urlPath := range urls {
		parsedURL, err := url.Parse(urlPath)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}

		dir, file := path.Split(parsedURL.Path)
		dir = strings.Trim(dir, "/")
		if dir != "" {
			paths = append(paths, strings.Split(dir, "/")...)
		}

		if file != "" {
			fileName := strings.TrimSuffix(file, filepath.Ext(file))
			filenames = append(filenames, fileName)
			ext := strings.TrimPrefix(filepath.Ext(file), ".")
			if ext != "" {
				extensions = append(extensions, ext)
			}
		}
	}

	slices.Sort(extensions)
	extensions = slices.Compact(extensions)

	slices.Sort(paths)
	paths = slices.Compact(paths)

	slices.Sort(filenames)
	filenames = slices.Compact(filenames)

	// 5 is arbitrarily used as a minimum value
	if len(paths) < 5 {
		return nil, fmt.Errorf("got %d paths need at least 5", len(paths))
	}

	if len(filenames) < 5 {
		return nil, fmt.Errorf("got %d paths need at least 5", len(filenames))
	}

	template.ImplantConfig.Extensions = extensions
	template.ImplantConfig.Paths = paths
	template.ImplantConfig.Files = filenames
	return template, nil
}
