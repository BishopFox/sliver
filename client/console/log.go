package console

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/moloch--/asciicast"
	"golang.org/x/exp/slog"
	"golang.org/x/term"
)

const (
	stdoutSyncMarkerSize = 16
	stdoutSyncWait       = 2 * time.Second
)

// ConsoleClientLogger is an io.Writer that sends data to the server.
type ConsoleClientLogger struct {
	name   string
	Stream rpcpb.SliverRPC_ClientLogClient
}

func (l *ConsoleClientLogger) Write(buf []byte) (int, error) {
	err := l.Stream.Send(&clientpb.ClientLogData{
		Stream: l.name,
		Data:   buf,
	})
	return len(buf), err
}

// optionalRemoteWriter is an io.Writer that forwards to an optional remote writer.
// It always returns len(buf), nil to avoid breaking io.MultiWriter/io.Copy when
// a remote stream disconnects during a server switch.
type optionalRemoteWriter struct {
	mu sync.RWMutex
	w  io.Writer
}

func (o *optionalRemoteWriter) Set(w io.Writer) {
	o.mu.Lock()
	o.w = w
	o.mu.Unlock()
}

func (o *optionalRemoteWriter) Write(buf []byte) (int, error) {
	o.mu.RLock()
	w := o.w
	o.mu.RUnlock()
	if w != nil {
		_, _ = w.Write(buf)
	}
	return len(buf), nil
}

func (con *SliverClient) ensureJSONRemoteWriter() {
	con.connMu.Lock()
	defer con.connMu.Unlock()

	if con.jsonRemoteWriter == nil {
		con.jsonRemoteWriter = &optionalRemoteWriter{}
	}
}

func (con *SliverClient) ensureAsciicastRemoteWriter() {
	con.connMu.Lock()
	defer con.connMu.Unlock()

	if con.asciicastRemoteWriter == nil {
		con.asciicastRemoteWriter = &optionalRemoteWriter{}
	}
}

func (con *SliverClient) refreshRemoteLogStreamsLocked() {
	// Only refresh if the console logging stack has been initialized.
	if con.jsonRemoteWriter == nil && con.asciicastRemoteWriter == nil {
		return
	}
	if con.Rpc == nil {
		con.setRemoteLogStreamsLocked(nil, nil)
		return
	}

	var jsonStream *ConsoleClientLogger
	var asciicastStream *ConsoleClientLogger

	if con.jsonRemoteWriter != nil {
		s, err := con.ClientLogStream("json")
		if err != nil {
			log.Printf("Could not get client json log stream: %s", err)
		} else {
			jsonStream = s
		}
	}
	if con.asciicastRemoteWriter != nil {
		s, err := con.ClientLogStream("asciicast")
		if err != nil {
			log.Printf("Could not get client asciicast log stream: %s", err)
		} else {
			asciicastStream = s
		}
	}

	con.setRemoteLogStreamsLocked(jsonStream, asciicastStream)
}

func (con *SliverClient) setRemoteLogStreamsLocked(jsonStream, asciicastStream *ConsoleClientLogger) {
	// Detach writers first so background pipes can't hit a closing stream.
	if con.jsonRemoteWriter != nil {
		con.jsonRemoteWriter.Set(nil)
	}
	if con.asciicastRemoteWriter != nil {
		con.asciicastRemoteWriter.Set(nil)
	}

	if con.jsonRemoteStream != nil {
		_ = con.jsonRemoteStream.CloseSend()
		con.jsonRemoteStream = nil
	}
	if con.asciicastRemoteStream != nil {
		_ = con.asciicastRemoteStream.CloseSend()
		con.asciicastRemoteStream = nil
	}

	if jsonStream != nil && con.jsonRemoteWriter != nil {
		con.jsonRemoteWriter.Set(jsonStream)
		con.jsonRemoteStream = jsonStream.Stream
	}
	if asciicastStream != nil && con.asciicastRemoteWriter != nil {
		con.asciicastRemoteWriter.Set(asciicastStream)
		con.asciicastRemoteStream = asciicastStream.Stream
	}
}

// ClientLogStream requires a log stream name, used to save the logs
// going through this stream in a specific log subdirectory/file.
func (con *SliverClient) ClientLogStream(name string) (*ConsoleClientLogger, error) {
	stream, err := con.Rpc.ClientLog(context.Background())
	if err != nil {
		return nil, err
	}
	return &ConsoleClientLogger{name: name, Stream: stream}, nil
}

func (con *SliverClient) setupLogger(writers ...io.Writer) {
	logWriter := io.MultiWriter(writers...)
	jsonOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	con.jsonHandler = slog.NewJSONHandler(logWriter, jsonOptions)

	// Log all commands before running them.
	con.connMu.Lock()
	defer con.connMu.Unlock()
	if !con.logCommandHookApplied {
		con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, con.logCommand)
		con.logCommandHookApplied = true
	}
}

