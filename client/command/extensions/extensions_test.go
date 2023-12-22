package extensions

import (
	"encoding/json"
	"testing"
)

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

const (
	sample1 = `{
	"name": "test1",
	"version": "1.0.0",
	"extension_author": "test",
	"original_author": "test",
	"repo_url": "https://example.com/",
	"commands":[
	{
		"command_name": "test1",
		"help": "some help",
		"files": [
			{
				"os": "windows",
				"arch": "amd64",
				"path": "foo/test1.dll"
			}
		]
	}
	]
}`

	sample2 = `{
	"name": "test2",
	"command_name": "test2",
	"help": "some help",
	"files": [
		{
			"os": "windows",
			"arch": "amd64",
			"path": "../../../../foo/test1.dll"
		}
	]
}`
	sample3 = `{
	"name": "test3",
	"version": "1.0.0",
	"extension_author": "test",
	"original_author": "test",
	"repo_url": "https://example.com/",
	"commands": [
		{
			"command_name": "test3",
			"help": "some help",
			"files": [
				{
					"os": "windows",
					"arch": "amd64",
					"path": "foo/test1.dll"
				}
			]
		}
	]
}`

	multicmd = `{
		"name": "example-multientry",
		"version": "0.0.0",
		"extension_author": "cs",
		"original_author": "cs",
		"repo_url": "no",
		"commands": [
			{
				"command_name": "startw",
				"help": "startw",
				"entrypoint": "StartW",
				"files": [
					{
						"os": "windows",
						"arch": "amd64",
						"path": "ex.dll"
					}
				]
			},
			{
				"command_name": "Test2",
				"help": "startw",
				"entrypoint": "Test2",
				"files": [
					{
						"os": "windows",
						"arch": "amd64",
						"path": "ex.dll"
					}
				]
			}
		]
	}`
)

func TestParseExtensionManifest(t *testing.T) {
	extManifest, err := ParseExtensionManifest([]byte(sample1))
	if err != nil {
		t.Fatalf("Error parsing extension manifest: %s", err)
	}
	if extManifest.Name != "test1" {
		t.Errorf("Expected extension name 'test1', got '%s'", extManifest.Name)
	}

	if extManifest.Version != "1.0.0" {
		t.Errorf("Expected extension version '1.0.0', got '%s'", extManifest.Version)
	}
	if extManifest.ExtensionAuthor != "test" {
		t.Errorf("Expected extension author 'test', got '%s'", extManifest.ExtensionAuthor)
	}
	if extManifest.OriginalAuthor != "test" {
		t.Errorf("Expected original author 'test', got '%s'", extManifest.OriginalAuthor)
	}
	if extManifest.RepoURL != "https://example.com/" {
		t.Errorf("Expected repo URL 'https://example.com/', got '%s'", extManifest.RepoURL)
	}
	for _, extCmd := range extManifest.ExtCommand { //should only be a single manfiest here, so should pass
		if extCmd.CommandName != "test1" {
			t.Errorf("Expected extension command name 'test1', got '%s'", extCmd.CommandName)
		}
		if extCmd.Help != "some help" {
			t.Errorf("Expected help 'some help', got '%s'", extCmd.Help)
		}
		if len(extCmd.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(extCmd.Files))
		}
		if extCmd.Files[0].OS != "windows" {
			t.Errorf("Expected OS 'windows', got '%s'", extCmd.Files[0].OS)
		}
		if extCmd.Files[0].Arch != "amd64" {
			t.Errorf("Expected Arch 'amd64', got '%s'", extCmd.Files[0].Arch)
		}
		if extCmd.Files[0].Path != "/foo/test1.dll" {
			t.Errorf("Expected path '/foo/test1.dll', got '%s'", extCmd.Files[0].Path)
		}
	}

	mextManifest2, err := ParseExtensionManifest([]byte(sample2)) //checking old manifests work good too
	if err != nil {
		t.Fatalf("Error parsing extension manifest (2): %s", err)
	}
	if mextManifest2.Name != "test2" {
		t.Errorf("Expected extension name 'test2', got '%s'", mextManifest2.Name)
	}
	for _, extManifest2 := range mextManifest2.ExtCommand {
		if extManifest2.CommandName != "test2" {
			t.Errorf("Expected extension command name 'test2', got '%s'", extManifest2.CommandName)
		}
		if extManifest2.Help != "some help" {
			t.Errorf("Expected help 'some help', got '%s'", extManifest2.Help)
		}
		if len(extManifest2.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(extManifest2.Files))
		}
		if extManifest2.Files[0].OS != "windows" {
			t.Errorf("Expected OS 'windows', got '%s'", extManifest2.Files[0].OS)
		}
		if extManifest2.Files[0].Arch != "amd64" {
			t.Errorf("Expected Arch 'amd64', got '%s'", extManifest2.Files[0].Arch)
		}
		if extManifest2.Files[0].Path != "/foo/test1.dll" {
			t.Errorf("Expected path '/foo/test1.dll', got '%s'", extManifest2.Files[0].Path)
		}
	}

}

