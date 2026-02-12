package assets

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2019  Bishop Fox
	版权所有 (C) 2019 Bishop Fox

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

// HTTPC2Config - Parent config file struct for implant/server
// HTTPC2Config - implant/server 的父级配置文件结构体
type HTTPC2Config struct {
	ImplantConfig HTTPC2ImplantConfig `json:"implant_config"`
	ServerConfig  HTTPC2ServerConfig  `json:"server_config"`
}

// HTTPC2ServerConfig - Server configuration options
// HTTPC2ServerConfig - server 配置选项
type HTTPC2ServerConfig struct {
	RandomVersionHeaders bool                   `json:"random_version_headers"`
	Headers              []NameValueProbability `json:"headers"`
	Cookies              []string               `json:"cookies"`
}

type NameValueProbability struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Probability int    `json:"probability"`
	Method      string `json:"method"`
}

// HTTPC2ImplantConfig - Implant configuration options
// HTTPC2ImplantConfig - implant 配置选项
// Procedural C2
// 过程式 C2
// ===============
// ===============
// .txt = rsakey
// .txt = rsakey
// .css = start
// .css = 启动
// .php = session
// .php = session
//
//	.js = poll
//	.js = 轮询
//
// .png = stop
// .png = 停止
// .woff = sliver shellcode
// .woff = sliver shellcode
type HTTPC2ImplantConfig struct {
	UserAgent         string `json:"user_agent"`
	ChromeBaseVersion int    `json:"chrome_base_version"`
	MacOSVersion      string `json:"macos_version"`

	NonceQueryArgChars string                 `json:"nonce_query_args"`
	URLParameters      []NameValueProbability `json:"url_parameters"`
	Headers            []NameValueProbability `json:"headers"`
	NonceQueryLength   int                    `json:"nonce_query_length"`
	NonceMode          string                 `json:"nonce_mode"`

	MaxFileGen    int `json:"max_files"`
	MinFileGen    int `json:"min_files"`
	MaxPathGen    int `json:"max_paths"`
	MinPathGen    int `json:"min_paths"`
	MaxPathLength int `json:"max_path_length"`
	MinPathLength int `json:"min_path_length"`

	Extensions []string `json:"extensions"`
	Files      []string `json:"files"`
	Paths      []string `json:"paths"`
}
