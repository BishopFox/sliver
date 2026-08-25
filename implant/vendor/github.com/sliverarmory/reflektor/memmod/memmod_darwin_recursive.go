//go:build darwin && !ios && (amd64 || arm64)

package memmod

import (
	"bytes"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	darwinLCLoadDylib       = uint32(0x0000000c)
	darwinLCIDDylib         = uint32(0x0000000d)
	darwinLCLoadWeakDylib   = uint32(0x80000018)
	darwinLCRPath           = uint32(0x8000001c)
	darwinLCReexportDylib   = uint32(0x8000001f)
	darwinLCLazyLoadDylib   = uint32(0x00000020)
	darwinLCLoadUpwardDylib = uint32(0x80000023)

	darwinMaxRecursiveImages = 512
	darwinMaxRecursiveBytes  = 1 << 30
	darwinMaxDependencyPath  = 4095
)

type darwinRecursiveImage struct {
	path  string
	image []byte
}

type darwinSystemImport struct {
	name string
	weak bool
}

type darwinRecursivePlan struct {
	rootPath      string
	dependencies  []darwinRecursiveImage
	systemImports []darwinSystemImport
}

type darwinLinkedDylib struct {
	name string
	weak bool
}

type darwinImageMetadata struct {
	installName string
	rpaths      []string
	linked      []darwinLinkedDylib
}

type darwinGraphNode struct {
	path     string
	image    []byte
	metadata darwinImageMetadata
	visiting bool
	visited  bool
}

type darwinGraphPlanner struct {
	reader        DependencyReader
	byPath        map[string]*darwinGraphNode
	byInstallName map[string]*darwinGraphNode
	dependencies  []*darwinGraphNode
	systemImports []darwinSystemImport
	systemIndex   map[string]int
	totalBytes    uint64
	executableDir string
}

// LoadLibraryRecursive prepares a Mach-O image together with all of its
// non-system, file-backed dependencies. The dependency bytes are captured now;
// CallExport later maps the complete graph in one private dyld transaction.
func LoadLibraryRecursive(data []byte, origin string, reader DependencyReader) (*Module, error) {
	module, err := LoadLibrary(data)
	if err != nil {
		return nil, err
	}

	plan, err := buildDarwinRecursivePlan(module.image, origin, reader)
	if err != nil {
		module.Free()
		return nil, err
	}
	module.recursive = plan
	return module, nil
}

func buildDarwinRecursivePlan(rootImage []byte, origin string, reader DependencyReader) (*darwinRecursivePlan, error) {
	if uint64(len(rootImage)) > darwinMaxRecursiveBytes {
		return nil, fmt.Errorf("recursive Mach-O graph exceeds %d bytes", darwinMaxRecursiveBytes)
	}
	rootPath, err := normalizeDarwinDependencyPath(origin)
	if err != nil {
		return nil, fmt.Errorf("resolve recursive root origin: %w", err)
	}
	metadata, err := inspectDarwinDependencies(rootImage)
	if err != nil {
		return nil, fmt.Errorf("inspect recursive root %q: %w", rootPath, err)
	}

	planner := &darwinGraphPlanner{
		reader:        reader,
		byPath:        make(map[string]*darwinGraphNode),
		byInstallName: make(map[string]*darwinGraphNode),
		systemIndex:   make(map[string]int),
		totalBytes:    uint64(len(rootImage)),
	}
	if executable, executableErr := os.Executable(); executableErr == nil {
		if executable, executableErr = normalizeDarwinDependencyPath(executable); executableErr == nil {
			planner.executableDir = filepath.Dir(executable)
		}
	}

	root := &darwinGraphNode{path: rootPath, image: rootImage, metadata: metadata}
	planner.byPath[rootPath] = root
	if err := planner.registerInstallName(root); err != nil {
		return nil, err
	}
	if err := planner.visit(root, nil); err != nil {
		return nil, err
	}

	plan := &darwinRecursivePlan{
		rootPath:      rootPath,
		dependencies:  make([]darwinRecursiveImage, 0, len(planner.dependencies)),
		systemImports: append([]darwinSystemImport(nil), planner.systemImports...),
	}
	for _, dependency := range planner.dependencies {
		plan.dependencies = append(plan.dependencies, darwinRecursiveImage{
			path:  dependency.path,
			image: dependency.image,
		})
	}
	return plan, nil
}

