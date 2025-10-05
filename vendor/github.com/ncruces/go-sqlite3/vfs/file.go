package vfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

type vfsOS struct{}

func (vfsOS) FullPathname(path string) (string, error) {
	link, err := evalSymlinks(path)
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(link)
	if err == nil && link != path {
		err = _OK_SYMLINK
	}
	return full, err
}

func evalSymlinks(path string) (string, error) {
	var file string
	_, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		path, file = filepath.Split(path)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, file), nil
}

func (vfsOS) Delete(path string, syncDir bool) error {
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return _IOERR_DELETE_NOENT
	}
	if err != nil {
		return err
	}
	if isUnix && syncDir {
		f, err := os.Open(filepath.Dir(path))
		if err != nil {
			return _OK
		}
		defer f.Close()
		err = osSync(f, 0, SYNC_FULL)
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
	// notest // OpenFilename is called instead
	if name == "" {
		return vfsOS{}.OpenFilename(nil, flags)
	}
	return nil, 0, _CANTOPEN
}

func (vfsOS) OpenFilename(name *Filename, flags OpenFlag) (File, OpenFlag, error) {
	oflags := _O_NOFOLLOW
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

	isCreate := flags&(OPEN_CREATE) != 0
	isJournl := flags&(OPEN_MAIN_JOURNAL|OPEN_SUPER_JOURNAL|OPEN_WAL) != 0

	var err error
	var f *os.File
	if name == nil {
		f, err = os.CreateTemp(os.Getenv("SQLITE_TMPDIR"), "*.db")
	} else {
		f, err = os.OpenFile(name.String(), oflags, 0666)
	}
	if err != nil {
		if name == nil {
			return nil, flags, _IOERR_GETTEMPPATH
		}
		if errors.Is(err, syscall.EISDIR) {
			return nil, flags, _CANTOPEN_ISDIR
		}
		if isCreate && isJournl && errors.Is(err, fs.ErrPermission) &&
			osAccess(name.String(), ACCESS_EXISTS) != nil {
			return nil, flags, _READONLY_DIRECTORY
		}
		return nil, flags, err
	}

	if modeof := name.URIParameter("modeof"); modeof != "" {
		if err = osSetMode(f, modeof); err != nil {
			f.Close()
			return nil, flags, _IOERR_FSTAT
		}
	}
	if isUnix && flags&OPEN_DELETEONCLOSE != 0 {
		os.Remove(f.Name())
	}

	file := vfsFile{
		File:  f,
		flags: flags | _FLAG_PSOW,
		shm:   NewSharedMemory(name.String()+"-shm", flags),
	}
	if osBatchAtomic(f) {
		file.flags |= _FLAG_ATOMIC
	}
	if isUnix && isCreate && isJournl {
		file.flags |= _FLAG_SYNC_DIR
	}
	return &file, flags, nil
}

type vfsFile struct {
	*os.File
	shm   SharedMemory
	lock  LockLevel
	flags OpenFlag
}

var (
	// Ensure these interfaces are implemented:
	_ FileLockState          = &vfsFile{}
	_ FileHasMoved           = &vfsFile{}
	_ FileSizeHint           = &vfsFile{}
	_ FilePersistWAL         = &vfsFile{}
	_ FilePowersafeOverwrite = &vfsFile{}
)

func (f *vfsFile) Close() error {
	if !isUnix && f.flags&OPEN_DELETEONCLOSE != 0 {
		defer os.Remove(f.Name())
	}
	if f.shm != nil {
		f.shm.Close()
	}
	f.Unlock(LOCK_NONE)
	return f.File.Close()
}

func (f *vfsFile) ReadAt(p []byte, off int64) (n int, err error) {
	return osReadAt(f.File, p, off)
}

func (f *vfsFile) WriteAt(p []byte, off int64) (n int, err error) {
	return osWriteAt(f.File, p, off)
}

func (f *vfsFile) Sync(flags SyncFlag) error {
	err := osSync(f.File, f.flags, flags)
	if err != nil {
		return err
	}
	if isUnix && f.flags&_FLAG_SYNC_DIR != 0 {
		f.flags ^= _FLAG_SYNC_DIR
		d, err := os.Open(filepath.Dir(f.File.Name()))
		if err != nil {
			return nil
		}
		defer d.Close()
		err = osSync(f.File, f.flags, flags)
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
	ret := IOCAP_SUBPAGE_READ
	if f.flags&_FLAG_ATOMIC != 0 {
		ret |= IOCAP_BATCH_ATOMIC
	}
	if f.flags&_FLAG_PSOW != 0 {
		ret |= IOCAP_POWERSAFE_OVERWRITE
	}
	if runtime.GOOS == "windows" {
		ret |= IOCAP_UNDELETABLE_WHEN_OPEN
	}
	return ret
}

func (f *vfsFile) SizeHint(size int64) error {
	return osAllocate(f.File, size)
}

func (f *vfsFile) HasMoved() (bool, error) {
	if runtime.GOOS == "windows" {
		return false, nil
	}
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	pi, err := os.Stat(f.Name())
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return !os.SameFile(fi, pi), nil
}

func (f *vfsFile) LockState() LockLevel     { return f.lock }
func (f *vfsFile) PowersafeOverwrite() bool { return f.flags&_FLAG_PSOW != 0 }
func (f *vfsFile) PersistWAL() bool         { return f.flags&_FLAG_KEEP_WAL != 0 }

func (f *vfsFile) SetPowersafeOverwrite(psow bool) {
	f.flags &^= _FLAG_PSOW
	if psow {
		f.flags |= _FLAG_PSOW
	}
}

func (f *vfsFile) SetPersistWAL(keepWAL bool) {
	f.flags &^= _FLAG_KEEP_WAL
	if keepWAL {
		f.flags |= _FLAG_KEEP_WAL
	}
}
