package console

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
	"net"
	"os"
	"strconv"
	"time"

	"github.com/maxlandon/readline"
	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var (
	// serverCallbacks - All items needed by the prompt when in Server menu.
	serverCallbacks = map[string]func() string{
		// Local working directory
		"{cwd}": func() string {
			cwd, err := os.Getwd()
			if err != nil {
				return "ERROR: Could not get working directory !"
			}
			return cwd
		},
		// Server IP
		"{server_ip}": func() string {
			return serverConfig.LHost
		},
		// Local IP address
		"{local_ip}": func() string {
			addrs, _ := net.InterfaceAddrs()
			var ip string
			for _, addr := range addrs {
				network, ok := addr.(*net.IPNet)
				if ok && !network.IP.IsLoopback() && network.IP.To4() != nil {
					ip = network.IP.String()
				}
			}
			return ip
		},
		// Jobs and/or listeners
		"{jobs}": func() string {
			req := &commonpb.Empty{}
			res, _ := transport.RPC.GetJobs(context.Background(), req, grpc.EmptyCallOption{})
			return strconv.Itoa(len(res.Active))
		},
		// Sessions
		"{sessions}": func() string {
			sReq := &commonpb.Empty{}
			sRes, _ := transport.RPC.GetSessions(context.Background(), sReq, grpc.EmptyCallOption{})
			return strconv.Itoa(len(sRes.Sessions))
		},
		"{timestamp}": func() string {
			return time.Now().Format("15:04:05.000")
		},
	}

	// serverColorCallbacks - All colors and effects needed in the main menu
	serverColorCallbacks = map[string]string{
		// Base readline colors
		"{blink}": "\033[5m", // blinking
		"{bold}":  readline.BOLD,
		"{dim}":   readline.DIM,
		"{fr}":    readline.RED,
		"{g}":     readline.GREEN,
		"{b}":     readline.BLUE,
		"{y}":     readline.YELLOW,
		"{fw}":    readline.FOREWHITE,
		"{bdg}":   readline.BACKDARKGRAY,
		"{br}":    readline.BACKRED,
		"{bg}":    readline.BACKGREEN,
		"{by}":    readline.BACKYELLOW,
		"{blb}":   readline.BACKLIGHTBLUE,
		"{reset}": readline.RESET,
		// Custom colors
		"{ly}":   "\033[38;5;187m",
		"{lb}":   "\033[38;5;117m", // like VSCode var keyword
		"{db}":   "\033[38;5;24m",
		"{bddg}": "\033[48;5;237m",
	}
)

var (
	sliverCallbacks = map[string]func() string{
		"{session_name}": func() string {
			return core.ActiveSession.Name
		},
		"{wd}": func() string {
			return core.ActiveSession.WorkingDir
		},
		"{user}": func() string {
			return core.ActiveSession.Username
		},
		"{host}": func() string {
			return core.ActiveSession.Hostname
		},
		"{address}": func() string {
			return core.ActiveSession.RemoteAddress
		},
		"{platform}": func() string {
			os := core.ActiveSession.OS
			arch := core.ActiveSession.Arch
			return fmt.Sprintf("%s/%s", os, arch)
		},
		"{os}": func() string {
			return core.ActiveSession.OS
		},
		"{arch}": func() string {
			return core.ActiveSession.Arch
		},
		"{status}": func() string {
			if core.ActiveSession.IsDead {
				return "DEAD"
			}
			return "up"
		},
		"{version}": func() string {
			return core.ActiveSession.Version
		},
		"{uid}": func() string {
			return core.ActiveSession.UID
		},
		"{gid}": func() string {
			return core.ActiveSession.GID
		},
		"{pid}": func() string {
			return strconv.Itoa(int(core.ActiveSession.PID))
		},
	}

	sliverColorCallbacks = map[string]string{
		// Base readline colors
		"{blink}": "\033[5m", // blinking
		"{bold}":  readline.BOLD,
		"{dim}":   readline.DIM, // for Base Dim
		"{fr}":    readline.RED, // for Base Fore Red
		"{g}":     readline.GREEN,
		"{b}":     readline.BLUE,
		"{y}":     readline.YELLOW,
		"{fw}":    readline.FOREWHITE, // for Base Fore White.
		"{dg}":    readline.BACKDARKGRAY,
		"{br}":    readline.BACKRED,
		"{bg}":    readline.BACKGREEN,
		"{by}":    readline.BACKYELLOW,
		"{blb}":   readline.BACKLIGHTBLUE,
		"{reset}": readline.RESET,
		// Custom colors
		"{ly}":   "\033[38;5;187m",
		"{lb}":   "\033[38;5;117m", // like VSCode var keyword
		"{bddg}": "\033[48;5;237m",
	}
)
