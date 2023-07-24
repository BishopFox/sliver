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
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/moloch--/asciicast"
	"golang.org/x/exp/slog"
	"golang.org/x/term"
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

func (con *SliverClient) startClientLog() error {
	if con.IsCompleting {
		return nil
	}

	if !con.Settings.ConsoleLogs {
		return nil
	}

	// Classic logs.
	clientLogFile := getConsoleLogFile()

	clientLogs, err := con.ClientLogStream("json")
	if err != nil {
		return fmt.Errorf("Could not get client log stream: %w", err)
	}

	con.setupLogger(clientLogFile, clientLogs)

	// Asciicast sessions.
	asciicastFile := getConsoleAsciicastFile()

	asciicastStream, err := con.ClientLogStream("asciicast")
	if err != nil {
		return fmt.Errorf("Could not get client log stream: %w", err)
	}

	err = con.setupAsciicastRecord(asciicastFile, asciicastStream)

	con.closeLogs = func() {
		// Local files
		clientLogFile.Close()
		asciicastFile.Close()

		// Server streams.
		clientLogs.Stream.CloseAndRecv()
		asciicastStream.Stream.CloseAndRecv()
	}

	return nil
}

func (con *SliverClient) CloseClientLog() {
	if con.closeLogs == nil {
		return
	}

	defer func() {
		con.closeLogs = nil
	}()

	con.closeLogs()
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
	con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, con.logCommand)
}

// logCommand logs non empty commands to the client log file.
func (con *SliverClient) logCommand(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	logger := slog.New(con.jsonHandler).With(slog.String("type", "command"))
	logger.Debug(strings.Join(args, " "))
	return args, nil
}

func (con *SliverClient) setupAsciicastRecord(logFile *os.File, server io.Writer) error {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width, height = 80, 80
	}

	// Save the asciicast to the server and a local file.
	destinations := io.MultiWriter(logFile, server)

	encoder := asciicast.NewEncoder(destinations, width, height)
	if err := encoder.WriteHeader(); err != nil {
		return err
	}

	// save existing stdout | MultiWriter writes to saved stdout and file
	out := os.Stdout
	multiOut := io.MultiWriter(out, encoder)

	// get pipe reader and writer | writes to pipe writer come out pipe reader
	read, write, _ := os.Pipe()

	// replace stdout,stderr with pipe writer | all writes to stdout,
	// stderr will go through pipe instead (fmt.print, log)
	os.Stdout = write
	os.Stderr = write

	go io.Copy(multiOut, read)

	return nil
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
		con.PrintWarnf(err.Error())
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
	con.printf(format, args...)
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

// PrintWarnf a warning message immediately below the last line of output.
func (con *SliverClient) PrintWarnf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Warn(fmt.Sprintf(format, args...))

	con.printf(Clearln+"⚠️  "+Normal+format, args...)
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
