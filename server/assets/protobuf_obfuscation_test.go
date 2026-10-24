package assets

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	protobufs "github.com/bishopfox/sliver/protobuf"
)

func parseProtobufSource(t *testing.T, src []byte) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "protobuf.pb.go", src, 0)
	if err != nil {
		t.Fatalf("parse protobuf source: %v", err)
	}
	return file
}

func namedValue(t *testing.T, file *ast.File, name string) (token.Token, ast.Expr) {
	t.Helper()
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, ident := range value.Names {
				if ident.Name == name && i < len(value.Values) {
					return gen.Tok, value.Values[i]
				}
			}
		}
	}
	t.Fatalf("declaration %s not found", name)
	return token.ILLEGAL, nil
}

func descriptorLiteralBytes(t *testing.T, expr ast.Expr) []byte {
	t.Helper()
	switch value := expr.(type) {
	case *ast.BasicLit:
		if value.Kind != token.STRING {
			t.Fatalf("descriptor literal has token %s, want STRING", value.Kind)
		}
		unquoted, err := strconv.Unquote(value.Value)
		if err != nil {
			t.Fatalf("unquote descriptor literal: %v", err)
		}
		return []byte(unquoted)
	case *ast.BinaryExpr:
		if value.Op != token.ADD {
			t.Fatalf("descriptor expression uses %s, want +", value.Op)
		}
		return append(descriptorLiteralBytes(t, value.X), descriptorLiteralBytes(t, value.Y)...)
	case *ast.ParenExpr:
		return descriptorLiteralBytes(t, value.X)
	case *ast.CompositeLit:
		decoded := make([]byte, len(value.Elts))
		for i, element := range value.Elts {
			literal, ok := element.(*ast.BasicLit)
			if !ok || literal.Kind != token.INT {
				t.Fatalf("descriptor byte %d has type %T, want integer literal", i, element)
			}
			n, err := strconv.ParseUint(literal.Value, 0, 8)
			if err != nil {
				t.Fatalf("parse descriptor byte %d: %v", i, err)
			}
			decoded[i] = byte(n)
		}
		return decoded
	default:
		t.Fatalf("unsupported descriptor expression %T", expr)
		return nil
	}
}

func decodedTransformedDescriptor(t *testing.T, file *ast.File, name string) []byte {
	t.Helper()
	_, expr := namedValue(t, file, name)
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		t.Fatalf("%s initializer = %T, want decoder call with two arguments", name, expr)
	}
	decoder, ok := call.Fun.(*ast.Ident)
	if !ok || (decoder.Name != "xorString" && decoder.Name != "xorBytes") {
		t.Fatalf("%s uses decoder %T, want xorString or xorBytes", name, call.Fun)
	}
	keyRef, ok := call.Args[1].(*ast.Ident)
	if !ok || keyRef.Name != "xorKey" {
		t.Fatalf("%s decoder does not use xorKey", name)
	}
	_, keyExpr := namedValue(t, file, "xorKey")
	key := descriptorLiteralBytes(t, keyExpr)
	if len(key) == 0 {
		t.Fatal("xorKey is empty")
	}
	encoded := descriptorLiteralBytes(t, call.Args[0])
	decoded := make([]byte, len(encoded))
	for i := range encoded {
		decoded[i] = encoded[i] ^ key[i%len(key)]
	}
	return decoded
}

func TestXorPBRawBytesPreservesEmbeddedDescriptors(t *testing.T) {
	for _, tc := range []struct {
		path string
		name string
	}{
		{"sliverpb/sliver.pb.go", "file_sliverpb_sliver_proto_rawDesc"},
		{"commonpb/common.pb.go", "file_commonpb_common_proto_rawDesc"},
		{"dnspb/dns.pb.go", "file_dnspb_dns_proto_rawDesc"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			source, err := protobufs.FS.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			_, originalExpr := namedValue(t, parseProtobufSource(t, source), tc.name)
			want := descriptorLiteralBytes(t, originalExpr)

			transformed, err := xorPBRawBytes(source)
			if err != nil {
				t.Fatalf("transform descriptor: %v", err)
			}
			got := decodedTransformedDescriptor(t, parseProtobufSource(t, transformed), tc.name)
			if !bytes.Equal(got, want) {
				t.Fatalf("decoded descriptor differs from embedded %s (%d versus %d bytes)", tc.path, len(got), len(want))
			}
		})
	}
}

