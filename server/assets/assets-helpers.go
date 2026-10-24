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
	if err := os.MkdirAll(goPathSrc, 0700); err != nil {
		return fmt.Errorf("create implant source directory %s: %w", goPathSrc, err)
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
	sliverpbGoSrc, err = xorPBRawBytes(sliverpbGoSrc)
	if err != nil {
		return fmt.Errorf("obfuscate sliverpb descriptor: %w", err)
	}
	sliverpbGoSrc, err = rewriteProtobufStructTags(sliverpbGoSrc)
	if err != nil {
		return fmt.Errorf("obfuscate sliverpb tags: %w", err)
	}
	sliverpbDir := filepath.Join(goPathSrc, "protobuf", "sliverpb")
	setupLog.Infof("Creating sliverpb directory: %s", sliverpbDir)
	if err := os.MkdirAll(sliverpbDir, 0700); err != nil {
		return fmt.Errorf("create sliverpb directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(sliverpbDir, "sliver.pb.go"), sliverpbGoSrc, 0600); err != nil {
		return fmt.Errorf("write sliver.pb.go: %w", err)
	}
	if err := os.WriteFile(filepath.Join(sliverpbDir, "constants.go"), sliverpbConstSrc, 0600); err != nil {
		return fmt.Errorf("write sliverpb constants.go: %w", err)
	}
	if err := os.WriteFile(filepath.Join(sliverpbDir, "capabilities.go"), sliverpbCapabilitiesSrc, 0600); err != nil {
		return fmt.Errorf("write sliverpb capabilities.go: %w", err)
	}

	// Common PB
	commonpbSrc, err := protobufs.FS.ReadFile("commonpb/common.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: common.pb.go")
		return err
	}
	commonpbSrc, err = xorPBRawBytes(commonpbSrc)
	if err != nil {
		return fmt.Errorf("obfuscate commonpb descriptor: %w", err)
	}
	commonpbSrc, err = rewriteProtobufStructTags(commonpbSrc)
	if err != nil {
		return fmt.Errorf("obfuscate commonpb tags: %w", err)
	}
	commonpbDir := filepath.Join(goPathSrc, "protobuf", "commonpb")
	if err := os.MkdirAll(commonpbDir, 0700); err != nil {
		return fmt.Errorf("create commonpb directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(commonpbDir, "common.pb.go"), commonpbSrc, 0600); err != nil {
		return fmt.Errorf("write common.pb.go: %w", err)
	}

	// DNS PB
	if includeDNS {
		dnspbSrc, err := protobufs.FS.ReadFile("dnspb/dns.pb.go")
		if err != nil {
			setupLog.Info("Static asset not found: dns.pb.go")
			return err
		}
		dnspbSrc, err = xorPBRawBytes(dnspbSrc)
		if err != nil {
			return fmt.Errorf("obfuscate dnspb descriptor: %w", err)
		}
		dnspbSrc, err = rewriteProtobufStructTags(dnspbSrc)
		if err != nil {
			return fmt.Errorf("obfuscate dnspb tags: %w", err)
		}
		dnspbDir := filepath.Join(goPathSrc, "protobuf", "dnspb")
		if err := os.MkdirAll(dnspbDir, 0700); err != nil {
			return fmt.Errorf("create dnspb directory: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dnspbDir, "dns.pb.go"), dnspbSrc, 0600); err != nil {
			return fmt.Errorf("write dns.pb.go: %w", err)
		}
	}
	return nil
}

type structTagPart struct {
	key   string
	value string
}

// parseStructTagParts parses the key/value pairs in a Go struct tag without
// assuming an order or a particular set of protobuf options.
func parseStructTagParts(tag string) ([]structTagPart, error) {
	var parts []structTagPart
	for tag != "" {
		tag = strings.TrimLeft(tag, " ")
		if tag == "" {
			break
		}
		colon := strings.IndexByte(tag, ':')
		if colon <= 0 || strings.ContainsAny(tag[:colon], " \t\n\r\"`") {
			return nil, fmt.Errorf("invalid struct tag key near %q", tag)
		}
		key := tag[:colon]
		tag = tag[colon+1:]
		if !strings.HasPrefix(tag, `"`) {
			return nil, fmt.Errorf("missing quoted value for struct tag %q", key)
		}
		end := 1
		for end < len(tag) {
			if tag[end] == '\\' {
				end += 2
				continue
			}
			if tag[end] == '"' {
				break
			}
			end++
		}
		if end >= len(tag) {
			return nil, fmt.Errorf("unterminated struct tag value for %q", key)
		}
		value, err := strconv.Unquote(tag[:end+1])
		if err != nil {
			return nil, fmt.Errorf("unquote struct tag %q: %w", key, err)
		}
		parts = append(parts, structTagPart{key: key, value: value})
		tag = tag[end+1:]
		if tag != "" && tag[0] != ' ' {
			return nil, fmt.Errorf("missing separator after struct tag %q", key)
		}
	}
	return parts, nil
}

func renderStructTagParts(parts []structTagPart) string {
	var out strings.Builder
	for i, part := range parts {
		if i != 0 {
			out.WriteByte(' ')
		}
		out.WriteString(part.key)
		out.WriteByte(':')
		out.WriteString(strconv.Quote(part.value))
	}
	return out.String()
}

func randomDifferentName(original string) string {
	for {
		candidate := pseudoRandStringRunes(len(original))
		if candidate != original {
			return candidate
		}
	}
}

func rewriteProtobufTagValue(value string) (string, string, string, error) {
	options := strings.Split(value, ",")
	nameIndex, enumIndex := -1, -1
	fieldNumberFound := false
	for i, option := range options {
		if _, err := strconv.ParseUint(option, 10, 32); err == nil {
			fieldNumberFound = true
		}
		if strings.HasPrefix(option, "name=") {
			if nameIndex >= 0 {
				return "", "", "", fmt.Errorf("duplicate protobuf field name")
			}
			nameIndex = i
		}
		if strings.HasPrefix(option, "enum=") {
			enumIndex = i
		}
	}
	if !fieldNumberFound || nameIndex < 0 || len(options[nameIndex]) == len("name=") {
		return "", "", "", fmt.Errorf("protobuf struct tag lacks a field number or name: %q", value)
	}
	originalName := strings.TrimPrefix(options[nameIndex], "name=")
	newName := randomDifferentName(originalName)
	options[nameIndex] = "name=" + newName
	if enumIndex >= 0 {
		enumName := strings.TrimPrefix(options[enumIndex], "enum=")
		if enumName == "" {
			return "", "", "", fmt.Errorf("empty protobuf enum name")
		}
		options[enumIndex] = "enum=" + randomDifferentName(enumName)
	}
	return strings.Join(options, ","), originalName, newName, nil
}

func rewriteJSONTagValue(value, originalName, newName string) (string, error) {
	options := strings.SplitN(value, ",", 2)
	if options[0] == "" || options[0] == "-" {
		return "", fmt.Errorf("unsupported JSON field name %q", options[0])
	}
	if options[0] == originalName {
		options[0] = newName
	} else {
		options[0] = randomDifferentName(options[0])
	}
	return strings.Join(options, ","), nil
}

func rewritePBStructTag(raw string, cache map[string]string) (string, bool, error) {
	if rewritten, ok := cache[raw]; ok {
		return rewritten, true, nil
	}
	parts, err := parseStructTagParts(raw)
	if err != nil {
		return "", false, err
	}
	protobufIndex, jsonIndex := -1, -1
	for i, part := range parts {
		switch part.key {
		case "protobuf":
			if protobufIndex >= 0 {
				return "", false, fmt.Errorf("duplicate protobuf struct tag")
			}
			protobufIndex = i
		case "json":
			if jsonIndex >= 0 {
				return "", false, fmt.Errorf("duplicate JSON struct tag")
			}
			jsonIndex = i
		}
	}
	if protobufIndex < 0 {
		return raw, false, nil
	}
	if jsonIndex < 0 {
		return "", false, fmt.Errorf("protobuf struct tag is missing a JSON tag")
	}
	protobufValue, originalName, newName, err := rewriteProtobufTagValue(parts[protobufIndex].value)
	if err != nil {
		return "", false, err
	}
	parts[protobufIndex].value = protobufValue
	parts[jsonIndex].value, err = rewriteJSONTagValue(parts[jsonIndex].value, originalName, newName)
	if err != nil {
		return "", false, err
	}
	rewritten := renderStructTagParts(parts)
	cache[raw] = rewritten
	return rewritten, true, nil
}

// rewriteProtobufStructTags rewrites generated protobuf struct tags after parsing their
// syntax, including packed fields and future option ordering changes.
func rewriteProtobufStructTags(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse protobuf source: %w", err)
	}
	cache := make(map[string]string)
	rewritten := 0
	var rewriteErr error
	ast.Inspect(fileAst, func(node ast.Node) bool {
		if rewriteErr != nil {
			return false
		}
		structType, ok := node.(*ast.StructType)
		if !ok || structType.Fields == nil {
			return true
		}
		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}
			raw, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				rewriteErr = fmt.Errorf("parse struct tag at %s: %w", fset.Position(field.Pos()), err)
				return false
			}
			newTag, changed, err := rewritePBStructTag(raw, cache)
			if err != nil {
				rewriteErr = fmt.Errorf("rewrite struct tag at %s: %w", fset.Position(field.Pos()), err)
				return false
			}
			if changed {
				rewritten++
				if strings.ContainsRune(newTag, '`') {
					field.Tag.Value = strconv.Quote(newTag)
				} else {
					field.Tag.Value = "`" + newTag + "`"
				}
			}
		}
		return true
	})
	if rewriteErr != nil {
		return nil, rewriteErr
	}
	if rewritten == 0 {
		return nil, fmt.Errorf("no protobuf struct tags found in source")
	}
	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, fileAst); err != nil {
		return nil, fmt.Errorf("print rewritten protobuf source: %w", err)
	}
	return out.Bytes(), nil
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

