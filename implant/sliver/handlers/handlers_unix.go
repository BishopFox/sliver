//go:build darwin || linux

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
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

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

				err := filepath.WalkDir(target, func(file string, _ fs.DirEntry, err error) error {
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
	var uidStr string
	var gidStr string
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

	uidStr = chownReq.Uid
	usr, err = user.Lookup(uidStr)
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	uid, err = strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		chown.Response.Err = err.Error()
		goto finished
	}

	gidStr = chownReq.Gid
	grp, err = user.LookupGroup(gidStr)
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

		err := filepath.WalkDir(target, func(file string, _ fs.DirEntry, err error) error {
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
