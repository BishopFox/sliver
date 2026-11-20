package workspace

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// ArchiveLimits bounds installed core bundles before they are made available
// to a build. Zero-valued fields use the defaults below.
type ArchiveLimits struct {
	MaxArchiveBytes int64
	MaxEntries      int
	MaxFileBytes    int64
	MaxTotalBytes   int64
}

const (
	defaultMaxArchiveBytes = 256 << 20
	defaultMaxEntries      = 50000
	defaultMaxFileBytes    = 128 << 20
	defaultMaxTotalBytes   = 512 << 20
)

// ExtractSourceArchive installs a ZIP, TAR, or TAR.GZ core source bundle.
// The destination must be empty. Archive entries are never allowed to create
// links or special files, and only owner-readable files are created.
func (w *Workspace) ExtractSourceArchive(archivePath string, limits ArchiveLimits) error {
	if w == nil {
		return errors.New("missing build workspace")
	}
	return extractArchive(archivePath, w.SourceDir, limits)
}

// ExtractMagefilesArchive installs a Magefile bundle with the same safeguards.
func (w *Workspace) ExtractMagefilesArchive(archivePath string, limits ArchiveLimits) error {
	if w == nil {
		return errors.New("missing build workspace")
	}
	return extractArchive(archivePath, w.MagefilesDir, limits)
}

func extractArchive(archivePath, dest string, limits ArchiveLimits) error {
	limits, err := normalizedLimits(limits)
	if err != nil {
		return err
	}
	info, err := os.Lstat(archivePath)
	if err != nil {
		return fmt.Errorf("stat core archive: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > limits.MaxArchiveBytes {
		return errors.New("core archive is not a regular file within the size limit")
	}
	if err := requireEmptyDir(dest); err != nil {
		return err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return fmt.Errorf("open archive destination: %w", err)
	}
	defer func() { _ = root.Close() }()

	name := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(name, ".zip"):
		return extractZIP(archivePath, root, limits)
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return extractTAR(archivePath, root, limits, true)
	case strings.HasSuffix(name, ".tar"):
		return extractTAR(archivePath, root, limits, false)
	default:
		return errors.New("unsupported core archive format")
	}
}

func normalizedLimits(limits ArchiveLimits) (ArchiveLimits, error) {
	if limits.MaxArchiveBytes < 0 || limits.MaxEntries < 0 || limits.MaxFileBytes < 0 || limits.MaxTotalBytes < 0 {
		return ArchiveLimits{}, errors.New("negative core archive limit")
	}
	if limits.MaxArchiveBytes == 0 {
		limits.MaxArchiveBytes = defaultMaxArchiveBytes
	}
	if limits.MaxEntries == 0 {
		limits.MaxEntries = defaultMaxEntries
	}
	if limits.MaxFileBytes == 0 {
		limits.MaxFileBytes = defaultMaxFileBytes
	}
	if limits.MaxTotalBytes == 0 {
		limits.MaxTotalBytes = defaultMaxTotalBytes
	}
	if limits.MaxFileBytes > limits.MaxTotalBytes {
		limits.MaxFileBytes = limits.MaxTotalBytes
	}
	if limits.MaxArchiveBytes > defaultMaxArchiveBytes ||
		limits.MaxEntries > defaultMaxEntries ||
		limits.MaxFileBytes > defaultMaxFileBytes ||
		limits.MaxTotalBytes > defaultMaxTotalBytes {
		return ArchiveLimits{}, errors.New("core archive limits exceed hard maximum")
	}
	return limits, nil
}

type extractionBudget struct {
	limits  ArchiveLimits
	entries int
	bytes   int64
	seen    map[string]archivePathEntry
}

type archivePathEntry struct {
	name      string
	directory bool
	explicit  bool
}

