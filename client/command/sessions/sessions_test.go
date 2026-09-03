package sessions

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/command/targettable"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

var sessionsTestNow = time.Date(2026, time.September, 3, 14, 25, 41, 0, time.UTC)

func TestRenderSessionsKeepsFullProcessPathWhenItFits(t *testing.T) {
	session := sessionsTestSession("windows", `C:\Users\vagrant\AppData\Local\Temp\sliver.exe`)
	sessions := sessionMap(session)
	con := sessionsTestConsole()
	hasIntegrity := true
	fullLayout := targettable.Layout{IncludeIntegrity: true, IncludeLocale: true}
	width := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, fullLayout, hasIntegrity, sessionsTestNow))

	rendered := renderedSessionTable(renderSessions(sessions, "", nil, con, width, sessionsTestNow))
	if !strings.Contains(rendered, session.Filename+" (456)") {
		t.Fatalf("full process path was not retained:\n%s", rendered)
	}
	assertSessionTableFits(t, rendered, width)
}

func TestRenderSessionsUsesTargetProcessBasenameBeforeOmittingColumns(t *testing.T) {
	tests := []struct {
		name         string
		operatingSys string
		filename     string
		basename     string
		hasIntegrity bool
	}{
		{
			name:         "windows path",
			operatingSys: "windows",
			filename:     `C:\Users\vagrant\AppData\Local\Temp\a-very-long-directory-name\sliver.exe`,
			basename:     "sliver.exe",
			hasIntegrity: true,
		},
		{
			name:         "POSIX path",
			operatingSys: "linux",
			filename:     "/opt/sliver/a-very-long-directory-name/implant/sliver",
			basename:     "sliver",
			hasIntegrity: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := sessionsTestSession(test.operatingSys, test.filename)
			sessions := sessionMap(session)
			con := sessionsTestConsole()
			fullLayout := targettable.Layout{IncludeIntegrity: test.hasIntegrity, IncludeLocale: true}
			basenameLayout := targettable.Layout{ProcessBasename: true, IncludeIntegrity: test.hasIntegrity, IncludeLocale: true}
			fullWidth := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, fullLayout, test.hasIntegrity, sessionsTestNow))
			basenameWidth := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, basenameLayout, test.hasIntegrity, sessionsTestNow))
			if basenameWidth >= fullWidth {
				t.Fatalf("test fixture does not make basename table narrower: basename=%d full=%d", basenameWidth, fullWidth)
			}

			rendered := renderedSessionTable(renderSessions(sessions, "", nil, con, basenameWidth, sessionsTestNow))
			if !strings.Contains(rendered, test.basename+" (456)") {
				t.Fatalf("process basename was not rendered:\n%s", rendered)
			}
			if strings.Contains(rendered, test.filename) {
				t.Fatalf("full process path was rendered after basename fallback:\n%s", rendered)
			}
			if !strings.Contains(rendered, "Locale") {
				t.Fatalf("Locale was omitted before it was needed:\n%s", rendered)
			}
			if test.hasIntegrity && !strings.Contains(rendered, "Integrity") {
				t.Fatalf("Integrity was omitted before it was needed:\n%s", rendered)
			}
			assertSessionTableFits(t, rendered, basenameWidth)
		})
	}
}

