package extension

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	experimentalsysfs "github.com/tetratelabs/wazero/experimental/sysfs"
	wazerosys "github.com/tetratelabs/wazero/sys"
	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

const (
	wasmMemoryFSMaxBytes     = int64(256 << 20)
	wasmMemoryFSMaxEntries   = int64(64 << 10)
	wasmMemoryFSMaxOpenFiles = int64(4 << 10)
	wasmMemoryFSMaxPathBytes = 4096
	wasmMemoryFSMaxNameBytes = 255
	wasmMemoryFSMaxLinkDepth = 40
)

var (
	wasmMemoryFSInitMu sync.RWMutex

	_ fs.FS              = (*WasmMemoryFS)(nil)
	_ fs.FS              = WasmMemoryFS{}
	_ fs.ReadDirFS       = (*WasmMemoryFS)(nil)
	_ fs.ReadFileFS      = (*WasmMemoryFS)(nil)
	_ experimentalsys.FS = (*WasmMemoryFS)(nil)

	_ fs.ReadDirFile       = (*wasmMemoryFSFile)(nil)
	_ io.Seeker            = (*wasmMemoryFSFile)(nil)
	_ experimentalsys.File = (*wasmMemoryFile)(nil)
)

// WasmMemoryFSOption configures a WasmMemoryFS.
type WasmMemoryFSOption func(*wasmMemoryFSConfig)

type wasmMemoryFSConfig struct {
	readOnly     bool
	maxBytes     int64
	maxEntries   int64
	maxOpenFiles int64
}

// WithWasmMemoryFSReadOnly configures the in-memory namespace as read-only.
// Reads and metadata queries remain available, while every mutating operation
// returns EROFS.
func WithWasmMemoryFSReadOnly() WasmMemoryFSOption {
	return func(config *wasmMemoryFSConfig) {
		config.readOnly = true
	}
}

// NewWasmMemoryFS creates a virtual filesystem which exposes files under
// /memfs and preserves the historical read-only host filesystem pass-through
// everywhere else. Initial file names are relative to /memfs. The input map
// and its byte slices are copied.
func NewWasmMemoryFS(files map[string][]byte, options ...WasmMemoryFSOption) (*WasmMemoryFS, error) {
	config := wasmMemoryFSConfig{
		maxBytes:     wasmMemoryFSMaxBytes,
		maxEntries:   wasmMemoryFSMaxEntries,
		maxOpenFiles: wasmMemoryFSMaxOpenFiles,
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	initialEntries := make(map[string]bool, len(files))
	var initialBytes int64
	for name, data := range files {
		segments, errno := normalizeWasmMemoryPath(name)
		if errno != 0 {
			return nil, fmt.Errorf("initialize memfs path %q: %w", name, errno)
		}
		if len(segments) == 0 {
			return nil, fmt.Errorf("initialize memfs path %q: %w", name, experimentalsys.EINVAL)
		}
		for index := range segments {
			entryName := strings.Join(segments[:index+1], "/")
			isDirectory := index < len(segments)-1
			if existingDirectory, exists := initialEntries[entryName]; exists {
				if !existingDirectory || !isDirectory {
					return nil, fmt.Errorf("initialize memfs path %q: %w", name, experimentalsys.EEXIST)
				}
				continue
			}
			initialEntries[entryName] = isDirectory
			if int64(len(initialEntries)) > config.maxEntries {
				return nil, fmt.Errorf("initialize memfs: %w", experimentalsys.ERANGE)
			}
		}
		if int64(len(data)) > config.maxBytes-initialBytes {
			return nil, fmt.Errorf("initialize memfs: %w", experimentalsys.ERANGE)
		}
		initialBytes += int64(len(data))
	}

	initialFiles := make(map[string][]byte, len(files))
	for name, data := range files {
		initialFiles[name] = append([]byte(nil), data...)
	}

	memoryFS := &WasmMemoryFS{
		memFS:        initialFiles,
		readOnly:     config.readOnly,
		maxBytes:     config.maxBytes,
		maxEntries:   config.maxEntries,
		maxOpenFiles: config.maxOpenFiles,
	}
	if err := memoryFS.ensureInitialized(); err != nil {
		return nil, err
	}
	return memoryFS, nil
}

// makeWasmMemFS retains the historical construction helper used by the Wasm
// extension runtime.
func makeWasmMemFS(files map[string][]byte, options ...WasmMemoryFSOption) (*WasmMemoryFS, error) {
	return NewWasmMemoryFS(files, options...)
}

// WasmMemoryFS is a writable in-memory filesystem rooted at /memfs. Filesystems
// returned by NewWasmMemoryFS are concurrency-safe. It also implements the
// legacy io/fs facade whose non-memfs paths are passed through to the same
// host-root os.DirFS used by earlier versions.
//
// The zero value and package-local legacy literals remain usable for
// compatibility, though callers should use NewWasmMemoryFS so malformed
// initial paths can be reported immediately. Because Open retains its
// historical value receiver, a legacy literal must finish a pointer-receiver
// operation, such as Stat("memfs"), before it is shared between goroutines.
type WasmMemoryFS struct {
	experimentalsys.UnimplementedFS

	initErr error

	// memFS is retained only so package-local legacy struct literals continue
	// to initialize lazily. It is cleared after initialization.
	memFS        map[string][]byte
	readOnly     bool
	maxBytes     int64
	maxEntries   int64
	maxOpenFiles int64

	state       *wasmMemoryFSState
	localFS     fs.FS
	localRoot   string
	passthrough experimentalsys.FS
}

type wasmMemoryFSState struct {
	mu sync.RWMutex

	root         *wasmMemoryNode
	nextInode    uint64
	usedBytes    int64
	maxBytes     int64
	entries      int64
	maxEntries   int64
	openFiles    int64
	maxOpenFiles int64
	readOnly     bool
}

type wasmMemoryNode struct {
	inode    uint64
	typeBits fs.FileMode
	perm     fs.FileMode

	data       []byte
	children   map[string]*wasmMemoryNode
	linkTarget string

	atim int64
	mtim int64
	ctim int64

	nlink     uint64
	openCount uint64
}

func (node *wasmMemoryNode) isDir() bool {
	return node.typeBits&fs.ModeDir != 0
}

func (node *wasmMemoryNode) isSymlink() bool {
	return node.typeBits&fs.ModeSymlink != 0
}

func (node *wasmMemoryNode) allocatedBytes() int64 {
	if node.isSymlink() {
		return int64(len(node.linkTarget))
	}
	if node.isDir() {
		return 0
	}
	return int64(len(node.data))
}

func (w *WasmMemoryFS) ensureInitialized() error {
	wasmMemoryFSInitMu.RLock()
	if w.initErr != nil || w.state != nil {
		err := w.initErr
		wasmMemoryFSInitMu.RUnlock()
		return err
	}
	wasmMemoryFSInitMu.RUnlock()

	// WasmMemoryFS historically implemented fs.FS with a value receiver. Keep
	// that source compatibility without copying a mutex by serializing the rare
	// lazy initialization here. Initialized filesystems only take the shared
	// read lock above, so unrelated guest filesystem operations remain
	// concurrent.
	wasmMemoryFSInitMu.Lock()
	defer wasmMemoryFSInitMu.Unlock()
	if w.initErr != nil || w.state != nil {
		return w.initErr
	}

	if w.maxBytes == 0 {
		w.maxBytes = wasmMemoryFSMaxBytes
	}
	if w.maxEntries == 0 {
		w.maxEntries = wasmMemoryFSMaxEntries
	}
	if w.maxOpenFiles == 0 {
		w.maxOpenFiles = wasmMemoryFSMaxOpenFiles
	}
	w.localRoot = wasmLocalFilesystemRoot()
	if w.localFS == nil {
		w.localFS = os.DirFS(w.localRoot)
	}
	w.passthrough = &experimentalsysfs.AdaptFS{FS: w.localFS}

	now := time.Now().UnixNano()
	rootMode := fs.ModeDir | 0o755
	if w.readOnly {
		rootMode = fs.ModeDir | 0o555
	}
	w.state = &wasmMemoryFSState{
		root: &wasmMemoryNode{
			inode:    1,
			typeBits: fs.ModeDir,
			perm:     rootMode.Perm(),
			children: map[string]*wasmMemoryNode{},
			atim:     now,
			mtim:     now,
			ctim:     now,
			nlink:    2,
		},
		nextInode:    2,
		maxBytes:     w.maxBytes,
		maxEntries:   w.maxEntries,
		maxOpenFiles: w.maxOpenFiles,
		readOnly:     w.readOnly,
	}

	keys := make([]string, 0, len(w.memFS))
	for name := range w.memFS {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		// {{if .Config.Debug}}
		log.Printf("[wasm ext] memfs file: %s (%d bytes)", name, len(w.memFS[name]))
		// {{end}}
		if errno := w.state.insertInitialFile(name, w.memFS[name]); errno != 0 {
			w.initErr = fmt.Errorf("initialize memfs path %q: %w", name, errno)
			break
		}
	}
	w.memFS = nil

	// {{if .Config.Debug}}
	log.Printf("[wasm ext] local filesystem root: %s", w.localRoot)
	// {{end}}
	return w.initErr
}

func wasmLocalFilesystemRoot() string {
	root := "/"
	cwd, err := os.Getwd()
	if err != nil {
		return root
	}
	if volume := filepath.VolumeName(cwd); volume != "" {
		root = volume + string(filepath.Separator)
	}
	return root
}

func (state *wasmMemoryFSState) insertInitialFile(name string, data []byte) experimentalsys.Errno {
	segments, errno := normalizeWasmMemoryPath(name)
	if errno != 0 {
		return errno
	}
	if len(segments) == 0 {
		return experimentalsys.EINVAL
	}
	if int64(len(data)) > state.maxBytes-state.usedBytes {
		return experimentalsys.ERANGE
	}

	directoryMode := fs.ModeDir | 0o755
	fileMode := fs.FileMode(0o644)
	if state.readOnly {
		directoryMode = fs.ModeDir | 0o555
		fileMode = 0o444
	}

	current := state.root
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current.children[segment]
		if !ok {
			next = state.newNode(directoryMode)
			if errno := state.addChildLocked(current, segment, next); errno != 0 {
				return errno
			}
		}
		if !next.isDir() {
			return experimentalsys.ENOTDIR
		}
		current = next
	}

	base := segments[len(segments)-1]
	if _, ok := current.children[base]; ok {
		return experimentalsys.EEXIST
	}
	node := state.newNode(fileMode)
	node.data = data
	state.usedBytes += int64(len(node.data))
	if errno := state.addChildLocked(current, base, node); errno != 0 {
		state.usedBytes -= int64(len(node.data))
		return errno
	}
	return 0
}

