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
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
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
			profileName = selectC2Profile(httpC2Profiles.Configs)
		}
	}

	profile, err := con.Rpc.GetHTTPC2ProfileByName(context.Background(), &clientpb.C2ProfileReq{Name: profileName})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	filter, _ := cmd.Flags().GetString("filter")
	PrintC2Profiles(profile, con, filter)
}

func ImportC2ProfileCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	protocols := []string{constants.HttpStr, constants.HttpsStr}
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
	confirm := false
	prompt := &survey.Confirm{Message: "Restart HTTP/S jobs?"}
	survey.AskOne(prompt, &confirm)
	if confirm {
		var restartJobReq clientpb.RestartJobReq
		jobs, err := con.Rpc.GetJobs(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		// reload jobs to include new profile
		for _, job := range jobs.Active {
			if job != nil && slices.Contains(protocols, job.Name) {
				restartJobReq.JobIDs = append(restartJobReq.JobIDs, job.ID)
			}
		}

		_, err = con.Rpc.RestartJobs(context.Background(), &restartJobReq)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
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
			profileName = selectC2Profile(httpC2Profiles.Configs)
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

	c2Profiles, err := con.Rpc.GetHTTPC2Profiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	var extensions []string
	for _, c2profile := range c2Profiles.Configs {
		confProfile, err := con.Rpc.GetHTTPC2ProfileByName(context.Background(), &clientpb.C2ProfileReq{Name: c2profile.Name})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		extensions = append(extensions, confProfile.ImplantConfig.StagerFileExtension)
		extensions = append(extensions, confProfile.ImplantConfig.StartSessionFileExtension)
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

	jsonProfile, err := updateC2Profile(extensions, config, urls)
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
		PrintC2Profiles(profile, con, "")
	}
}

// convert protobuf to json
func C2ConfigToJSON(profileName string, profile *clientpb.HTTPC2Config) (*assets.HTTPC2Config, error) {
	implantConfig := assets.HTTPC2ImplantConfig{
		UserAgent:           profile.ImplantConfig.UserAgent,
		ChromeBaseVersion:   int(profile.ImplantConfig.ChromeBaseVersion),
		MacOSVersion:        profile.ImplantConfig.MacOSVersion,
		NonceQueryArgChars:  profile.ImplantConfig.NonceQueryArgChars,
		MaxFiles:            int(profile.ImplantConfig.MaxFiles),
		MinFiles:            int(profile.ImplantConfig.MinFiles),
		MaxPaths:            int(profile.ImplantConfig.MaxPaths),
		MinPaths:            int(profile.ImplantConfig.MinFiles),
		StagerFileExt:       profile.ImplantConfig.StagerFileExtension,
		PollFileExt:         profile.ImplantConfig.PollFileExtension,
		StartSessionFileExt: profile.ImplantConfig.StartSessionFileExtension,
		SessionFileExt:      profile.ImplantConfig.SessionFileExtension,
		CloseFileExt:        profile.ImplantConfig.CloseFileExtension,
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
		stagerFiles  []string
		pollFiles    []string
		sessionFiles []string
		closeFiles   []string
		stagerPaths  []string
		pollPaths    []string
		sessionPaths []string
		closePaths   []string
	)

	for _, pathSegment := range profile.ImplantConfig.PathSegments {
		if pathSegment.IsFile {
			switch pathSegment.SegmentType {
			case 0:
				pollFiles = append(pollFiles, pathSegment.Value)
			case 1:
				sessionFiles = append(sessionFiles, pathSegment.Value)
			case 2:
				closeFiles = append(closeFiles, pathSegment.Value)
			case 3:
				stagerFiles = append(stagerFiles, pathSegment.Value)
			}
		} else {
			switch pathSegment.SegmentType {
			case 0:
				pollPaths = append(pollPaths, pathSegment.Value)
			case 1:
				sessionPaths = append(sessionPaths, pathSegment.Value)
			case 2:
				closePaths = append(closePaths, pathSegment.Value)
			case 3:
				stagerPaths = append(stagerPaths, pathSegment.Value)
			}
		}
	}

	implantConfig.PollFiles = pollFiles
	implantConfig.SessionFiles = sessionFiles
	implantConfig.CloseFiles = closeFiles
	implantConfig.StagerFiles = stagerFiles
	implantConfig.PollPaths = pollPaths
	implantConfig.SessionPaths = sessionPaths
	implantConfig.ClosePaths = closePaths
	implantConfig.StagerPaths = stagerPaths

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
	for _, poll := range config.ImplantConfig.PollFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 0,
			Value:       poll,
		})
	}

	for _, session := range config.ImplantConfig.SessionFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 1,
			Value:       session,
		})
	}

	for _, close := range config.ImplantConfig.CloseFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 2,
			Value:       close,
		})
	}

	for _, stager := range config.ImplantConfig.StagerFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 3,
			Value:       stager,
		})
	}

	// paths
	for _, poll := range config.ImplantConfig.PollPaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 0,
			Value:       poll,
		})
	}

	for _, session := range config.ImplantConfig.SessionPaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 1,
			Value:       session,
		})
	}

	for _, close := range config.ImplantConfig.ClosePaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 2,
			Value:       close,
		})
	}

	for _, stager := range config.ImplantConfig.StagerPaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 3,
			Value:       stager,
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
		UserAgent:                 config.ImplantConfig.UserAgent,
		ChromeBaseVersion:         int32(config.ImplantConfig.ChromeBaseVersion),
		MacOSVersion:              config.ImplantConfig.MacOSVersion,
		NonceQueryArgChars:        config.ImplantConfig.NonceQueryArgChars,
		ExtraURLParameters:        httpC2UrlParameters,
		Headers:                   httpC2Headers,
		MaxFiles:                  int32(config.ImplantConfig.MaxFiles),
		MinFiles:                  int32(config.ImplantConfig.MinFiles),
		MaxPaths:                  int32(config.ImplantConfig.MaxPaths),
		MinPaths:                  int32(config.ImplantConfig.MinFiles),
		StagerFileExtension:       config.ImplantConfig.StagerFileExt,
		PollFileExtension:         config.ImplantConfig.PollFileExt,
		StartSessionFileExtension: config.ImplantConfig.StartSessionFileExt,
		SessionFileExtension:      config.ImplantConfig.SessionFileExt,
		CloseFileExtension:        config.ImplantConfig.CloseFileExt,
		PathSegments:              pathSegments,
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
func PrintC2Profiles(profile *clientpb.HTTPC2Config, con *console.SliverClient, filter string) {

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

	var serverCookies []string
	for _, cookie := range profile.ServerConfig.Cookies {
		serverCookies = append(serverCookies, cookie.Name)
	}

	// Client side configuration
	var clientHeaders []string
	for _, header := range profile.ImplantConfig.Headers {
		clientHeaders = append(clientHeaders, header.Name)
	}

	var clientUrlParams []string
	for _, clientUrlParam := range profile.ImplantConfig.ExtraURLParameters {
		clientUrlParams = append(clientUrlParams, clientUrlParam.Name)
	}

	var (
		pollPaths    []string
		pollFiles    []string
		sessionPaths []string
		sessionFiles []string
		closePaths   []string
		closeFiles   []string
	)
	for _, segment := range profile.ImplantConfig.PathSegments {
		if segment.IsFile {
			switch segment.SegmentType {
			case 0:
				pollFiles = append(pollFiles, segment.Value)
			case 1:
				sessionFiles = append(sessionFiles, segment.Value)
			case 2:
				closeFiles = append(closeFiles, segment.Value)
			}
		} else {
			switch segment.SegmentType {
			case 0:
				pollPaths = append(pollPaths, segment.Value)
			case 1:
				sessionPaths = append(sessionPaths, segment.Value)
			case 2:
				closePaths = append(closePaths, segment.Value)
			}
		}
	}

	var displayName []string
	var values []string

	switch filter {
	case "extensions":
		displayName = []string{
			"Stager file extension",
			"Start session file extension",
			"Session file extension",
			"Poll file extension",
			"Close file extension",
		}
		values = []string{
			profile.ImplantConfig.StagerFileExtension,
			profile.ImplantConfig.StartSessionFileExtension,
			profile.ImplantConfig.SessionFileExtension,
			profile.ImplantConfig.PollFileExtension,
			profile.ImplantConfig.CloseFileExtension,
		}

	case "headers":
		displayName = []string{
			"Server Headers",
			"Clients Headers",
		}
		values = []string{
			strings.Join(serverHeaders[:], ","),
			strings.Join(clientHeaders[:], ","),
		}
	case "poll":
		displayName = []string{
			"Poll paths",
			"Poll files",
		}
		values = []string{
			strings.Join(pollPaths[:], ","),
			strings.Join(pollFiles[:], ","),
		}
	case "session":
		displayName = []string{
			"Session paths",
			"Session files",
		}
		values = []string{
			strings.Join(sessionPaths[:], ","),
			strings.Join(sessionFiles[:], ","),
		}
	case "close":
		displayName = []string{
			"Close paths",
			"Close files",
		}
		values = []string{
			strings.Join(closePaths[:], ","),
			strings.Join(closeFiles[:], ","),
		}
	default:
		displayName = []string{
			"Server Headers",
			"Server Cookies",
			"Randomize Server Headers",
			"Clients Headers",
			"Extra URL Parameters",
			"User agent",
			"Chrome base version",
			"MacOS version",
			"Nonce query arg chars",
			"Max files",
			"Min files",
			"Max paths",
			"Min paths",
			"Stager file extension",
			"Start session file extension",
			"Session file extension",
			"Poll file extension",
			"Close file extension",
			"Poll paths",
			"Poll files",
			"Session paths",
			"Session files",
			"Close paths",
			"Close files",
		}
		values = []string{
			strings.Join(serverHeaders[:], ","),
			strings.Join(serverCookies[:], ","),
			strconv.FormatBool(profile.ServerConfig.RandomVersionHeaders),
			strings.Join(clientHeaders[:], ","),
			strings.Join(clientUrlParams[:], ","),
			profile.ImplantConfig.UserAgent,
			fmt.Sprintf("%d", profile.ImplantConfig.ChromeBaseVersion),
			profile.ImplantConfig.MacOSVersion,
			profile.ImplantConfig.NonceQueryArgChars,
			fmt.Sprintf("%d", profile.ImplantConfig.MaxFiles),
			fmt.Sprintf("%d", profile.ImplantConfig.MinFiles),
			fmt.Sprintf("%d", profile.ImplantConfig.MaxPaths),
			fmt.Sprintf("%d", profile.ImplantConfig.MinPaths),
			profile.ImplantConfig.StagerFileExtension,
			profile.ImplantConfig.StartSessionFileExtension,
			profile.ImplantConfig.SessionFileExtension,
			profile.ImplantConfig.PollFileExtension,
			profile.ImplantConfig.CloseFileExtension,
			strings.Join(pollPaths[:], ","),
			strings.Join(pollFiles[:], ","),
			strings.Join(sessionPaths[:], ","),
			strings.Join(sessionFiles[:], ","),
			strings.Join(closePaths[:], ","),
			strings.Join(closeFiles[:], ","),
		}
	}

	for i := range displayName {
		tw.AppendRow(table.Row{
			displayName[i],
			values[i],
		})
	}

	con.Println(tw.Render())
	con.Println("\n")
}

