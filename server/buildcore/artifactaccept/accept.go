// Package artifactaccept validates and identifies finished build artifacts.
package artifactaccept

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/util"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	resultFileName             = "result.json"
	maxResultBytes             = 16 << 10
	maxPrimaryBytes      int64 = 512 << 20
	maxArchiveEntries          = 8192
	maxArchiveEntryBytes       = 512 << 20
	maxArchiveTotalBytes       = 1 << 30
	maxCompressionRatio        = 1000
	maxARMembers               = 65536
	maxARNameTableBytes        = 16 << 20
)

const arMemberHeaderSize = 60

var arMagic = []byte("!<arch>\n")

// Result is the location of a Mage target's one primary output. Its fields
// are location and display hints only: the frozen plan determines artifact
// type, and the server computes the output's size and hash itself.
type Result struct {
	Primary     string `json:"primary"`
	DisplayName string `json:"displayName"`
}

// Accept reads a successful Mage target's output under binDir and verifies it
// before any transformer receives the bytes. Archive inspection never extracts
// files into the filesystem.
func Accept(binDir string, expected buildplan.Artifact) ([]byte, buildplan.ArtifactInfo, error) {
	dirInfo, err := os.Lstat(binDir)
	if err != nil {
		return nil, buildplan.ArtifactInfo{}, fmt.Errorf("open core output directory: %w", err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return nil, buildplan.ArtifactInfo{}, errors.New("core output directory is not a directory")
	}
	manifestBytes, err := readRegularFile(filepath.Join(binDir, resultFileName), maxResultBytes)
	if err != nil {
		return nil, buildplan.ArtifactInfo{}, fmt.Errorf("read core result: %w", err)
	}
	var result Result
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return nil, buildplan.ArtifactInfo{}, fmt.Errorf("parse core result: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return nil, buildplan.ArtifactInfo{}, errors.New("core result contains trailing data")
	}
	if err := validateBasename(result.Primary); err != nil || result.Primary == resultFileName {
		return nil, buildplan.ArtifactInfo{}, errors.New("core result primary must be a file basename under bin")
	}
	if err := validateBasename(result.DisplayName); err != nil {
		return nil, buildplan.ArtifactInfo{}, errors.New("core result displayName must be a basename")
	}
	data, err := readRegularFile(filepath.Join(binDir, result.Primary), maxPrimaryBytes)
	if err != nil {
		return nil, buildplan.ArtifactInfo{}, fmt.Errorf("read core primary: %w", err)
	}
	info, err := InspectBytes(data, expected, result.DisplayName)
	if err != nil {
		return nil, buildplan.ArtifactInfo{}, fmt.Errorf("validate core primary: %w", err)
	}
	return data, info, nil
}

// InspectBytes validates a completed in-memory step output against its frozen
// artifact type and target, then computes its size and SHA-256. It does not
// retain bytes or read from the filesystem. Callers keep the bytes private
// until the final transform has been accepted and stored.
func InspectBytes(data []byte, expected buildplan.Artifact, displayName string) (buildplan.ArtifactInfo, error) {
	if err := validateBasename(displayName); err != nil {
		return buildplan.ArtifactInfo{}, errors.New("artifact display name must be a basename")
	}
	if len(data) == 0 {
		return buildplan.ArtifactInfo{}, errors.New("artifact is empty")
	}
	if int64(len(data)) > maxPrimaryBytes {
		return buildplan.ArtifactInfo{}, fmt.Errorf("artifact exceeds %d byte limit", maxPrimaryBytes)
	}
	if err := validateBinary(data, expected); err != nil {
		return buildplan.ArtifactInfo{}, fmt.Errorf("artifact format or target is invalid: %w", err)
	}
	hash := sha256.Sum256(data)
	return buildplan.ArtifactInfo{
		Type: expected, Name: displayName, Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:]),
	}, nil
}

func validateBasename(name string) error {
	if !util.ValidPortableFileName(name) {
		return errors.New("invalid basename")
	}
	return nil
}

