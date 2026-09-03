package assets

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
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	protobufs "github.com/bishopfox/sliver/protobuf"
	"github.com/bishopfox/sliver/util"
	"github.com/ulikunitz/xz"
)

const (
	zigDirName = "zig"
)

var (
	//go:embed traffic-encoders/*.wasm
	trafficEncoderFS embed.FS
)

func unpackDefaultTrafficEncoders(force bool) error {
	encoders, err := trafficEncoderFS.ReadDir("traffic-encoders")
	if err != nil {
		return err
	}
	for _, encoder := range encoders {
		if encoder.IsDir() {
			continue
		}
		encoderName := path.Base(encoder.Name())
		encoderPath := path.Join("traffic-encoders", encoderName)
		encoderBytes, err := trafficEncoderFS.ReadFile(encoderPath)
		if err != nil {
			return err
		}

		localPath := filepath.Join(GetTrafficEncoderDir(), encoderName)
		if _, err := os.Stat(localPath); os.IsNotExist(err) || force {
			err = os.WriteFile(localPath, encoderBytes, 0600)
			if err != nil {
				return err
			}
		} else {
			setupLog.Infof("Skipping unpacking %s, already exists", encoderName)
		}
	}
	return nil
}

func unzipBuf(src []byte, dest string) ([]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, err
	}
	return extractZipReader(reader, dest, false)
}

func pseudoRandStringRunes(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[util.Intn(len(letterRunes))]
	}
	return string(b)
}

func setupZig(appDir string) error {
	setupLog.Infof("Unpacking to '%s'", appDir)
	zigRootPath := filepath.Join(appDir, zigDirName)
	setupLog.Infof("zig path = %s", zigRootPath)
	if _, err := os.Stat(zigRootPath); !os.IsNotExist(err) {
		setupLog.Info("Removing old zig root directory")
		os.Chmod(zigRootPath, 0700)
		err = util.ChmodR(zigRootPath, 0600, 0700) // Make sure everything is writable before we try to rm
		if err != nil {
			setupLog.Warnf("Failed to modify file system permissions of old zig root directory %s", err)
		}
		err = os.RemoveAll(zigRootPath)
		if err != nil {
			setupLog.Warnf("Failed to cleanup old zig root directory %s", err)
		}
	}
	os.MkdirAll(zigRootPath, 0700)

	// extract xz archive
	if runtime.GOOS != "windows" {
		// Everything except windows
		zigXzFSPath := path.Join("fs", runtime.GOOS, runtime.GOARCH, "zig.tar.xz")
		zigXzBuf, err := assetsFs.ReadFile(zigXzFSPath)
		if err != nil {
			setupLog.Errorf("static asset not found: %s", zigXzFSPath)
			return err
		}
		xzReader, err := xz.NewReader(bytes.NewReader(zigXzBuf))
		if err != nil {
			setupLog.Errorf("NewReader error %s", err)
			return err
		}
		// Extract tar archive
		setupLog.Infof("Unpacking zig.tar.xz to %s", zigRootPath)
		return untarSkipTopLevel(zigRootPath, xzReader)
	} else {
		// Windows only, since it's an awful operating system
		zigZipFSPath := path.Join("fs", runtime.GOOS, runtime.GOARCH, "zig.zip")
		zigZipBuf, err := assetsFs.ReadFile(zigZipFSPath)
		if err != nil {
			setupLog.Errorf("static asset not found: %s", zigZipFSPath)
			return err
		}
		reader, err := zip.NewReader(bytes.NewReader(zigZipBuf), int64(len(zigZipBuf)))
		if err != nil {
			setupLog.Errorf("zip.NewReader error %s", err)
			return err
		}
		err = unzipSkipTopLevel(zigRootPath, reader)
		if err != nil {
			setupLog.Infof("Failed to unzip file %s -> %s", zigZipFSPath, zigRootPath)
			return err
		}
		return nil
	}
}

