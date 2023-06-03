//go:build windows
// +build windows

package clr

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// origSTDOUT is a Windows Handle to the program's original STDOUT
var origSTDOUT = windows.Stdout

// origSTDERR is a Windows Handle to the program's original STDERR
var origSTDERR = windows.Stderr

// rSTDOUT is an io.Reader for STDOUT
var rSTDOUT *os.File

// wSTDOUT is an io.Writer for STDOUT
var wSTDOUT *os.File

// rSTDERR is an io.Reader for STDERR
var rSTDERR *os.File

// wSTDERR is an io.Writer for STDERR
var wSTDERR *os.File

// Stdout is a buffer to collect anything written to STDOUT
// The CLR will return the COR_E_TARGETINVOCATION error on subsequent Invoke_3 calls if the
// redirected STDOUT writer is EVER closed while the parent process is running (e.g., a C2 Agent)
// The redirected STDOUT reader will never recieve EOF and therefore reads will block and that is
// why a buffer is used to stored anything that has been written to STDOUT while subsequent calls block
var Stdout bytes.Buffer

// Stderr is a buffer to collect anything written to STDERR
var Stderr bytes.Buffer

// errors is used to capture an errors from a goroutine
var errors = make(chan error)

// mutex ensures exclusive access to read/write on STDOUT/STDERR by one routine at a time
var mutex = &sync.Mutex{}

// RedirectStdoutStderr redirects the program's STDOUT/STDERR to an *os.File that can be read from this Go program
// The CLR executes assemblies outside of Go and therefore STDOUT/STDERR can't be captured using normal functions
// Intended to be used with a Command & Control framework so STDOUT/STDERR can be captured and returned
func RedirectStdoutStderr() (err error) {
	// Create a new reader and writer for STDOUT
	rSTDOUT, wSTDOUT, err = os.Pipe()
	if err != nil {
		err = fmt.Errorf("there was an error calling the os.Pipe() function to create a new STDOUT:\n%s", err)
		return
	}

	// Create a new reader and writer for STDERR
	rSTDERR, wSTDERR, err = os.Pipe()
	if err != nil {
		err = fmt.Errorf("there was an error calling the os.Pipe() function to create a new STDERR:\n%s", err)
		return
	}

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")

	// Ensure the process has a console because if it doesn't there will be no output to capture
	hConsole, _, _ := getConsoleWindow.Call()
	if hConsole == 0 {
		// https://learn.microsoft.com/en-us/windows/console/allocconsole
		allocConsole := kernel32.NewProc("AllocConsole")
		// BOOL WINAPI AllocConsole(void);
		ret, _, err := allocConsole.Call()
		// A process can be associated with only one console, so the AllocConsole function fails if the calling process
		// already has a console. So long as any console exists we are good to go and therefore don't care about errors
		if ret == 0 {
			return fmt.Errorf("there was an error calling kernel32!AllocConsole with return code %d: %s", ret, err)
		}

		// Get a handle to the newly created/allocated console
		hConsole, _, _ = getConsoleWindow.Call()

		user32 := windows.NewLazySystemDLL("user32.dll")
		showWindow := user32.NewProc("ShowWindow")
		// Hide the console window
		ret, _, err = showWindow.Call(hConsole, windows.SW_HIDE)
		if err != syscall.Errno(0) {
			return fmt.Errorf("there was an error calling user32!ShowWindow with return %+v: %s", ret, err)
		}
	}

	// Set STDOUT/STDERR to the new files from os.Pipe()
	// https://docs.microsoft.com/en-us/windows/console/setstdhandle
	if err = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(wSTDOUT.Fd())); err != nil {
		err = fmt.Errorf("there was an error calling the windows.SetStdHandle function for STDOUT:\n%s", err)
		return
	}

	if err = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(wSTDERR.Fd())); err != nil {
		err = fmt.Errorf("there was an error calling the windows.SetStdHandle function for STDERR:\n%s", err)
		return
	}

	// Start STDOUT/STDERR buffer and collection
	go BufferStdout()
	go BufferStderr()

	return
}

