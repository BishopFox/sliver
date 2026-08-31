package shell

import "testing"

func TestNormalizeWindowsShellPath(t *testing.T) {
	tests := map[string]string{
		"":                                    "",
		`C:/Windows/System32/cmd.exe`:         `C:\Windows\System32\cmd.exe`,
		`C:\Windows\System32\cmd.exe`:         `C:\Windows\System32\cmd.exe`,
		`//server/share/tools/powershell.exe`: `\\server\share\tools\powershell.exe`,
		`\\?\C:/Windows/System32/WindowsPowerShell/v1.0/powershell.exe`: `\\?\C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
	}
	for input, want := range tests {
		if got := normalizeWindowsShellPath(input); got != want {
			t.Errorf("normalizeWindowsShellPath(%q) = %q, want %q", input, got, want)
		}
	}
}
