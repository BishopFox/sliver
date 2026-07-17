#!/bin/bash

set -e

cleanup() {
  [ ! -z "$1" ] && echo "Error $1 occurred on $2"

  mv "${tempDir}/vendor" "${pwd}/../vendor"
  mv "${tempDir}/go.mod" "${pwd}/../go-mod"
  mv "${tempDir}/go.sum" "${pwd}/../go-sum"
  cd ..
  rm -rf "$tempDir"
}

# we are expecting from run from sliver/implant via 'go generate'
cd scripts

# copy Go module related files
pwd="$(pwd)"
tempDir="$(mktemp -d)"
cp ../go-mod "${tempDir}/go.mod"
cp ../go-sum "${tempDir}/go.sum"
mv ../vendor "${tempDir}/vendor"


# Trap when a build fails so we can reset the environment
trap 'cleanup $? $LINENO' ERR

# build Go file with all imported packages
go run update-vendor.go "$tempDir"
cd "$tempDir"
# update vendor dir
# go get gvisor.dev/gvisor/runsc@go
go mod tidy -compat=1.25
go mod vendor

# The lesnuages/go-winio fork provides RemoteClientMode, which Sliver needs,
# but its checked-in syscall wrappers predate syscall.SyscallN. Regenerate the
# wrappers with the pinned x/sys generator so Go 1.26 Windows/ARM64 debug builds
# do not link the oversized syscall.Syscall15 wrapper.
(
  cd vendor/github.com/lesnuages/go-winio
  go run -mod=mod golang.org/x/sys/windows/mkwinsyscall \
    -output zsyscall_windows.go \
    file.go pipe.go sd.go fileinfo.go privilege.go backup.go hvsock.go
)

# move updated files back
cleanup
