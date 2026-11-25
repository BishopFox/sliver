package mage

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	mageCompletionMarker    = "# begin mage tab completion"
	mageCompletionMarkerEnd = "# end mage tab completion"
)

// installCompletion installs shell tab completion for the given shell.
func installCompletion(stdout io.Writer, shell string) error {
	shell = strings.ToLower(strings.TrimSpace(shell))
	switch shell {
	case "bash":
		return installBashCompletion(stdout)
	case "zsh":
		return installZshCompletion(stdout)
	case "fish":
		return installFishCompletion(stdout)
	case "powershell", "pwsh":
		return installPowerShellCompletion(stdout)
	default:
		return fmt.Errorf("unsupported shell %q; supported shells: bash, zsh, fish, powershell (or pwsh)", shell)
	}
}

// mageExePath returns the path to use for the mage executable in generated
// completion scripts. It prefers the unresolved executable path (preserving
// symlinks) so completions survive package-manager upgrades. Falls back to
// "mage" if the path cannot be determined.
func mageExePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "mage"
	}
	return exe
}

// completionConfigDir returns the directory for mage completion config files.
func completionConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mage"), nil
}

// writeCompletionFile writes the completion script to the given path,
// creating parent directories as needed.
func writeCompletionFile(path, content string) error {
	dir := filepath.Dir(path)
	// #nosec -- path is constructed internally from trusted locations.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}
	// #nosec -- path is constructed internally from trusted locations.
	return os.WriteFile(path, []byte(content), 0o600)
}

// addGuardedBlock adds a guarded block of content to the given file.
// If a guarded block already exists, it is replaced. Otherwise the block
// is appended. The file is created if it doesn't exist.
func addGuardedBlock(path, content string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	block := mageCompletionMarker + "\n" + content + "\n" + mageCompletionMarkerEnd

	existingStr := string(existing)
	beforeStart, afterStart, foundStart := strings.Cut(existingStr, mageCompletionMarker)
	if foundStart {
		_, afterEnd, foundEnd := strings.Cut(afterStart, mageCompletionMarkerEnd)
		if foundEnd {
			newContent := beforeStart + block + afterEnd
			// #nosec -- path is constructed internally from trusted locations.
			return os.WriteFile(path, []byte(newContent), 0o600)
		}
	}

	// Append to file, creating parent directories if needed
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o750); mkdirErr != nil {
		return mkdirErr
	}
	f, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if openErr != nil {
		return openErr
	}
	defer func() { _ = f.Close() }()

	// Add a newline before the block if the file is non-empty and doesn't end with one
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		if _, writeErr := f.WriteString("\n"); writeErr != nil {
			return writeErr
		}
	}
	_, writeErr := f.WriteString(block + "\n")
	return writeErr
}

func installBashCompletion(stdout io.Writer) error {
	bin := mageExePath()
	script := bashCompletionScript(bin)

	dir, err := completionConfigDir()
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(dir, "completion.bash")
	if writeErr := writeCompletionFile(scriptPath, script); writeErr != nil {
		return fmt.Errorf("could not write completion script: %w", writeErr)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// On macOS, bash reads .bash_profile for login shells (the default
	// terminal behavior) rather than .bashrc. Use .bashrc if it exists,
	// otherwise fall back to .bash_profile.
	rcFile := filepath.Join(home, ".bashrc")
	if _, err := os.Stat(rcFile); os.IsNotExist(err) {
		rcFile = filepath.Join(home, ".bash_profile")
	}

	sourceLine := fmt.Sprintf(`[ -f '%s' ] && source '%s'`, scriptPath, scriptPath)
	if err := addGuardedBlock(rcFile, sourceLine); err != nil {
		_, _ = fmt.Fprintf(stdout, "Installed bash completion to %s\n", scriptPath)
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "Could not update %s: %v\n", rcFile, err)
		_, _ = fmt.Fprintln(stdout, "To enable, add the following line to your shell profile:")
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "  source '%s'\n", scriptPath)
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Installed bash completion to %s\n", scriptPath)
	_, _ = fmt.Fprintf(stdout, "Updated %s\n", rcFile)
	_, _ = fmt.Fprintf(stdout, "Run 'source %s' or restart your shell to enable completions.\n", rcFile)
	return nil
}