func (budget *extractionBudget) addPath(name string, directory bool) (string, error) {
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	if !directory && strings.HasSuffix(name, "/") {
		return "", fmt.Errorf("core archive file %q has a directory path", name)
	}
	name = strings.TrimSuffix(name, "/")
	local, err := safeEntryPath(name)
	if err != nil {
		return "", err
	}
	budget.entries++
	if budget.entries > budget.limits.MaxEntries {
		return "", errors.New("core archive has too many entries")
	}
	if err := budget.recordPath(name, directory); err != nil {
		return "", err
	}
	return local, nil
}

func (budget *extractionBudget) recordPath(name string, directory bool) error {
	parts := strings.Split(name, "/")
	for index := range parts {
		current := strings.Join(parts[:index+1], "/")
		key := norm.NFC.String(cases.Fold().String(norm.NFC.String(current)))
		isLeaf := index == len(parts)-1
		entry, exists := budget.seen[key]
		if exists {
			if entry.name != current || !entry.directory && (!isLeaf || directory) ||
				isLeaf && (entry.explicit || !directory) {
				return fmt.Errorf("duplicate or conflicting core archive path %q", name)
			}
			if isLeaf {
				entry.explicit = true
				budget.seen[key] = entry
			}
			continue
		}
		budget.seen[key] = archivePathEntry{
			name: current, directory: !isLeaf || directory, explicit: isLeaf,
		}
	}
	return nil
}

func (budget *extractionBudget) allowFile(size int64) error {
	if size < 0 || size > budget.limits.MaxFileBytes || size > budget.limits.MaxTotalBytes-budget.bytes {
		return errors.New("core archive exceeds uncompressed size limit")
	}
	return nil
}

func (budget *extractionBudget) addBytes(size int64) error {
	if err := budget.allowFile(size); err != nil {
		return err
	}
	budget.bytes += size
	return nil
}

func extractZIP(archivePath string, root *os.Root, limits ArchiveLimits) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open core ZIP: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limits.MaxArchiveBytes {
		return errors.New("core ZIP is not a regular file within the size limit")
	}
	if err := precheckZIPDirectory(file, info.Size(), limits.MaxEntries); err != nil {
		return err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("parse core ZIP: %w", err)
	}
	if len(reader.File) > limits.MaxEntries {
		return errors.New("core ZIP has too many entries")
	}
	budget := extractionBudget{limits: limits, seen: make(map[string]archivePathEntry, len(reader.File))}
	for _, entry := range reader.File {
		if err := extractZIPEntry(root, &budget, entry); err != nil {
			return err
		}
	}
	return nil
}

func extractZIPEntry(root *os.Root, budget *extractionBudget, entry *zip.File) error {
	mode := entry.Mode()
	isDir := mode.IsDir()
	if isDir != strings.HasSuffix(entry.Name, "/") || entry.Flags&1 != 0 ||
		entry.Method != zip.Store && entry.Method != zip.Deflate {
		return fmt.Errorf("unsupported ZIP entry %q", entry.Name)
	}
	local, err := budget.addPath(entry.Name, isDir)
	if err != nil {
		return err
	}
	if isDir {
		if mode&fs.ModeType != fs.ModeDir {
			return fmt.Errorf("unsupported ZIP directory type %q", entry.Name)
		}
		if entry.UncompressedSize64 != 0 || entry.CompressedSize64 != 0 {
			return fmt.Errorf("core ZIP directory %q has content", entry.Name)
		}
		if err := root.MkdirAll(local, 0o700); err != nil {
			return fmt.Errorf("create core ZIP directory: %w", err)
		}
		return nil
	}
	return extractZIPFile(root, budget, entry, mode, local)
}