// logCommand logs non empty commands to the client log file.
func (con *SliverClient) logCommand(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	logger := slog.New(con.jsonHandler).With(slog.String("type", "command"))

	// Attach active target context if available
	if session := con.ActiveTarget.GetSession(); session != nil {
		logger = logger.With(
			slog.String("id", session.ID),
			slog.String("name", session.Name),
			slog.String("hostname", session.Hostname),
			slog.String("username", session.Username),
		)
	} else if beacon := con.ActiveTarget.GetBeacon(); beacon != nil {
		logger = logger.With(
			slog.String("id", beacon.ID),
			slog.String("name", beacon.Name),
			slog.String("hostname", beacon.Hostname),
			slog.String("username", beacon.Username),
		)
	}

	logger.Debug(strings.Join(args, " "))
	return args, nil
}

func (con *SliverClient) setupAsciicastRecord(logFile *os.File, server io.Writer) {
	x, y, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		x, y = 80, 80
	}

	// Save the asciicast to the server and a local file.
	destinations := io.MultiWriter(logFile, server)

	encoder := asciicast.NewEncoder(destinations, x, y)
	encoder.WriteHeader()

	// save existing stdout | MultiWriter writes to saved stdout and file
	out := os.Stdout
	mw := io.MultiWriter(out, encoder)

	// get pipe reader and writer | writes to pipe writer come out pipe reader
	r, w, _ := os.Pipe()

	// replace stdout,stderr with pipe writer | all writes to stdout,
	// stderr will go through pipe instead (fmt.print, log)
	os.Stdout = w
	os.Stderr = w

	marker := make([]byte, stdoutSyncMarkerSize)
	if _, err := rand.Read(marker); err != nil {
		copy(marker, []byte(fmt.Sprintf("sliver-sync-%d", time.Now().UnixNano())))
	}

	done := make(chan struct{})
	con.stdoutPipeWriter = w
	con.stdoutPipeDone = done
	con.stdoutSyncMarker = marker
	con.stdoutSyncAcks = map[uint64]chan struct{}{}

	go con.copyStdoutPipe(r, mw, done)
}

func buildStdoutSyncFrame(marker []byte, seq uint64) []byte {
	frame := make([]byte, len(marker)*2+8)
	copy(frame, marker)
	binary.BigEndian.PutUint64(frame[len(marker):], seq)
	copy(frame[len(marker)+8:], marker)
	return frame
}

func pendingMarkerPrefixLen(pending []byte, marker []byte) int {
	max := len(marker) - 1
	if max > len(pending) {
		max = len(pending)
	}
	for size := max; size > 0; size-- {
		if bytes.Equal(pending[len(pending)-size:], marker[:size]) {
			return size
		}
	}
	return 0
}

func drainStdoutPipeBuffer(dst io.Writer, pending []byte, marker []byte, ack func(uint64)) []byte {
	if len(pending) == 0 {
		return pending
	}
	if len(marker) == 0 {
		_, _ = dst.Write(pending)
		return pending[:0]
	}

	frameLen := len(marker)*2 + 8
	for {
		idx := bytes.Index(pending, marker)
		if idx == -1 {
			keep := pendingMarkerPrefixLen(pending, marker)
			flush := len(pending) - keep
			if flush > 0 {
				_, _ = dst.Write(pending[:flush])
				pending = pending[flush:]
			}
			return pending
		}

		if idx > 0 {
			_, _ = dst.Write(pending[:idx])
			pending = pending[idx:]
		}
		if len(pending) < frameLen {
			return pending
		}
		if bytes.Equal(pending[len(marker)+8:frameLen], marker) {
			if ack != nil {
				ack(binary.BigEndian.Uint64(pending[len(marker) : len(marker)+8]))
			}
			pending = pending[frameLen:]
			continue
		}

		_, _ = dst.Write(pending[:1])
		pending = pending[1:]
	}
}

func (con *SliverClient) copyStdoutPipe(src *os.File, dst io.Writer, done chan struct{}) {
	defer close(done)
	defer src.Close()

	readBuf := make([]byte, 4096)
	pending := make([]byte, 0, 4096)
	for {
		n, err := src.Read(readBuf)
		if n > 0 {
			pending = append(pending, readBuf[:n]...)
			pending = drainStdoutPipeBuffer(dst, pending, con.stdoutSyncMarker, con.ackStdoutSync)
		}
		if err != nil {
			break
		}
	}
	if len(pending) > 0 {
		_, _ = dst.Write(pending)
	}
}

func (con *SliverClient) ackStdoutSync(seq uint64) {
	con.stdoutSyncAcksMu.Lock()
	ack := con.stdoutSyncAcks[seq]
	delete(con.stdoutSyncAcks, seq)
	con.stdoutSyncAcksMu.Unlock()

	if ack != nil {
		close(ack)
	}
}

func (con *SliverClient) clearStdoutSync(seq uint64) {
	con.stdoutSyncAcksMu.Lock()
	delete(con.stdoutSyncAcks, seq)
	con.stdoutSyncAcksMu.Unlock()
}

func (con *SliverClient) syncOutputHook() error {
	con.syncOutput()
	return nil
}