func (state *wasmMemoryFSState) newNode(mode fs.FileMode) *wasmMemoryNode {
	now := time.Now().UnixNano()
	node := &wasmMemoryNode{
		inode:    state.nextInode,
		typeBits: mode.Type(),
		perm:     mode.Perm(),
		atim:     now,
		mtim:     now,
		ctim:     now,
		nlink:    1,
	}
	state.nextInode++
	if node.isDir() {
		node.children = map[string]*wasmMemoryNode{}
		node.nlink = 2
	}
	return node
}

func (state *wasmMemoryFSState) addChildLocked(parent *wasmMemoryNode, name string, node *wasmMemoryNode) experimentalsys.Errno {
	if state.entries >= state.maxEntries {
		return experimentalsys.ERANGE
	}
	parent.children[name] = node
	state.entries++
	if node.isDir() {
		parent.nlink++
	}
	return 0
}

func (state *wasmMemoryFSState) removeChildLocked(parent *wasmMemoryNode, name string) (*wasmMemoryNode, bool) {
	node, ok := parent.children[name]
	if !ok {
		return nil, false
	}
	delete(parent.children, name)
	if state.entries > 0 {
		state.entries--
	}
	if node.isDir() && parent.nlink > 0 {
		parent.nlink--
	}
	return node, true
}

// splitWasmMemoryPath decides the namespace before normalizing the path. This
// ensures a traversal such as memfs/../host never falls through to the host.
func splitWasmMemoryPath(name string) (string, bool) {
	trimmed := strings.TrimLeft(name, "/")
	for offset := 0; offset < len(trimmed); {
		for offset < len(trimmed) && trimmed[offset] == '/' {
			offset++
		}
		start := offset
		for offset < len(trimmed) && trimmed[offset] != '/' {
			offset++
		}
		segment := trimmed[start:offset]
		if segment == "" || segment == "." {
			continue
		}
		if segment != "memfs" {
			return "", false
		}
		for offset < len(trimmed) && trimmed[offset] == '/' {
			offset++
		}
		return trimmed[offset:], true
	}
	return "", false
}

