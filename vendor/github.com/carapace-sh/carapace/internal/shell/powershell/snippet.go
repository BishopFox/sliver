package powershell

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

const snippetOriginal = `using namespace System.Management.Automation
using namespace System.Management.Automation.Language
Function _%v_completer {
    [System.Diagnostics.CodeAnalysis.SuppressMessageAttribute("PSAvoidUsingInvokeExpression", "", Scope="Function", Target="*")]
    param($wordToComplete, $commandAst, $cursorPosition)
    $commandElements = $commandAst.CommandElements

    # double quoted value works but seems single quoted needs some fixing (e.g. "example 'acti" -> "example acti")
    $elems = @()
    foreach ($_ in $commandElements) {
      if ($_.Extent.StartOffset -gt $cursorPosition) {
          break
      }
      $t = $_.Extent.Text
      if ($_.Extent.EndOffset -gt $cursorPosition) {
          $t = $t.Substring(0, $_.Extent.Text.get_Length() - ($_.Extent.EndOffset - $cursorPosition))
      }

      if ($t.Substring(0,1) -eq "'"){
        $t = $t.Substring(1)
      }
      if ($t.get_Length() -gt 0 -and $t.Substring($t.get_Length()-1) -eq "'"){
        $t = $t.Substring(0,$t.get_Length()-1)
      }
      if ($t.get_Length() -eq 0){
        $t = '""'
      }
      $elems += $t.replace('` + "`" + `,', ',') # quick fix
    }

    $completions = @(
      if (!$wordToComplete) {
        %v _carapace powershell $($elems| ForEach-Object {$_}) '' | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('` + "`" + `e[', "` + "`" + `e["), [CompletionResultType]::ParameterValue, $_.ToolTip.replace('` + "`" + `e[', "` + "`" + `e[")) }
      } else {
        %v _carapace powershell $($elems| ForEach-Object {$_}) | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('` + "`" + `e[', "` + "`" + `e["), [CompletionResultType]::ParameterValue, $_.ToolTip.replace('` + "`" + `e[', "` + "`" + `e[")) }
      }
    )

    if ($completions.count -eq 0) {
      return "" # prevent default file completion
    }

    $completions
}
Register-ArgumentCompleter -Native -ScriptBlock (Get-Item "Function:_%v_completer").ScriptBlock -CommandName '%v'%v,'%v.exe'
`

const snippetMultiSingle = `using namespace System.Management.Automation
using namespace System.Management.Automation.Language

$_%[2]v_completion = {
    [System.Diagnostics.CodeAnalysis.SuppressMessageAttribute("PSAvoidUsingInvokeExpression", "", Scope="Function", Target="*")]
    param($wordToComplete, $commandAst, $cursorPosition)
    $commandElements = $commandAst.CommandElements

    # double quoted value works but seems single quoted needs some fixing (e.g. "example 'acti" -> "example acti")
    $elems = @()
    foreach ($_ in $commandElements) {
      if ($_.Extent.StartOffset -gt $cursorPosition) {
          break
      }
      $t = $_.Extent.Text
      if ($_.Extent.EndOffset -gt $cursorPosition) {
          $t = $t.Substring(0, $_.Extent.Text.get_Length() - ($_.Extent.EndOffset - $cursorPosition))
      }

      if ($t.Substring(0,1) -eq "'"){
        $t = $t.Substring(1)
      }
      if ($t.get_Length() -gt 0 -and $t.Substring($t.get_Length()-1) -eq "'"){
        $t = $t.Substring(0,$t.get_Length()-1)
      }
      if ($t.get_Length() -eq 0){
        $t = '""'
      }
      $elems += $t.replace('` + "`" + `,', ',') # quick fix
    }

    $completions = @(
      if (!$wordToComplete) {
        %[1]v %[3]v _carapace powershell $($elems| ForEach-Object {$_}) '' | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('` + "`" + `e[', "` + "`" + `e["), [CompletionResultType]::ParameterValue, $_.ToolTip.replace('` + "`" + `e[', "` + "`" + `e[")) }
      } else {
        %[1]v %[3]v _carapace powershell $($elems| ForEach-Object {$_}) | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('` + "`" + `e[', "` + "`" + `e["), [CompletionResultType]::ParameterValue, $_.ToolTip.replace('` + "`" + `e[', "` + "`" + `e[")) }
      }
    )

    if ($completions.count -eq 0) {
      return "" # prevent default file completion
    }

    $completions
}

%[4]v
`

