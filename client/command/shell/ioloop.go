package shell

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
)

// runAttachedIO forwards local stdin to a shell tunnel while attached to the shell.
// It returns:
// - detached=true when user hit escape (Ctrl-])
// - closeRequested=true when user requested shell close (EOF/exit/logout)
func runAttachedIO(tunnel *core.TunnelIO, con *console.SliverClient, remoteOS string) (detached bool, closeRequested bool) {
	// We can't pipe stdin directly into the tunnel: if the gRPC tunnel send path blocks
	// (network hang, server gone, etc.) tunnel.Write can block forever and "lock" the
	// client in shell mode. Instead, buffer stdin locally and look for a local escape key.
	const (
		shellEscapeByte           = byte(0x1d) // Ctrl-]
		stdinBufSize              = 32 * 1024
		stdinQueueDepth           = 128
		finalInputDeliveryTimeout = 2 * time.Second
	)

	type inputChunk struct {
		data      []byte
		delivered chan bool
	}
	stdinQueue := make(chan inputChunk, stdinQueueDepth)
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
			case <-tunnel.Done():
				return
			case chunk, ok := <-stdinQueue:
				if !ok {
					return
				}
				if len(chunk.data) == 0 {
					if chunk.delivered != nil {
						chunk.delivered <- true
					}
					continue
				}
				delivered := false
				select {
				case <-stopWriter:
				case <-tunnel.Done():
				case tunnel.Send <- chunk.data:
					delivered = true
				}
				if chunk.delivered != nil {
					chunk.delivered <- delivered
				}
				if !delivered {
					return
				}
			}
		}
	}()

	defer func() {
		close(stopWriter)
		close(stdinQueue)
		writerWG.Wait()
	}()

	stdin, err := newCancellableShellInput(os.Stdin, newFilterReader(os.Stdin))
	if err != nil {
		con.PrintErrorf("Failed to prepare shell input: %s\n", err)
		return false, true
	}
	defer func() {
		if err := stdin.Close(); err != nil {
			log.Printf("Failed to restore shell input state: %v", err)
		}
	}()

	buf := make([]byte, stdinBufSize)
	lineBuf := make([]byte, 0, 128)
	inEscSeq := false
	log.Printf("Reading from stdin (escape=Ctrl-]) ...")

	for {
		n, err, tunnelClosed := stdin.Read(buf, tunnel.Done())
		if tunnelClosed {
			closeRequested = true
			break
		}
		if n > 0 {
			data := buf[:n]
			if i := bytes.IndexByte(data, shellEscapeByte); i >= 0 {
				if i > 0 {
					// Best-effort: don't block stdin if the tunnel is wedged.
					dataCopy := append([]byte(nil), data[:i]...)
					select {
					case stdinQueue <- inputChunk{data: dataCopy}:
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
					if isShellExitCommand(string(lineBuf), remoteOS == "windows") {
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
			var delivered chan bool
			if exitRequested {
				delivered = make(chan bool, 1)
			}
			chunk := inputChunk{data: dataCopy, delivered: delivered}
			queued := false
			if exitRequested {
				timer := time.NewTimer(finalInputDeliveryTimeout)
				select {
				case stdinQueue <- chunk:
					queued = true
				case <-tunnel.Done():
				case <-timer.C:
				}
				timer.Stop()
			} else {
				select {
				case stdinQueue <- chunk:
				default:
					// Drop input if the tunnel send path is blocked; still allow escape.
				}
			}

			if exitRequested {
				if queued {
					timer := time.NewTimer(finalInputDeliveryTimeout)
					select {
					case <-delivered:
					case <-tunnel.Done():
					case <-timer.C:
					}
					timer.Stop()
				}
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

func isShellExitCommand(line string, caseInsensitive bool) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}

	command := fields[0]
	if caseInsensitive {
		command = strings.ToLower(command)
	}
	switch command {
	case "exit", "logout":
		return true
	default:
		return false
	}
}