func normalizeWasmMemoryPath(name string) ([]string, experimentalsys.Errno) {
	if len(name) > wasmMemoryFSMaxPathBytes {
		return nil, experimentalsys.ENAMETOOLONG
	}
	if strings.IndexByte(name, 0) >= 0 {
		return nil, experimentalsys.EINVAL
	}

	segments := make([]string, 0, strings.Count(name, "/")+1)
	for _, segment := range strings.Split(strings.TrimLeft(name, "/"), "/") {
		switch segment {
		case "", ".":
			continue
		case "..":
			// Never allow a raw guest path to traverse out of the directory
			// it named. In particular, memfs/../host must not be re-routed
			// into the host pass-through namespace.
			return nil, experimentalsys.EINVAL
		default:
			if len(segment) > wasmMemoryFSMaxNameBytes {
				return nil, experimentalsys.ENAMETOOLONG
			}
			segments = append(segments, segment)
		}
	}
	return segments, 0
}

func resolveWasmMemoryLinkPath(base []string, target string, remaining []string) ([]string, experimentalsys.Errno) {
	if target == "" || path.IsAbs(target) || strings.IndexByte(target, 0) >= 0 {
		return nil, experimentalsys.ENOENT
	}
	segments := append([]string(nil), base...)
	for _, segment := range strings.Split(target, "/") {
		switch segment {
		case "", ".":
			continue
		case "..":
			if len(segments) == 0 {
				return nil, experimentalsys.EINVAL
			}
			segments = segments[:len(segments)-1]
		default:
			if len(segment) > wasmMemoryFSMaxNameBytes {
				return nil, experimentalsys.ENAMETOOLONG
			}
			segments = append(segments, segment)
		}
	}
	segments = append(segments, remaining...)
	if len(strings.Join(segments, "/")) > wasmMemoryFSMaxPathBytes {
		return nil, experimentalsys.ENAMETOOLONG
	}
	return segments, 0
}

func (state *wasmMemoryFSState) lookupLocked(name string, followFinal bool) (*wasmMemoryNode, experimentalsys.Errno) {
	segments, errno := normalizeWasmMemoryPath(name)
	if errno != 0 {
		return nil, errno
	}
	return state.lookupSegmentsLocked(segments, followFinal, 0)
}

func (state *wasmMemoryFSState) lookupSegmentsLocked(segments []string, followFinal bool, depth int) (*wasmMemoryNode, experimentalsys.Errno) {
	if depth > wasmMemoryFSMaxLinkDepth {
		return nil, experimentalsys.ELOOP
	}
	current := state.root
	for index, segment := range segments {
		if !current.isDir() {
			return nil, experimentalsys.ENOTDIR
		}
		next, ok := current.children[segment]
		if !ok {
			return nil, experimentalsys.ENOENT
		}
		if next.isSymlink() && (followFinal || index < len(segments)-1) {
			resolved, errno := resolveWasmMemoryLinkPath(segments[:index], next.linkTarget, segments[index+1:])
			if errno != 0 {
				return nil, errno
			}
			return state.lookupSegmentsLocked(resolved, followFinal, depth+1)
		}
		current = next
	}
	return current, 0
}

// createPathSegmentsLocked resolves all symbolic links and returns a path
// whose final entry does not yet exist. This lets O_CREAT follow a dangling
// final symlink without replacing the link itself.
func (state *wasmMemoryFSState) createPathSegmentsLocked(segments []string, depth int) ([]string, experimentalsys.Errno) {
	if depth > wasmMemoryFSMaxLinkDepth {
		return nil, experimentalsys.ELOOP
	}
	if len(segments) == 0 {
		return nil, experimentalsys.EEXIST
	}
	current := state.root
	for index, segment := range segments {
		if !current.isDir() {
			return nil, experimentalsys.ENOTDIR
		}
		next, ok := current.children[segment]
		if !ok {
			if index != len(segments)-1 {
				return nil, experimentalsys.ENOENT
			}
			return segments, 0
		}
		if next.isSymlink() {
			resolved, errno := resolveWasmMemoryLinkPath(segments[:index], next.linkTarget, segments[index+1:])
			if errno != 0 {
				return nil, errno
			}
			return state.createPathSegmentsLocked(resolved, depth+1)
		}
		current = next
	}
	return nil, experimentalsys.EEXIST
}

func (state *wasmMemoryFSState) parentLocked(name string) (*wasmMemoryNode, string, experimentalsys.Errno) {
	segments, errno := normalizeWasmMemoryPath(name)
	if errno != 0 {
		return nil, "", errno
	}
	if len(segments) == 0 {
		return nil, "", experimentalsys.EINVAL
	}
	parent, errno := state.lookupSegmentsLocked(segments[:len(segments)-1], true, 0)
	if errno != 0 {
		return nil, "", errno
	}
	if !parent.isDir() {
		return nil, "", experimentalsys.ENOTDIR
	}
	return parent, segments[len(segments)-1], 0
}

func (state *wasmMemoryFSState) releaseNodeLocked(node *wasmMemoryNode) {
	if node.nlink != 0 || node.openCount != 0 {
		return
	}
	state.usedBytes -= node.allocatedBytes()
	if state.usedBytes < 0 {
		state.usedBytes = 0
	}
	node.data = nil
	node.linkTarget = ""
}

func (state *wasmMemoryFSState) unlinkNodeLocked(node *wasmMemoryNode) {
	if node.isDir() {
		node.nlink = 0
	} else if node.nlink > 0 {
		node.nlink--
	}
	node.ctim = time.Now().UnixNano()
	state.releaseNodeLocked(node)
}

func wasmMemoryStat(node *wasmMemoryNode, readOnly bool) wazerosys.Stat_t {
	mode := node.typeBits | node.perm
	if readOnly {
		mode &^= 0o222
	}
	size := int64(len(node.data))
	if node.isDir() {
		size = 0
	} else if node.isSymlink() {
		size = int64(len(node.linkTarget))
	}
	return wazerosys.Stat_t{
		Dev:   1,
		Ino:   node.inode,
		Mode:  mode,
		Nlink: node.nlink,
		Size:  size,
		Atim:  node.atim,
		Mtim:  node.mtim,
		Ctim:  node.ctim,
	}
}