func (con *SliverClient) syncOutput() {
	if con.stdoutPipeWriter == nil {
		_ = os.Stdout.Sync()
		return
	}
	if len(con.stdoutSyncMarker) == 0 {
		return
	}

	con.stdoutSyncMu.Lock()
	con.stdoutSyncSeq++
	seq := con.stdoutSyncSeq
	ack := make(chan struct{})
	con.stdoutSyncAcksMu.Lock()
	if con.stdoutSyncAcks == nil {
		con.stdoutSyncAcks = map[uint64]chan struct{}{}
	}
	con.stdoutSyncAcks[seq] = ack
	con.stdoutSyncAcksMu.Unlock()
	frame := buildStdoutSyncFrame(con.stdoutSyncMarker, seq)
	_, err := con.stdoutPipeWriter.Write(frame)
	con.stdoutSyncMu.Unlock()
	if err != nil {
		con.clearStdoutSync(seq)
		return
	}

	select {
	case <-ack:
	case <-time.After(stdoutSyncWait):
		con.clearStdoutSync(seq)
	}
}

func getConsoleLogFile() *os.File {
	logsDir := assets.GetConsoleLogsDir()
	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("%s.log", dateTime))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalf("Could not open log file: %s", err)
	}
	return logFile
}

func getConsoleAsciicastFile() *os.File {
	logsDir := assets.GetConsoleLogsDir()
	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("asciicast_%s.log", dateTime))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalf("Could not open log file: %s", err)
	}
	return logFile
}

// FlushOutput drains any piped stdout before exiting.
func (con *SliverClient) FlushOutput() {
	if con.stdoutPipeWriter == nil {
		_ = os.Stdout.Sync()
		return
	}

	con.stdoutPipeOnce.Do(func() {
		_ = con.stdoutPipeWriter.Close()
	})

	if con.stdoutPipeDone != nil {
		select {
		case <-con.stdoutPipeDone:
		case <-time.After(500 * time.Millisecond):
		}
	}
}

//
// -------------------------- [ Logging ] -----------------------------
//
// Logging function below differ slightly from their counterparts in client/log package:
// These below will print their output regardless of the currently active menu (server/implant),
// while those in the log package tie their output to the current menu.

// PrintAsyncResponse - Print the generic async response information.
func (con *SliverClient) PrintAsyncResponse(resp *commonpb.Response) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	beacon, err := con.Rpc.GetBeacon(ctx, &clientpb.Beacon{ID: resp.BeaconID})
	if err != nil {
		con.PrintWarnf("%s", err.Error())
		return
	}
	con.PrintInfof("Tasked beacon %s (%s)\n", beacon.Name, strings.Split(resp.TaskID, "-")[0])
}

func (con *SliverClient) Printf(format string, args ...any) {
	logger := slog.NewLogLogger(con.jsonHandler, slog.LevelInfo)
	logger.Printf(format, args...)

	con.printf(format, args...)
}

// Println prints an output without status and immediately below the last line of output.
func (con *SliverClient) Println(args ...any) {
	logger := slog.New(con.jsonHandler)
	format := strings.Repeat("%s", len(args))
	logger.Info(fmt.Sprintf(format, args))
	con.printf(format+"\n", args...)
}

// PrintInfof prints an info message immediately below the last line of output.
func (con *SliverClient) PrintInfof(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+Info+format, args...)
}

// PrintSuccessf prints a success message immediately below the last line of output.
func (con *SliverClient) PrintSuccessf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+Success+format, args...)
}

// PrintSuccess prints a success message with a default or provided message.
func (con *SliverClient) PrintSuccess(args ...any) {
	if len(args) == 0 {
		con.PrintSuccessf("Success")
		return
	}
	con.PrintSuccessf("%s", fmt.Sprint(args...))
}

// PrintWarnf a warning message immediately below the last line of output.
func (con *SliverClient) PrintWarnf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Warn(fmt.Sprintf(format, args...))

	con.printf(Clearln+"⚠️  "+format, args...)
}

// PrintErrorf prints an error message immediately below the last line of output.
func (con *SliverClient) PrintErrorf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Error(fmt.Sprintf(format, args...))

	con.printf(Clearln+Warn+format, args...)
}

// PrintEventInfof prints an info message with a leading/trailing newline for emphasis.
func (con *SliverClient) PrintEventInfof(format string, args ...any) {
	logger := slog.New(con.jsonHandler).With(slog.String("type", "event"))

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+"\r\n"+Info+format+"\r", args...)
}

// PrintEventErrorf prints an error message with a leading/trailing newline for emphasis.
func (con *SliverClient) PrintEventErrorf(format string, args ...any) {
	logger := slog.New(con.jsonHandler).With(slog.String("type", "event"))

	logger.Error(fmt.Sprintf(format, args...))

	con.printf(Clearln+"\r\n"+Warn+format+"\r", args...)
}

// PrintEventSuccessf a success message with a leading/trailing newline for emphasis.
func (con *SliverClient) PrintEventSuccessf(format string, args ...any) {
	logger := slog.New(con.jsonHandler).With(slog.String("type", "event"))

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+"\r\n"+Success+format+"\r", args...)
}