func descriptorStringLiterals(expr ast.Expr) ([]*ast.BasicLit, error) {
	var literals []*ast.BasicLit
	var walk func(ast.Expr) error
	walk = func(expr ast.Expr) error {
		switch node := expr.(type) {
		case *ast.BasicLit:
			if node.Kind != token.STRING {
				return fmt.Errorf("descriptor expression contains a non-string literal")
			}
			if _, err := strconv.Unquote(node.Value); err != nil {
				return fmt.Errorf("decode descriptor string literal: %w", err)
			}
			literals = append(literals, node)
			return nil
		case *ast.BinaryExpr:
			if node.Op != token.ADD {
				return fmt.Errorf("descriptor expression uses operator %s", node.Op)
			}
			if err := walk(node.X); err != nil {
				return err
			}
			return walk(node.Y)
		case *ast.ParenExpr:
			return walk(node.X)
		default:
			return fmt.Errorf("unsupported descriptor expression %T", expr)
		}
	}
	if err := walk(expr); err != nil {
		return nil, err
	}
	return literals, nil
}

func descriptorByteLiterals(expr *ast.CompositeLit) ([]*ast.BasicLit, []byte, error) {
	arrayType, ok := expr.Type.(*ast.ArrayType)
	if !ok || arrayType.Len != nil {
		return nil, nil, fmt.Errorf("descriptor must be a byte slice literal")
	}
	elementType, ok := arrayType.Elt.(*ast.Ident)
	if !ok || (elementType.Name != "byte" && elementType.Name != "uint8") {
		return nil, nil, fmt.Errorf("descriptor must be a byte slice literal")
	}
	literals := make([]*ast.BasicLit, 0, len(expr.Elts))
	values := make([]byte, 0, len(expr.Elts))
	for _, element := range expr.Elts {
		literal, ok := element.(*ast.BasicLit)
		if !ok || literal.Kind != token.INT {
			return nil, nil, fmt.Errorf("descriptor byte slice has unsupported element %T", element)
		}
		value, err := strconv.ParseUint(literal.Value, 0, 8)
		if err != nil {
			return nil, nil, fmt.Errorf("decode descriptor byte %q: %w", literal.Value, err)
		}
		literals = append(literals, literal)
		values = append(values, byte(value))
	}
	return literals, values, nil
}