func extractZIPFile(root *os.Root, budget *extractionBudget, entry *zip.File, mode fs.FileMode, local string) error {
	limits := budget.limits
	if !mode.IsRegular() {
		return fmt.Errorf("unsupported ZIP entry type %q", entry.Name)
	}
	if entry.UncompressedSize64 > uint64(limits.MaxFileBytes) ||
		entry.UncompressedSize64 > uint64(limits.MaxTotalBytes-budget.bytes) {
		return errors.New("core ZIP exceeds uncompressed size limit")
	}
	in, err := entry.Open()
	if err != nil {
		return fmt.Errorf("open core ZIP entry: %w", err)
	}
	maxFile := limits.MaxFileBytes
	if remaining := limits.MaxTotalBytes - budget.bytes; remaining < maxFile {
		maxFile = remaining
	}
	size, writeErr := writeBounded(root, local, mode, in, maxFile)
	closeErr := in.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return fmt.Errorf("close core ZIP entry: %w", closeErr)
	}
	if uint64(size) != entry.UncompressedSize64 {
		_ = root.Remove(local)
		return fmt.Errorf("core ZIP entry %q size does not match its header", entry.Name)
	}
	return budget.addBytes(size)
}

// precheckZIPDirectory bounds the number of central directory records before
// archive/zip allocates its File slice. ZIP64 and self-extracting archives are
// unnecessary for a bounded core source bundle and are rejected here.
func precheckZIPDirectory(file *os.File, size int64, maxEntries int) error {
	const endRecordSize = 22
	if size < endRecordSize {
		return errors.New("core ZIP has no end record")
	}
	tailSize := min(size, endRecordSize+65535)
	tail := make([]byte, tailSize)
	if _, err := file.ReadAt(tail, size-tailSize); err != nil {
		return fmt.Errorf("read core ZIP end record: %w", err)
	}
	for index := len(tail) - endRecordSize; index >= 0; index-- {
		if string(tail[index:index+4]) != "PK\x05\x06" {
			continue
		}
		end := tail[index : index+endRecordSize]
		if int(binary.LittleEndian.Uint16(end[20:22])) != len(tail)-index-endRecordSize {
			continue
		}
		if binary.LittleEndian.Uint16(end[4:6]) != 0 || binary.LittleEndian.Uint16(end[6:8]) != 0 {
			return errors.New("multi-disk core ZIP is unsupported")
		}
		count := binary.LittleEndian.Uint16(end[10:12])
		if count == 0xffff || binary.LittleEndian.Uint16(end[8:10]) != count || int(count) > maxEntries {
			return errors.New("core ZIP entry count is invalid or exceeds limit")
		}
		centralSize := binary.LittleEndian.Uint32(end[12:16])
		centralOffset := binary.LittleEndian.Uint32(end[16:20])
		endOffset := size - tailSize + int64(index)
		if centralSize == 0xffffffff || centralOffset == 0xffffffff ||
			int64(centralOffset)+int64(centralSize) != endOffset {
			return errors.New("core ZIP central directory is invalid or ZIP64")
		}
		return validateZIPCentralDirectory(file, int64(centralOffset), endOffset, count)
	}
	return errors.New("core ZIP has no valid end record")
}

func validateZIPCentralDirectory(file *os.File, position, endOffset int64, count uint16) error {
	for range count {
		if endOffset-position < 46 {
			return errors.New("core ZIP central directory is truncated")
		}
		var header [46]byte
		if _, err := file.ReadAt(header[:], position); err != nil {
			return fmt.Errorf("read core ZIP central directory: %w", err)
		}
		if string(header[:4]) != "PK\x01\x02" {
			return errors.New("core ZIP has an invalid central directory entry")
		}
		entrySize := int64(46) + int64(binary.LittleEndian.Uint16(header[28:30])) +
			int64(binary.LittleEndian.Uint16(header[30:32])) + int64(binary.LittleEndian.Uint16(header[32:34]))
		position += entrySize
		if position > endOffset {
			return errors.New("core ZIP central directory entry exceeds its boundary")
		}
	}
	if position != endOffset {
		return errors.New("core ZIP central directory count does not match its contents")
	}
	return nil
}

