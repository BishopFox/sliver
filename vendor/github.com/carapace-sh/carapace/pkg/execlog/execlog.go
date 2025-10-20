package execlog

import (
	shlex "github.com/carapace-sh/carapace-shlex"
	"github.com/carapace-sh/carapace/internal/log"
	"github.com/carapace-sh/carapace/third_party/golang.org/x/sys/execabs"
)

type Cmd struct {
	*execabs.Cmd
}

// Command is like execabs.Command but logs args on execution.
func Command(name string, arg ...string) *Cmd {
	cmd := &Cmd{
		execabs.Command(name, arg...),
	}
	return cmd
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	log.LOG.Printf("executing %#v", shlex.Join(c.Args))
	return c.Cmd.CombinedOutput()
}

func (c *Cmd) Output() ([]byte, error) {
	log.LOG.Printf("executing %#v", shlex.Join(c.Args))
	return c.Cmd.Output()
}

func (c *Cmd) Run() error {
	log.LOG.Printf("executing %#v", shlex.Join(c.Args))
	return c.Cmd.Run()
}

func (c *Cmd) Start() error {
	log.LOG.Printf("executing %#v", shlex.Join(c.Args))
	return c.Cmd.Start()
}

// Command is the same as execabs.Command.
func LookPath(file string) (string, error) {
	return execabs.LookPath(file)
}
