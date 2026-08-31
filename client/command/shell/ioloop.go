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

const (
	shellEscapeByte           = byte(0x1d) // Ctrl-]
	stdinBufSize              = 32 * 1024
	stdinQueueDepth           = 128
	finalInputDeliveryTimeout = 2 * time.Second
)

type shellInputChunk struct {
	data      []byte
	delivered chan bool
}

type shellLineState struct {
	lineBuf         []byte
	inEscSeq        bool
	caseInsensitive bool
}

// runAttachedIO forwards local stdin to a shell tunnel while attached to the shell.
// It returns:
// - detached=true when user hit escape (Ctrl-])
// - closeRequested=true when user requested shell close (EOF/exit/logout)
func runAttachedIO(tunnel *core.TunnelIO, con *console.SliverClient, remoteOS string) (detached bool, closeRequested bool) {
	// We can't pipe stdin directly into the tunnel: if the gRPC tunnel send path blocks
	// (network hang, server gone, etc.) tunnel.Write can block forever and "lock" the
	// client in shell mode. Instead, buffer stdin locally and look for a local escape key.
	stdinQueue := make(chan shellInputChunk, stdinQueueDepth)
	stopWriter := make(chan struct{})
	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go runShellInputWriter(tunnel, stdinQueue, stopWriter, &writerWG)

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
	lineState := shellLineState{
		lineBuf:         make([]byte, 0, 128),
		caseInsensitive: remoteOS == "windows",
	}
	log.Printf("Reading from stdin (escape=Ctrl-]) ...")

	for {
		n, tunnelClosed, err := stdin.Read(buf, tunnel.Done())
		if tunnelClosed {
			closeRequested = true
			break
		}
		if n > 0 {
			detached, closeRequested = forwardShellInput(buf[:n], &lineState, stdinQueue, tunnel.Done())
			if detached || closeRequested {
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

func runShellInputWriter(tunnel *core.TunnelIO, stdinQueue <-chan shellInputChunk, stop <-chan struct{}, writerWG *sync.WaitGroup) {
	defer writerWG.Done()
	defer recoverShellInputWriter()

	for {
		select {
		case <-stop:
			return
		case <-tunnel.Done():
			return
		case chunk, ok := <-stdinQueue:
			if !ok || !deliverShellInputChunk(tunnel, chunk, stop) {
				return
			}
		}
	}
}

func recoverShellInputWriter() {
	// Tunnel teardown can race with writes (server-side close, disconnect, etc.).
	// Don't let a send-on-closed-channel panic bring down the whole client.
	if r := recover(); r != nil {
		log.Printf("Shell tunnel writer stopped: %v", r)
	}
}

func deliverShellInputChunk(tunnel *core.TunnelIO, chunk shellInputChunk, stop <-chan struct{}) bool {
	if len(chunk.data) == 0 {
		notifyShellInputDelivery(chunk.delivered, true)
		return true
	}

	delivered := false
	select {
	case <-stop:
	case <-tunnel.Done():
	case tunnel.Send <- chunk.data:
		delivered = true
	}
	notifyShellInputDelivery(chunk.delivered, delivered)
	return delivered
}

func notifyShellInputDelivery(delivery chan<- bool, delivered bool) {
	if delivery != nil {
		delivery <- delivered
	}
}

func forwardShellInput(data []byte, lineState *shellLineState, stdinQueue chan<- shellInputChunk, tunnelDone <-chan struct{}) (detached bool, closeRequested bool) {
	if i := bytes.IndexByte(data, shellEscapeByte); i >= 0 {
		if i > 0 {
			queueShellInputBestEffort(stdinQueue, data[:i])
		}
		return true, false
	}

	exitRequested := lineState.exitRequested(data)
	if exitRequested {
		deliverFinalShellInput(stdinQueue, data, tunnelDone)
		return false, true
	}

	queueShellInputBestEffort(stdinQueue, data)
	return false, false
}

func queueShellInputBestEffort(stdinQueue chan<- shellInputChunk, data []byte) {
	dataCopy := append([]byte(nil), data...)
	select {
	case stdinQueue <- shellInputChunk{data: dataCopy}:
	default:
		// Drop input if the tunnel send path is blocked; still allow escape.
	}
}

func deliverFinalShellInput(stdinQueue chan<- shellInputChunk, data []byte, tunnelDone <-chan struct{}) {
	delivered := make(chan bool, 1)
	chunk := shellInputChunk{
		data:      append([]byte(nil), data...),
		delivered: delivered,
	}

	queueTimer := time.NewTimer(finalInputDeliveryTimeout)
	queued := false
	select {
	case stdinQueue <- chunk:
		queued = true
	case <-tunnelDone:
	case <-queueTimer.C:
	}
	queueTimer.Stop()
	if !queued {
		return
	}

	deliveryTimer := time.NewTimer(finalInputDeliveryTimeout)
	select {
	case <-delivered:
	case <-tunnelDone:
	case <-deliveryTimer.C:
	}
	deliveryTimer.Stop()
}

func (s *shellLineState) exitRequested(data []byte) bool {
	exitRequested := false
	for _, b := range data {
		if s.inEscSeq {
			if b >= 0x40 && b <= 0x7e {
				s.inEscSeq = false
			}
			continue
		}

		switch b {
		case '\r', '\n':
			if isShellExitCommand(string(s.lineBuf), s.caseInsensitive) {
				exitRequested = true
			}
			s.lineBuf = s.lineBuf[:0]
		case 0x1b:
			s.inEscSeq = true
		case 0x08, 0x7f:
			if len(s.lineBuf) > 0 {
				s.lineBuf = s.lineBuf[:len(s.lineBuf)-1]
			}
		default:
			// Keep printable bytes only; ignore control sequences (arrows, etc.).
			if b >= 0x20 && b <= 0x7e {
				s.lineBuf = append(s.lineBuf, b)
			}
		}
	}
	return exitRequested
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
