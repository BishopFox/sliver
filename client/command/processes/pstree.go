package processes

import (
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/xlab/treeprint"
)

// A PsTree is a tree of *commonpb.Process
type PsTree struct {
	tree       treeprint.Tree
	implantPID int32
}

// NewPsTree creates a new PsTree
func NewPsTree(pid int32) *PsTree {
	return &PsTree{
		tree:       treeprint.New(),
		implantPID: pid,
	}
}

func (t *PsTree) AddProcess(process *commonpb.Process) {
	if existingParent := t.tree.FindByMeta(process.Ppid); existingParent != nil {
		procName := t.filterProc(process)
		existingParent.AddMetaBranch(process.Pid, procName)
	} else {
		t.tree.AddMetaBranch(process.Pid, process.Executable)
	}
}

func (t *PsTree) filterProc(process *commonpb.Process) string {
	color := console.Normal
	if process.Pid == t.implantPID {
		color = console.Green
	}
	if secTool, ok := knownSecurityTools[process.Executable]; ok {
		color = secTool[0]
	}
	return fmt.Sprintf(color+"%s"+console.Normal, process.Executable)
}

func (t *PsTree) String() string {
	return t.tree.String()
}