func TestRenderSessionsOmitsIntegrityBeforeLocale(t *testing.T) {
	session := sessionsTestSession("windows", `C:\Users\vagrant\AppData\Local\Temp\a-very-long-directory-name\sliver.exe`)
	session.Integrity = "System-With-An-Extremely-Long-Integrity-Label"
	session.Locale = "extraordinarily-long-locale"
	sessions := sessionMap(session)
	con := sessionsTestConsole()
	hasIntegrity := true

	withoutIntegrity := targettable.Layout{ProcessBasename: true, IncludeLocale: true}
	withoutIntegrityOrLocale := targettable.Layout{ProcessBasename: true}
	withoutIntegrityWidth := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, withoutIntegrity, hasIntegrity, sessionsTestNow))
	withoutIntegrityOrLocaleWidth := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, withoutIntegrityOrLocale, hasIntegrity, sessionsTestNow))
	if withoutIntegrityOrLocaleWidth >= withoutIntegrityWidth {
		t.Fatalf("test fixture does not make Locale omission narrower: without Locale=%d with Locale=%d", withoutIntegrityOrLocaleWidth, withoutIntegrityWidth)
	}

	t.Run("Integrity omitted first", func(t *testing.T) {
		rendered := renderedSessionTable(renderSessions(sessions, "", nil, con, withoutIntegrityWidth, sessionsTestNow))
		if strings.Contains(rendered, "Integrity") || strings.Contains(rendered, session.Integrity) {
			t.Fatalf("Integrity column was retained:\n%s", rendered)
		}
		if !strings.Contains(rendered, "Locale") || !strings.Contains(rendered, session.Locale) {
			t.Fatalf("Locale should remain after Integrity is omitted:\n%s", rendered)
		}
		if !strings.Contains(rendered, "Process (PID)") {
			t.Fatalf("renderer unexpectedly used the compact fallback:\n%s", rendered)
		}
		assertSessionTableFits(t, rendered, withoutIntegrityWidth)
	})

	t.Run("Locale omitted second", func(t *testing.T) {
		rendered := renderedSessionTable(renderSessions(sessions, "", nil, con, withoutIntegrityOrLocaleWidth, sessionsTestNow))
		if strings.Contains(rendered, "Integrity") || strings.Contains(rendered, "Locale") {
			t.Fatalf("optional columns were retained:\n%s", rendered)
		}
		if !strings.Contains(rendered, "Process (PID)") {
			t.Fatalf("renderer unexpectedly used the compact fallback:\n%s", rendered)
		}
		assertSessionTableFits(t, rendered, withoutIntegrityOrLocaleWidth)
	})
}

func TestRenderSessionsUsesDirectionalRelativeLastMessage(t *testing.T) {
	tests := []struct {
		name        string
		lastCheckin time.Time
		want        string
	}{
		{name: "past", lastCheckin: sessionsTestNow.Add(-36 * time.Second), want: "36s ago"},
		{name: "future clock skew", lastCheckin: sessionsTestNow.Add(5 * time.Second), want: "in 5s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := sessionsTestSession("linux", "/opt/sliver/sliver")
			session.LastCheckin = test.lastCheckin.Unix()
			rendered := renderedSessionTable(renderSessions(sessionMap(session), "", nil, sessionsTestConsole(), 10_000, sessionsTestNow))
			if !strings.Contains(rendered, test.want) {
				t.Fatalf("relative Last Message %q was not rendered:\n%s", test.want, rendered)
			}
			if strings.Contains(rendered, "2026") || strings.Contains(rendered, test.lastCheckin.Format(time.UnixDate)) {
				t.Fatalf("Last Message unexpectedly contains a calendar date:\n%s", rendered)
			}
		})
	}
}

func TestRenderSessionsOmitsIntegrityForNonWindowsSessions(t *testing.T) {
	session := sessionsTestSession("linux", "/opt/sliver/sliver")
	rendered := renderedSessionTable(renderSessions(sessionMap(session), "", nil, sessionsTestConsole(), 10_000, sessionsTestNow))
	if strings.Contains(rendered, "Integrity") {
		t.Fatalf("non-Windows sessions table contains an Integrity column:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Locale") {
		t.Fatalf("wide non-Windows sessions table is missing Locale:\n%s", rendered)
	}
}

func TestRenderSessionsCompactFallbackPreservesLegacyColumns(t *testing.T) {
	session := sessionsTestSession("windows", `C:\Users\vagrant\AppData\Local\Temp\sliver.exe`)
	sessions := sessionMap(session)
	con := sessionsTestConsole()
	compactWidth := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, targettable.Layout{Compact: true}, true, sessionsTestNow))
	withoutOptionalWidth := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, targettable.Layout{ProcessBasename: true}, true, sessionsTestNow))
	if compactWidth >= withoutOptionalWidth {
		t.Fatalf("test fixture does not require compact fallback: compact=%d detailed=%d", compactWidth, withoutOptionalWidth)
	}
	rendered := renderedSessionTable(renderSessions(sessions, "", nil, con, compactWidth, sessionsTestNow))
	header := strings.SplitN(rendered, "\n", 2)[0]

	for _, retained := range []string{"ID", "Transport", "Remote Address", "Hostname", "Username", "Operating System", "Health"} {
		if !strings.Contains(header, retained) {
			t.Fatalf("compact fallback is missing legacy column %q:\n%s", retained, rendered)
		}
	}
	for _, omitted := range []string{"Process (PID)", "Integrity", "Locale", "Last Message"} {
		if strings.Contains(header, omitted) {
			t.Fatalf("compact fallback unexpectedly contains column %q:\n%s", omitted, rendered)
		}
	}
	if strings.Contains(rendered, session.Name) {
		t.Fatalf("compact fallback unexpectedly contains the session name:\n%s", rendered)
	}
	assertSessionTableFits(t, rendered, compactWidth)
}