func (planner *darwinGraphPlanner) visit(node *darwinGraphNode, inheritedRPaths []string) error {
	if node.visited || node.visiting {
		return nil
	}
	node.visiting = true
	defer func() { node.visiting = false }()

	searchPaths := appendUniqueDarwinPaths(
		planner.expandRPaths(node.path, node.metadata.rpaths),
		inheritedRPaths,
	)
	for _, linked := range node.metadata.linked {
		if isDarwinRecursiveSystemInstallName(linked.name) {
			planner.addSystemImport(linked.name, linked.weak)
			continue
		}

		if existing := planner.findExisting(linked.name, node.path, searchPaths); existing != nil {
			if err := planner.visit(existing, searchPaths); err != nil {
				return err
			}
			continue
		}
		if planner.reader == nil {
			return fmt.Errorf("resolve dependency %q imported by %q: dependency reader is nil", linked.name, node.path)
		}

		request := DependencyRequest{
			Name:         linked.name,
			ImporterPath: node.path,
			SearchPaths:  append([]string(nil), searchPaths...),
		}
		dependency, err := planner.reader(request)
		if err != nil {
			if errors.Is(err, ErrDependencyNotFound) {
				if systemName := planner.expandedSystemInstallName(linked.name, node.path, searchPaths); systemName != "" {
					planner.addSystemImport(systemName, linked.weak)
					continue
				}
			}
			if linked.weak && errors.Is(err, ErrDependencyNotFound) {
				continue
			}
			return fmt.Errorf("resolve dependency %q imported by %q: %w", linked.name, node.path, err)
		}
		if len(dependency.Data) == 0 {
			return fmt.Errorf("resolve dependency %q imported by %q: reader returned an empty image", linked.name, node.path)
		}
		dependencyPath, err := normalizeDarwinDependencyPath(dependency.Path)
		if err != nil {
			return fmt.Errorf("resolve dependency %q imported by %q: invalid returned path: %w", linked.name, node.path, err)
		}
		if isDarwinRecursiveSystemInstallName(dependencyPath) {
			planner.addSystemImport(dependencyPath, linked.weak)
			continue
		}

		if existing := planner.byPath[dependencyPath]; existing != nil {
			if err := planner.visit(existing, searchPaths); err != nil {
				return err
			}
			continue
		}
		if len(planner.dependencies)+1 >= darwinMaxRecursiveImages {
			return fmt.Errorf("recursive Mach-O graph exceeds %d images", darwinMaxRecursiveImages)
		}
		if planner.totalBytes > darwinMaxRecursiveBytes || uint64(len(dependency.Data)) > darwinMaxRecursiveBytes-planner.totalBytes {
			return fmt.Errorf("recursive Mach-O graph exceeds %d bytes", darwinMaxRecursiveBytes)
		}

		image, err := selectCurrentArchMachOSlice(dependency.Data)
		if err != nil {
			return fmt.Errorf("inspect dependency %q at %q: %w", linked.name, dependencyPath, err)
		}
		metadata, err := inspectDarwinDependencies(image)
		if err != nil {
			return fmt.Errorf("inspect dependency %q at %q: %w", linked.name, dependencyPath, err)
		}
		cloned := append([]byte(nil), image...)
		dependencyNode := &darwinGraphNode{
			path:     dependencyPath,
			image:    cloned,
			metadata: metadata,
		}
		planner.byPath[dependencyPath] = dependencyNode
		planner.dependencies = append(planner.dependencies, dependencyNode)
		planner.totalBytes += uint64(len(dependency.Data))
		if err := planner.registerInstallName(dependencyNode); err != nil {
			return err
		}
		if err := planner.visit(dependencyNode, searchPaths); err != nil {
			return err
		}
	}

	node.visited = true
	return nil
}

