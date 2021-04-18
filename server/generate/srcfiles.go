package generate

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

// These files get rendered as part of the build process.

// If you add a file to `implant/sliver/` it won't be automatically included
// as part of the build by the server, you have to add it here.

var (
	srcFiles = []string{
		"constants/constants.go",

		"encoders/base64.go",
		"encoders/combos.go",
		"encoders/encoders.go",
		"encoders/english-words.go",
		"encoders/english.go",
		"encoders/gzip.go",
		"encoders/hex.go",
		"encoders/images.go",

		"evasion/evasion.go",
		"evasion/evasion_darwin.go",
		"evasion/evasion_linux.go",
		"evasion/evasion_windows.go",

		"forwarder/forwarder.go",
		"forwarder/socks.go",
		"forwarder/portforward.go",

		// C files for DLL
		"sliver.h",
		"sliver.c",

		"handlers/rpc-handlers.go",
		"handlers/tun-handlers.go",
		"handlers/pivot-handlers.go",
		"handlers/special-handlers.go",
		"handlers/handlers_darwin.go",
		"handlers/handlers_linux.go",
		"handlers/handlers_windows.go",
		"handlers/handlers.go",

		"hostuuid/uuid.go",
		"hostuuid/uuid_windows.go",
		"hostuuid/uuid_darwin.go",
		"hostuuid/uuid_linux.go",

		"limits/limits.go",
		"limits/limits_windows.go",
		"limits/limits_darwin.go",
		"limits/limits_linux.go",

		"netstat/netstat.go",
		"netstat/netstat_windows.go",
		"netstat/netstat_linux.go",
		"netstat/netstat_darwin.go",

		"procdump/dump.go",
		"procdump/dump_windows.go",
		"procdump/dump_linux.go",
		"procdump/dump_darwin.go",

		"proxy/provider_darwin.go",
		"proxy/provider_linux.go",
		"proxy/provider_windows.go",
		"proxy/provider.go",
		"proxy/proxy.go",
		"proxy/url.go",

		"ps/ps.go",
		"ps/ps_windows.go",
		"ps/ps_linux.go",
		"ps/ps_darwin.go",

		"registry/registry.go",
		"registry/registry_windows.go",

		"service/service.go",
		"service/service_windows.go",

		"shell/shell.go",
		"shell/shell_windows.go",
		"shell/shell_darwin.go",
		"shell/pty/pty_darwin.go",
		"shell/shell_linux.go",
		"shell/pty/pty_linux.go",

		"shell/pty/run.go",
		"shell/pty/util.go",
		"shell/pty/doc.go",
		"shell/pty/types.go",
		"shell/pty/ztypes_386.go",
		"shell/pty/ztypes_amd64.go",
		"shell/pty/ioctl.go",
		"shell/pty/ioctl_bsd.go",
		"shell/pty/ioctl_darwin.go",
		"shell/pty/pty_unsupported.go",

		"taskrunner/task.go",
		"taskrunner/task_windows.go",
		"taskrunner/task_darwin.go",
		"taskrunner/task_linux.go",

		"priv/priv.go",
		"priv/priv_windows.go",

		"pivots/named-pipe.go",
		"pivots/named-pipe_windows.go",
		"pivots/tcp.go",
		"pivots/pivots.go",

		"screen/screen.go",
		"screen/screenshot_darwin.go",
		"screen/screenshot_linux.go",
		"screen/screenshot_windows.go",

		"syscalls/syscalls.go",
		"syscalls/syscalls_windows.go",
		"syscalls/types_windows.go",
		"syscalls/zsyscalls_windows.go",

		"transports/crypto.go",
		"transports/tcp-mtls.go",
		"netstack/tun.go",
		"transports/tcp-wg.go",
		"transports/tcp-http.go",
		"transports/udp-dns.go",
		"transports/named-pipe.go",
		"transports/tcp-pivot.go",
		"transports/transports.go",

		"version/version.go",
		"version/version_windows.go",
		"version/version_linux.go",
		"version/version_darwin.go",

		"winhttp/winhttp.go",

		"sliver.go",
	}

	// PURE GO only - For compiling to unsupported platforms
	genericSrcFiles = []string{
		"constants/constants.go",

		"encoders/base64.go",
		"encoders/combos.go",
		"encoders/encoders.go",
		"encoders/english-words.go",
		"encoders/english.go",
		"encoders/gzip.go",
		"encoders/hex.go",
		"encoders/images.go",

		"handlers/handlers_default.go",
		"handlers/handlers.go",

		"hostuuid/uuid_default.go",

		"limits/limits.go",
		"limits/limits_default.go",

		"pivots/named-pipe.go",
		"pivots/tcp.go",
		"pivots/pivots.go",

		"proxy/provider.go",
		"proxy/provider_default.go",
		"proxy/proxy.go",
		"proxy/url.go",

		"transports/crypto.go",
		"transports/tcp-mtls.go",
		"netstack/tun.go",
		"transports/tcp-wg.go",
		"transports/tcp-http.go",
		"transports/udp-dns.go",
		"transports/named-pipe.go",
		"transports/tcp-pivot.go",
		"transports/transports.go",

		"version/version_default.go",
		"version/version.go",

		"sliver.go",
	}
)
