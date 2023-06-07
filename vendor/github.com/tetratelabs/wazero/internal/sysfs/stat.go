package sysfs

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func defaultStatFile(f *os.File) (fsapi.Stat_t, syscall.Errno) {
	if t, err := f.Stat(); err != nil {
		return fsapi.Stat_t{}, platform.UnwrapOSError(err)
	} else {
		return statFromFileInfo(t), 0
	}
}

func StatFromDefaultFileInfo(t fs.FileInfo) fsapi.Stat_t {
	st := fsapi.Stat_t{}
	st.Ino = 0
	st.Dev = 0
	st.Mode = t.Mode()
	st.Nlink = 1
	st.Size = t.Size()
	mtim := t.ModTime().UnixNano() // Set all times to the mod time
	st.Atim = mtim
	st.Mtim = mtim
	st.Ctim = mtim
	return st
}
