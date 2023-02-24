// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build dummy

// This file exists purely to prevent the Go toolchain from stripping
// away the C source directories and files when `go mod vendor` is used
// to populate a `vendor/` directory of a project depending on this package.

package purego

import (
	_ "github.com/ebitengine/purego/internal/abi"
)