func (planner *darwinGraphPlanner) expandedSystemInstallName(name string, importerPath string, searchPaths []string) string {
	for _, candidate := range planner.dependencyCandidates(name, importerPath, searchPaths) {
		candidate, err := normalizeDarwinDependencyPath(candidate)
		if err == nil && isDarwinRecursiveSystemInstallName(candidate) {
			return candidate
		}
	}
	return ""
}

func (planner *darwinGraphPlanner) registerInstallName(node *darwinGraphNode) error {
	installName := strings.TrimSpace(node.metadata.installName)
	if installName == "" {
		return nil
	}
	if existing := planner.byInstallName[installName]; existing != nil && existing != node {
		return fmt.Errorf("conflicting Mach-O install name %q for %q and %q", installName, existing.path, node.path)
	}
	planner.byInstallName[installName] = node
	return nil
}

func (planner *darwinGraphPlanner) findExisting(name string, importerPath string, searchPaths []string) *darwinGraphNode {
	if existing := planner.byInstallName[name]; existing != nil {
		return existing
	}
	for _, candidate := range planner.dependencyCandidates(name, importerPath, searchPaths) {
		candidate, err := normalizeDarwinDependencyPath(candidate)
		if err != nil {
			continue
		}
		if existing := planner.byPath[candidate]; existing != nil {
			return existing
		}
	}
	return nil
}

func (planner *darwinGraphPlanner) dependencyCandidates(name string, importerPath string, searchPaths []string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	importerDir := filepath.Dir(importerPath)
	switch {
	case strings.HasPrefix(name, "@loader_path/"):
		return []string{filepath.Join(importerDir, strings.TrimPrefix(name, "@loader_path/"))}
	case strings.HasPrefix(name, "@executable_path/") && planner.executableDir != "":
		return []string{filepath.Join(planner.executableDir, strings.TrimPrefix(name, "@executable_path/"))}
	case strings.HasPrefix(name, "@rpath/"):
		relative := strings.TrimPrefix(name, "@rpath/")
		candidates := make([]string, 0, len(searchPaths))
		for _, searchPath := range searchPaths {
			candidates = append(candidates, filepath.Join(searchPath, relative))
		}
		return candidates
	case filepath.IsAbs(name):
		return []string{name}
	default:
		return []string{filepath.Join(importerDir, name)}
	}
}

func (planner *darwinGraphPlanner) expandRPaths(importerPath string, rpaths []string) []string {
	importerDir := filepath.Dir(importerPath)
	expanded := make([]string, 0, len(rpaths))
	for _, rpath := range rpaths {
		switch {
		case rpath == "@loader_path":
			rpath = importerDir
		case strings.HasPrefix(rpath, "@loader_path/"):
			rpath = filepath.Join(importerDir, strings.TrimPrefix(rpath, "@loader_path/"))
		case rpath == "@executable_path" && planner.executableDir != "":
			rpath = planner.executableDir
		case strings.HasPrefix(rpath, "@executable_path/") && planner.executableDir != "":
			rpath = filepath.Join(planner.executableDir, strings.TrimPrefix(rpath, "@executable_path/"))
		case rpath == "@rpath" || strings.HasPrefix(rpath, "@rpath/"):
			// An @rpath inside LC_RPATH cannot be expanded without an already
			// established runpath stack, so do not guess a process-relative path.
			continue
		}
		rpath, err := normalizeDarwinDependencyPath(rpath)
		if err == nil {
			expanded = append(expanded, rpath)
		}
	}
	return appendUniqueDarwinPaths(expanded, nil)
}

func (planner *darwinGraphPlanner) addSystemImport(name string, weak bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if index, exists := planner.systemIndex[name]; exists {
		if planner.systemImports[index].weak && !weak {
			planner.systemImports[index].weak = false
		}
		return
	}
	planner.systemIndex[name] = len(planner.systemImports)
	planner.systemImports = append(planner.systemImports, darwinSystemImport{name: name, weak: weak})
}

