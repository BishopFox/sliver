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

// If you add a file to `sliver/` it won't be automatically included
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

		// C files for DLL
		"sliver.h",
		"sliver.c",

		"handlers/generic-rpc-handlers.go",
		"handlers/generic-tun-handlers.go",
		"handlers/generic-pivot-handlers.go",
		"handlers/special-handlers.go",
		"handlers/handlers_darwin.go",
		"handlers/handlers_linux.go",
		"handlers/handlers_windows.go",
		"handlers/handlers.go",

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

		"sc/screenshot_darwin.go",
		"sc/screenshot_linux.go",
		"sc/screenshot_windows.go",
		"sc/screenshot.go",

		"syscalls/syscalls.go",
		"syscalls/syscalls_windows.go",
		"syscalls/types_windows.go",
		"syscalls/zsyscalls_windows.go",

		"transports/crypto.go",
		"transports/tcp-mtls.go",
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

		// *** 3rd Party ***

		"3rdparty/BurntSushi/xgb/dri2/dri2.go",
		"3rdparty/BurntSushi/xgb/res/res.go",
		"3rdparty/BurntSushi/xgb/shape/shape.go",
		"3rdparty/BurntSushi/xgb/bigreq/bigreq.go",
		"3rdparty/BurntSushi/xgb/xf86vidmode/xf86vidmode.go",
		"3rdparty/BurntSushi/xgb/record/record.go",
		"3rdparty/BurntSushi/xgb/shm/shm.go",
		"3rdparty/BurntSushi/xgb/auth.go",
		"3rdparty/BurntSushi/xgb/sync.go",
		"3rdparty/BurntSushi/xgb/xevie/xevie.go",
		"3rdparty/BurntSushi/xgb/screensaver/screensaver.go",
		"3rdparty/BurntSushi/xgb/randr/randr.go",
		"3rdparty/BurntSushi/xgb/xtest/xtest.go",
		"3rdparty/BurntSushi/xgb/render/render.go",
		"3rdparty/BurntSushi/xgb/xvmc/xvmc.go",
		"3rdparty/BurntSushi/xgb/dpms/dpms.go",
		"3rdparty/BurntSushi/xgb/xcmisc/xcmisc.go",
		"3rdparty/BurntSushi/xgb/composite/composite.go",
		"3rdparty/BurntSushi/xgb/doc.go",
		"3rdparty/BurntSushi/xgb/xprint/xprint.go",
		"3rdparty/BurntSushi/xgb/ge/ge.go",
		"3rdparty/BurntSushi/xgb/xselinux/xselinux.go",
		"3rdparty/BurntSushi/xgb/cookie.go",
		"3rdparty/BurntSushi/xgb/xinerama/xinerama.go",
		"3rdparty/BurntSushi/xgb/xproto/xproto_test.go",
		"3rdparty/BurntSushi/xgb/xproto/xproto.go",
		"3rdparty/BurntSushi/xgb/xgb.go",
		"3rdparty/BurntSushi/xgb/xfixes/xfixes.go",
		"3rdparty/BurntSushi/xgb/xv/xv.go",
		"3rdparty/BurntSushi/xgb/help.go",
		"3rdparty/BurntSushi/xgb/glx/glx.go",
		"3rdparty/BurntSushi/xgb/damage/damage.go",
		"3rdparty/BurntSushi/xgb/xf86dri/xf86dri.go",
		"3rdparty/BurntSushi/xgb/conn.go",
		"3rdparty/BurntSushi/xgb/xgbgen/misc.go",
		"3rdparty/BurntSushi/xgb/xgbgen/field.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_union.go",
		"3rdparty/BurntSushi/xgb/xgbgen/size.go",
		"3rdparty/BurntSushi/xgb/xgbgen/type.go",
		"3rdparty/BurntSushi/xgb/xgbgen/request_reply.go",
		"3rdparty/BurntSushi/xgb/xgbgen/xml.go",
		"3rdparty/BurntSushi/xgb/xgbgen/protocol.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_list.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go.go",
		"3rdparty/BurntSushi/xgb/xgbgen/doc.go",
		"3rdparty/BurntSushi/xgb/xgbgen/translation.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_struct.go",
		"3rdparty/BurntSushi/xgb/xgbgen/context.go",
		"3rdparty/BurntSushi/xgb/xgbgen/aligngap.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_single_field.go",
		"3rdparty/BurntSushi/xgb/xgbgen/xml_fields.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_request_reply.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_event.go",
		"3rdparty/BurntSushi/xgb/xgbgen/expression.go",
		"3rdparty/BurntSushi/xgb/xgbgen/go_error.go",
		"3rdparty/BurntSushi/xgb/xgbgen/main.go",

		"3rdparty/gen2brain/shm/shm_test.go",
		"3rdparty/gen2brain/shm/shm.go",
		"3rdparty/gen2brain/shm/shm_openbsd.go",
		"3rdparty/gen2brain/shm/shm_netbsd.go",
		"3rdparty/gen2brain/shm/shm_dragonfly.go",
		"3rdparty/gen2brain/shm/shm_linux_386.go",
		"3rdparty/gen2brain/shm/shm_freebsd.go",
		"3rdparty/gen2brain/shm/shm_darwin.go",
		"3rdparty/gen2brain/shm/shm_solaris.go",
		"3rdparty/gen2brain/shm/shm_linux_amd64.go",

		"3rdparty/kbinani/screenshot/screenshot_openbsd.go",
		"3rdparty/kbinani/screenshot/screenshot_darwin.go",
		"3rdparty/kbinani/screenshot/internal/xwindow/xwindow.go",
		"3rdparty/kbinani/screenshot/internal/util/util.go",
		"3rdparty/kbinani/screenshot/screenshot_linux.go",
		"3rdparty/kbinani/screenshot/screenshot_freebsd.go",
		"3rdparty/kbinani/screenshot/screenshot_solaris.go",
		"3rdparty/kbinani/screenshot/screenshot_netbsd.go",
		"3rdparty/kbinani/screenshot/screenshot_windows.go",
		"3rdparty/kbinani/screenshot/screenshot_go1.9_or_earlier_darwin.go",
		"3rdparty/kbinani/screenshot/screenshot.go",

		"3rdparty/winio/hvsock.go",
		"3rdparty/winio/zsyscall_windows.go",
		"3rdparty/winio/file.go",
		"3rdparty/winio/pipe.go",
		"3rdparty/winio/sd.go",
		"3rdparty/winio/pkg/guid/guid.go",
		
		// "go.mod",

	}
)