// A regular file is opened only if its directory entry is not a symlink. The
// identity check also catches replacement of that entry between Lstat and Open.
func readRegularFile(filePath string, maxBytes int64) ([]byte, error) {
	before, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	if before.Size() > maxBytes {
		return nil, fmt.Errorf("file exceeds %d byte limit", maxBytes)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, errors.New("file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes || int64(len(data)) != before.Size() {
		return nil, errors.New("file changed size while reading or exceeds limit")
	}
	return data, nil
}

func validateBinary(data []byte, expected buildplan.Artifact) error {
	switch expected.Container {
	case "pe":
		if expected.Target.OS != "windows" {
			return errors.New("PE output requires a Windows target")
		}
		return validatePE(data, expected)
	case "elf":
		if expected.Target.OS != "linux" {
			return errors.New("ELF output requires a Linux target")
		}
		return validateELF(data, expected)
	case "macho":
		if expected.Target.OS != "darwin" {
			return errors.New("Mach-O output requires a Darwin target")
		}
		return validateMachO(data, expected)
	case "ar":
		if expected.Kind != "go-archive" && expected.Kind != "c-archive" {
			return errors.New("unexpected ar artifact kind")
		}
		return validateAR(bytes.NewReader(data), uint64(len(data)))
	case "zip":
		if expected.Kind != "c-archive-bundle" {
			return errors.New("unexpected ZIP artifact kind")
		}
		return validateZIP(data, true)
	case "raw":
		if expected.Kind != "shellcode" && expected.Kind != "stage" {
			return errors.New("unexpected raw artifact kind")
		}
		return nil // Raw machine code has no portable header to identify its target.
	default:
		return fmt.Errorf("unsupported artifact container %q", expected.Container)
	}
}

func validatePE(data []byte, expected buildplan.Artifact) error {
	file, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	want := map[string]uint16{"386": 0x014c, "amd64": 0x8664, "arm64": 0xaa64}[expected.Target.Arch]
	if want == 0 || file.Machine != want {
		return fmt.Errorf("PE machine %#x does not match %s", file.Machine, expected.Target.Arch)
	}
	if expected.Target.Arch == "386" {
		if _, ok := file.OptionalHeader.(*pe.OptionalHeader32); !ok {
			return errors.New("PE optional header does not match 32-bit target")
		}
	} else {
		if _, ok := file.OptionalHeader.(*pe.OptionalHeader64); !ok {
			return errors.New("PE optional header does not match 64-bit target")
		}
	}
	dll := file.Characteristics&0x2000 != 0
	switch expected.Kind {
	case "shared-library":
		if !dll {
			return errors.New("PE shared library lacks DLL characteristic")
		}
	case "executable", "service-executable":
		if dll {
			return errors.New("PE executable has DLL characteristic")
		}
	default:
		return fmt.Errorf("unexpected PE artifact kind %q", expected.Kind)
	}
	return nil
}

func validateELF(data []byte, expected buildplan.Artifact) error {
	file, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	want := map[string]elf.Machine{"386": elf.EM_386, "amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64}[expected.Target.Arch]
	if want == 0 || file.Machine != want {
		return fmt.Errorf("ELF machine %s does not match %s", file.Machine, expected.Target.Arch)
	}
	wantClass := elf.ELFCLASS64
	if expected.Target.Arch == "386" {
		wantClass = elf.ELFCLASS32
	}
	if file.Class != wantClass {
		return fmt.Errorf("ELF class %s does not match %s", file.Class, expected.Target.Arch)
	}
	switch expected.Kind {
	case "shared-library", "go-plugin", "go-shared-library":
		if file.Type != elf.ET_DYN {
			return fmt.Errorf("ELF shared output has type %s", file.Type)
		}
	case "executable":
		if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN { // PIE is ET_DYN.
			return fmt.Errorf("ELF executable has type %s", file.Type)
		}
	default:
		return fmt.Errorf("unexpected ELF artifact kind %q", expected.Kind)
	}
	return nil
}

func validateMachO(data []byte, expected buildplan.Artifact) error {
	file, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	want := map[string]macho.Cpu{"amd64": macho.CpuAmd64, "arm64": macho.CpuArm64}[expected.Target.Arch]
	if want == 0 || file.Cpu != want {
		return fmt.Errorf("Mach-O CPU %s does not match %s", file.Cpu, expected.Target.Arch)
	}
	switch expected.Kind {
	case "executable":
		if file.Type != macho.TypeExec {
			return fmt.Errorf("Mach-O executable has type %s", file.Type)
		}
	case "shared-library", "go-shared-library":
		if file.Type != macho.TypeDylib {
			return fmt.Errorf("Mach-O shared output has type %s", file.Type)
		}
	case "go-plugin":
		if file.Type != macho.TypeBundle && file.Type != macho.TypeDylib {
			return fmt.Errorf("Mach-O plugin has type %s", file.Type)
		}
	default:
		return fmt.Errorf("unexpected Mach-O artifact kind %q", expected.Kind)
	}
	return nil
}

func validateZIP(data []byte, requireCArchive bool) error {
	reader, err := openZIP(data)
	if err != nil {
		return err
	}
	count := len(reader.File)
	names := zipNames{
		seen: make(map[string]bool, count), impliedDirs: make(map[string]bool, count),
		spellings: make(map[string]string, count),
	}
	var total uint64
	archiveCount, headerCount := 0, 0
	for _, entry := range reader.File {
		name, isDir, err := safeArchiveName(entry.Name)
		if err != nil {
			return fmt.Errorf("unsafe ZIP entry %q: %w", entry.Name, err)
		}
		if err := names.add(entry.Name, name, isDir); err != nil {
			return err
		}
		if err := validateZIPEntryHeader(entry, isDir, total); err != nil {
			return err
		}
		if isDir {
			continue
		}
		total += entry.UncompressedSize64
		isArchive := strings.EqualFold(path.Ext(name), ".a")
		if isArchive {
			archiveCount++
		}
		if strings.EqualFold(path.Ext(name), ".h") {
			headerCount++
		}
		if err := inspectZIPEntry(entry, requireCArchive && isArchive); err != nil {
			return err
		}
	}
	if requireCArchive && (archiveCount != 1 || headerCount == 0) {
		return errors.New("a C archive ZIP must contain exactly one .a and at least one .h")
	}
	return nil
}

func openZIP(data []byte) (*zip.Reader, error) {
	if err := precheckZIPDirectory(data); err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	count := len(reader.File)
	if count == 0 || count > maxArchiveEntries {
		return nil, fmt.Errorf("ZIP entry count must be between 1 and %d", maxArchiveEntries)
	}
	return reader, nil
}

type zipNames struct {
	seen        map[string]bool
	impliedDirs map[string]bool
	spellings   map[string]string
}

func (names zipNames) add(rawName, name string, isDir bool) error {
	key := norm.NFC.String(cases.Fold().String(norm.NFC.String(name)))
	if names.seen[key] || !isDir && names.impliedDirs[key] {
		return fmt.Errorf("duplicate or conflicting ZIP entry %q", rawName)
	}
	names.seen[key] = true
	parts := strings.Split(key, "/")
	originalParts := strings.Split(name, "/")
	for i := range parts {
		canonical := strings.Join(parts[:i+1], "/")
		spelling := strings.Join(originalParts[:i+1], "/")
		if prior, exists := names.spellings[canonical]; exists && prior != spelling {
			return fmt.Errorf("case or Unicode alias in ZIP entry %q", rawName)
		}
		names.spellings[canonical] = spelling
	}
	for i := 1; i < len(parts); i++ {
		ancestor := strings.Join(parts[:i], "/")
		if names.seen[ancestor] && !names.impliedDirs[ancestor] {
			return fmt.Errorf("ZIP entry %q has file ancestor", rawName)
		}
		names.impliedDirs[ancestor] = true
	}
	if isDir {
		names.impliedDirs[key] = true
	}
	return nil
}

func validateZIPEntryHeader(entry *zip.File, isDir bool, total uint64) error {
	mode := entry.Mode()
	if mode&os.ModeSymlink != 0 || mode&os.ModeType != 0 && !mode.IsDir() || isDir != mode.IsDir() && mode&os.ModeType != 0 {
		return fmt.Errorf("ZIP entry %q is a symlink, special file, or mismatched directory", entry.Name)
	}
	if entry.Flags&1 != 0 { // encrypted ZIP entries cannot be verified here.
		return fmt.Errorf("ZIP entry %q is encrypted", entry.Name)
	}
	if entry.Method != zip.Store && entry.Method != zip.Deflate {
		return fmt.Errorf("ZIP entry %q uses unsupported compression", entry.Name)
	}
	if isDir {
		if entry.UncompressedSize64 != 0 || entry.CompressedSize64 != 0 {
			return fmt.Errorf("ZIP directory %q has contents", entry.Name)
		}
		return nil
	}
	return validateZIPEntrySize(entry, total)
}

func validateZIPEntrySize(entry *zip.File, total uint64) error {
	if entry.UncompressedSize64 > maxArchiveEntryBytes ||
		entry.UncompressedSize64 > maxArchiveTotalBytes-total {
		return errors.New("ZIP expansion exceeds limit")
	}
	if entry.UncompressedSize64 > 0 && (entry.CompressedSize64 == 0 ||
		entry.UncompressedSize64/entry.CompressedSize64 > maxCompressionRatio) {
		return fmt.Errorf("ZIP entry %q has excessive expansion ratio", entry.Name)
	}
	return nil
}

func inspectZIPEntry(entry *zip.File, inspectAR bool) error {
	content, err := entry.Open()
	if err != nil {
		return fmt.Errorf("open ZIP entry %q: %w", entry.Name, err)
	}
	if inspectAR {
		err = inspectZIPArchive(content, entry.UncompressedSize64)
	} else {
		var read uint64
		read, err = copyBounded(content, entry.UncompressedSize64)
		if err == nil && read != entry.UncompressedSize64 {
			err = errors.New("ZIP entry size differs from header")
		}
	}
	closeErr := content.Close()
	if err != nil {
		return fmt.Errorf("inspect ZIP entry %q: %w", entry.Name, err)
	}
	if closeErr != nil {
		return fmt.Errorf("close ZIP entry %q: %w", entry.Name, closeErr)
	}
	return nil
}

func inspectZIPArchive(content io.Reader, size uint64) error {
	if err := validateAR(content, size); err != nil {
		return err
	}
	var tail [1]byte
	_, err := content.Read(tail[:])
	if errors.Is(err, io.EOF) {
		return nil // Force ZIP's CRC check at EOF.
	}
	if err == nil {
		return errors.New("ZIP archive member has trailing data")
	}
	return err
}

// Bound the central directory's declared entry count before archive/zip
// allocates its File slice. ZIP64 and self-extracting ZIPs are unnecessary for
// core outputs and are rejected rather than trusting their extended metadata.
func precheckZIPDirectory(data []byte) error {
	const endRecordSize = 22
	if len(data) < endRecordSize {
		return errors.New("ZIP has no end record")
	}
	start := len(data) - endRecordSize - 65535
	if start < 0 {
		start = 0
	}
	for offset := len(data) - endRecordSize; offset >= start; offset-- {
		if !bytes.Equal(data[offset:offset+4], []byte("PK\x05\x06")) {
			continue
		}
		end := data[offset : offset+endRecordSize]
		if int(binary.LittleEndian.Uint16(end[20:22])) != len(data)-offset-endRecordSize {
			continue
		}
		if binary.LittleEndian.Uint16(end[4:6]) != 0 || binary.LittleEndian.Uint16(end[6:8]) != 0 {
			return errors.New("multi-disk ZIP is unsupported")
		}
		entriesOnDisk := binary.LittleEndian.Uint16(end[8:10])
		entries := binary.LittleEndian.Uint16(end[10:12])
		if entries == 0xffff || entriesOnDisk != entries || entries == 0 || entries > maxArchiveEntries {
			return errors.New("ZIP entry count is invalid or exceeds limit")
		}
		centralSize := binary.LittleEndian.Uint32(end[12:16])
		centralOffset := binary.LittleEndian.Uint32(end[16:20])
		if centralSize == 0xffffffff || centralOffset == 0xffffffff ||
			uint64(centralOffset)+uint64(centralSize) != uint64(offset) {
			return errors.New("ZIP central directory is invalid or ZIP64")
		}
		return nil
	}
	return errors.New("ZIP has no valid end record")
}

func copyBounded(reader io.Reader, declared uint64) (uint64, error) {
	// The extra byte detects a forged uncompressed size without trusting the ZIP
	// central directory. The per-entry limit was checked before this call.
	read, err := io.Copy(io.Discard, io.LimitReader(reader, int64(declared)+1))
	if err != nil {
		return 0, err
	}
	if uint64(read) > declared {
		return 0, errors.New("archive member exceeds declared size")
	}
	return uint64(read), nil
}

func safeArchiveName(name string) (string, bool, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return "", false, errors.New("absolute or invalid path")
	}
	isDir := strings.HasSuffix(name, "/")
	trimmed := strings.TrimSuffix(name, "/")
	for _, component := range strings.Split(trimmed, "/") {
		if component == "" || component == "." || component == ".." {
			return "", false, errors.New("empty or traversal component")
		}
	}
	return trimmed, isDir, nil
}

func validateAR(reader io.Reader, size uint64) error {
	if size < uint64(len(arMagic)) {
		return errors.New("ar archive is truncated")
	}
	var magic [8]byte
	if _, err := io.ReadFull(reader, magic[:]); err != nil {
		return err
	}
	if !bytes.Equal(magic[:], arMagic) {
		return errors.New("ar archive has invalid magic")
	}
	remaining := size - uint64(len(magic))
	var members arMembers
	for count := 0; remaining > 0; count++ {
		if count >= maxARMembers {
			return errors.New("ar archive has too many members")
		}
		name, memberSize, err := readARMemberHeader(reader, remaining)
		if err != nil {
			return err
		}
		remaining -= arMemberHeaderSize + memberSize
		if err := members.validateMember(reader, name, memberSize); err != nil {
			return err
		}
		if memberSize%2 == 1 {
			if err := readARPadding(reader, remaining); err != nil {
				return err
			}
			remaining--
		}
	}
	if members.regular == 0 {
		return errors.New("ar archive has no file members")
	}
	return nil
}

func readARMemberHeader(reader io.Reader, remaining uint64) (string, uint64, error) {
	var header [arMemberHeaderSize]byte
	headerSize := uint64(len(header))
	if remaining < headerSize {
		return "", 0, errors.New("ar archive has truncated member header")
	}
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return "", 0, err
	}
	if string(header[58:60]) != "`\n" {
		return "", 0, errors.New("ar archive has invalid member header")
	}
	// Member sizes are passed to io.CopyN, which accepts an int64.
	size, err := strconv.ParseUint(strings.TrimSpace(string(header[48:58])), 10, 63)
	if err != nil || size > remaining-headerSize {
		return "", 0, errors.New("ar archive has invalid member size")
	}
	return strings.TrimSpace(string(header[:16])), size, nil
}