const snippetMulti = `using namespace System.Management.Automation
using namespace System.Management.Automation.Language
%[4]v
$_%[1]v_completer = {
    [System.Diagnostics.CodeAnalysis.SuppressMessageAttribute("PSAvoidUsingInvokeExpression", "", Scope="Function", Target="*")]
    param($wordToComplete, $commandAst, $cursorPosition)
    $commandElements = $commandAst.CommandElements

    $elems = @()
    foreach ($_ in $commandElements) {
      if ($_.Extent.StartOffset -gt $cursorPosition) {
          break
      }
      $t = $_.Extent.Text
      if ($_.Extent.EndOffset -gt $cursorPosition) {
          $t = $t.Substring(0, $_.Extent.Text.get_Length() - ($_.Extent.EndOffset - $cursorPosition))
      }

      if ($t.Substring(0,1) -eq "'"){
        $t = $t.Substring(1)
      }
      if ($t.get_Length() -gt 0 -and $t.Substring($t.get_Length()-1) -eq "'"){
        $t = $t.Substring(0,$t.get_Length()-1)
      }
      if ($t.get_Length() -eq 0){
        $t = '""'
      }
      $elems += $t.replace('` + "`" + `,', ',') # quick fix
    }

    $completions = @(
      if (!$wordToComplete) {
        %[2]v ($elems[0] -replace ('\.exe$', '')) _carapace powershell $($elems| ForEach-Object {$_}) '' | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('` + "`" + `e[', "` + "`" + `e["), [CompletionResultType]::ParameterValue, $_.ToolTip.replace('` + "`" + `e[', "` + "`" + `e[")) }
      } else {
        %[2]v ($elems[0] -replace ('\.exe$', '')) _carapace powershell $($elems| ForEach-Object {$_}) | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('` + "`" + `e[', "` + "`" + `e["), [CompletionResultType]::ParameterValue, $_.ToolTip.replace('` + "`" + `e[', "` + "`" + `e[")) }
      }
    )

    if ($completions.count -eq 0) {
      return "" # prevent default file completion
    }

    $completions
}

%[3]v
`

func sanitize(s string) string {
	return strings.ReplaceAll(s, "-", "__")
}

// Snippet creates the powershell completion script.
func Snippet(cmd *cobra.Command) string {
	return SnippetSingle(cmd.Name(), false)
}

// SnippetMulti creates a multi-completer powershell completion script.
func SnippetMulti(names []string, defaultName string, snippetFuncs string) string {
	complete := make([]string, len(names))
	for i, name := range names {
		prefix := " # "
		if runtime.GOOS == "windows" {
			prefix = ""
		}
		complete[i] = fmt.Sprintf(`Register-ArgumentCompleter -Native -ScriptBlock $_%v_completer -CommandName '%v'%v'%v.exe'`, sanitize(defaultName), name, prefix, name)
	}
	return fmt.Sprintf(snippetMulti, sanitize(defaultName), uid.Executable(), strings.Join(complete, "\n"), snippetFuncs)
}

// SnippetSingle creates a single-command powershell completion script.
// When explicitCommand is true, the command name is included in the invocation
// (for multi-completer subcommands). When false, the executable is invoked
// without an explicit command (standalone mode).
func SnippetSingle(command string, explicitCommand bool) string {
	prefix := " # "
	if runtime.GOOS == "windows" {
		prefix = ""
	}

	if explicitCommand {
		complete := fmt.Sprintf(`Register-ArgumentCompleter -Native -ScriptBlock $_%v_completion -CommandName '%v'%v'%v.exe'`, sanitize(command), command, prefix, command)
		return fmt.Sprintf(snippetMultiSingle, uid.Executable(), sanitize(command), command, complete)
	}

	return fmt.Sprintf(snippetOriginal,
		sanitize(command),
		uid.Executable(),
		uid.Executable(),
		sanitize(command),
		command,
		prefix,
		command)
}
