package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
)

func main() {
	switch os.Args[1] {
	case "ls":
		var repeat bool
		if len(os.Args) == 4 {
			repeat = os.Args[3] == "repeat"
		}
		// Go doesn't open with O_DIRECTORY, so we don't end up with ENOTDIR,
		// rather EBADF trying to read the directory later.
		if err := mainLs(os.Args[2], repeat); errors.Is(err, syscall.EBADF) {
			fmt.Println("ENOTDIR")
		} else if err != nil {
			panic(err)
		}
	case "stat":
		if err := mainStat(); err != nil {
			panic(err)
		}
	case "sock":
		// TODO: undefined: net.FileListener
		// See https://github.com/tinygo-org/tinygo/pull/2748
	case "nonblock":
		// TODO: undefined: syscall.SetNonblock
		// See https://github.com/tinygo-org/tinygo/issues/3840
	}
}

func mainLs(path string, repeat bool) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()

	if err = printFileNames(d); err != nil {
		return err
	} else if repeat {
		// rewind
		if _, err = d.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return printFileNames(d)
	}
	return nil
}

func printFileNames(d *os.File) error {
	if names, err := d.Readdirnames(-1); err != nil {
		return err
	} else {
		for _, n := range names {
			fmt.Println("./" + n)
		}
	}
	return nil
}

func mainStat() error {
	var isatty = func(name string, fd uintptr) error {
		f := os.NewFile(fd, "")
		if st, err := f.Stat(); err != nil {
			return err
		} else {
			ttyMode := fs.ModeDevice | fs.ModeCharDevice
			isatty := st.Mode()&ttyMode == ttyMode
			fmt.Println(name, "isatty:", isatty)
			return nil
		}
	}

	for fd, name := range []string{"stdin", "stdout", "stderr", "/"} {
		if err := isatty(name, uintptr(fd)); err != nil {
			return err
		}
	}
	return nil
}
