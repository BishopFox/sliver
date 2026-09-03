//go:build darwin || linux

package e2e

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"golang.org/x/sys/unix"
)

const (
	e2eSecretFDHelperEnv = "SLIVER_E2E_SECRET_FD_HELPER"
	e2eSecretFDTestValue = "runtime-only-e2e-secret"
)

//nolint:gocyclo // Parent and exec-helper branches jointly prove the descriptor inheritance boundary.
func TestReadBoundedE2EFileDescriptorDoesNotLeakSourceToChild(t *testing.T) {
	if rawDescriptor := os.Getenv(e2eSecretFDHelperEnv); rawDescriptor != "" {
		descriptor, err := strconv.Atoi(rawDescriptor)
		if err != nil {
			t.Fatalf("parse helper descriptor: %v", err)
		}
		if _, err := unix.Seek(descriptor, 0, io.SeekStart); err != nil {
			if errors.Is(err, unix.EBADF) {
				return
			}
			// A descriptor reused for an unrelated non-seekable runtime resource
			// cannot expose the seekable secret file supplied by the parent.
			return
		}
		payload := make([]byte, len(e2eSecretFDTestValue))
		read, _ := unix.Read(descriptor, payload)
		if string(payload[:read]) == e2eSecretFDTestValue {
			t.Fatalf("runtime-only source descriptor %d survived exec", descriptor)
		}
		return
	}

	secretFile, err := os.CreateTemp(t.TempDir(), "sliver-e2e-secret-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = secretFile.Close() }()
	if _, err := secretFile.WriteString(e2eSecretFDTestValue); err != nil {
		t.Fatal(err)
	}
	if _, err := secretFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	descriptor := int(secretFile.Fd())
	flags, err := unix.FcntlInt(secretFile.Fd(), unix.F_GETFD, 0)
	if err != nil {
		t.Fatal(err)
	}
	// os.CreateTemp opens close-on-exec. Clear the bit to model a descriptor
	// inherited from the manual shell invocation documented for this suite.
	if _, err := unix.FcntlInt(secretFile.Fd(), unix.F_SETFD, flags&^unix.FD_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	payload, err := readBoundedE2EFileDescriptor(context.Background(), descriptor, len(e2eSecretFDTestValue)+1)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != e2eSecretFDTestValue {
		t.Fatalf("descriptor payload = %q, want %q", payload, e2eSecretFDTestValue)
	}
	flags, err = unix.FcntlInt(secretFile.Fd(), unix.F_GETFD, 0)
	if err != nil {
		t.Fatal(err)
	}
	if flags&unix.FD_CLOEXEC == 0 {
		t.Fatal("source descriptor was not marked close-on-exec")
	}

	command := exec.Command(os.Args[0], "-test.run=^TestReadBoundedE2EFileDescriptorDoesNotLeakSourceToChild$")
	command.Env = envWith(os.Environ(), e2eSecretFDHelperEnv, strconv.Itoa(descriptor))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("descriptor inheritance helper: %v\n%s", err, output)
	}
}
