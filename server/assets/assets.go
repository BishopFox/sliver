package assets

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"archive/zip"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	insecureRand "math/rand"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	ver "github.com/bishopfox/sliver/client/version"
	protobufs "github.com/bishopfox/sliver/protobuf"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	// GoDirName - The directory to store the go compiler/toolchain files in
	GoDirName = "go"

	goPathDirName   = "gopath"
	versionFileName = "version"
	envVarName      = "SLIVER_ROOT_DIR"
)

var (
	setupLog = log.NamedLogger("assets", "setup")
)

// GetRootAppDir - Get the Sliver app dir, default is: ~/.sliver/
func GetRootAppDir() string {

	value := os.Getenv(envVarName)

	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = path.Join(user.HomeDir, ".sliver")
	} else {
		dir = value
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			setupLog.Fatalf("Cannot write to sliver root dir %s", err)
		}
	}
	return dir
}

func assetVersion() string {
	appDir := GetRootAppDir()
	data, err := os.ReadFile(path.Join(appDir, versionFileName))
	if err != nil {
		setupLog.Infof("No version detected %s", err)
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveAssetVersion(appDir string) {
	versionFilePath := path.Join(appDir, versionFileName)
	fVer, _ := os.Create(versionFilePath)
	defer fVer.Close()
	fVer.Write([]byte(ver.GitCommit))
}

// Setup - Extract or create local assets
func Setup(force bool, echo bool) {
	appDir := GetRootAppDir()
	localVer := assetVersion()
	if force || localVer == "" || localVer != ver.GitCommit {
		setupLog.Infof("Version mismatch %v != %v", localVer, ver.GitCommit)
		if echo {
			fmt.Printf(`
Sliver  Copyright (C) 2022  Bishop Fox
This program comes with ABSOLUTELY NO WARRANTY; for details type 'licenses'.
This is free software, and you are welcome to redistribute it
under certain conditions; type 'licenses' for details.`)
			fmt.Printf("\n\nUnpacking assets ...\n")
		}
		setupGo(appDir)
		setupCodenames(appDir)
		saveAssetVersion(appDir)
	}
}

// English - Extracts the english dictionary for the english encoder
func English() []string {
	rawEnglish, err := assetsFs.ReadFile("fs/english.txt")
	if err != nil {
		return []string{}
	}
	englishWords := strings.Split(string(rawEnglish), "\n")
	return englishWords
}

// GetGPGPublicKey - Return the GPG public key from assets
func GetGPGPublicKey() (*packet.PublicKey, error) {
	rawPublicKey, err := assetsFs.ReadFile("fs/sliver.asc")
	if err != nil {
		return nil, err
	}
	// Decode armored public key
	block, err := armor.Decode(bytes.NewReader(rawPublicKey))
	if err != nil {
		return nil, fmt.Errorf("error decoding public key: %s", err)
	}
	if block.Type != "PGP PUBLIC KEY BLOCK" {
		return nil, errors.New("not an armored public key")
	}

	// Read the key
	pack, err := packet.Read(block.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading public key: %s", err)
	}

	// Was it really a public key file ? If yes, get the PublicKey
	publicKey, ok := pack.(*packet.PublicKey)
	if !ok {
		return nil, errors.New("invalid public key")
	}
	return publicKey, nil
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
	goZipFSPath := filepath.Join("fs", runtime.GOOS, runtime.GOARCH, "go.zip")
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
	garbleAssetPath := filepath.Join("fs", runtime.GOOS, runtime.GOARCH, garbleFileName)
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

	return setupSGN(appDir)
}

func setupSGN(appDir string) error {
	goBinPath := filepath.Join(appDir, "go", "bin")
	sgnZipFSPath := filepath.Join("fs", runtime.GOOS, runtime.GOARCH, "sgn.zip")
	sgnZip, err := assetsFs.ReadFile(sgnZipFSPath)
	if err != nil {
		setupLog.Errorf("static asset not found: %s", sgnZipFSPath)
		return err
	}
	_, err = unzipBuf(sgnZip, goBinPath)
	return err
}

// SetupGoPath - Extracts dependencies to goPathSrc
func SetupGoPath(goPathSrc string) error {

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
	sliverpbGoSrc = xorPBRawBytes(sliverpbGoSrc)
	sliverpbGoSrc = stripSliverpb(sliverpbGoSrc)
	sliverpbDir := filepath.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "sliverpb")
	os.MkdirAll(sliverpbDir, 0700)
	os.WriteFile(filepath.Join(sliverpbDir, "sliver.pb.go"), sliverpbGoSrc, 0600)
	os.WriteFile(filepath.Join(sliverpbDir, "constants.go"), sliverpbConstSrc, 0600)

	// Common PB
	commonpbSrc, err := protobufs.FS.ReadFile("commonpb/common.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: common.pb.go")
		return err
	}
	commonpbSrc = xorPBRawBytes(commonpbSrc)
	commonpbDir := filepath.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "commonpb")
	os.MkdirAll(commonpbDir, 0700)
	os.WriteFile(filepath.Join(commonpbDir, "common.pb.go"), commonpbSrc, 0600)

	// DNS PB
	dnspbSrc, err := protobufs.FS.ReadFile("dnspb/dns.pb.go")
	if err != nil {
		setupLog.Info("Static asset not found: dns.pb.go")
		return err
	}
	dnspbSrc = xorPBRawBytes(dnspbSrc)
	dnspbDir := filepath.Join(goPathSrc, "github.com", "bishopfox", "sliver", "protobuf", "dnspb")
	os.MkdirAll(dnspbDir, 0700)
	os.WriteFile(filepath.Join(dnspbDir, "dns.pb.go"), dnspbSrc, 0600)
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