func extractTAR(archivePath string, root *os.Root, limits ArchiveLimits, compressed bool) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open core TAR: %w", err)
	}
	defer func() { _ = file.Close() }()
	var stream io.Reader = file
	if compressed {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("open core gzip stream: %w", err)
		}
		defer func() { _ = gz.Close() }()
		stream = gz
	}
	// Count TAR headers and padding as well as file bytes. A bounded reader also
	// stops malformed compressed streams that never reach an end-of-archive block.
	streamLimit := limits.MaxTotalBytes + int64(limits.MaxEntries)*4096 + 1<<20
	if streamLimit < limits.MaxTotalBytes {
		return errors.New("core TAR decompression limit overflows")
	}
	bounded := &io.LimitedReader{R: stream, N: streamLimit}
	return extractTAREntries(root, limits, bounded)
}

func extractTAREntries(root *os.Root, limits ArchiveLimits, bounded *io.LimitedReader) error {
	reader := tar.NewReader(bounded)
	budget := extractionBudget{limits: limits, seen: map[string]archivePathEntry{}}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return finishTARStream(bounded)
		}
		if err != nil {
			return fmt.Errorf("read core TAR entry: %w", err)
		}
		if header.Typeflag == tar.TypeDir && (header.Name == "." || header.Name == "./") {
			continue
		}
		if err := extractTAREntry(root, &budget, header, reader); err != nil {
			return err
		}
	}
}

func finishTARStream(bounded *io.LimitedReader) error {
	if bounded.N == 0 {
		return errors.New("core TAR exceeds decompression limit")
	}
	if _, err := io.Copy(io.Discard, bounded); err != nil {
		return fmt.Errorf("finish core TAR stream: %w", err)
	}
	if bounded.N == 0 {
		return errors.New("core TAR exceeds decompression limit")
	}
	return nil
}

func extractTAREntry(root *os.Root, budget *extractionBudget, header *tar.Header, reader io.Reader) error {
	local, err := budget.addPath(header.Name, header.Typeflag == tar.TypeDir)
	if err != nil {
		return err
	}
	switch header.Typeflag {
	case tar.TypeDir:
		if header.Size != 0 {
			return fmt.Errorf("core TAR directory %q has content", header.Name)
		}
		if err := root.MkdirAll(local, 0o700); err != nil {
			return fmt.Errorf("create core TAR directory: %w", err)
		}
	case tar.TypeReg:
		if err := budget.allowFile(header.Size); err != nil {
			return err
		}
		size, err := writeBounded(root, local, fs.FileMode(header.Mode), reader, budget.limits.MaxFileBytes)
		if err != nil {
			return err
		}
		if size != header.Size {
			_ = root.Remove(local)
			return fmt.Errorf("core TAR entry %q size does not match its header", header.Name)
		}
		return budget.addBytes(size)
	default:
		return fmt.Errorf("unsupported TAR entry type %q", header.Name)
	}
	return nil
}

func writeBounded(root *os.Root, name string, originalMode fs.FileMode, source io.Reader, maxBytes int64) (int64, error) {
	if err := root.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return 0, fmt.Errorf("create core archive parent: %w", err)
	}
	mode := fs.FileMode(0o600)
	if originalMode&0o100 != 0 {
		mode = 0o700
	}
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return 0, fmt.Errorf("create core archive file: %w", err)
	}
	// Read one byte past the per-file limit to detect a stream that exceeds its
	// advertised uncompressed size. The partially written file is removed.
	count, copyErr := io.Copy(file, io.LimitReader(source, maxBytes+1))
	closeErr := file.Close()
	if count > maxBytes || copyErr != nil || closeErr != nil {
		_ = root.Remove(name)
		if count > maxBytes {
			return 0, errors.New("core archive file exceeds uncompressed size limit")
		}
		return 0, errors.Join(copyErr, closeErr)
	}
	return count, nil
}
