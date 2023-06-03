// Package powershell provides powershell completion
package powershell

import (
	"fmt"

	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the powershell completion script.
func Snippet(cmd *cobra.Command) string {
	return fmt.Sprintf(`using namespace System.Management.Automation
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
      $elems += $t.replace('`+"`"+`,', ',') # quick fix
    }

    $completions = @(
      if (!$wordToComplete) {
        %v _carapace powershell $($elems| ForEach-Object {$_}) '' | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('`+"`"+`e[', "`+"`"+`e["), [CompletionResultType]::ParameterValue, $_.ToolTip) }
      } else {
        %v _carapace powershell $($elems| ForEach-Object {$_}) | ConvertFrom-Json | ForEach-Object { [CompletionResult]::new($_.CompletionText, $_.ListItemText.replace('`+"`"+`e[', "`+"`"+`e["), [CompletionResultType]::ParameterValue, $_.ToolTip) }
      }
    )

    if ($completions.count -eq 0) {
      return "" # prevent default file completion
    }

    $completions
}
Register-ArgumentCompleter -Native -CommandName '%v' -ScriptBlock (Get-Item "Function:_%v_completer").ScriptBlock
`, cmd.Name(), uid.Executable(), uid.Executable(), cmd.Name(), cmd.Name())
}