func installZshCompletion(stdout io.Writer) error {
	bin := mageExePath()
	script := zshCompletionScript(bin)

	dir, err := completionConfigDir()
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(dir, "completion.zsh")
	if err := writeCompletionFile(scriptPath, script); err != nil {
		return fmt.Errorf("could not write completion script: %w", err)
	}

	// Honor ZDOTDIR if set, otherwise use $HOME
	zdotdir := os.Getenv("ZDOTDIR")
	if zdotdir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		zdotdir = home
	}

	rcFile := filepath.Join(zdotdir, ".zshrc")
	sourceLine := fmt.Sprintf(`[ -f '%s' ] && source '%s'`, scriptPath, scriptPath)
	if err := addGuardedBlock(rcFile, sourceLine); err != nil {
		_, _ = fmt.Fprintf(stdout, "Installed zsh completion to %s\n", scriptPath)
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "Could not update %s: %v\n", rcFile, err)
		_, _ = fmt.Fprintln(stdout, "To enable, add the following line to your .zshrc:")
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "  source '%s'\n", scriptPath)
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Installed zsh completion to %s\n", scriptPath)
	_, _ = fmt.Fprintf(stdout, "Updated %s\n", rcFile)
	_, _ = fmt.Fprintf(stdout, "Run 'source %s' or restart your shell to enable completions.\n", rcFile)
	return nil
}

func installFishCompletion(stdout io.Writer) error {
	bin := mageExePath()
	script := fishCompletionScript(bin)

	// Honor XDG_CONFIG_HOME if set, otherwise use ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(home, ".config")
	}

	scriptPath := filepath.Join(configDir, "fish", "completions", "mage.fish")
	if err := writeCompletionFile(scriptPath, script); err != nil {
		return fmt.Errorf("could not write completion script: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Installed fish completion to %s\n", scriptPath)
	_, _ = fmt.Fprintf(stdout, "Fish loads completions automatically. Restart your shell or run 'source %s' to enable.\n", scriptPath)
	return nil
}

func installPowerShellCompletion(stdout io.Writer) error {
	bin := mageExePath()
	script := powerShellCompletionScript(bin)

	dir, err := completionConfigDir()
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(dir, "completion.ps1")
	if err := writeCompletionFile(scriptPath, script); err != nil {
		return fmt.Errorf("could not write completion script: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Installed PowerShell completion to %s\n", scriptPath)

	profilePath := discoverPowerShellProfile()
	if profilePath == "" {
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintln(stdout, "Could not detect your PowerShell profile path.")
		_, _ = fmt.Fprintln(stdout, "To enable, add the following line to your PowerShell profile")
		_, _ = fmt.Fprintln(stdout, "(run 'echo $PROFILE' in PowerShell to see the profile path):")
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "  . %q\n", scriptPath)
		return nil
	}

	sourceLine := fmt.Sprintf(". %q", scriptPath)
	if err := addGuardedBlock(profilePath, sourceLine); err != nil {
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "Could not update %s: %v\n", profilePath, err)
		_, _ = fmt.Fprintln(stdout, "To enable, add the following line to your PowerShell profile:")
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintf(stdout, "  . %q\n", scriptPath)
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Updated %s\n", profilePath)
	_, _ = fmt.Fprintln(stdout, "Restart PowerShell to enable completions.")
	return nil
}

