package console

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/bishopfox/sliver/client/theme"
)

type promptPalette struct {
	Default string
	Mods    map[int]string
}

type promptColors struct {
	Primary   promptPalette
	Secondary promptPalette
	Default   promptPalette
	Success   promptPalette
	Warning   promptPalette
	Danger    promptPalette
}

type promptStyles struct {
	Normal    TextStyle
	Bold      TextStyle
	Underline TextStyle

	Primary   TextStyle
	Secondary TextStyle
	Default   TextStyle
	Success   TextStyle
	Warning   TextStyle
	Danger    TextStyle

	BoldPrimary   TextStyle
	BoldSecondary TextStyle
	BoldDefault   TextStyle
	BoldSuccess   TextStyle
	BoldWarning   TextStyle
	BoldDanger    TextStyle

	// Legacy names used across the codebase.
	Black  TextStyle
	Red    TextStyle
	Green  TextStyle
	Orange TextStyle
	Blue   TextStyle
	Purple TextStyle
	Cyan   TextStyle
	Gray   TextStyle
}

type promptTarget struct {
	SessionName string
	BeaconName  string

	// Suffix is a pre-rendered, themed " (name)" segment matching the default prompt.
	Suffix string
}

type promptTemplateContext struct {
	Now time.Time

	IsServer  bool
	Connected bool

	Operator string
	Host     string
	Port     int
	HostPort string

	Target promptTarget

	Colors promptColors
	Styles promptStyles
}

var promptTemplateCache struct {
	mu  sync.RWMutex
	src string
	tpl *template.Template
}

func toPromptPalette(p theme.Palette) promptPalette {
	out := promptPalette{
		Default: p.Default,
		Mods:    map[int]string{},
	}
	for k, v := range p.Mods {
		out.Mods[k] = v
	}
	return out
}

func promptTemplateForSource(src string) (*template.Template, error) {
	promptTemplateCache.mu.RLock()
	if promptTemplateCache.tpl != nil && promptTemplateCache.src == src {
		t := promptTemplateCache.tpl
		promptTemplateCache.mu.RUnlock()
		return t, nil
	}
	promptTemplateCache.mu.RUnlock()

	t, err := template.New("sliver-prompt").Option("missingkey=zero").Parse(src)
	if err != nil {
		return nil, err
	}

	promptTemplateCache.mu.Lock()
	promptTemplateCache.src = src
	promptTemplateCache.tpl = t
	promptTemplateCache.mu.Unlock()

	return t, nil
}

func renderPromptTemplate(con *SliverClient, src string) (string, error) {
	if con == nil {
		return "", nil
	}

	details, _, ok := con.CurrentConnection()
	operator := ""
	host := ""
	port := 0
	hostPort := ""
	if ok && details != nil && details.Config != nil {
		operator = strings.TrimSpace(details.Config.Operator)
		host = strings.TrimSpace(details.Config.LHost)
		port = int(details.Config.LPort)
	}
	if host != "" && port != 0 {
		hostPort = fmt.Sprintf("%s:%d", host, port)
	}

	// Active target context.
	var suffix string
	sessionName := ""
	beaconName := ""
	if session := con.ActiveTarget.GetSession(); session != nil {
		sessionName = session.Name
		suffix = StyleBoldRed.Render(" (" + sessionName + ")")
	} else if beacon := con.ActiveTarget.GetBeacon(); beacon != nil {
		beaconName = beacon.Name
		suffix = StyleBoldBlue.Render(" (" + beaconName + ")")
	}

	th := theme.Current()
	ctx := promptTemplateContext{
		Now:       time.Now(),
		IsServer:  con.IsServer,
		Connected: ok && details != nil && details.Config != nil,
		Operator:  operator,
		Host:      host,
		Port:      port,
		HostPort:  hostPort,
		Target: promptTarget{
			SessionName: sessionName,
			BeaconName:  beaconName,
			Suffix:      suffix,
		},
		Colors: promptColors{
			Primary:   toPromptPalette(th.Primary),
			Secondary: toPromptPalette(th.Secondary),
			Default:   toPromptPalette(th.Default),
			Success:   toPromptPalette(th.Success),
			Warning:   toPromptPalette(th.Warning),
			Danger:    toPromptPalette(th.Danger),
		},
		Styles: promptStyles{
			Normal:    StyleNormal,
			Bold:      StyleBold,
			Underline: StyleUnderline,

			Primary:   StylePrimary,
			Secondary: StyleSecondary,
			Default:   StyleDefault,
			Success:   StyleSuccess,
			Warning:   StyleWarning,
			Danger:    StyleDanger,

			BoldPrimary:   StyleBoldPrimary,
			BoldSecondary: StyleBoldSecondary,
			BoldDefault:   StyleBoldDefault,
			BoldSuccess:   StyleBoldSuccess,
			BoldWarning:   StyleBoldWarning,
			BoldDanger:    StyleBoldDanger,

			Black:  StyleBlack,
			Red:    StyleRed,
			Green:  StyleGreen,
			Orange: StyleOrange,
			Blue:   StyleBlue,
			Purple: StylePurple,
			Cyan:   StyleCyan,
			Gray:   StyleGray,
		},
	}

	tpl, err := promptTemplateForSource(src)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}