func knownWasmMemoryOpenFlags() experimentalsys.Oflag {
	return experimentalsys.O_RDWR |
		experimentalsys.O_WRONLY |
		experimentalsys.O_APPEND |
		experimentalsys.O_CREAT |
		experimentalsys.O_DIRECTORY |
		experimentalsys.O_DSYNC |
		experimentalsys.O_EXCL |
		experimentalsys.O_NOFOLLOW |
		experimentalsys.O_NONBLOCK |
		experimentalsys.O_RSYNC |
		experimentalsys.O_SYNC |
		experimentalsys.O_TRUNC
}

func (w *WasmMemoryFS) memoryState() (*wasmMemoryFSState, experimentalsys.Errno) {
	if err := w.ensureInitialized(); err != nil {
		return nil, experimentalsys.EINVAL
	}
	return w.state, 0
}

// OpenFile implements experimental/sys.FS. Only the exact memfs namespace is
// mutable; all other paths use wazero's same fs.FS adapter as the historical
// WithFS configuration.
func (w *WasmMemoryFS) OpenFile(name string, flag experimentalsys.Oflag, perm fs.FileMode) (experimentalsys.File, experimentalsys.Errno) {
	state, errno := w.memoryState()
	if errno != 0 {
		return nil, errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.OpenFile(name, flag, perm)
	}
	return state.openFile(subpath, flag, perm)
}

//nolint:gocyclo // Open flag validation and creation are one atomic filesystem state transition.
func (state *wasmMemoryFSState) openFile(name string, flag experimentalsys.Oflag, perm fs.FileMode) (experimentalsys.File, experimentalsys.Errno) {
	if flag & ^knownWasmMemoryOpenFlags() != 0 {
		return nil, experimentalsys.EINVAL
	}
	accessMode := flag & (experimentalsys.O_RDWR | experimentalsys.O_WRONLY)
	if accessMode == experimentalsys.O_RDWR|experimentalsys.O_WRONLY {
		return nil, experimentalsys.EINVAL
	}
	readable := accessMode != experimentalsys.O_WRONLY
	writable := accessMode == experimentalsys.O_WRONLY || accessMode == experimentalsys.O_RDWR
	if flag&experimentalsys.O_APPEND != 0 && !writable {
		return nil, experimentalsys.EINVAL
	}
	if flag&experimentalsys.O_TRUNC != 0 && !writable {
		return nil, experimentalsys.EINVAL
	}
	if flag&experimentalsys.O_CREAT != 0 && flag&experimentalsys.O_DIRECTORY != 0 {
		return nil, experimentalsys.EINVAL
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly && (writable || flag&(experimentalsys.O_CREAT|experimentalsys.O_TRUNC|experimentalsys.O_APPEND) != 0) {
		return nil, experimentalsys.EROFS
	}

	_, entryErrno := state.lookupLocked(name, false)
	if entryErrno != 0 && entryErrno != experimentalsys.ENOENT {
		return nil, entryErrno
	}
	if entryErrno == 0 && flag&experimentalsys.O_CREAT != 0 && flag&experimentalsys.O_EXCL != 0 {
		return nil, experimentalsys.EEXIST
	}

	followFinal := flag&experimentalsys.O_NOFOLLOW == 0
	node, lookupErrno := state.lookupLocked(name, followFinal)
	if lookupErrno == experimentalsys.ENOENT && flag&experimentalsys.O_CREAT != 0 {
		if state.openFiles >= state.maxOpenFiles {
			return nil, experimentalsys.ERANGE
		}
		segments, pathErrno := normalizeWasmMemoryPath(name)
		if pathErrno != 0 {
			return nil, pathErrno
		}
		createSegments, createErrno := state.createPathSegmentsLocked(segments, 0)
		if createErrno != 0 {
			return nil, createErrno
		}
		createName := strings.Join(createSegments, "/")
		parent, base, parentErrno := state.parentLocked(createName)
		if parentErrno != 0 {
			return nil, parentErrno
		}
		if _, exists := parent.children[base]; exists {
			return nil, experimentalsys.EEXIST
		}
		node = state.newNode(perm.Perm())
		if addErrno := state.addChildLocked(parent, base, node); addErrno != 0 {
			return nil, addErrno
		}
		parent.mtim = time.Now().UnixNano()
		parent.ctim = parent.mtim
		lookupErrno = 0
	}
	if lookupErrno != 0 {
		return nil, lookupErrno
	}
	if node.isSymlink() {
		return nil, experimentalsys.ELOOP
	}
	if flag&experimentalsys.O_DIRECTORY != 0 && !node.isDir() {
		return nil, experimentalsys.ENOTDIR
	}
	if node.isDir() && writable {
		return nil, experimentalsys.EISDIR
	}
	if state.openFiles >= state.maxOpenFiles {
		return nil, experimentalsys.ERANGE
	}
	if flag&experimentalsys.O_TRUNC != 0 && !node.isDir() {
		if truncateErrno := state.truncateNodeLocked(node, 0); truncateErrno != 0 {
			return nil, truncateErrno
		}
	}

	node.openCount++
	state.openFiles++
	return &wasmMemoryFile{
		state:    state,
		node:     node,
		readable: readable,
		writable: writable,
		append:   flag&experimentalsys.O_APPEND != 0,
	}, 0
}

// Lstat implements experimental/sys.FS.
func (w *WasmMemoryFS) Lstat(name string) (wazerosys.Stat_t, experimentalsys.Errno) {
	return w.statPath(name, false)
}

// Stat implements experimental/sys.FS.
func (w *WasmMemoryFS) Stat(name string) (wazerosys.Stat_t, experimentalsys.Errno) {
	return w.statPath(name, true)
}

func (w *WasmMemoryFS) statPath(name string, followFinal bool) (wazerosys.Stat_t, experimentalsys.Errno) {
	state, errno := w.memoryState()
	if errno != 0 {
		return wazerosys.Stat_t{}, errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		if followFinal {
			return w.passthrough.Stat(name)
		}
		return w.passthrough.Lstat(name)
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	node, errno := state.lookupLocked(subpath, followFinal)
	if errno != 0 {
		return wazerosys.Stat_t{}, errno
	}
	return wasmMemoryStat(node, state.readOnly), 0
}

// Mkdir implements experimental/sys.FS.
func (w *WasmMemoryFS) Mkdir(name string, perm fs.FileMode) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.Mkdir(name, perm)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	segments, errno := normalizeWasmMemoryPath(subpath)
	if errno != 0 {
		return errno
	}
	if len(segments) == 0 {
		return experimentalsys.EEXIST
	}
	parent, base, errno := state.parentLocked(subpath)
	if errno != 0 {
		return errno
	}
	if _, exists := parent.children[base]; exists {
		return experimentalsys.EEXIST
	}
	if errno := state.addChildLocked(parent, base, state.newNode(fs.ModeDir|perm.Perm())); errno != 0 {
		return errno
	}
	parent.mtim = time.Now().UnixNano()
	parent.ctim = parent.mtim
	return 0
}

// Chmod implements experimental/sys.FS.
func (w *WasmMemoryFS) Chmod(name string, perm fs.FileMode) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.Chmod(name, perm)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	node, errno := state.lookupLocked(subpath, true)
	if errno != 0 {
		return errno
	}
	node.perm = perm.Perm()
	node.ctim = time.Now().UnixNano()
	return 0
}

