package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	registryE2EHive       = "HKCU"
	registryE2EParentPath = `Software`
)

func (s *suite) exerciseWindowsRegistry(target implantTarget, transport string) (resultErr error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate registry fixture name: %w", err)
	}
	fixtureName := "SliverE2E-" + hex.EncodeToString(tokenBytes)
	fixturePath := registryE2EParentPath + `\` + fixtureName
	childName := "nested"
	rootMayExist := false

	defer func() {
		if !rootMayExist {
			return
		}
		if err := s.cleanupWindowsRegistry(target, fixtureName, fixturePath, childName); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean registry fixture %s\\%s: %w", registryE2EHive, fixturePath, err))
		}
	}()

	if err := s.step(target, transport, "RegistryCreateKey", "unique HKCU subtree and child", func() error {
		// Arm cleanup before issuing the mutating RPC so a response timeout cannot
		// leave a key that the implant successfully created behind.
		rootMayExist = true
		if err := s.registryCreateKey(target, registryE2EParentPath, fixtureName); err != nil {
			return fmt.Errorf("create fixture root: %w", err)
		}
		if err := s.registryCreateKey(target, fixturePath, childName); err != nil {
			return fmt.Errorf("create fixture child: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	type registryValue struct {
		name        string
		valueType   uint32
		stringValue string
		byteValue   []byte
		dwordValue  uint32
		qwordValue  uint64
		expected    string
	}
	values := []registryValue{
		{name: "string-value", valueType: sliverpb.RegistryTypeString, stringValue: "sliver-e2e-" + fixtureName, expected: "sliver-e2e-" + fixtureName},
		{name: "binary-value", valueType: sliverpb.RegistryTypeBinary, byteValue: []byte{0x00, 0x7f, 0x80, 0xff}, expected: "[0 127 128 255]"},
		{name: "dword-value", valueType: sliverpb.RegistryTypeDWORD, dwordValue: 0x5a17c0de, expected: "0x5a17c0de"},
		{name: "qword-value", valueType: sliverpb.RegistryTypeQWORD, qwordValue: 0x0123456789abcdef, expected: "0x123456789abcdef"},
	}

	if err := s.step(target, transport, "RegistryWrite", "string binary DWORD and QWORD values", func() error {
		for _, value := range values {
			if err := s.registryWrite(target, fixturePath, value.name, value.valueType, value.stringValue, value.byteValue, value.dwordValue, value.qwordValue); err != nil {
				return fmt.Errorf("write %s: %w", value.name, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "RegistryRead", "exact typed value round trips", func() error {
		for _, value := range values {
			got, err := s.registryRead(target, fixturePath, value.name)
			if err != nil {
				return fmt.Errorf("read %s: %w", value.name, err)
			}
			if got != value.expected {
				return fmt.Errorf("read %s got %q, want %q", value.name, got, value.expected)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "RegistryListSubKeys", "exact disposable child inventory", func() error {
		subkeys, err := s.registryListSubKeys(target, fixturePath)
		if err != nil {
			return err
		}
		if !slices.Equal(subkeys, []string{childName}) {
			return fmt.Errorf("subkeys got %v, want [%s]", subkeys, childName)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "RegistryListValues", "exact typed value inventory", func() error {
		names, err := s.registryListValues(target, fixturePath)
		if err != nil {
			return err
		}
		want := make([]string, 0, len(values))
		for _, value := range values {
			want = append(want, value.name)
		}
		sort.Strings(want)
		if !slices.Equal(names, want) {
			return fmt.Errorf("value names got %v, want %v", names, want)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "RegistryDeleteKey", "remove only disposable child and root", func() error {
		if err := s.registryDeleteKey(target, fixturePath, childName); err != nil {
			return fmt.Errorf("delete fixture child: %w", err)
		}
		if err := s.registryDeleteKey(target, registryE2EParentPath, fixtureName); err != nil {
			return fmt.Errorf("delete fixture root: %w", err)
		}
		if err := s.requireRegistryKeyAbsent(target, fixtureName); err != nil {
			return err
		}
		rootMayExist = false
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (s *suite) cleanupWindowsRegistry(target implantTarget, fixtureName string, fixturePath string, childName string) error {
	// DeleteKey cannot recursively delete subkeys, so always attempt the known
	// child first. Delete errors are superseded by an authoritative parent
	// listing when it proves the unique root is absent.
	childErr := s.registryDeleteKey(target, fixturePath, childName)
	rootErr := s.registryDeleteKey(target, registryE2EParentPath, fixtureName)
	if err := s.requireRegistryKeyAbsent(target, fixtureName); err != nil {
		return errors.Join(childErr, rootErr, err)
	}
	return nil
}

func (s *suite) requireRegistryKeyAbsent(target implantTarget, fixtureName string) error {
	subkeys, err := s.registryListSubKeys(target, registryE2EParentPath)
	if err != nil {
		return fmt.Errorf("verify fixture root removal: %w", err)
	}
	if slices.Contains(subkeys, fixtureName) {
		return fmt.Errorf("registry fixture root %q still exists", fixtureName)
	}
	return nil
}

func (s *suite) registryCreateKey(target implantTarget, path string, key string) error {
	_, err := invokeRPC(s, target, "RegistryCreateKey", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegistryCreateKey, error) {
		return s.rpc.RegistryCreateKey(ctx, &sliverpb.RegistryCreateKeyReq{
			Hive: registryE2EHive, Path: path, Key: key, Request: request,
		})
	}, func(response *sliverpb.RegistryCreateKey) *commonpb.Response { return response.GetResponse() })
	return err
}

func (s *suite) registryWrite(target implantTarget, path string, key string, valueType uint32, stringValue string, byteValue []byte, dwordValue uint32, qwordValue uint64) error {
	_, err := invokeRPC(s, target, "RegistryWrite", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegistryWrite, error) {
		return s.rpc.RegistryWrite(ctx, &sliverpb.RegistryWriteReq{
			Hive: registryE2EHive, Path: path, Key: key, Type: valueType, Request: request,
			StringValue: stringValue, ByteValue: byteValue, DWordValue: dwordValue, QWordValue: qwordValue,
		})
	}, func(response *sliverpb.RegistryWrite) *commonpb.Response { return response.GetResponse() })
	return err
}

func (s *suite) registryRead(target implantTarget, path string, key string) (string, error) {
	response, err := invokeRPC(s, target, "RegistryRead", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegistryRead, error) {
		return s.rpc.RegistryRead(ctx, &sliverpb.RegistryReadReq{
			Hive: registryE2EHive, Path: path, Key: key, Request: request,
		})
	}, func(response *sliverpb.RegistryRead) *commonpb.Response { return response.GetResponse() })
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func (s *suite) registryDeleteKey(target implantTarget, path string, key string) error {
	_, err := invokeRPC(s, target, "RegistryDeleteKey", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegistryDeleteKey, error) {
		return s.rpc.RegistryDeleteKey(ctx, &sliverpb.RegistryDeleteKeyReq{
			Hive: registryE2EHive, Path: path, Key: key, Request: request,
		})
	}, func(response *sliverpb.RegistryDeleteKey) *commonpb.Response { return response.GetResponse() })
	return err
}

func (s *suite) registryListSubKeys(target implantTarget, path string) ([]string, error) {
	response, err := invokeRPC(s, target, "RegistryListSubKeys", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegistrySubKeyList, error) {
		return s.rpc.RegistryListSubKeys(ctx, &sliverpb.RegistrySubKeyListReq{
			Hive: registryE2EHive, Path: path, Request: request,
		})
	}, func(response *sliverpb.RegistrySubKeyList) *commonpb.Response { return response.GetResponse() })
	if err != nil {
		return nil, err
	}
	sort.Strings(response.Subkeys)
	return response.Subkeys, nil
}

func (s *suite) registryListValues(target implantTarget, path string) ([]string, error) {
	response, err := invokeRPC(s, target, "RegistryListValues", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegistryValuesList, error) {
		return s.rpc.RegistryListValues(ctx, &sliverpb.RegistryListValuesReq{
			Hive: registryE2EHive, Path: path, Request: request,
		})
	}, func(response *sliverpb.RegistryValuesList) *commonpb.Response { return response.GetResponse() })
	if err != nil {
		return nil, err
	}
	sort.Strings(response.ValueNames)
	return response.ValueNames, nil
}
