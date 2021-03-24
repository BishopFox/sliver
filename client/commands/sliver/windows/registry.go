package windows

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
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var validHives = []string{
	"HKCU",
	"HKLM",
	"HKCC",
	"HKPD",
	"HKU",
	"HKCR",
}

var ValidTypes = []string{
	"binary",
	"dword",
	"qword",
	"string",
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

// Registry - Windows registry management
type Registry struct{}

// Execute - Windows registry management. Requires a subcommand.
func (r *Registry) Execute(args []string) (err error) {
	return
}

// RegistryRead - Read values from the Windows registry.
type RegistryRead struct {
	Positional struct {
		KeyPath string `description:"path to registry key" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Hive     string `long:"hive" short:"H" description:"registry hive" default:"HKCU"`
		Hostname string `long:"hostname" short:"o" description:"remove host to read values from"`
	} `group:"read options"`
}

// Execute - Read values from the Windows registry.
func (rr *RegistryRead) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	hostname := rr.Options.Hostname
	hive := rr.Options.Hive

	regPath := rr.Positional.KeyPath
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	regRead, err := transport.RPC.RegistryRead(context.Background(), &sliverpb.RegistryReadReq{
		Hive:     hive,
		Path:     regPath,
		Key:      key,
		Hostname: hostname,
		Request:  cctx.Request(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	if regRead.Response != nil && regRead.Response.Err != "" {
		fmt.Printf(util.Error+"Error: %s", regRead.Response.Err)
		return
	}
	fmt.Println(regRead.Value)

	return
}

// RegistryWrite - Write values to the Windows registry.
type RegistryWrite struct {
	Positional struct {
		Key   string `description:"registry key name" required:"1"`
		Value string `description:"registry key value" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Hive     string `long:"hive" short:"H" description:"registry hive" default:"HKCU"`
		Hostname string `long:"hostname" short:"o" description:"remove host to write values to"`
		Type     string `long:"type" short:"T" description:"type of value to write (if binary, you must provide a path with --path)" default:"string"`
		Path     string `long:"path" short:"p" description:"path to the binary file to write"`
	} `group:"write options"`
}

// Execute - Write values to the Windows registry.
func (rw *RegistryWrite) Execute(args []string) (err error) {
	var (
		dwordValue  uint32
		qwordValue  uint64
		stringValue string
		binaryValue []byte
	)
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	binPath := rw.Options.Path
	hostname := rw.Options.Hostname
	flagType := rw.Options.Type
	valType, err := getType(flagType)
	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}
	hive := rw.Options.Hive

	regPath := rw.Positional.Key
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	value := rw.Positional.Value
	switch valType {
	case sliverpb.RegistryType_BINARY:
		var (
			v   []byte
			err error
		)
		if binPath == "" {
			v, err = hex.DecodeString(value)
			if err != nil {
				fmt.Printf(util.Error+"Error: %v", err)
				return err
			}
		} else {
			v, err = ioutil.ReadFile(binPath)
			if err != nil {
				fmt.Printf(util.Error+"Error: %v", err)
				return err
			}
		}
		binaryValue = v
	case sliverpb.RegistryType_DWORD:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			fmt.Printf(util.Error+"Error: %v", err)
			return err
		}
		dwordValue = uint32(v)
	case sliverpb.RegistryType_QWORD:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			fmt.Printf(util.Error+"Error: %v", err)
			return err
		}
		qwordValue = v
	case sliverpb.RegistryType_STRING:
		stringValue = value
	default:
		fmt.Printf(util.Error + "Invalid type")
		return
	}
	regWrite, err := transport.RPC.RegistryWrite(context.Background(), &sliverpb.RegistryWriteReq{
		Request:     cctx.Request(session),
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
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}
	if regWrite.Response != nil && regWrite.Response.Err != "" {
		fmt.Printf(util.Error+"Error: %v", regWrite.Response.Err)
		return
	}
	fmt.Printf(util.Error + "Value written to registry\n")

	return
}

// RegistryCreateKey - Create a registry key .
type RegistryCreateKey struct {
	Positional struct {
		Key string `description:"registry key name" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Hive     string `long:"hive" short:"H" description:"registry hive" default:"HKCU"`
		Hostname string `long:"hostname" short:"o" description:"remove host to write values to"`
	} `group:"write options"`
}

// Execute - Create a registry key
func (rck *RegistryCreateKey) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	hostname := rck.Options.Hostname
	hive := rck.Options.Hive

	regPath := rck.Positional.Key
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	createKeyResp, err := transport.RPC.RegistryCreateKey(context.Background(), &sliverpb.RegistryCreateKeyReq{
		Hive:     hive,
		Path:     regPath,
		Key:      key,
		Hostname: hostname,
		Request:  cctx.Request(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	if createKeyResp.Response != nil && createKeyResp.Response.Err != "" {
		fmt.Printf(util.Warn+"Error: %s", createKeyResp.Response.Err)
		return
	}
	fmt.Printf(util.Info+"Key created at %s\\%s", regPath, key)

	return
}