// Rename implements experimental/sys.FS.
func (w *WasmMemoryFS) Rename(from, to string) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	fromSubpath, fromMemory := splitWasmMemoryPath(from)
	toSubpath, toMemory := splitWasmMemoryPath(to)
	if fromMemory != toMemory {
		return experimentalsys.ENOSYS
	}
	if !fromMemory {
		return w.passthrough.Rename(from, to)
	}
	return state.rename(fromSubpath, toSubpath)
}

//nolint:gocyclo // Rename compatibility checks must execute under one filesystem lock.
func (state *wasmMemoryFSState) rename(from, to string) experimentalsys.Errno {
	fromSegments, errno := normalizeWasmMemoryPath(from)
	if errno != 0 {
		return errno
	}
	toSegments, errno := normalizeWasmMemoryPath(to)
	if errno != 0 {
		return errno
	}
	if len(fromSegments) == 0 || len(toSegments) == 0 {
		return experimentalsys.EINVAL
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	fromParent, fromBase, errno := state.parentLocked(from)
	if errno != 0 {
		return errno
	}
	node, exists := fromParent.children[fromBase]
	if !exists {
		return experimentalsys.ENOENT
	}
	if strings.Join(fromSegments, "/") == strings.Join(toSegments, "/") {
		return 0
	}
	toParent, toBase, errno := state.parentLocked(to)
	if errno != 0 {
		return errno
	}
	if node.isDir() && wasmMemoryDirectoryContains(node, toParent) {
		return experimentalsys.EINVAL
	}
	if target, ok := toParent.children[toBase]; ok {
		if target == node {
			return 0
		}
		switch {
		case node.isDir() && !target.isDir():
			return experimentalsys.ENOTDIR
		case !node.isDir() && target.isDir():
			return experimentalsys.EISDIR
		case target.isDir() && len(target.children) != 0:
			return experimentalsys.ENOTEMPTY
		}
		state.removeChildLocked(toParent, toBase)
		state.unlinkNodeLocked(target)
	}
	state.removeChildLocked(fromParent, fromBase)
	if errno := state.addChildLocked(toParent, toBase, node); errno != 0 {
		// Removing the source always frees an entry, so reaching this branch
		// would indicate broken quota accounting. Restore the original name.
		_ = state.addChildLocked(fromParent, fromBase, node)
		return errno
	}
	now := time.Now().UnixNano()
	fromParent.mtim, fromParent.ctim = now, now
	toParent.mtim, toParent.ctim = now, now
	node.ctim = now
	return 0
}

func wasmMemoryDirectoryContains(root, candidate *wasmMemoryNode) bool {
	if root == candidate {
		return true
	}
	for _, child := range root.children {
		if child.isDir() && wasmMemoryDirectoryContains(child, candidate) {
			return true
		}
	}
	return false
}

// Rmdir implements experimental/sys.FS.
func (w *WasmMemoryFS) Rmdir(name string) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.Rmdir(name)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	parent, base, errno := state.parentLocked(subpath)
	if errno != 0 {
		return errno
	}
	node, ok := parent.children[base]
	if !ok {
		return experimentalsys.ENOENT
	}
	if !node.isDir() {
		return experimentalsys.ENOTDIR
	}
	if len(node.children) != 0 {
		return experimentalsys.ENOTEMPTY
	}
	state.removeChildLocked(parent, base)
	state.unlinkNodeLocked(node)
	parent.mtim = time.Now().UnixNano()
	parent.ctim = parent.mtim
	return 0
}

// Unlink implements experimental/sys.FS.
func (w *WasmMemoryFS) Unlink(name string) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.Unlink(name)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	parent, base, errno := state.parentLocked(subpath)
	if errno != 0 {
		return errno
	}
	node, ok := parent.children[base]
	if !ok {
		return experimentalsys.ENOENT
	}
	if node.isDir() {
		return experimentalsys.EISDIR
	}
	state.removeChildLocked(parent, base)
	state.unlinkNodeLocked(node)
	parent.mtim = time.Now().UnixNano()
	parent.ctim = parent.mtim
	return 0
}

// Link implements experimental/sys.FS.
func (w *WasmMemoryFS) Link(oldName, newName string) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	oldSubpath, oldMemory := splitWasmMemoryPath(oldName)
	newSubpath, newMemory := splitWasmMemoryPath(newName)
	if oldMemory != newMemory {
		return experimentalsys.ENOSYS
	}
	if !oldMemory {
		return w.passthrough.Link(oldName, newName)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	node, errno := state.lookupLocked(oldSubpath, false)
	if errno != 0 {
		return errno
	}
	if node.isDir() {
		return experimentalsys.EPERM
	}
	parent, base, errno := state.parentLocked(newSubpath)
	if errno != 0 {
		return errno
	}
	if _, exists := parent.children[base]; exists {
		return experimentalsys.EEXIST
	}
	if errno := state.addChildLocked(parent, base, node); errno != 0 {
		return errno
	}
	node.nlink++
	now := time.Now().UnixNano()
	node.ctim = now
	parent.mtim, parent.ctim = now, now
	return 0
}

