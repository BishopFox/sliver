package alias

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2021  Bishop Fox
	版权所有 (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	本程序是自由软件：你可以再发布和/或修改它
	it under the terms of the GNU General Public License as published by
	在自由软件基金会发布的 GNU General Public License 条款下，
	the Free Software Foundation, either version 3 of the License, or
	可以使用许可证第 3 版，或
	(at your option) any later version.
	（由你选择）任何更高版本。

	This program is distributed in the hope that it will be useful,
	发布本程序是希望它能发挥作用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但不提供任何担保；甚至不包括
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	对适销性或特定用途适用性的默示担保。请参阅
	GNU General Public License for more details.
	GNU General Public License 以获取更多细节。

	You should have received a copy of the GNU General Public License
	你应当已随本程序收到一份 GNU General Public License 副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	如果没有，请参见 <https://www.gnu.org/licenses/>。
*/

import "testing"

const (
	sample1 = `{
	"name": "test1",
	"command_name": "test1",
	"version": "1.0.0",
	"repo_url": "https://example.com/",
	"help": "some help",
	"entrypoint": "foo",
	"allow_args": true,
	"default_args": "bar",
	"is_reflective": true,
	"is_assembly": true,
	"files": [
		{
			"os": "windows",
			"arch": "amd64",
			"path": "foo/test1.dll"
		}
	]
}`
)

func TestParseAliasManifest(t *testing.T) {
	alias1, err := ParseAliasManifest([]byte(sample1))
	if err != nil {
		t.Errorf("Error parsing alias manifest: %s", err)
	}

	if alias1.Name != "test1" {
		t.Errorf("Expected name to be 'test1', got '%s'", alias1.Name)
	}

	if alias1.CommandName != "test1" {
		t.Errorf("Expected command_name to be 'test1', got '%s'", alias1.CommandName)
	}

	if alias1.Version != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got '%s'", alias1.Version)
	}

	if alias1.RepoURL != "https://example.com/" {
		t.Errorf("Expected repo_url to be 'https://example.com/', got '%s'", alias1.RepoURL)
	}

	if alias1.Help != "some help" {
		t.Errorf("Expected help to be 'some help', got '%s'", alias1.Help)
	}

	if alias1.Entrypoint != "foo" {
		t.Errorf("Expected entrypoint to be 'foo', got '%s'", alias1.Entrypoint)
	}

	if !alias1.AllowArgs {
		t.Errorf("Expected allow_args to be true, got false")
	}

	if alias1.DefaultArgs != "bar" {
		t.Errorf("Expected default_args to be 'bar', got '%s'", alias1.DefaultArgs)
	}

	if !alias1.IsReflective {
		t.Errorf("Expected is_reflective to be true, got false")
	}

	if !alias1.IsAssembly {
		t.Errorf("Expected is_assembly to be true, got false")
	}
}
