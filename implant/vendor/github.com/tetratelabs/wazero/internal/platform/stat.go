package platform

import "os"

// StatTimes returns platform-specific values if os.FileInfo Sys is available.
// Otherwise, it returns the mod time for all values.
func StatTimes(t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64) {
	if t.Sys() == nil { // possibly fake filesystem
		return mtimes(t)
	}
	return statTimes(t)
}

func mtimes(t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64) {
	mtimeNsec = t.ModTime().UnixNano()
	atimeNsec = mtimeNsec
	ctimeNsec = mtimeNsec
	return
}