// Symlink implements experimental/sys.FS.
func (w *WasmMemoryFS) Symlink(oldName, linkName string) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	linkSubpath, memoryPath := splitWasmMemoryPath(linkName)
	if !memoryPath {
		return w.passthrough.Symlink(oldName, linkName)
	}
	if oldName == "" || path.IsAbs(oldName) || strings.IndexByte(oldName, 0) >= 0 {
		return experimentalsys.EPERM
	}
	if len(oldName) > wasmMemoryFSMaxPathBytes {
		return experimentalsys.ENAMETOOLONG
	}
	for _, segment := range strings.Split(oldName, "/") {
		if len(segment) > wasmMemoryFSMaxNameBytes {
			return experimentalsys.ENAMETOOLONG
		}
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	if int64(len(oldName)) > state.maxBytes-state.usedBytes {
		return experimentalsys.ERANGE
	}
	parent, base, errno := state.parentLocked(linkSubpath)
	if errno != 0 {
		return errno
	}
	if _, exists := parent.children[base]; exists {
		return experimentalsys.EEXIST
	}
	node := state.newNode(fs.ModeSymlink | 0o777)
	node.linkTarget = oldName
	if errno := state.addChildLocked(parent, base, node); errno != 0 {
		return errno
	}
	state.usedBytes += int64(len(oldName))
	parent.mtim = time.Now().UnixNano()
	parent.ctim = parent.mtim
	return 0
}

// Readlink implements experimental/sys.FS.
func (w *WasmMemoryFS) Readlink(name string) (string, experimentalsys.Errno) {
	state, errno := w.memoryState()
	if errno != 0 {
		return "", errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.Readlink(name)
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	node, errno := state.lookupLocked(subpath, false)
	if errno != 0 {
		return "", errno
	}
	if !node.isSymlink() {
		return "", experimentalsys.EINVAL
	}
	return node.linkTarget, 0
}

// Utimens implements experimental/sys.FS.
func (w *WasmMemoryFS) Utimens(name string, atim, mtim int64) experimentalsys.Errno {
	state, errno := w.memoryState()
	if errno != 0 {
		return errno
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.passthrough.Utimens(name, atim, mtim)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.readOnly {
		return experimentalsys.EROFS
	}
	node, errno := state.lookupLocked(subpath, true)
	if errno != 0 {
		return errno
	}
	setWasmMemoryTimes(node, atim, mtim)
	return 0
}

func setWasmMemoryTimes(node *wasmMemoryNode, atim, mtim int64) {
	if atim == experimentalsys.UTIME_OMIT && mtim == experimentalsys.UTIME_OMIT {
		return
	}
	if atim != experimentalsys.UTIME_OMIT {
		node.atim = atim
	}
	if mtim != experimentalsys.UTIME_OMIT {
		node.mtim = mtim
	}
	node.ctim = time.Now().UnixNano()
}

type wasmMemoryFile struct {
	experimentalsys.UnimplementedFile

	mu sync.Mutex

	state *wasmMemoryFSState
	node  *wasmMemoryNode

	offset    int64
	directory int
	readable  bool
	writable  bool
	append    bool
	closed    bool
}

// Dev implements experimental/sys.File.
func (file *wasmMemoryFile) Dev() (uint64, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return 0, experimentalsys.EBADF
	}
	return 1, 0
}

// Ino implements experimental/sys.File.
func (file *wasmMemoryFile) Ino() (wazerosys.Inode, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return 0, experimentalsys.EBADF
	}
	return file.node.inode, 0
}

// IsDir implements experimental/sys.File.
func (file *wasmMemoryFile) IsDir() (bool, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return false, experimentalsys.EBADF
	}
	return file.node.isDir(), 0
}

// IsAppend implements experimental/sys.File.
func (file *wasmMemoryFile) IsAppend() bool {
	file.mu.Lock()
	defer file.mu.Unlock()
	return !file.closed && file.append
}

// SetAppend implements experimental/sys.File.
func (file *wasmMemoryFile) SetAppend(enabled bool) experimentalsys.Errno {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return experimentalsys.EBADF
	}
	if file.node.isDir() {
		return experimentalsys.EISDIR
	}
	if !file.writable {
		return experimentalsys.EBADF
	}
	if file.state.readOnly {
		return experimentalsys.EROFS
	}
	file.append = enabled
	return 0
}

// Stat implements experimental/sys.File.
func (file *wasmMemoryFile) Stat() (wazerosys.Stat_t, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return wazerosys.Stat_t{}, experimentalsys.EBADF
	}
	file.state.mu.RLock()
	defer file.state.mu.RUnlock()
	return wasmMemoryStat(file.node, file.state.readOnly), 0
}

// Read implements experimental/sys.File.
func (file *wasmMemoryFile) Read(buffer []byte) (int, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed || !file.readable {
		return 0, experimentalsys.EBADF
	}
	if file.node.isDir() {
		return 0, experimentalsys.EISDIR
	}
	file.state.mu.Lock()
	defer file.state.mu.Unlock()
	read := readWasmMemoryAt(file.node, buffer, file.offset)
	file.offset += int64(read)
	if read > 0 && !file.state.readOnly {
		file.node.atim = time.Now().UnixNano()
	}
	return read, 0
}

// Pread implements experimental/sys.File.
func (file *wasmMemoryFile) Pread(buffer []byte, offset int64) (int, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed || !file.readable {
		return 0, experimentalsys.EBADF
	}
	if offset < 0 {
		return 0, experimentalsys.EINVAL
	}
	if file.node.isDir() {
		return 0, experimentalsys.EISDIR
	}
	file.state.mu.Lock()
	defer file.state.mu.Unlock()
	read := readWasmMemoryAt(file.node, buffer, offset)
	if read > 0 && !file.state.readOnly {
		file.node.atim = time.Now().UnixNano()
	}
	return read, 0
}

func readWasmMemoryAt(node *wasmMemoryNode, buffer []byte, offset int64) int {
	if len(buffer) == 0 || offset < 0 || offset >= int64(len(node.data)) {
		return 0
	}
	return copy(buffer, node.data[int(offset):])
}

// Seek implements experimental/sys.File.
//
//nolint:govet // The experimental/sys.File interface requires an Errno result.
func (file *wasmMemoryFile) Seek(offset int64, whence int) (int64, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return 0, experimentalsys.EBADF
	}
	if file.node.isDir() {
		if offset == 0 && whence == io.SeekStart {
			file.directory = 0
			return 0, 0
		}
		return 0, experimentalsys.EINVAL
	}

	var base int64
	switch whence {
	case io.SeekStart:
		base = 0
	case io.SeekCurrent:
		base = file.offset
	case io.SeekEnd:
		file.state.mu.RLock()
		base = int64(len(file.node.data))
		file.state.mu.RUnlock()
	default:
		return 0, experimentalsys.EINVAL
	}
	position, ok := addWasmMemoryOffset(base, offset)
	if !ok || position < 0 {
		return 0, experimentalsys.EINVAL
	}
	file.offset = position
	return position, 0
}