func TestParseMultipleCmdManifest(t *testing.T) {
	mextManifest, err := ParseExtensionManifest([]byte(multicmd))
	if err != nil {
		t.Errorf("error parsing manifest: %s", err)
	}
	if mextManifest.Name != "example-multientry" {
		t.Errorf("expected name example-multientry, got %s", mextManifest.Name)
	}

	if mextManifest.ExtCommand[0].CommandName != "startw" {
		t.Errorf("expected commandname startw, got %s", mextManifest.ExtCommand[0].CommandName)
	}
	if mextManifest.ExtCommand[1].CommandName != "Test2" {
		t.Errorf("expected commandname Test2, got %s", mextManifest.ExtCommand[1].CommandName)
	}
	if mextManifest.ExtCommand[0].Entrypoint != "StartW" {
		t.Errorf("expected entrypoint StartW, got %s", mextManifest.ExtCommand[0].Entrypoint)
	}
	if mextManifest.ExtCommand[1].Entrypoint != "Test2" {
		t.Errorf("expected entrypoint Test2, got %s", mextManifest.ExtCommand[1].Entrypoint)
	}
	if mextManifest.ExtCommand[0].Files[0].Path != "/ex.dll" { //path cleaning adds a root path here? I am not sure if this should be a bug or not... works fine in prod
		t.Errorf("expected path ex.dll, got %s", mextManifest.ExtCommand[0].Files[0].Path)
	}
	if mextManifest.ExtCommand[1].Files[0].Path != "/ex.dll" { //path cleaning adds a root path here? I am not sure if this should be a bug or not... works fine in prod
		t.Errorf("expected path ex.dll, got %s", mextManifest.ExtCommand[0].Files[0].Path)
	}
	//maybe some more? args?
}

func TestParseExtensionManifestErrors(t *testing.T) {
	sample3, err := ParseExtensionManifest([]byte(sample3))
	if err != nil {
		t.Fatalf("Failed to parse initial sample3: %s", err)
	}

	missingName := (*sample3)
	missingName.Name = ""
	data, _ := json.Marshal(missingName)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing name error, got none")
	}

	missingCmdName := (*sample3)
	missingCmdName.ExtCommand[0].CommandName = ""
	data, _ = json.Marshal(missingCmdName)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing command name error, got none")
	}

	missingHelp := (*sample3)
	missingHelp.ExtCommand[0].Help = ""
	data, _ = json.Marshal(missingHelp)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing help error, got none")
	}

	missingFiles := (*sample3)
	missingFiles.ExtCommand[0].Files = []*extensionFile{}
	data, _ = json.Marshal(missingFiles)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files error, got none")
	}

	missingFileOS := (*sample3)
	missingFileOS.ExtCommand[0].Files = []*extensionFile{
		{
			OS:   "",
			Arch: "amd64",
			Path: "foo/test1.dll",
		},
	}
	data, _ = json.Marshal(missingFileOS)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files.os error, got none")
	}

	missingFileArch := (*sample3)
	missingFileArch.ExtCommand[0].Files = []*extensionFile{
		{
			OS:   "windows",
			Arch: "",
			Path: "foo/test1.dll",
		},
	}
	data, _ = json.Marshal(missingFileArch)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files.arch error, got none")
	}

	missingFilePath := (*sample3)
	missingFilePath.ExtCommand[0].Files = []*extensionFile{
		{
			OS:   "windows",
			Arch: "amd64",
			Path: "",
		},
	}
	data, _ = json.Marshal(missingFilePath)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files.path error, got none")
	}

	invalidPaths := []string{
		"../../../../../",
		"/../../../../..",
		".",
		"/",
	}
	for _, invalidPath := range invalidPaths {
		missingFilePath2 := (*sample3)
		missingFilePath2.ExtCommand[0].Files = []*extensionFile{
			{
				OS:   "windows",
				Arch: "amd64",
				Path: invalidPath,
			},
		}
		data, _ = json.Marshal(missingFilePath2)
		_, err = ParseExtensionManifest(data)
		if err == nil {
			t.Fatalf("Expected missing files.path error, got none")
		}
	}
}