// discoverPowerShellProfile attempts to find the PowerShell profile path.
// It tries running pwsh/powershell to query $PROFILE, then falls back to
// well-known default locations.
func discoverPowerShellProfile() string {
	// Try querying pwsh (PowerShell Core) first, then powershell (Windows PowerShell)
	for _, shell := range []string{"pwsh", "powershell"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := exec.CommandContext(ctx, shell, "-NoProfile", "-NonInteractive", "-Command", "echo $PROFILE").Output()
		cancel()
		if err == nil {
			if p := strings.TrimSpace(string(out)); p != "" {
				return p
			}
		}
	}

	// Fall back to well-known default locations
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	if runtime.GOOS == "windows" {
		// Windows PowerShell default
		return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	// PowerShell Core on macOS/Linux
	return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
}

// bashCompletionScript returns a bash completion script that uses mage -autocomplete.
func bashCompletionScript(mageBin string) string {
	return `_mage_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == -* ]]; then
        local flags="-l -h -v -f -debug -t -d -w -keep -compile -clean -init -version -gocmd -goos -goarch -ldflags -autocomplete -install -multiline"
        COMPREPLY=($(compgen -W "$flags" -- "$cur"))
        return
    fi
    local IFS=$'\n'
    COMPREPLY=($(compgen -W "$('` + mageBin + `' -autocomplete 2>/dev/null)" -- "$cur"))
}
complete -F _mage_completions mage
`
}

// zshCompletionScript returns a zsh completion script that uses mage -autocomplete.
func zshCompletionScript(mageBin string) string {
	return `#compdef mage
_mage() {
    local -a targets
    if [[ "$PREFIX" == -* ]]; then
        local -a flags
        flags=(
            '-l:list mage targets in this directory'
            '-h:show this help'
            '-v:show verbose output when running mage targets'
            '-f:force recreation of compiled magefile'
            '-debug:turn on debug messages'
            '-t:timeout in duration parsable format'
            '-d:directory to read magefiles from'
            '-w:working directory where magefiles will run'
            '-keep:keep intermediate mage files around after running'
            '-compile:output a static binary to the given path'
            '-clean:clean out old generated binaries from CACHE_DIR'
            '-init:create a starting template if no mage files exist'
            '-version:show version info for the mage binary'
            '-gocmd:use the given go binary to compile the output'
            '-goos:set GOOS for binary produced with -compile'
            '-goarch:set GOARCH for binary produced with -compile'
            '-ldflags:set ldflags for binary produced with -compile'
            '-autocomplete:print target names for shell completion'
            '-install:install shell completion for the given shell'
            '-multiline:retain line returns in help text'
        )
        _describe 'flag' flags
        return
    fi
    targets=(${(f)"$('` + mageBin + `' -autocomplete 2>/dev/null)"})
    _describe 'target' targets
}
if (( $+functions[compdef] )); then
    compdef _mage mage
else
    autoload -Uz compinit && compinit
    compdef _mage mage
fi
`
}

// fishCompletionScript returns a fish completion script that uses mage -autocomplete.
func fishCompletionScript(mageBin string) string {
	return `# mage tab completion for fish
complete -c mage -f
complete -c mage -a '('` + mageBin + `' -autocomplete 2>/dev/null)' -d 'mage target'
complete -c mage -s l -d 'list mage targets in this directory'
complete -c mage -s h -d 'show this help'
complete -c mage -s v -d 'show verbose output when running mage targets'
complete -c mage -s f -d 'force recreation of compiled magefile'
complete -c mage -l debug -d 'turn on debug messages'
complete -c mage -s t -r -d 'timeout in duration parsable format'
complete -c mage -s d -r -F -d 'directory to read magefiles from'
complete -c mage -s w -r -F -d 'working directory where magefiles will run'
complete -c mage -l keep -d 'keep intermediate mage files around after running'
complete -c mage -l compile -r -F -d 'output a static binary to the given path'
complete -c mage -l clean -d 'clean out old generated binaries from CACHE_DIR'
complete -c mage -l init -d 'create a starting template if no mage files exist'
complete -c mage -l version -d 'show version info for the mage binary'
complete -c mage -l gocmd -r -d 'use the given go binary to compile the output'
complete -c mage -l goos -r -d 'set GOOS for binary produced with -compile'
complete -c mage -l goarch -r -d 'set GOARCH for binary produced with -compile'
complete -c mage -l ldflags -r -d 'set ldflags for binary produced with -compile'
complete -c mage -l autocomplete -d 'print target names for shell completion'
complete -c mage -l install -r -a 'bash zsh fish powershell pwsh' -d 'install shell completion'
complete -c mage -l multiline -d 'retain line returns in help text'
`
}

// powerShellCompletionScript returns a PowerShell completion script that uses mage -autocomplete.
func powerShellCompletionScript(mageBin string) string {
	return `# mage tab completion for PowerShell
Register-ArgumentCompleter -CommandName mage -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    if ($wordToComplete.StartsWith('-')) {
        $flags = @(
            @{N='-l'; D='list mage targets in this directory'},
            @{N='-h'; D='show this help'},
            @{N='-v'; D='show verbose output'},
            @{N='-f'; D='force recreation of compiled magefile'},
            @{N='-debug'; D='turn on debug messages'},
            @{N='-t'; D='timeout in duration parsable format'},
            @{N='-d'; D='directory to read magefiles from'},
            @{N='-w'; D='working directory where magefiles will run'},
            @{N='-keep'; D='keep intermediate mage files around'},
            @{N='-compile'; D='output a static binary to the given path'},
            @{N='-clean'; D='clean out old generated binaries'},
            @{N='-init'; D='create a starting template'},
            @{N='-version'; D='show version info'},
            @{N='-gocmd'; D='use the given go binary'},
            @{N='-goos'; D='set GOOS for -compile'},
            @{N='-goarch'; D='set GOARCH for -compile'},
            @{N='-ldflags'; D='set ldflags for -compile'},
            @{N='-autocomplete'; D='print target names for shell completion'},
            @{N='-install'; D='install shell completion'},
            @{N='-multiline'; D='retain line returns in help text'}
        )
        $flags | Where-Object { $_.N -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_.N, $_.N, 'ParameterValue', $_.D)
        }
    } else {
        (& '` + mageBin + `' -autocomplete 2>$null) -split "` + "`n" + `" |
            Where-Object { $_ -ne '' -and $_ -like "$wordToComplete*" } |
            ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
    }
}
`
}
