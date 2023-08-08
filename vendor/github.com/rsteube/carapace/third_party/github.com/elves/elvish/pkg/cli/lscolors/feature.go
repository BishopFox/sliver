package lscolors

import (
	"os"
)

type feature int

const (
	featureInvalid feature = iota

	featureOrphanedSymlink
	featureSymlink

	featureMultiHardLink

	featureNamedPipe
	featureSocket
	featureDoor
	featureBlockDevice
	featureCharDevice

	featureWorldWritableStickyDirectory
	featureWorldWritableDirectory
	featureStickyDirectory
	featureDirectory

	featureCapability

	featureSetuid
	featureSetgid
	featureExecutable

	featureRegular
)

// Some platforms, such as Windows, have simulated UNIX style permission masks.
// On Windows the only two permission masks are 0o666 (RW) and 0o444 (RO).
const worldWritable = 0o002

func determineFeature(fname string, mh bool) (feature, error) {
	stat, err := os.Lstat(fname)

	if err != nil {
		return featureInvalid, err
	}

	m := stat.Mode()

	// Symlink and OrphanedSymlink has highest precedence
	if is(m, os.ModeSymlink) {
		_, err := os.Stat(fname)
		if err != nil {
			return featureOrphanedSymlink, nil
		}
		return featureSymlink, nil
	}

	// featureMultiHardLink
	if mh && isMultiHardlink(stat) {
		return featureMultiHardLink, nil
	}

	// type bits features
	switch {
	case is(m, os.ModeNamedPipe):
		return featureNamedPipe, nil
	case is(m, os.ModeSocket): // Never on Windows
		return featureSocket, nil
	case isDoor(stat):
		return featureDoor, nil
	case is(m, os.ModeCharDevice):
		return featureCharDevice, nil
	case is(m, os.ModeDevice):
		// There is no dedicated os.Mode* flag for block device. On all
		// supported Unix platforms, when os.ModeDevice is set but
		// os.ModeCharDevice is not, the file is a block device (i.e.
		// syscall.S_IFBLK is set). On Windows, this branch is unreachable.
		//
		// On Plan9, this in inaccurate.
		return featureBlockDevice, nil
	case is(m, os.ModeDir):
		// Perm bits features for directory
		perm := m.Perm()
		switch {
		case is(m, os.ModeSticky) && is(perm, worldWritable):
			return featureWorldWritableStickyDirectory, nil
		case is(perm, worldWritable):
			return featureWorldWritableDirectory, nil
		case is(m, os.ModeSticky):
			return featureStickyDirectory, nil
		default:
			return featureDirectory, nil
		}
	}

	// TODO(xiaq): Support featureCapacity

	// Perm bits features for regular files
	switch {
	case is(m, os.ModeSetuid):
		return featureSetuid, nil
	case is(m, os.ModeSetgid):
		return featureSetgid, nil
	case IsExecutable(stat):
		return featureExecutable, nil
	}

	// Check extension
	return featureRegular, nil
}

func is(m, p os.FileMode) bool {
	return m&p == p
}