// SetupGo - Unzip Go compiler assets
func setupGo(appDir string) error {
	setupLog.Infof("Unpacking to '%s'", appDir)
	goRootPath := filepath.Join(appDir, GoDirName)
	setupLog.Infof("GOPATH = %s", goRootPath)
	if _, err := os.Stat(goRootPath); !os.IsNotExist(err) {
		setupLog.Info("Removing old go root directory")
		os.Chmod(goRootPath, 0700)
		err = util.ChmodR(goRootPath, 0600, 0700) // Make sure everything is writable before we try to rm
		if err != nil {
			setupLog.Warnf("Failed to modify file system permissions of old go root directory %s", err)
		}
		err = os.RemoveAll(goRootPath)
		if err != nil {
			setupLog.Warnf("Failed to cleanup old go root directory %s", err)
		}
	}
	os.MkdirAll(goRootPath, 0700)

	// Go compiler and stdlib
	goZipFSPath := path.Join("fs", runtime.GOOS, runtime.GOARCH, "go.zip")
	goZip, err := assetsFs.ReadFile(goZipFSPath)
	if err != nil {
		setupLog.Errorf("static asset not found: %s", goZipFSPath)
		return err
	}

	goZipPath := filepath.Join(appDir, "go.zip")
	defer os.Remove(goZipPath)
	os.WriteFile(goZipPath, goZip, 0600)
	_, err = unzip(goZipPath, appDir)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s", goZipPath, appDir)
		return err
	}

	goSrcZip, err := assetsFs.ReadFile("fs/src.zip")
	if err != nil {
		setupLog.Info("static asset not found: src.zip")
		return err
	}
	goSrcZipPath := filepath.Join(appDir, "src.zip")
	defer os.Remove(goSrcZipPath)
	os.WriteFile(goSrcZipPath, goSrcZip, 0600)
	_, err = unzip(goSrcZipPath, goRootPath)
	if err != nil {
		setupLog.Infof("Failed to unzip file %s -> %s/go", goSrcZipPath, appDir)
		return err
	}

	garbleFileName := "garble"
	if runtime.GOOS == "windows" {
		garbleFileName = "garble.exe"
	}
	garbleAssetPath := path.Join("fs", runtime.GOOS, runtime.GOARCH, garbleFileName)
	garbleFile, err := assetsFs.ReadFile(garbleAssetPath)
	if err != nil {
		setupLog.Errorf("Static asset not found: %s", garbleFile)
		return err
	}
	garbleLocalPath := filepath.Join(appDir, "go", "bin", garbleFileName)
	err = os.WriteFile(garbleLocalPath, garbleFile, 0700)
	if err != nil {
		setupLog.Errorf("Failed to write garble %s", err)
		return err
	}

	return nil
}

// SetupGoPath - Extracts dependencies to goPathSrc
func SetupGoPath(goPathSrc string, includeDNS bool) error {

	// GOPATH setup
	if _, err := os.Stat(goPathSrc); os.IsNotExist(err) {
		setupLog.Infof("Creating GOPATH directory: %s", goPathSrc)
		os.MkdirAll(goPathSrc, 0700)
	}

	// Sliver PB
	sliverpbGoSrc, err := protobufs.FS.ReadFile("sliverpb/sliver.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: sliver.pb.go")
		return err
	}
	sliverpbConstSrc, err := protobufs.FS.ReadFile("sliverpb/constants.go")
	if err != nil {
		setupLog.Info("Static asset not found: constants.go")
		return err
	}
	sliverpbCapabilitiesSrc, err := protobufs.FS.ReadFile("sliverpb/capabilities.go")
	if err != nil {
		setupLog.Info("Static asset not found: capabilities.go")
		return err
	}
	sliverpbGoSrc = xorPBRawBytes(sliverpbGoSrc)
	sliverpbGoSrc = stripSliverpb(sliverpbGoSrc)
	sliverpbDir := filepath.Join(goPathSrc, "protobuf", "sliverpb")
	setupLog.Infof("Creating sliverpb directory: %s", sliverpbDir)
	os.MkdirAll(sliverpbDir, 0700)
	os.WriteFile(filepath.Join(sliverpbDir, "sliver.pb.go"), sliverpbGoSrc, 0600)
	os.WriteFile(filepath.Join(sliverpbDir, "constants.go"), sliverpbConstSrc, 0600)
	err = os.WriteFile(filepath.Join(sliverpbDir, "capabilities.go"), sliverpbCapabilitiesSrc, 0600)
	if err != nil {
		setupLog.Errorf("Failed to write capabilities.go: %s", err)
		return err
	}

	// Common PB
	commonpbSrc, err := protobufs.FS.ReadFile("commonpb/common.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: common.pb.go")
		return err
	}
	commonpbSrc = xorPBRawBytes(commonpbSrc)
	commonpbDir := filepath.Join(goPathSrc, "protobuf", "commonpb")
	os.MkdirAll(commonpbDir, 0700)
	os.WriteFile(filepath.Join(commonpbDir, "common.pb.go"), commonpbSrc, 0600)

	// DNS PB
	if includeDNS {
		dnspbSrc, err := protobufs.FS.ReadFile("dnspb/dns.pb.go")
		if err != nil {
			setupLog.Info("Static asset not found: dns.pb.go")
			return err
		}
		dnspbSrc = xorPBRawBytes(dnspbSrc)
		dnspbDir := filepath.Join(goPathSrc, "protobuf", "dnspb")
		os.MkdirAll(dnspbDir, 0700)
		os.WriteFile(filepath.Join(dnspbDir, "dns.pb.go"), dnspbSrc, 0600)
	}
	return nil
}