func inspectDarwinDependencies(image []byte) (darwinImageMetadata, error) {
	file, err := macho.NewFile(bytes.NewReader(image))
	if err != nil {
		return darwinImageMetadata{}, fmt.Errorf("parse Mach-O dependency commands: %w", err)
	}
	defer file.Close()

	var metadata darwinImageMetadata
	for _, load := range file.Loads {
		raw := load.Raw()
		if len(raw) < 8 {
			return darwinImageMetadata{}, errors.New("Mach-O load command is shorter than 8 bytes")
		}
		command := file.ByteOrder.Uint32(raw[:4])
		switch command {
		case darwinLCLoadDylib, darwinLCLoadWeakDylib, darwinLCReexportDylib,
			darwinLCLazyLoadDylib, darwinLCLoadUpwardDylib, darwinLCIDDylib:
			if len(raw) < 24 {
				return darwinImageMetadata{}, fmt.Errorf("Mach-O dylib command %#x is shorter than 24 bytes", command)
			}
			name, err := darwinLoadCommandString(raw, file.ByteOrder.Uint32(raw[8:12]))
			if err != nil {
				return darwinImageMetadata{}, fmt.Errorf("read Mach-O dylib command %#x: %w", command, err)
			}
			if command == darwinLCIDDylib {
				if metadata.installName != "" && metadata.installName != name {
					return darwinImageMetadata{}, fmt.Errorf("multiple Mach-O install names %q and %q", metadata.installName, name)
				}
				metadata.installName = name
				continue
			}
			metadata.linked = append(metadata.linked, darwinLinkedDylib{
				name: name,
				weak: command == darwinLCLoadWeakDylib,
			})
		case darwinLCRPath:
			if len(raw) < 12 {
				return darwinImageMetadata{}, errors.New("Mach-O LC_RPATH command is shorter than 12 bytes")
			}
			path, err := darwinLoadCommandString(raw, file.ByteOrder.Uint32(raw[8:12]))
			if err != nil {
				return darwinImageMetadata{}, fmt.Errorf("read Mach-O LC_RPATH: %w", err)
			}
			metadata.rpaths = append(metadata.rpaths, path)
		}
	}
	return metadata, nil
}

func darwinLoadCommandString(command []byte, offset uint32) (string, error) {
	if offset >= uint32(len(command)) {
		return "", fmt.Errorf("string offset %d is outside a %d-byte load command", offset, len(command))
	}
	data := command[offset:]
	if nul := bytes.IndexByte(data, 0); nul >= 0 {
		data = data[:nul]
	}
	if len(data) == 0 {
		return "", errors.New("load command string is empty")
	}
	if len(data) > darwinMaxDependencyPath {
		return "", fmt.Errorf("load command path exceeds %d bytes", darwinMaxDependencyPath)
	}
	return string(data), nil
}

func normalizeDarwinDependencyPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is empty")
	}
	if strings.ContainsRune(path, 0) {
		return "", errors.New("path contains NUL")
	}
	if len(path) > darwinMaxDependencyPath {
		return "", fmt.Errorf("path exceeds %d bytes", darwinMaxDependencyPath)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	if evaluated, evalErr := filepath.EvalSymlinks(absolute); evalErr == nil {
		absolute = evaluated
	}
	return absolute, nil
}

func appendUniqueDarwinPaths(first []string, second []string) []string {
	paths := make([]string, 0, len(first)+len(second))
	seen := make(map[string]struct{}, len(first)+len(second))
	for _, group := range [][]string{first, second} {
		for _, path := range group {
			path = filepath.Clean(strings.TrimSpace(path))
			if path == "" || path == "." {
				continue
			}
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	return paths
}

func isDarwinRecursiveSystemInstallName(installName string) bool {
	return isDarwinSystemInstallName(installName) ||
		strings.HasPrefix(installName, "/System/iOSSupport/usr/lib/") ||
		strings.HasPrefix(installName, "/System/iOSSupport/System/Library/") ||
		strings.HasPrefix(installName, "/System/DriverKit/")
}
