package processes

import (
	"fmt"
	"sort"

	"github.com/xlab/treeprint"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// A PsTree is a tree of *commonpb.Process
type PsTree struct {
	printableTree treeprint.Tree // only used for rendering
	implantPID    int32
	procTree      *node // used to hold the tree structure
}

type node struct {
	Value    *commonpb.Process // The process information
	Children map[int32]*node   // The children of this node
	Parent   *node             // The parent of this node
}

// NewPsTree creates a new PsTree
func NewPsTree(pid int32) *PsTree {
	return &PsTree{
		printableTree: treeprint.New(),
		implantPID:    pid,
		procTree:      &node{Value: &commonpb.Process{Pid: -1}},
	}
}

func (n *node) insert(proc *commonpb.Process) {
	if n.Value == nil {
		n.Value = proc
		return
	}
	if parent := n.findParent(proc); parent != nil {
		if parent.Children == nil {
			parent.Children = make(map[int32]*node)
		}
		parent.Children[proc.Pid] = &node{Value: proc, Parent: parent}
	} else {
		if n.Children == nil {
			n.Children = make(map[int32]*node)
		}
		n.Children[proc.Pid] = &node{Value: proc, Parent: n}
	}
}

func (n *node) findParent(proc *commonpb.Process) *node {
	// Empty node
	if n.Value == nil {
		return nil
	}
	// Skip self when called from reorder
	// otherwise things might explode, see #1340
	if n.Value.Pid == proc.Pid {
		return nil
	}
	// Found parent
	if n.Value.Pid == proc.Ppid {
		return n
	}
	for _, child := range n.Children {
		if p := child.findParent(proc); p != nil {
			return p
		}
	}
	return nil
}

func reorder(root *node) {
	toDelete := make([]int32, 0)
	for pid, child := range root.Children {
		// skip root node
		if child.Value.Pid == -1 {
			continue
		}
		// Skip edge case of [System Process] on Windows
		if child.Value.Ppid == pid {
			continue
		}
		// only focus on children without parent
		if child.Parent.Value.Pid == -1 {
			if parent := root.findParent(child.Value); parent != nil {
				child.Parent = parent
				if parent.Children == nil {
					parent.Children = make(map[int32]*node)
				}
				// copy to new parent
				parent.Children[pid] = child
				// mark for deletion
				toDelete = append(toDelete, pid)
			}
		}
	}
	// delete children that were moved
	for _, pid := range toDelete {
		delete(root.Children, pid)
	}
}

func (t *PsTree) AddProcess(proc *commonpb.Process) {
	t.procTree.insert(proc)
}

func (t *PsTree) filterProc(proc *commonpb.Process) string {
	color := console.Normal
	if proc.Pid == t.implantPID {
		color = console.Green
	}
	if secTool, ok := knownSecurityTools[proc.Executable]; ok {
		color = secTool[0]
	}
	return fmt.Sprintf(color+"%s"+console.Normal, proc.Executable)
}

func (t *PsTree) String() string {
	reorder(t.procTree)
	return t.Print()
}

func (t *PsTree) Print() string {
	topLevelPIDs := make([]int, 0, len(t.procTree.Children))
	for k := range t.procTree.Children {
		topLevelPIDs = append(topLevelPIDs, int(k))
	}
	sort.Ints(topLevelPIDs)
	for _, pid := range topLevelPIDs {
		current := t.printableTree.AddMetaBranch(pid, t.filterProc(t.procTree.Children[int32(pid)].Value))
		t.addToTree(current, t.procTree.Children[int32(pid)].Children)
	}
	return t.printableTree.String()
}

func (t *PsTree) addToTree(tree treeprint.Tree, procs map[int32]*node) {
	for pid, proc := range procs {
		if proc.Value.Pid == -1 {
			continue
		}
		procName := t.filterProc(proc.Value)

		procNode := tree.FindByMeta(pid)
		if procNode == nil {
			if len(proc.Children) > 0 {
				procNode = tree.AddMetaBranch(pid, procName)
			} else {
				procNode = tree.AddMetaNode(pid, procName)
			}
		}
		t.addToTree(procNode, proc.Children)
	}
}