func addWasmMemoryOffset(base, delta int64) (int64, bool) {
	if delta > 0 && base > math.MaxInt64-delta {
		return 0, false
	}
	if delta < 0 && base < math.MinInt64-delta {
		return 0, false
	}
	return base + delta, true
}

// Readdir implements experimental/sys.File.
func (file *wasmMemoryFile) Readdir(count int) ([]experimentalsys.Dirent, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed || !file.node.isDir() {
		return nil, experimentalsys.EBADF
	}
	file.state.mu.RLock()
	defer file.state.mu.RUnlock()
	entries := makeWasmMemoryDirents(file.node)
	if file.directory >= len(entries) {
		return []experimentalsys.Dirent{}, 0
	}
	end := len(entries)
	if count > 0 && end-file.directory > count {
		end = file.directory + count
	}
	entries = entries[file.directory:end]
	file.directory = end
	return entries, 0
}

func makeWasmMemoryDirents(node *wasmMemoryNode) []experimentalsys.Dirent {
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]experimentalsys.Dirent, 0, len(names))
	for _, name := range names {
		child := node.children[name]
		entries = append(entries, experimentalsys.Dirent{Ino: child.inode, Name: name, Type: child.typeBits})
	}
	return entries
}

// Write implements experimental/sys.File.
func (file *wasmMemoryFile) Write(buffer []byte) (int, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed || !file.writable {
		return 0, experimentalsys.EBADF
	}
	if file.node.isDir() {
		return 0, experimentalsys.EISDIR
	}
	file.state.mu.Lock()
	defer file.state.mu.Unlock()
	if file.state.readOnly {
		return 0, experimentalsys.EROFS
	}
	offset := file.offset
	if file.append {
		offset = int64(len(file.node.data))
	}
	written, errno := file.state.writeNodeAtLocked(file.node, buffer, offset)
	if errno == 0 {
		file.offset = offset + int64(written)
	}
	return written, errno
}

// Pwrite implements experimental/sys.File.
func (file *wasmMemoryFile) Pwrite(buffer []byte, offset int64) (int, experimentalsys.Errno) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed || !file.writable {
		return 0, experimentalsys.EBADF
	}
	if file.node.isDir() {
		return 0, experimentalsys.EISDIR
	}
	if file.append {
		return 0, experimentalsys.EINVAL
	}
	file.state.mu.Lock()
	defer file.state.mu.Unlock()
	if file.state.readOnly {
		return 0, experimentalsys.EROFS
	}
	return file.state.writeNodeAtLocked(file.node, buffer, offset)
}

func (state *wasmMemoryFSState) writeNodeAtLocked(node *wasmMemoryNode, buffer []byte, offset int64) (int, experimentalsys.Errno) {
	if offset < 0 {
		return 0, experimentalsys.EINVAL
	}
	if len(buffer) == 0 {
		return 0, 0
	}
	end, ok := addWasmMemoryOffset(offset, int64(len(buffer)))
	if !ok || end > state.maxBytes {
		return 0, experimentalsys.ERANGE
	}
	oldSize := int64(len(node.data))
	if end > oldSize {
		growth := end - oldSize
		if growth > state.maxBytes-state.usedBytes {
			return 0, experimentalsys.ERANGE
		}
		data := make([]byte, int(end))
		copy(data, node.data)
		node.data = data
		state.usedBytes += growth
	}
	copy(node.data[int(offset):], buffer)
	now := time.Now().UnixNano()
	node.mtim, node.ctim = now, now
	return len(buffer), 0
}

// Truncate implements experimental/sys.File.
func (file *wasmMemoryFile) Truncate(size int64) experimentalsys.Errno {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed || !file.writable {
		return experimentalsys.EBADF
	}
	if file.node.isDir() {
		return experimentalsys.EISDIR
	}
	file.state.mu.Lock()
	defer file.state.mu.Unlock()
	if file.state.readOnly {
		return experimentalsys.EROFS
	}
	return file.state.truncateNodeLocked(file.node, size)
}

func (state *wasmMemoryFSState) truncateNodeLocked(node *wasmMemoryNode, size int64) experimentalsys.Errno {
	if size < 0 {
		return experimentalsys.EINVAL
	}
	if size > state.maxBytes {
		return experimentalsys.ERANGE
	}
	oldSize := int64(len(node.data))
	if size > oldSize {
		growth := size - oldSize
		if growth > state.maxBytes-state.usedBytes {
			return experimentalsys.ERANGE
		}
		data := make([]byte, int(size))
		copy(data, node.data)
		node.data = data
		state.usedBytes += growth
	} else if size < oldSize {
		data := make([]byte, int(size))
		copy(data, node.data[:int(size)])
		node.data = data
		state.usedBytes -= oldSize - size
	}
	now := time.Now().UnixNano()
	node.mtim, node.ctim = now, now
	return 0
}

// Sync implements experimental/sys.File.
func (file *wasmMemoryFile) Sync() experimentalsys.Errno {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return experimentalsys.EBADF
	}
	return 0
}

// Datasync implements experimental/sys.File.
func (file *wasmMemoryFile) Datasync() experimentalsys.Errno {
	return file.Sync()
}

// Utimens implements experimental/sys.File.
func (file *wasmMemoryFile) Utimens(atim, mtim int64) experimentalsys.Errno {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return experimentalsys.EBADF
	}
	file.state.mu.Lock()
	defer file.state.mu.Unlock()
	if file.state.readOnly {
		return experimentalsys.EROFS
	}
	setWasmMemoryTimes(file.node, atim, mtim)
	return 0
}

// Close implements experimental/sys.File.
func (file *wasmMemoryFile) Close() experimentalsys.Errno {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return 0
	}
	file.closed = true
	file.state.mu.Lock()
	if file.node.openCount > 0 {
		file.node.openCount--
	}
	if file.state.openFiles > 0 {
		file.state.openFiles--
	}
	file.state.releaseNodeLocked(file.node)
	file.state.mu.Unlock()
	return 0
}