func stripSliverpb(src []byte) []byte {
	out := src
	re := regexp.MustCompile(`protobuf:"[a-z]+,\d+,[a-z]+,name=(?P<FieldName1>[a-zA-Z0-9]+),proto3(,enum=(?P<EnumName>[a-zA-Z\.]+))?" json:"(?P<FiledName2>[a-zA-Z0-9]+),[a-z]+"`)
	found := re.FindAllSubmatch(src, -1)
	for _, x := range found {
		line := x[0]     // line that matched
		typeName := x[1] // first named capturing group (FieldName1)
		enumName := x[3]
		if string(enumName) != "" {
			newEnumName := pseudoRandStringRunes(len(enumName))
			newEnumLine := bytes.ReplaceAll(line, enumName, []byte(newEnumName))
			out = bytes.ReplaceAll(out, line, []byte(newEnumLine))
			line = newEnumLine
		}
		// we don't care about FieldName2 because its value is the same as FieldName1
		newItem := pseudoRandStringRunes(len(typeName))
		newLine := bytes.ReplaceAll(line, typeName, []byte(newItem))
		out = bytes.ReplaceAll(out, line, []byte(newLine))
	}
	return out
}

// UntarSkipTopLevel - Untar a tar file, skipping the top level directory
func untarSkipTopLevel(dst string, r io.Reader) error {
	tr := tar.NewReader(r)
	topLevel, err := tr.Next()
	if err == io.EOF {
		return fmt.Errorf("no files found in tar")
	}
	if err != nil {
		return err
	}
	if topLevel.Typeflag != tar.TypeDir {
		return fmt.Errorf("expected top level to be a directory, got %v", topLevel.Typeflag)
	}
	topLevelName, err := archiveTopLevel(topLevel.Name)
	if err != nil {
		return err
	}
	root, err := openArchiveRoot(dst, 0o700)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		entryPath, err := archivePathBelowTopLevel(header.Name, topLevelName)
		if err != nil {
			return err
		}
		if entryPath == "" {
			continue
		}

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if err := root.MkdirAll(entryPath, 0o700); err != nil {
				return fmt.Errorf("create archive directory: %w", err)
			}

		// if it's a file create it
		case tar.TypeReg, tar.TypeRegA: //nolint:staticcheck // TypeRegA is valid in legacy tar archives.
			if err := writeArchiveFile(root, entryPath, os.FileMode(header.Mode), 0o700, tr); err != nil {
				return err
			}
		}
	}
}

// UnzipSkipTopLevel - Unzip a zip file, skipping the top level directory
func unzipSkipTopLevel(dst string, z *zip.Reader) error {
	_, err := extractZipReader(z, dst, true)
	return err
}