func TestRenderSessionsFiltersCanonicalValuesHiddenByLayout(t *testing.T) {
	session := sessionsTestSession("windows", `C:\hidden-directory-token\sliver.exe`)
	session.Integrity = "hidden-integrity-token"
	session.Locale = "hidden-locale-token"
	sessions := sessionMap(session)
	con := sessionsTestConsole()
	layout := targettable.Layout{ProcessBasename: true}
	width := renderedSessionTableWidth(buildSessionsTable(sessions, "", nil, con, layout, true, sessionsTestNow))

	tests := []struct {
		name        string
		filter      string
		filterRegex *regexp.Regexp
		hiddenValue string
	}{
		{name: "substring in shortened path", filter: "hidden-directory-token", hiddenValue: "hidden-directory-token"},
		{name: "regexp in omitted columns", filterRegex: regexp.MustCompile(`hidden-(?:integrity|locale)-token`), hiddenValue: "hidden-integrity-token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered := renderedSessionTable(renderSessions(sessions, test.filter, test.filterRegex, con, width, sessionsTestNow))
			if !strings.Contains(rendered, ShortSessionID(session.ID)) {
				t.Fatalf("session matching a canonical value was filtered out:\n%s", rendered)
			}
			if strings.Contains(rendered, test.hiddenValue) {
				t.Fatalf("hidden canonical value leaked into rendered layout:\n%s", rendered)
			}
			if !strings.Contains(rendered, "Process (PID)") || strings.Contains(rendered, "Locale") || strings.Contains(rendered, "Integrity") {
				t.Fatalf("test did not exercise the shortened, omitted-column layout:\n%s", rendered)
			}
			assertSessionTableFits(t, rendered, width)
		})
	}
}

func sessionsTestSession(operatingSystem string, filename string) *clientpb.Session {
	return &clientpb.Session{
		ID:            "01234567-89ab-cdef-0123-456789abcdef",
		Name:          "winterfell-session",
		Transport:     "mtls",
		RemoteAddress: "10.13.37.10:61839",
		Hostname:      "winterfell",
		Username:      `winterfell\vagrant`,
		Filename:      filename,
		PID:           456,
		Integrity:     "High",
		OS:            operatingSystem,
		Arch:          "amd64",
		Locale:        "en-US",
		LastCheckin:   sessionsTestNow.Add(-36 * time.Second).Unix(),
	}
}

func sessionsTestConsole() *console.SliverClient {
	return &console.SliverClient{
		Settings: &assets.ClientSettings{TableStyle: settings.SliverDefault.Name},
	}
}

func sessionMap(sessions ...*clientpb.Session) map[string]*clientpb.Session {
	result := make(map[string]*clientpb.Session, len(sessions))
	for _, session := range sessions {
		result[session.ID] = session
	}
	return result
}

func renderedSessionTable(writer table.Writer) string {
	return text.StripEscape(writer.Render())
}

func renderedSessionTableWidth(writer table.Writer) int {
	return text.LongestLineLen(writer.Render())
}

func assertSessionTableFits(t *testing.T, rendered string, width int) {
	t.Helper()
	if got := text.LongestLineLen(rendered); got > width {
		t.Fatalf("rendered table width %d exceeds terminal width %d:\n%s", got, width, rendered)
	}
}