func TestXorPBRawBytesRejectsUnsupportedDescriptorShapes(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"missing descriptor", "package p\nconst other = \"abc\"\n"},
		{"unsupported suffix", "package p\nconst suffix = \"def\"\nconst file_x_rawDesc = \"abc\" + suffix\n"},
		{"grouped declaration", "package p\nconst (\nfile_x_rawDesc = \"abc\"\nother = \"def\"\n)\n"},
		{"unsupported byte element", "package p\nvar value = byte(1)\nvar file_x_rawDesc = []byte{0x00, value}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.src)
			original := bytes.Clone(source)
			if _, err := xorPBRawBytes(source); err == nil {
				t.Fatal("expected an unsupported descriptor shape to fail")
			}
			if !bytes.Equal(source, original) {
				t.Fatal("failed transform modified its input")
			}
		})
	}
}

func TestXorPBRawBytesSupportsByteLiterals(t *testing.T) {
	const source = "package p\nvar file_x_rawDesc = []byte{0x00, 1, 0xff}\n"
	transformed, err := xorPBRawBytes([]byte(source))
	if err != nil {
		t.Fatalf("transform byte descriptor: %v", err)
	}
	got := decodedTransformedDescriptor(t, parseProtobufSource(t, transformed), "file_x_rawDesc")
	if want := []byte{0, 1, 255}; !bytes.Equal(got, want) {
		t.Fatalf("decoded byte descriptor = %v, want %v", got, want)
	}
}

type protobufFieldTag struct {
	structName string
	fieldName  string
	protobuf   string
	json       string
	raw        string
}

func protobufFieldTags(t *testing.T, src []byte) []protobufFieldTag {
	t.Helper()
	file := parseProtobufSource(t, src)
	tags := []protobufFieldTag{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				if field.Tag == nil {
					continue
				}
				raw, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					t.Fatalf("unquote %s struct tag: %v", typeSpec.Name.Name, err)
				}
				protoTag, ok := reflect.StructTag(raw).Lookup("protobuf")
				if !ok {
					continue
				}
				jsonTag := reflect.StructTag(raw).Get("json")
				for _, name := range field.Names {
					tags = append(tags, protobufFieldTag{typeSpec.Name.Name, name.Name, protoTag, jsonTag, raw})
				}
			}
		}
	}
	return tags
}

func nameFromTag(t *testing.T, value string) string {
	t.Helper()
	for _, part := range strings.Split(value, ",") {
		if strings.HasPrefix(part, "name=") {
			return strings.TrimPrefix(part, "name=")
		}
	}
	t.Fatalf("protobuf tag %q has no name", value)
	return ""
}

func enumFromTag(value string) string {
	for _, part := range strings.Split(value, ",") {
		if strings.HasPrefix(part, "enum=") {
			return strings.TrimPrefix(part, "enum=")
		}
	}
	return ""
}

func tagWithoutNames(value string) string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		if strings.HasPrefix(part, "name=") {
			parts[i] = "name=*"
		} else if strings.HasPrefix(part, "enum=") {
			parts[i] = "enum=*"
		}
	}
	return strings.Join(parts, ",")
}

