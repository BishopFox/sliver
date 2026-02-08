package console

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/bishopfox/sliver/client/theme"
	"github.com/charmbracelet/lipgloss"
)

type promptPalette struct {
	Default lipgloss.Color
	Mods    map[int]lipgloss.Color
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
		Default: lipgloss.Color(p.Default),
		Mods:    map[int]lipgloss.Color{},
	}
	for k, v := range p.Mods {
		out.Mods[k] = lipgloss.Color(v)
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
