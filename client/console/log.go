package console

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slog"
	"maze.io/x/asciicast"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func (con *SliverConsoleClient) setupLogger(logFile *os.File) {
	// File and Sliver RPC streams
	logWriter := io.MultiWriter(logFile)

	// JSON
	jsonOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	con.jsonHandler = slog.NewJSONHandler(logWriter, jsonOptions)

	// Log all commands before running them.
	con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, con.logCommand)
}

func getConsoleLogFile() *os.File {
	logsDir := assets.GetConsoleLogsDir()
	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("%s.log", dateTime))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalf("Could not open log file: %s", err)
	}
	logFile.Write([]byte(fmt.Sprintf("Sliver Console Log - %s\n\n", dateTime)))
	return logFile
}

// logCommand logs non empty commands to the client log file.
func (con *SliverConsoleClient) logCommand(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	logger := slog.New(con.jsonHandler).With(slog.String("type", "command"))
	logger.Debug(strings.Join(args, " "))

	return args, nil
}

//
// -------------------------- [ Logging ] -----------------------------
//
// Logging function below differ slightly from their counterparts in client/log package:
// These below will print their output regardless of the currently active menu (server/implant),
// while those in the log package tie their output to the current menu.

// PrintAsyncResponse - Print the generic async response information
func (con *SliverConsoleClient) PrintAsyncResponse(resp *commonpb.Response) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	beacon, err := con.Rpc.GetBeacon(ctx, &clientpb.Beacon{ID: resp.BeaconID})
	if err != nil {
		con.PrintWarnf(err.Error())
		return
	}
	con.PrintInfof("Tasked beacon %s (%s)", beacon.Name, strings.Split(resp.TaskID, "-")[0])
}

func (con *SliverConsoleClient) Printf(format string, args ...any) {
	logger := slog.NewLogLogger(con.jsonHandler, slog.LevelInfo)
	logger.Printf(format, args...)

	con.printf(format, args...)
}

// Println prints an output without status and immediately below the last line of output.
func (con *SliverConsoleClient) Println(args ...any) {
	logger := slog.New(con.jsonHandler)

	format := strings.Repeat("%s", len(args))
	logger.Info(fmt.Sprintf(format, args))

	con.printf(format, args...)
}

// PrintInfof prints an info message immediately below the last line of output.
func (con *SliverConsoleClient) PrintInfof(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+Info+format, args...)
}

// PrintSuccessf prints a success message immediately below the last line of output.
func (con *SliverConsoleClient) PrintSuccessf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+Success+format, args...)
}

// PrintWarnf a warning message immediately below the last line of output.
func (con *SliverConsoleClient) PrintWarnf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Warn(fmt.Sprintf(format, args...))

	con.printf(Clearln+"⚠️  "+Normal+format, args...)
}

// PrintErrorf prints an error message immediately below the last line of output.
func (con *SliverConsoleClient) PrintErrorf(format string, args ...any) {
	logger := slog.New(con.jsonHandler)

	logger.Error(fmt.Sprintf(format, args...))

	con.printf(Clearln+Warn+format, args...)
}

// PrintEventInfof prints an info message with a leading/trailing newline for emphasis.
func (con *SliverConsoleClient) PrintEventInfof(format string, args ...any) {
	logger := slog.New(con.jsonHandler).With(slog.String("type", "event"))

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+"\n"+Info+format+"\r", args...)
}

// PrintEventErrorf prints an error message with a leading/trailing newline for emphasis.
func (con *SliverConsoleClient) PrintEventErrorf(format string, args ...any) {
	logger := slog.New(con.jsonHandler).With(slog.String("type", "event"))

	logger.Error(fmt.Sprintf(format, args...))

	con.printf(Clearln+"\n"+Warn+format+"\r", args...)
}

// PrintEventSuccessf a success message with a leading/trailing newline for emphasis.
func (con *SliverConsoleClient) PrintEventSuccessf(format string, args ...any) {
	logger := slog.New(con.jsonHandler).With(slog.String("type", "event"))

	logger.Info(fmt.Sprintf(format, args...))

	con.printf(Clearln+"\n"+Success+format+"\r", args...)
}

func (con *SliverConsoleClient) setupAsciicastRecord(logFile *os.File) {
	encoder := asciicast.NewEncoder(logFile, 80, 80)

	go io.Copy(encoder, os.Stdout)
}

func getConsoleAsciicastFile() *os.File {
	logsDir := assets.GetConsoleLogsDir()
	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("asciicast_%s.log", dateTime))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalf("Could not open log file: %s", err)
	}
	logFile.Write([]byte(fmt.Sprintf("Sliver Console Log - %s\n\n", dateTime)))
	return logFile
}