type arMembers struct {
	nameTable []byte
	regular   int
}

func (members *arMembers) validateMember(reader io.Reader, name string, size uint64) error {
	switch {
	case name == "//":
		if size > maxARNameTableBytes || members.nameTable != nil {
			return errors.New("ar archive has oversized or duplicate name table")
		}
		members.nameTable = make([]byte, size)
		_, err := io.ReadFull(reader, members.nameTable)
		return err
	case name == "/" || name == "/SYM64/":
		_, err := io.CopyN(io.Discard, reader, int64(size))
		return err
	case strings.HasPrefix(name, "#1/"):
		var err error
		name, size, err = readARExtendedName(reader, name, size)
		if err != nil {
			return err
		}
	case strings.HasPrefix(name, "/"):
		var err error
		name, err = members.lookupName(name)
		if err != nil {
			return err
		}
	}
	name = strings.TrimSuffix(name, "/")
	if _, _, err := safeArchiveName(name); err != nil {
		return fmt.Errorf("unsafe ar member %q: %w", name, err)
	}
	members.regular++
	_, err := io.CopyN(io.Discard, reader, int64(size))
	return err
}

func readARExtendedName(reader io.Reader, name string, size uint64) (string, uint64, error) {
	nameSize, err := strconv.ParseUint(strings.TrimPrefix(name, "#1/"), 10, 64)
	if err != nil || nameSize == 0 || nameSize > size || nameSize > 4096 {
		return "", 0, errors.New("ar archive has invalid extended name")
	}
	nameBytes := make([]byte, nameSize)
	if _, err := io.ReadFull(reader, nameBytes); err != nil {
		return "", 0, err
	}
	return strings.TrimRight(string(nameBytes), "\x00"), size - nameSize, nil
}

func (members *arMembers) lookupName(name string) (string, error) {
	offset, err := strconv.ParseUint(strings.TrimPrefix(name, "/"), 10, 64)
	if err != nil || offset >= uint64(len(members.nameTable)) {
		return "", errors.New("ar archive has invalid long-name reference")
	}
	end := bytes.IndexByte(members.nameTable[offset:], '\n')
	if end < 0 {
		return "", errors.New("ar archive has unterminated long name")
	}
	return string(members.nameTable[offset : offset+uint64(end)]), nil
}

func readARPadding(reader io.Reader, remaining uint64) error {
	if remaining == 0 {
		return errors.New("ar archive is missing alignment byte")
	}
	var pad [1]byte
	if _, err := io.ReadFull(reader, pad[:]); err != nil {
		return err
	}
	if pad[0] != '\n' {
		return errors.New("ar archive has invalid alignment byte")
	}
	return nil
}