// xorPBRawBytes hides the one generated raw descriptor in a protobuf source.
// The complete expression is validated before changing the AST so a new protoc
// output shape cannot result in a partially encoded descriptor.
func xorPBRawBytes(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse protobuf source: %w", err)
	}
	declaration, valueSpec, descriptorName, err := findRawDescriptor(fileAst)
	if err != nil {
		return nil, err
	}
	if (declaration.Tok != token.CONST && declaration.Tok != token.VAR) ||
		len(declaration.Specs) != 1 || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
		return nil, fmt.Errorf("%s has unsupported declaration shape", descriptorName)
	}

	functionName, stringLiterals, byteLiterals, byteValues, err := rawDescriptorLiterals(declaration, valueSpec, descriptorName)
	if err != nil {
		return nil, err
	}
	var xorKey [8]byte
	if _, err := rand.Read(xorKey[:]); err != nil {
		return nil, fmt.Errorf("generate protobuf descriptor key: %w", err)
	}
	encodeRawDescriptorLiterals(declaration, functionName, stringLiterals, byteLiterals, byteValues, xorKey[:])
	valueSpec.Values[0] = &ast.CallExpr{
		Fun: ast.NewIdent(functionName),
		Args: []ast.Expr{
			valueSpec.Values[0],
			ast.NewIdent("xorKey"),
		},
	}
	if err := appendRawDescriptorHelpers(fileAst, functionName, xorKey[:]); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, fileAst); err != nil {
		return nil, fmt.Errorf("print rewritten protobuf source: %w", err)
	}
	return out.Bytes(), nil
}