// RestoreStdoutStderr returns the program's original STDOUT/STDERR handles before they were redirected an *os.File
// Previously instantiated CLRs will continue to use the REDIRECTED STDOUT/STDERR handles and will not resume
// using the restored handles
func RestoreStdoutStderr() error {
	if err := windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, origSTDOUT); err != nil {
		return fmt.Errorf("there was an error calling the windows.SetStdHandle function to restore the original STDOUT handle:\n%s", err)
	}
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, origSTDERR); err != nil {
		return fmt.Errorf("there was an error calling the windows.SetStdHandle function to restore the original STDERR handle:\n%s", err)
	}
	return nil
}

// ReadStdoutStderr reads from the REDIRECTED STDOUT/STDERR
// Only use when RedirectStdoutStderr was previously called
func ReadStdoutStderr() (stdout string, stderr string, err error) {
	debugPrint("Entering into io.ReadStdoutStderr()...")

	// Sleep for one Microsecond to wait for STDOUT/STDERR goroutines to finish reading
	// Race condition between reading the buffers and reading STDOUT/STDERR to the buffers
	// Can't close STDOUT/STDERR writers once the CLR invokes on assembly and EOF is not
	// returned because parent program is perpetually running
	time.Sleep(1 * time.Microsecond)

	// Check the error channel to see if any of the goroutines generated an error
	if len(errors) > 0 {
		var totalErrors string
		for e := range errors {
			totalErrors += e.Error()
		}
		err = fmt.Errorf(totalErrors)
		return
	}

	// Read STDOUT Buffer
	if Stdout.Len() > 0 {
		stdout = Stdout.String()
		Stdout.Reset()
	}

	// Read STDERR Buffer
	if Stderr.Len() > 0 {
		stderr = Stderr.String()
		Stderr.Reset()
	}
	return
}

// CloseStdoutStderr closes the Reader/Writer for the previously redirected STDOUT/STDERR
// that was changed to an *os.File
func CloseStdoutStderr() (err error) {
	err = rSTDOUT.Close()
	if err != nil {
		err = fmt.Errorf("there was an error closing the STDOUT Reader:\n%s", err)
		return
	}

	err = wSTDOUT.Close()
	if err != nil {
		err = fmt.Errorf("there was an error closing the STDOUT Writer:\n%s", err)
		return
	}

	err = rSTDERR.Close()
	if err != nil {
		err = fmt.Errorf("there was an error closing the STDERR Reader:\n%s", err)
		return
	}

	err = wSTDERR.Close()
	if err != nil {
		err = fmt.Errorf("there was an error closing the STDERR Writer:\n%s", err)
		return
	}
	return nil
}

// BufferStdout is designed to be used as a go routine to monitor for data written to the REDIRECTED STDOUT
// and collect it into a buffer so that it can be collected and sent back to a server
func BufferStdout() {
	debugPrint("Entering into io.BufferStdout()...")
	stdoutReader := bufio.NewReader(rSTDOUT)
	for {
		// Standard STDOUT buffer size is 4k
		buf := make([]byte, 4096)
		line, err := stdoutReader.Read(buf)
		if err != nil {
			errors <- fmt.Errorf("there was an error reading from STDOUT in io.BufferStdout:\n%s", err)
		}
		if line > 0 {
			// Remove null bytes and add contents to the buffer
			Stdout.Write(bytes.TrimRight(buf, "\x00"))
		}
	}
}

// BufferStderr is designed to be used as a go routine to monitor for data written to the REDIRECTED STDERR
// and collect it into a buffer so that it can be collected and sent back to a server
func BufferStderr() {
	debugPrint("Entering into io.BufferStderr()...")
	stderrReader := bufio.NewReader(rSTDERR)
	for {
		// Standard STDOUT buffer size is 4k
		buf := make([]byte, 4096)
		line, err := stderrReader.Read(buf)
		if err != nil {
			errors <- fmt.Errorf("there was an error reading from STDOUT in io.BufferStdout:\n%s", err)
		}
		if line > 0 {
			// Remove null bytes and add contents to the buffer
			Stderr.Write(bytes.TrimRight(buf, "\x00"))
		}
	}
}
