package vfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/ncruces/go-sqlite3/util/osutil"
)

type vfsOS struct{}

func (vfsOS) FullPathname(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return path, nil
		}
		return "", err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		err = _OK_SYMLINK
	}
	return path, err
}

func (vfsOS) Delete(path string, syncDir bool) error {
	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return _IOERR_DELETE_NOENT
		}
		return err
	}
	if runtime.GOOS != "windows" && syncDir {
		f, err := os.Open(filepath.Dir(path))
		if err != nil {
			return _OK
		}
		defer f.Close()
		err = osSync(f, false, false)
		if err != nil {
			return _IOERR_DIR_FSYNC
		}
	}
	return nil
}

func (vfsOS) Access(name string, flags AccessFlag) (bool, error) {
	err := osAccess(name, flags)
	if flags == ACCESS_EXISTS {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
	} else {
		if errors.Is(err, fs.ErrPermission) {
			return false, nil
		}
	}
	return err == nil, err
}

func (vfsOS) Open(name string, flags OpenFlag) (File, OpenFlag, error) {
	return nil, 0, _CANTOPEN
}

func (vfsOS) OpenFilename(name *Filename, flags OpenFlag) (File, OpenFlag, error) {
	var oflags int
	if flags&OPEN_EXCLUSIVE != 0 {
		oflags |= os.O_EXCL
	}
	if flags&OPEN_CREATE != 0 {
		oflags |= os.O_CREATE
	}
	if flags&OPEN_READONLY != 0 {
		oflags |= os.O_RDONLY
	}
	if flags&OPEN_READWRITE != 0 {
		oflags |= os.O_RDWR
	}

	var err error
	var f *os.File
	if name == nil {
		f, err = os.CreateTemp("", "*.db")
	} else {
		f, err = osutil.OpenFile(name.String(), oflags, 0666)
	}
	if err != nil {
		if errors.Is(err, syscall.EISDIR) {
			return nil, flags, _CANTOPEN_ISDIR
		}
		return nil, flags, err
	}

	if modeof := name.URIParameter("modeof"); modeof != "" {
		if err = osSetMode(f, modeof); err != nil {
			f.Close()
			return nil, flags, _IOERR_FSTAT
		}
	}
	if flags&OPEN_DELETEONCLOSE != 0 {
		os.Remove(f.Name())
	}

	file := vfsFile{
		File:     f,
		psow:     true,
		readOnly: flags&OPEN_READONLY != 0,
		syncDir: runtime.GOOS != "windows" &&
			flags&(OPEN_CREATE) != 0 &&
			flags&(OPEN_MAIN_JOURNAL|OPEN_SUPER_JOURNAL|OPEN_WAL) != 0,
		shm: NewSharedMemory(name.String()+"-shm", flags),
	}
	return &file, flags, nil
}

type vfsFile struct {
	*os.File
	shm      SharedMemory
	lock     LockLevel
	readOnly bool
	keepWAL  bool
	syncDir  bool
	psow     bool
}

var (
	// Ensure these interfaces are implemented:
	_ FileLockState          = &vfsFile{}
	_ FileHasMoved           = &vfsFile{}
	_ FileSizeHint           = &vfsFile{}
	_ FilePersistentWAL      = &vfsFile{}
	_ FilePowersafeOverwrite = &vfsFile{}
)

func (f *vfsFile) Close() error {
	if f.shm != nil {
		f.shm.Close()
	}
	return f.File.Close()
}

func (f *vfsFile) Sync(flags SyncFlag) error {
	dataonly := (flags & SYNC_DATAONLY) != 0
	fullsync := (flags & 0x0f) == SYNC_FULL

	err := osSync(f.File, fullsync, dataonly)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && f.syncDir {
		f.syncDir = false
		d, err := os.Open(filepath.Dir(f.File.Name()))
		if err != nil {
			return nil
		}
		defer d.Close()
		err = osSync(d, false, false)
		if err != nil {
			return _IOERR_DIR_FSYNC
		}
	}
	return nil
}

func (f *vfsFile) Size() (int64, error) {
	return f.Seek(0, io.SeekEnd)
}

func (f *vfsFile) SectorSize() int {
	return _DEFAULT_SECTOR_SIZE
}

func (f *vfsFile) DeviceCharacteristics() DeviceCharacteristic {
	var res DeviceCharacteristic
	if osBatchAtomic(f.File) {
		res |= IOCAP_BATCH_ATOMIC
	}
	if f.psow {
		res |= IOCAP_POWERSAFE_OVERWRITE
	}
	return res
}

func (f *vfsFile) SizeHint(size int64) error {
	return osAllocate(f.File, size)
}

func (f *vfsFile) HasMoved() (bool, error) {
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	pi, err := os.Stat(f.Name())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	return !os.SameFile(fi, pi), nil
}

func (f *vfsFile) LockState() LockLevel            { return f.lock }
func (f *vfsFile) PowersafeOverwrite() bool        { return f.psow }
func (f *vfsFile) PersistentWAL() bool             { return f.keepWAL }
func (f *vfsFile) SetPowersafeOverwrite(psow bool) { f.psow = psow }
func (f *vfsFile) SetPersistentWAL(keepWAL bool)   { f.keepWAL = keepWAL }