func assertRewrittenProtobufFieldTag(t *testing.T, index int, old, newTag protobufFieldTag) {
	t.Helper()
	if old.structName != newTag.structName || old.fieldName != newTag.fieldName {
		t.Fatalf("field %d changed from %s.%s to %s.%s", index, old.structName, old.fieldName, newTag.structName, newTag.fieldName)
	}
	if tagWithoutNames(old.protobuf) != tagWithoutNames(newTag.protobuf) {
		t.Fatalf("wire tag changed for %s.%s: %q -> %q", old.structName, old.fieldName, old.protobuf, newTag.protobuf)
	}
	oldName := nameFromTag(t, old.protobuf)
	newName := nameFromTag(t, newTag.protobuf)
	if oldName == newName || len(oldName) != len(newName) {
		t.Fatalf("field name was not rewritten with the same length for %s.%s: %q -> %q", old.structName, old.fieldName, oldName, newName)
	}
	oldJSON := strings.SplitN(old.json, ",", 2)
	newJSON := strings.SplitN(newTag.json, ",", 2)
	if len(oldJSON) != len(newJSON) || oldJSON[0] == newJSON[0] || len(oldJSON[0]) != len(newJSON[0]) || strings.Join(oldJSON[1:], ",") != strings.Join(newJSON[1:], ",") {
		t.Fatalf("JSON tag options changed for %s.%s: %q -> %q", old.structName, old.fieldName, old.json, newTag.json)
	}
	oldEnum, newEnum := enumFromTag(old.protobuf), enumFromTag(newTag.protobuf)
	if oldEnum != "" && (newEnum == oldEnum || len(newEnum) != len(oldEnum)) {
		t.Fatalf("enum tag was not rewritten with the same length for %s.%s: %q -> %q", old.structName, old.fieldName, oldEnum, newEnum)
	}
}

func TestStripSliverpbRewritesEveryFieldTagWithoutChangingWireNumbers(t *testing.T) {
	source, err := protobufs.FS.ReadFile("sliverpb/sliver.pb.go")
	if err != nil {
		t.Fatal(err)
	}
	transformed, err := stripSliverpb(source)
	if err != nil {
		t.Fatalf("rewrite protobuf tags: %v", err)
	}
	before := protobufFieldTags(t, source)
	after := protobufFieldTags(t, transformed)
	if len(before) != 656 || len(after) != len(before) {
		t.Fatalf("protobuf tag coverage = %d original, %d rewritten; want 656 each", len(before), len(after))
	}
	rewrittenByOriginal := make(map[string]string)
	packedFound := false
	for i, old := range before {
		newTag := after[i]
		assertRewrittenProtobufFieldTag(t, i, old, newTag)
		if old.structName == "PivotStartListenerReq" && old.fieldName == "Options" {
			packedFound = true
			if !strings.Contains(newTag.protobuf, ",packed,") {
				t.Fatalf("packed Options tag lost packed option: %q", newTag.protobuf)
			}
		}
		if earlier, exists := rewrittenByOriginal[old.raw]; exists && earlier != newTag.raw {
			t.Fatalf("identical original tag %q rewrote inconsistently", old.raw)
		}
		rewrittenByOriginal[old.raw] = newTag.raw
	}
	if !packedFound {
		t.Fatal("packed PivotStartListenerReq.Options tag not found")
	}
}

func TestStripSliverpbRejectsMissingTags(t *testing.T) {
	if _, err := stripSliverpb([]byte("package p\ntype Empty struct{}\n")); err == nil {
		t.Fatal("expected a source without protobuf field tags to fail")
	}
}

func TestSetupGoPathReportsTransformedFileWriteErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		includeDNS bool
	}{
		{"sliver", "protobuf/sliverpb/sliver.pb.go", false},
		{"common", "protobuf/commonpb/common.pb.go", false},
		{"dns", "protobuf/dnspb/dns.pb.go", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			destination := t.TempDir()
			blocker := filepath.Join(destination, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(blocker, 0o700); err != nil {
				t.Fatal(err)
			}
			err := SetupGoPath(destination, tc.includeDNS)
			if err == nil || !strings.Contains(err.Error(), filepath.Base(blocker)) {
				t.Fatalf("SetupGoPath() error = %v, want path %s", err, blocker)
			}
		})
	}
}
