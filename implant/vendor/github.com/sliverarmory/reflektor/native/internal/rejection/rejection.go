// Package rejection owns native-loader errors shared by the public package and
// platform backends without introducing a dependency on Reflektor's root or
// memmod packages.
package rejection

import "errors"

// ErrGoSharedLibraryUnsupported reports that the native-only loader was asked
// to load an image containing a Go runtime.
var ErrGoSharedLibraryUnsupported = errors.New("reflektor/native: Go c-shared images are unsupported; use package reflektor")
