// Package fish provides fish completion
package fish

import (
	"fmt"

	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/spf13/cobra"
)

// Snippet creates the fish completion script.
func Snippet(cmd *cobra.Command) string {
	return fmt.Sprintf(`function _%[1]v_completion
  set --local data
  IFS='' set data (echo (commandline -cp)'' | sed "s/ \$/ ''/" | xargs %[2]v _carapace fish 2>/dev/null)
  if [ $status -eq 1 ]
    IFS='' set data (echo (commandline -cp)"'" | sed "s/ \$/ ''/" | xargs %[2]v _carapace fish 2>/dev/null)
    if [ $status -eq 1 ]
      IFS='' set data (echo (commandline -cp)'"' | sed "s/ \$/ ''/" | xargs %[2]v _carapace fish 2>/dev/null)
    end
  end
  echo $data
end

complete -e '%[1]v'
complete -c '%[1]v' -f -a '(_%[1]v_completion)' -r
`, cmd.Name(), uid.Executable())
}