func findRawDescriptor(fileAst *ast.File) (*ast.GenDecl, *ast.ValueSpec, string, error) {
	var declaration *ast.GenDecl
	var valueSpec *ast.ValueSpec
	var descriptorName string
	for _, decl := range fileAst.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			candidate, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range candidate.Names {
				if !strings.HasSuffix(name.Name, "_rawDesc") {
					continue
				}
				if declaration != nil {
					return nil, nil, "", fmt.Errorf("multiple protobuf raw descriptors found")
				}
				declaration, valueSpec, descriptorName = genDecl, candidate, name.Name
			}
		}
	}
	if declaration == nil {
		return nil, nil, "", fmt.Errorf("no protobuf raw descriptor found")
	}
	return declaration, valueSpec, descriptorName, nil
}

func rawDescriptorLiterals(declaration *ast.GenDecl, valueSpec *ast.ValueSpec, descriptorName string) (string, []*ast.BasicLit, []*ast.BasicLit, []byte, error) {
	var stringLiterals []*ast.BasicLit
	var byteLiterals []*ast.BasicLit
	var byteValues []byte
	var err error
	functionName := "xorString"
	if byteExpr, ok := valueSpec.Values[0].(*ast.CompositeLit); ok {
		if declaration.Tok != token.VAR {
			return "", nil, nil, nil, fmt.Errorf("%s byte descriptor is not a variable", descriptorName)
		}
		byteLiterals, byteValues, err = descriptorByteLiterals(byteExpr)
		functionName = "xorBytes"
	} else {
		stringLiterals, err = descriptorStringLiterals(valueSpec.Values[0])
	}
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("%s: %w", descriptorName, err)
	}
	return functionName, stringLiterals, byteLiterals, byteValues, nil
}

func encodeRawDescriptorLiterals(declaration *ast.GenDecl, functionName string, stringLiterals, byteLiterals []*ast.BasicLit, byteValues, xorKey []byte) {
	if functionName == "xorString" {
		offset := 0
		for _, literal := range stringLiterals {
			plain, _ := strconv.Unquote(literal.Value) // Validated above.
			encoded := []byte(plain)
			for i := range encoded {
				encoded[i] ^= xorKey[(offset+i)%len(xorKey)]
			}
			offset += len(encoded)
			literal.Value = strconv.Quote(string(encoded))
		}
		declaration.Tok = token.VAR
	} else {
		for i, literal := range byteLiterals {
			literal.Value = fmt.Sprintf("0x%02x", byteValues[i]^xorKey[i%len(xorKey)])
		}
	}
}

func appendRawDescriptorHelpers(fileAst *ast.File, functionName string, xorKey []byte) error {
	helper := `func xorString(input string, key []byte) string {
	out := make([]byte, len(input))
	for i := 0; i < len(input); i++ {
		out[i] = input[i] ^ key[i%len(key)]
	}
	return string(out)
}`
	if functionName == "xorBytes" {
		helper = `func xorBytes(input []byte, key []byte) []byte {
	out := make([]byte, len(input))
	for i := range input {
		out[i] = input[i] ^ key[i%len(key)]
	}
	return out
}`
	}
	helperDecl, err := parseHelperDecl(helper)
	if err != nil {
		return err
	}
	fileAst.Decls = append(fileAst.Decls, helperDecl)

	xorTokens := make([]ast.Expr, len(xorKey))
	for i, b := range xorKey {
		xorTokens[i] = &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("0x%x", b)}
	}
	fileAst.Decls = append(fileAst.Decls, &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("xorKey")},
			Values: []ast.Expr{&ast.CompositeLit{Type: ast.NewIdent("[]byte"), Elts: xorTokens}},
		}},
	})
	return nil
}

func parseHelperDecl(src string) (ast.Decl, error) {
	fileAst, err := parser.ParseFile(token.NewFileSet(), "", "package assets\n"+src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse protobuf descriptor helper: %w", err)
	}
	return fileAst.Decls[0], nil
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
