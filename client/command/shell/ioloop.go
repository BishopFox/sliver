package shell

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
)

// runAttachedIO forwards local stdin to a shell tunnel while attached to the shell.
// It returns:
// - detached=true when user hit escape (Ctrl-])
// - closeRequested=true when user requested shell close (EOF/exit/logout)
func runAttachedIO(tunnel *core.TunnelIO, con *console.SliverClient) (detached bool, closeRequested bool) {
	// We can't pipe stdin directly into the tunnel: if the gRPC tunnel send path blocks
	// (network hang, server gone, etc.) tunnel.Write can block forever and "lock" the
	// client in shell mode. Instead, buffer stdin locally and look for a local escape key.
	const (
		shellEscapeByte = byte(0x1d) // Ctrl-]
		stdinBufSize    = 32 * 1024
		stdinQueueDepth = 128
	)

	stdinQueue := make(chan []byte, stdinQueueDepth)
	stopWriter := make(chan struct{})
	var writerWG sync.WaitGroup
	writerWG.Add(1)

	go func() {
		defer writerWG.Done()
		defer func() {
			// Tunnel teardown can race with writes (server-side close, disconnect, etc.).
			// Don't let a send-on-closed-channel panic bring down the whole client.
			if r := recover(); r != nil {
				log.Printf("Shell tunnel writer stopped: %v", r)
			}
		}()

		for {
			select {
			case <-stopWriter:
				return
			case data, ok := <-stdinQueue:
				if !ok {
					return
				}
				if len(data) == 0 {
					continue
				}
				select {
				case <-stopWriter:
					return
				case tunnel.Send <- data:
				}
			}
		}
	}()

	defer func() {
		close(stopWriter)
		close(stdinQueue)
		writerWG.Wait()
	}()

	stdin := newFilterReader(os.Stdin)
	buf := make([]byte, stdinBufSize)
	lineBuf := make([]byte, 0, 128)
	inEscSeq := false
	log.Printf("Reading from stdin (escape=Ctrl-]) ...")

	for {
		n, err := stdin.Read(buf)
		if n > 0 {
			data := buf[:n]
			if i := bytes.IndexByte(data, shellEscapeByte); i >= 0 {
				if i > 0 {
					// Best-effort: don't block stdin if the tunnel is wedged.
					dataCopy := append([]byte(nil), data[:i]...)
					select {
					case stdinQueue <- dataCopy:
					default:
					}
				}
				detached = true
				break
			}

			exitRequested := false
			for _, b := range data {
				if inEscSeq {
					if b >= 0x40 && b <= 0x7e {
						inEscSeq = false
					}
					continue
				}

				switch b {
				case '\r', '\n':
					line := strings.TrimSpace(string(lineBuf))
					if line == "exit" || line == "logout" {
						exitRequested = true
					}
					lineBuf = lineBuf[:0]
				case 0x1b:
					inEscSeq = true
				case 0x08, 0x7f:
					if len(lineBuf) > 0 {
						lineBuf = lineBuf[:len(lineBuf)-1]
					}
				default:
					// Keep printable bytes only; ignore control sequences (arrows, etc.).
					if b >= 0x20 && b <= 0x7e {
						lineBuf = append(lineBuf, b)
					}
				}
			}

			dataCopy := append([]byte(nil), data...)
			select {
			case stdinQueue <- dataCopy:
			default:
				// Drop input if the tunnel send path is blocked; still allow escape.
			}

			if exitRequested {
				closeRequested = true
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				con.PrintErrorf("Error reading from stdin: %s\n", err)
			}
			closeRequested = true
			break
		}
	}

	log.Printf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()
	return detached, closeRequested
}