func xorPBRawBytes(src []byte) []byte {
	var (
		fileAst    *ast.File
		err        error
		fset       = token.NewFileSet()
		parserMode = parser.ParseComments
	)
	fileAst, err = parser.ParseFile(fset, "", src, parserMode)
	if err != nil {
		// Panic because this is mandatory for the agent to work
		panic(err)
	}
	var xorKey [8]byte
	// generate random xor key
	if _, err := rand.Read(xorKey[:]); err != nil {
		// Panic because this is mandatory for the agent to work
		panic(err)
	}

	ast.Inspect(fileAst, func(n ast.Node) bool {
		node, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		for _, spec := range node.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for valueIndex, id := range valueSpec.Names {
				if !strings.HasSuffix(id.Name, "_rawDesc") || strings.HasSuffix(id.Name, "_rawDescData") {
					continue
				}
				if valueIndex >= len(valueSpec.Values) {
					continue
				}

				switch rawDesc := valueSpec.Values[valueIndex].(type) {
				case *ast.CompositeLit:
					for i, v := range rawDesc.Elts {
						elt := v.(*ast.BasicLit)
						elt.Value = xorByte(elt.Value, xorKey[i%len(xorKey)])
					}
					valueSpec.Values[valueIndex] = &ast.CallExpr{
						Fun: ast.NewIdent("xorBytes"),
						Args: []ast.Expr{
							rawDesc,
							ast.NewIdent("xorKey"),
						},
					}
				default:
					offset := 0
					if xorStringExpr(rawDesc, xorKey, &offset) {
						node.Tok = token.VAR
						valueSpec.Values[valueIndex] = &ast.CallExpr{
							Fun: ast.NewIdent("xorString"),
							Args: []ast.Expr{
								rawDesc,
								ast.NewIdent("xorKey"),
							},
						}
					}
				}
			}
		}
		return true
	})

	fileAst.Decls = append(fileAst.Decls,
		parseHelperDecl(`func xorBytes(input []byte, key []byte) []byte {
	out := make([]byte, len(input))
	for i := range input {
		out[i] = input[i] ^ key[i%len(key)]
	}
	return out
}`),
		parseHelperDecl(`func xorString(input string, key []byte) string {
	out := make([]byte, len(input))
	for i := 0; i < len(input); i++ {
		out[i] = input[i] ^ key[i%len(key)]
	}
	return string(out)
}`),
	)

	xorTokens := make([]ast.Expr, len(xorKey))
	// map xorKey to a slice of ast.BasicLit
	for i, b := range xorKey {
		xorTokens[i] = &ast.BasicLit{
			Kind:  token.INT,
			Value: fmt.Sprintf("0x%x", b),
		}
	}

	// add the global xorKey variable to the AST
	fileAst.Decls = append(fileAst.Decls, ast.Decl(&ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("xorKey")},
				Values: []ast.Expr{
					&ast.CompositeLit{
						Type: ast.NewIdent("[]byte"),
						Elts: xorTokens,
					},
				},
			},
		},
	}))

	outBuff := bytes.Buffer{}
	// Render the AST as Go code
	printer.Fprint(&outBuff, fset, fileAst)
	return outBuff.Bytes()
}

func parseHelperDecl(src string) ast.Decl {
	fileAst, err := parser.ParseFile(token.NewFileSet(), "", "package assets\n"+src, 0)
	if err != nil {
		panic(err)
	}
	return fileAst.Decls[0]
}

func xorByte(raw string, key byte) string {
	// strip 0x
	raw = raw[2:]
	if len(raw) == 1 {
		// Because we got `0x8` at some point
		raw = fmt.Sprintf("0%s", raw)
	}
	hexByte, err := hex.DecodeString(raw)
	if err != nil {
		panic(err)
	}
	newByte := hex.EncodeToString([]byte{hexByte[0] ^ key})
	return fmt.Sprintf("0x%s", newByte)
}

func xorStringExpr(expr ast.Expr, key [8]byte, offset *int) bool {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind != token.STRING {
			return false
		}
		raw, err := strconv.Unquote(node.Value)
		if err != nil {
			panic(err)
		}
		rawBytes := []byte(raw)
		for i := range rawBytes {
			rawBytes[i] ^= key[(*offset+i)%len(key)]
		}
		*offset += len(rawBytes)
		node.Value = strconv.Quote(string(rawBytes))
		return true
	case *ast.BinaryExpr:
		if node.Op != token.ADD {
			return false
		}
		leftOkay := xorStringExpr(node.X, key, offset)
		rightOkay := xorStringExpr(node.Y, key, offset)
		return leftOkay && rightOkay
	case *ast.ParenExpr:
		return xorStringExpr(node.X, key, offset)
	default:
		return false
	}
}

func unzip(src string, dest string) ([]string, error) {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return extractZipReader(&reader.Reader, dest, false)
}

func openArchiveRoot(dest string, mode fs.FileMode) (*os.Root, error) {
	if err := os.MkdirAll(dest, mode); err != nil {
		return nil, fmt.Errorf("create archive destination: %w", err)
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return nil, fmt.Errorf("open archive destination: %w", err)
	}
	return root, nil
}