func selectC2Profile(c2profiles []*clientpb.HTTPC2Config) string {
	c2profile := ""
	var choices []string
	for _, c2profile := range c2profiles {
		choices = append(choices, c2profile.Name)
	}

	prompt := &survey.Select{
		Message: "Select a c2 profile",
		Options: choices,
	}
	survey.AskOne(prompt, &c2profile, nil)

	return c2profile
}

func updateC2Profile(usedExtensions []string, template *assets.HTTPC2Config, urls []string) (*assets.HTTPC2Config, error) {
	// update the template with the urls

	var (
		paths              []string
		filenames          []string
		extensions         []string
		filteredExtensions []string
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

	for _, extension := range extensions {
		if !slices.Contains(usedExtensions, extension) {
			filteredExtensions = append(filteredExtensions, extension)
		}
	}

	slices.Sort(paths)
	paths = slices.Compact(paths)

	slices.Sort(filenames)
	filenames = slices.Compact(filenames)

	// 5 is arbitrarily used as a minimum value, it only has to be 5 for the extensions, the others can be lower
	if len(filteredExtensions) < 5 {
		return nil, fmt.Errorf("got %d unused extensions, need at least 5", len(filteredExtensions))
	}

	if len(paths) < 5 {
		return nil, fmt.Errorf("got %d paths need at least 5", len(paths))
	}

	if len(filenames) < 5 {
		return nil, fmt.Errorf("got %d paths need at least 5", len(filenames))
	}

	// shuffle extensions
	for i := len(extensions) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		extensions[i], extensions[j] = extensions[j], extensions[i]
	}

	template.ImplantConfig.PollFileExt = extensions[0]
	template.ImplantConfig.StagerFileExt = extensions[1]
	template.ImplantConfig.StartSessionFileExt = extensions[2]
	template.ImplantConfig.SessionFileExt = extensions[3]
	template.ImplantConfig.CloseFileExt = extensions[4]

	// randomly distribute the paths and filenames into the different segment types
	template.ImplantConfig.CloseFiles = []string{}
	template.ImplantConfig.SessionFiles = []string{}
	template.ImplantConfig.PollFiles = []string{}
	template.ImplantConfig.StagerFiles = []string{}
	template.ImplantConfig.ClosePaths = []string{}
	template.ImplantConfig.SessionPaths = []string{}
	template.ImplantConfig.PollPaths = []string{}
	template.ImplantConfig.StagerPaths = []string{}

	for _, path := range paths {
		switch rand.Intn(4) {
		case 0:
			template.ImplantConfig.PollPaths = append(template.ImplantConfig.PollPaths, path)
		case 1:
			template.ImplantConfig.SessionPaths = append(template.ImplantConfig.SessionPaths, path)
		case 2:
			template.ImplantConfig.ClosePaths = append(template.ImplantConfig.ClosePaths, path)
		case 3:
			template.ImplantConfig.StagerPaths = append(template.ImplantConfig.StagerPaths, path)
		}
	}

	for _, filename := range filenames {
		switch rand.Intn(4) {
		case 0:
			template.ImplantConfig.PollFiles = append(template.ImplantConfig.PollFiles, filename)
		case 1:
			template.ImplantConfig.SessionFiles = append(template.ImplantConfig.SessionFiles, filename)
		case 2:
			template.ImplantConfig.CloseFiles = append(template.ImplantConfig.CloseFiles, filename)
		case 3:
			template.ImplantConfig.StagerFiles = append(template.ImplantConfig.StagerFiles, filename)
		}
	}

	return template, nil
}