// Open implements fs.FS. The /memfs namespace is exposed through a read-only
// io/fs handle, while all other paths retain the existing host pass-through.
func (w WasmMemoryFS) Open(name string) (fs.File, error) {
	if err := w.ensureInitialized(); err != nil {
		return nil, err
	}
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath {
		return w.localFS.Open(strings.TrimPrefix(name, "/"))
	}

	w.state.mu.Lock()
	node, errno := w.state.lookupLocked(subpath, true)
	if errno != 0 {
		w.state.mu.Unlock()
		return nil, wasmMemoryPathError("open", name, errno)
	}
	if w.state.openFiles >= w.state.maxOpenFiles {
		w.state.mu.Unlock()
		return nil, wasmMemoryPathError("open", name, experimentalsys.ERANGE)
	}
	node.openCount++
	w.state.openFiles++
	segments, _ := normalizeWasmMemoryPath(subpath)
	displayName := "memfs"
	if len(segments) > 0 {
		displayName += "/" + strings.Join(segments, "/")
	}
	w.state.mu.Unlock()
	return &wasmMemoryFSFile{state: w.state, node: node, name: displayName}, nil
}

// ReadFile implements fs.ReadFileFS.
func (w *WasmMemoryFS) ReadFile(name string) ([]byte, error) {
	file, err := w.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ReadDir implements fs.ReadDirFS.
func (w *WasmMemoryFS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := w.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	directory, ok := file.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	return directory.ReadDir(-1)
}

type wasmMemoryFSFile struct {
	mu sync.Mutex

	state *wasmMemoryFSState
	node  *wasmMemoryNode
	name  string

	offset    int64
	directory int
	closed    bool
}

// Stat implements fs.File.
func (file *wasmMemoryFSFile) Stat() (fs.FileInfo, error) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return nil, fs.ErrClosed
	}
	file.state.mu.RLock()
	defer file.state.mu.RUnlock()
	return newWasmMemoryFileInfo(file.name, file.node, file.state.readOnly), nil
}

// Read implements fs.File.
func (file *wasmMemoryFSFile) Read(buffer []byte) (int, error) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return 0, fs.ErrClosed
	}
	if file.node.isDir() {
		return 0, fs.ErrInvalid
	}
	file.state.mu.Lock()
	read := readWasmMemoryAt(file.node, buffer, file.offset)
	file.offset += int64(read)
	if read > 0 && !file.state.readOnly {
		file.node.atim = time.Now().UnixNano()
	}
	file.state.mu.Unlock()
	if read == 0 && len(buffer) != 0 {
		return 0, io.EOF
	}
	return read, nil
}

// Seek implements io.Seeker.
func (file *wasmMemoryFSFile) Seek(offset int64, whence int) (int64, error) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return 0, fs.ErrClosed
	}
	if file.node.isDir() {
		if offset == 0 && whence == io.SeekStart {
			file.directory = 0
			return 0, nil
		}
		return 0, fs.ErrInvalid
	}
	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = file.offset
	case io.SeekEnd:
		file.state.mu.RLock()
		base = int64(len(file.node.data))
		file.state.mu.RUnlock()
	default:
		return 0, fs.ErrInvalid
	}
	position, ok := addWasmMemoryOffset(base, offset)
	if !ok || position < 0 {
		return 0, fs.ErrInvalid
	}
	file.offset = position
	return position, nil
}

// ReadDir implements fs.ReadDirFile.
func (file *wasmMemoryFSFile) ReadDir(count int) ([]fs.DirEntry, error) {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return nil, fs.ErrClosed
	}
	if !file.node.isDir() {
		return nil, fs.ErrInvalid
	}
	file.state.mu.RLock()
	defer file.state.mu.RUnlock()
	entries := makeWasmMemoryFSDirEntries(file.node, file.state.readOnly)
	if file.directory >= len(entries) {
		if count > 0 {
			return nil, io.EOF
		}
		return []fs.DirEntry{}, nil
	}
	end := len(entries)
	if count > 0 && end-file.directory > count {
		end = file.directory + count
	}
	entries = entries[file.directory:end]
	file.directory = end
	return entries, nil
}

func makeWasmMemoryFSDirEntries(node *wasmMemoryNode, readOnly bool) []fs.DirEntry {
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, &wasmMemoryDirEntry{info: newWasmMemoryFileInfo(name, node.children[name], readOnly)})
	}
	return entries
}

// Close implements fs.File.
func (file *wasmMemoryFSFile) Close() error {
	file.mu.Lock()
	defer file.mu.Unlock()
	if file.closed {
		return nil
	}
	file.closed = true
	file.state.mu.Lock()
	if file.node.openCount > 0 {
		file.node.openCount--
	}
	if file.state.openFiles > 0 {
		file.state.openFiles--
	}
	file.state.releaseNodeLocked(file.node)
	file.state.mu.Unlock()
	return nil
}

type wasmMemoryFileInfo struct {
	name string
	stat wazerosys.Stat_t
}

func newWasmMemoryFileInfo(name string, node *wasmMemoryNode, readOnly bool) *wasmMemoryFileInfo {
	return &wasmMemoryFileInfo{name: name, stat: wasmMemoryStat(node, readOnly)}
}

func (info *wasmMemoryFileInfo) Name() string       { return info.name }
func (info *wasmMemoryFileInfo) Size() int64        { return info.stat.Size }
func (info *wasmMemoryFileInfo) Mode() fs.FileMode  { return info.stat.Mode }
func (info *wasmMemoryFileInfo) ModTime() time.Time { return time.Unix(0, info.stat.Mtim) }
func (info *wasmMemoryFileInfo) IsDir() bool        { return info.stat.Mode.IsDir() }
func (info *wasmMemoryFileInfo) Sys() any {
	stat := info.stat
	return &stat
}

type wasmMemoryDirEntry struct {
	info *wasmMemoryFileInfo
}

func (entry *wasmMemoryDirEntry) Name() string               { return entry.info.Name() }
func (entry *wasmMemoryDirEntry) IsDir() bool                { return entry.info.IsDir() }
func (entry *wasmMemoryDirEntry) Type() fs.FileMode          { return entry.info.Mode().Type() }
func (entry *wasmMemoryDirEntry) Info() (fs.FileInfo, error) { return entry.info, nil }

func wasmMemoryPathError(operation, name string, errno experimentalsys.Errno) error {
	var err error = errno
	switch errno {
	case experimentalsys.ENOENT:
		err = fs.ErrNotExist
	case experimentalsys.EEXIST:
		err = fs.ErrExist
	case experimentalsys.EACCES, experimentalsys.EPERM, experimentalsys.EROFS:
		err = fs.ErrPermission
	case experimentalsys.EINVAL:
		err = fs.ErrInvalid
	case experimentalsys.EBADF:
		err = fs.ErrClosed
	}
	return &fs.PathError{Op: operation, Path: name, Err: err}
}