func xorPBRawBytes(src []byte) []byte {
	var (
		fileAst                   *ast.File
		err                       error
		sliverpbVarName           = "file_sliverpb_sliver_proto_rawDesc"
		sliverPbProtoInitFuncName = "file_sliverpb_sliver_proto_init"
		fset                      = token.NewFileSet()
		parserMode                = parser.ParseComments
	)
	fileAst, err = parser.ParseFile(fset, "", src, parserMode)
	if err != nil {
		// Panic because this is mandatory for the agent to work
		panic(err)
	}
	var xorKey [8]byte
	// generate random xor key
	if _, err := insecureRand.Read(xorKey[:]); err != nil {
		// Panic because this is mandatory for the agent to work
		panic(err)
	}

	ast.Inspect(fileAst, func(n ast.Node) bool {
		switch node := n.(type) {
		// Look for the protobuf init function and decode the XORed
		// data at the top of the function
		case *ast.FuncDecl:
			// add an ast.AssignStmt at the top of the function Body
			// that calls the xor function on the file_sliverpb_sliver_proto_rawDesc object
			if node.Name.Name == sliverPbProtoInitFuncName {
				newStmt := &ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent(sliverpbVarName),
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("xor"),
							Args: []ast.Expr{
								ast.NewIdent(sliverpbVarName),
								ast.NewIdent("xorKey"),
							},
						},
					},
				}
				node.Body.List = append([]ast.Stmt{newStmt}, node.Body.List...)
			}

		// Look for the protobuf rawDesc variable and XOR each byte
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, id := range spec.Names {
						if id.Name == sliverpbVarName {
							values := spec.Values[0].(*ast.CompositeLit).Elts
							// XOR each value of the slice
							for i, v := range values {
								elt := v.(*ast.BasicLit)
								elt.Value = xorByte(elt.Value, xorKey[i%len(xorKey)])
							}

						}
					}
				default:
				}
			}
		default:
		}
		return true
	})

	// Add the XOR function to the AST
	fileAst.Decls = append(fileAst.Decls, ast.Decl(&ast.FuncDecl{
		Name: ast.NewIdent("xor"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("input")},
						Type:  ast.NewIdent("[]byte"),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("key")},
						Type:  ast.NewIdent("[]byte"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("[]byte"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent("out"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("make"),
							Args: []ast.Expr{
								&ast.ArrayType{
									Elt: ast.NewIdent("byte"),
								},
								&ast.CallExpr{
									Fun: ast.NewIdent("len"),
									Args: []ast.Expr{
										ast.NewIdent("input"),
									},
								},
							},
						},
					},
				},
				&ast.RangeStmt{
					Key:   ast.NewIdent("i"),
					Value: ast.NewIdent("_"),
					Tok:   token.DEFINE,
					X:     ast.NewIdent("input"),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{
									&ast.IndexExpr{
										X:     ast.NewIdent("out"),
										Index: ast.NewIdent("i"),
									},
								},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{
									&ast.BinaryExpr{
										X: &ast.IndexExpr{
											X:     ast.NewIdent("input"),
											Index: ast.NewIdent("i"),
										},
										Op: token.XOR,
										Y: &ast.IndexExpr{
											X: ast.NewIdent("key"),
											Index: &ast.BinaryExpr{
												X: &ast.Ident{
													Name: "i",
												},
												Op: token.REM,
												Y: &ast.CallExpr{
													Fun: ast.NewIdent("len"),
													Args: []ast.Expr{
														ast.NewIdent("key"),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				&ast.ReturnStmt{
					Results: []ast.Expr{
						ast.NewIdent("out"),
					},
				},
			},
		},
	}))

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

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	reader, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer reader.Close()

	for _, file := range reader.File {

		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		fPath := filepath.Clean(filepath.Join(dest, file.Name))
		if !strings.HasPrefix(fPath, filepath.Clean(dest)) {
			panic("illegal zip file path")
		}
		filenames = append(filenames, fPath)

		if file.FileInfo().IsDir() {
			os.MkdirAll(fPath, 0700)
		} else {
			if err = os.MkdirAll(filepath.Dir(fPath), 0700); err != nil {
				return filenames, err
			}
			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return filenames, err
			}
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			if err != nil {
				return filenames, err
			}
		}
	}
	return filenames, nil
}

func unzipBuf(src []byte, dest string) ([]string, error) {

	var filenames []string

	reader, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return filenames, err
	}

	for _, file := range reader.File {

		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		fPath := filepath.Join(dest, file.Name)
		filenames = append(filenames, fPath)

		if file.FileInfo().IsDir() {
			os.MkdirAll(fPath, 0700)
		} else {
			if err = os.MkdirAll(filepath.Dir(fPath), 0700); err != nil {
				return filenames, err
			}
			outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return filenames, err
			}
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			if err != nil {
				return filenames, err
			}
		}
	}
	return filenames, nil
}

func pseudoRandStringRunes(n int) string {
	insecureRand.Seed(time.Now().UnixNano())
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[insecureRand.Intn(len(letterRunes))]
	}
	return string(b)
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
