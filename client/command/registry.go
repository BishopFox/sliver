package command

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

var validHives = []string{
	"HKCU",
	"HKLM",
	"HKCC",
	"HKPD",
	"HKU",
	"HKCR",
}

var validTypes = []string{
	"binary",
	"dword",
	"qword",
	"string",
}

func checkHive(hive string) error {
	for _, h := range validHives {
		if h == hive {
			return nil
		}
	}
	return fmt.Errorf("invalid hive %s", hive)
}

func getType(t string) (sliverpb.RegistryType, error) {
	var res sliverpb.RegistryType
	switch t {
	case "binary":
		res = sliverpb.RegistryType_BINARY
	case "dword":
		res = sliverpb.RegistryType_DWORD
	case "qword":
		res = sliverpb.RegistryType_QWORD
	case "string":
		res = sliverpb.RegistryType_STRING
	default:
		return res, fmt.Errorf("invalid type %s", t)
	}
	return res, nil
}

// registry read --hostname aa.bc.local --hive HKCU "software\google\chrome\blbeacon\version"
func registryReadCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if session.OS != "windows" {
		fmt.Printf(Warn + "Error: command not supported on this operating system.")
		return
	}

	hostname := ctx.Flags.String("hostname")
	hive := ctx.Flags.String("hive")
	if err := checkHive(hive); err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "you must provide a path")
		return
	}
	regPath := ctx.Args[0]
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	regRead, err := rpc.RegistryRead(context.Background(), &sliverpb.RegistryReadReq{
		Hive:     hive,
		Path:     regPath,
		Key:      key,
		Hostname: hostname,
		Request:  ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if regRead.Response != nil && regRead.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s", regRead.Response.Err)
		return
	}
	fmt.Println(regRead.Value)
}

// registry write --hive HKCU --type dword "software\google\chrome\blbeacon\hello" 32
func registryWriteCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	var (
		dwordValue  uint32
		qwordValue  uint64
		stringValue string
		binaryValue []byte
	)
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if session.OS != "windows" {
		fmt.Printf(Warn + "Error: command not supported on this operating system.")
		return
	}
	binPath := ctx.Flags.String("path")
	hostname := ctx.Flags.String("hostname")
	flagType := ctx.Flags.String("type")
	valType, err := getType(flagType)
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	hive := ctx.Flags.String("hive")
	if err := checkHive(hive); err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if len(ctx.Args) != 2 {
		fmt.Printf(Warn + "you must provide a path and a value to write")
		return
	}
	regPath := ctx.Args[0]
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	value := ctx.Args[1]
	switch valType {
	case sliverpb.RegistryType_BINARY:
		var (
			v   []byte
			err error
		)
		if binPath == "" {
			v, err = hex.DecodeString(value)
			if err != nil {
				fmt.Printf(Warn+"Error: %v", err)
				return
			}
		} else {
			v, err = ioutil.ReadFile(binPath)
			if err != nil {
				fmt.Printf(Warn+"Error: %v", err)
				return
			}
		}
		binaryValue = v
	case sliverpb.RegistryType_DWORD:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}
		dwordValue = uint32(v)
	case sliverpb.RegistryType_QWORD:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}
		qwordValue = v
	case sliverpb.RegistryType_STRING:
		stringValue = value
	default:
		fmt.Printf(Warn + "Invalid type")
		return
	}
	regWrite, err := rpc.RegistryWrite(context.Background(), &sliverpb.RegistryWriteReq{
		Request:     ActiveSession.Request(ctx),
		Hostname:    hostname,
		Hive:        hive,
		Path:        regPath,
		Type:        valType,
		Key:         key,
		StringValue: stringValue,
		DWordValue:  dwordValue,
		QWordValue:  qwordValue,
		ByteValue:   binaryValue,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if regWrite.Response != nil && regWrite.Response.Err != "" {
		fmt.Printf(Warn+"Error: %v", regWrite.Response.Err)
		return
	}
	fmt.Printf(Info + "Value written to registry\n")
}

func regCreateKeyCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	hostname := ctx.Flags.String("hostname")
	hive := ctx.Flags.String("hive")
	if err := checkHive(hive); err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "you must provide a path")
		return
	}
	regPath := ctx.Args[0]
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	createKeyResp, err := rpc.RegistryCreateKey(context.Background(), &sliverpb.RegistryCreateKeyReq{
		Hive:     hive,
		Path:     regPath,
		Key:      key,
		Hostname: hostname,
		Request:  ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if createKeyResp.Response != nil && createKeyResp.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s", createKeyResp.Response.Err)
		return
	}
	fmt.Printf(Info+"Key created at %s\\%s", regPath, key)
}