func cleanArchiveName(name string) (string, error) {
	archiveName := strings.TrimSuffix(name, "/")
	if archiveName == "." || !fs.ValidPath(archiveName) {
		return "", fmt.Errorf("invalid path in archive: %q", name)
	}
	return archiveName, nil
}

func archiveEntryPath(name string) (string, error) {
	archiveName, err := cleanArchiveName(name)
	if err != nil {
		return "", err
	}
	entryPath, err := filepath.Localize(archiveName)
	if err != nil {
		return "", fmt.Errorf("invalid path in archive %q: %w", name, err)
	}
	return entryPath, nil
}

func archiveTopLevel(name string) (string, error) {
	topLevel, err := cleanArchiveName(name)
	if err != nil {
		return "", err
	}
	if strings.Contains(topLevel, "/") {
		return "", fmt.Errorf("invalid top-level directory in archive: %q", name)
	}
	if _, err := filepath.Localize(topLevel); err != nil {
		return "", fmt.Errorf("invalid top-level directory in archive %q: %w", name, err)
	}
	return topLevel, nil
}

func archivePathBelowTopLevel(name, topLevel string) (string, error) {
	archiveName, err := cleanArchiveName(name)
	if err != nil {
		return "", err
	}
	if archiveName == topLevel {
		return "", nil
	}
	prefix := topLevel + "/"
	if !strings.HasPrefix(archiveName, prefix) {
		return "", fmt.Errorf("archive path %q is outside top-level directory %q", name, topLevel)
	}
	entryPath, err := filepath.Localize(strings.TrimPrefix(archiveName, prefix))
	if err != nil {
		return "", fmt.Errorf("invalid path in archive %q: %w", name, err)
	}
	return entryPath, nil
}

func writeArchiveFile(root *os.Root, name string, mode, parentMode fs.FileMode, src io.Reader) error {
	if err := root.MkdirAll(filepath.Dir(name), parentMode); err != nil {
		return fmt.Errorf("create archive parent directory: %w", err)
	}
	out, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("write archive file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close archive file: %w", closeErr)
	}
	return nil
}

func extractZipReader(reader *zip.Reader, dest string, skipTopLevel bool) ([]string, error) {
	var filenames []string
	topLevel := ""
	if skipTopLevel {
		if len(reader.File) == 0 {
			return nil, fmt.Errorf("no files found in zip")
		}
		if !reader.File[0].FileInfo().IsDir() {
			return nil, fmt.Errorf("expected top level to be a directory, got %s", reader.File[0].Name)
		}
		var err error
		topLevel, err = archiveTopLevel(reader.File[0].Name)
		if err != nil {
			return nil, err
		}
	}

	root, err := openArchiveRoot(dest, 0o700)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	for _, file := range reader.File {
		entryPath, err := archiveEntryPath(file.Name)
		if skipTopLevel {
			entryPath, err = archivePathBelowTopLevel(file.Name, topLevel)
		}
		if err != nil {
			return filenames, err
		}
		if entryPath == "" {
			continue
		}
		filenames = append(filenames, filepath.Join(dest, entryPath))

		if file.FileInfo().IsDir() {
			if err := root.MkdirAll(entryPath, 0o700); err != nil {
				return filenames, fmt.Errorf("create archive directory: %w", err)
			}
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}
		writeErr := writeArchiveFile(root, entryPath, file.Mode(), 0o700, rc)
		closeErr := rc.Close()
		if writeErr != nil {
			return filenames, writeErr
		}
		if closeErr != nil {
			return filenames, fmt.Errorf("close zip entry: %w", closeErr)
		}
	}
	return filenames, nil
}

func setupCodenames(appDir string) error {
	nouns, err := assetsFs.ReadFile("fs/nouns.txt")
	if err != nil {
		setupLog.Infof("nouns.txt asset not found")
		return err
	}

	adjectives, err := assetsFs.ReadFile("fs/adjectives.txt")
	if err != nil {
		setupLog.Infof("adjectives.txt asset not found")
		return err
	}

	err = os.WriteFile(filepath.Join(appDir, "nouns.txt"), nouns, 0600)
	if err != nil {
		setupLog.Infof("Failed to write noun data to: %s", appDir)
		return err
	}

	err = os.WriteFile(filepath.Join(appDir, "adjectives.txt"), adjectives, 0600)
	if err != nil {
		setupLog.Infof("Failed to write adjective data to: %s", appDir)
		return err
	}
	return nil
}
