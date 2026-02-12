package processes

import (
	"sort"

	"github.com/xlab/treeprint"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// A PsTree is a tree of *commonpb.Process
// A PsTree 是 *commonpb.Process 的树
type PsTree struct {
	printableTree treeprint.Tree // only used for rendering
	printableTree treeprint.Tree // 仅用于渲染
	implantPID    int32
	procTree      *node // used to hold the tree structure
	procTree      *node // 用于保存树结构
}

type node struct {
	Value    *commonpb.Process // The process information
	Value    *commonpb.Process // The进程信息
	Children map[int32]*node   // The children of this node
	Children map[int32]*node   // 该节点的 The 子节点
	Parent   *node             // The parent of this node
	Parent   *node             // The 该节点的父节点
}

// NewPsTree creates a new PsTree
// NewPsTree 创建一个新的 PsTree
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
	// Empty 节点
	if n.Value == nil {
		return nil
	}
	// Skip self when called from reorder
	// 从重新排序中调用时的 Skip self
	// otherwise things might explode, see #1340
	// 否则事情可能会爆炸，请参阅#1340
	if n.Value.Pid == proc.Pid {
		return nil
	}
	// Found parent
	// Found 家长
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
		// 跳过根节点
		if child.Value.Pid == -1 {
			continue
		}
		// Skip edge case of [System Process] on Windows
		// Windows 上 [System Process] 的 Skip 边缘情况
		if child.Value.Ppid == pid {
			continue
		}
		// only focus on children without parent
		// 只关注没有父母的孩子
		if child.Parent.Value.Pid == -1 {
			if parent := root.findParent(child.Value); parent != nil {
				child.Parent = parent
				if parent.Children == nil {
					parent.Children = make(map[int32]*node)
				}
				// copy to new parent
				// 复制到新的父级
				parent.Children[pid] = child
				// mark for deletion
				// 标记为删除
				toDelete = append(toDelete, pid)
			}
		}
	}
	// delete children that were moved
	// 删除被移动的子项
	for _, pid := range toDelete {
		delete(root.Children, pid)
	}
}

func (t *PsTree) AddProcess(proc *commonpb.Process) {
	t.procTree.insert(proc)
}

func (t *PsTree) filterProc(proc *commonpb.Process) string {
	style := console.StyleNormal
	if proc.Pid == t.implantPID {
		style = console.StyleGreen
	}
	if _, ok := knownSecurityTools[proc.Executable]; ok {
		style = console.StyleRed
	}
	return style.Render(proc.Executable)
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
