package command

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

func getType(t string) (uint32, error) {
	var res uint32
	switch t {
	case "binary":
		res = sliverpb.RegistryTypeBinary
	case "dword":
		res = sliverpb.RegistryTypeDWORD
	case "qword":
		res = sliverpb.RegistryTypeQWORD
	case "string":
		res = sliverpb.RegistryTypeString
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

	regPath := ctx.Args.String("registry-path")
	if regPath == "" {
		fmt.Printf(Warn + "you must provide a path")
		return
	}
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

	regPath := ctx.Args.String("registry-path")
	value := ctx.Args.String("value")
	if regPath == "" || value == "" {
		fmt.Printf(Warn + "you must provide a path and a value to write")
		return
	}
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	switch valType {
	case sliverpb.RegistryTypeBinary:
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
	case sliverpb.RegistryTypeDWORD:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}
		dwordValue = uint32(v)
	case sliverpb.RegistryTypeQWORD:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			fmt.Printf(Warn+"Error: %v", err)
			return
		}
		qwordValue = v
	case sliverpb.RegistryTypeString:
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

	regPath := ctx.Args.String("registry-path")
	if regPath == "" {
		fmt.Printf(Warn + "you must provide a path")
		return
	}
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
