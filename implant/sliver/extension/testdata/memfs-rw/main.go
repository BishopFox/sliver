package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	memFSRoot     = "/memfs/wasi-integration"
	createdPath   = memFSRoot + "/created.txt"
	persistedPath = memFSRoot + "/persisted.txt"
	removedPath   = memFSRoot + "/removed.txt"
	removedDir    = memFSRoot + "/removed-dir"
	seedPath      = "/memfs/seed.txt"
	blockedPath   = "/memfs/blocked.txt"

	persistedContents = "created-appended"
)

func main() {
	if len(os.Args) != 2 {
		fail("usage: memfs-rw <write|read|readonly>")
	}

	var err error
	switch os.Args[1] {
	case "write":
		err = writeAndMutate()
	case "read":
		err = readPersistedState()
	case "readonly":
		err = verifyReadOnly()
	default:
		err = fmt.Errorf("unknown operation %q", os.Args[1])
	}
	if err != nil {
		fail(err.Error())
	}
}

func writeAndMutate() error {
	if err := os.Mkdir(memFSRoot, 0o750); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	file, err := os.OpenFile(createdPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o640)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if _, err = file.WriteString("created"); err != nil {
		_ = file.Close()
		return fmt.Errorf("write: %w", err)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return fmt.Errorf("seek: %w", err)
	}
	contents := make([]byte, len("created"))
	if _, err = io.ReadFull(file, contents); err != nil {
		_ = file.Close()
		return fmt.Errorf("read after seek: %w", err)
	}
	if !bytes.Equal(contents, []byte("created")) {
		_ = file.Close()
		return fmt.Errorf("read after seek = %q", contents)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close created file: %w", err)
	}

	file, err = os.OpenFile(createdPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("open append: %w", err)
	}
	if _, err = file.WriteString("-appended"); err != nil {
		_ = file.Close()
		return fmt.Errorf("append: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close appended file: %w", err)
	}

	if err = os.Rename(createdPath, persistedPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	if err = expectFileContents(persistedPath, persistedContents); err != nil {
		return err
	}

	if err = os.WriteFile(removedPath, []byte("remove-me"), 0o600); err != nil {
		return fmt.Errorf("create removable file: %w", err)
	}
	if err = os.Remove(removedPath); err != nil {
		return fmt.Errorf("remove file: %w", err)
	}
	if _, err = os.Stat(removedPath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat removed file: got %v, want not-exist", err)
	}

	if err = os.Mkdir(removedDir, 0o700); err != nil {
		return fmt.Errorf("create removable directory: %w", err)
	}
	if err = os.Remove(removedDir); err != nil {
		return fmt.Errorf("remove directory: %w", err)
	}
	if _, err = os.Stat(removedDir); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat removed directory: got %v, want not-exist", err)
	}

	fmt.Println("memfs-rw-write-ok")
	return nil
}

func readPersistedState() error {
	if err := expectFileContents(persistedPath, persistedContents); err != nil {
		return fmt.Errorf("read state from prior execution: %w", err)
	}
	fmt.Println("memfs-rw-state-ok")
	return nil
}

func verifyReadOnly() error {
	if err := expectFileContents(seedPath, "seed-data"); err != nil {
		return fmt.Errorf("read seed: %w", err)
	}
	if err := os.WriteFile(blockedPath, []byte("blocked"), 0o600); err == nil {
		return errors.New("creating a file on read-only memfs unexpectedly succeeded")
	}
	if _, err := os.Stat(blockedPath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat blocked file: got %v, want not-exist", err)
	}
	if file, err := os.OpenFile(seedPath, os.O_WRONLY|os.O_TRUNC, 0); err == nil {
		_ = file.Close()
		return errors.New("truncating a file on read-only memfs unexpectedly succeeded")
	}
	if err := expectFileContents(seedPath, "seed-data"); err != nil {
		return fmt.Errorf("seed changed after rejected write: %w", err)
	}
	fmt.Println("memfs-readonly-ok")
	return nil
}

func expectFileContents(name, expected string) error {
	contents, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if string(contents) != expected {
		return fmt.Errorf("read %s = %q, want %q", name, contents, expected)
	}
	return nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
