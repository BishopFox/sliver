package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	bspinner "charm.land/bubbles/v2/spinner"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

const (
	minConnectSpinnerDuration = 900 * time.Millisecond
	minConnectStatusDuration  = 350 * time.Millisecond
)

type connectResult struct {
	rpc  rpcpb.SliverRPCClient
	conn *grpc.ClientConn
	err  error
}

func connectWithSpinner(out io.Writer, target string, connect func(transport.ConnectStatusFn) (rpcpb.SliverRPCClient, *grpc.ClientConn, error)) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
	statusCh := make(chan string, 8)
	resultCh := make(chan connectResult, 1)

	go func() {
		rpc, conn, err := connect(func(status string) {
			sendStatus(statusCh, status)
		})
		resultCh <- connectResult{rpc: rpc, conn: conn, err: err}
	}()

	currentStatus := ""
	currentStatusSince := time.Time{}
	queuedStatus := ""
	lastWidth := 0
	spinner := bspinner.New(bspinner.WithSpinner(bspinner.Line))
	ticker := time.NewTicker(spinner.Spinner.FPS)
	defer ticker.Stop()
	startedAt := time.Now()
	var pendingResult *connectResult

	setStatus := func(status string) {
		status = strings.TrimSpace(status)
		if status == "" || status == currentStatus {
			return
		}
		currentStatus = status
		currentStatusSince = time.Now()
	}

	stageStatus := func(status string) {
		status = strings.TrimSpace(status)
		if status == "" {
			return
		}
		if currentStatus == "" || time.Since(currentStatusSince) >= minConnectStatusDuration {
			queuedStatus = ""
			setStatus(status)
			return
		}
		queuedStatus = status
	}

	flushQueuedStatus := func() {
		if queuedStatus == "" || time.Since(currentStatusSince) < minConnectStatusDuration {
			return
		}
		nextStatus := queuedStatus
		queuedStatus = ""
		setStatus(nextStatus)
	}

	canReturnPendingResult := func() bool {
		if pendingResult == nil {
			return false
		}
		if pendingResult.err != nil {
			return true
		}
		if time.Since(startedAt) < minConnectSpinnerDuration {
			return false
		}
		if queuedStatus != "" {
			return false
		}
		if !currentStatusSince.IsZero() && time.Since(currentStatusSince) < minConnectStatusDuration {
			return false
		}
		return true
	}

	render := func() {
		line := fmt.Sprintf("%s %s", spinner.View(), formatConnectSpinnerMessage(target, currentStatus))
		lastWidth = writeSpinnerLine(out, line, lastWidth)
	}

	render()
	for {
		select {
		case status := <-statusCh:
			stageStatus(status)
			spinner, _ = spinner.Update(spinner.Tick())
			render()

		case result := <-resultCh:
			pendingResult = &result

		case <-ticker.C:
			spinner, _ = spinner.Update(spinner.Tick())
			flushQueuedStatus()
			render()
			if canReturnPendingResult() {
				result := *pendingResult
				pendingResult = nil
				clearSpinnerLine(out, lastWidth)
				return result.rpc, result.conn, result.err
			}
		}

		if pendingResult != nil && pendingResult.err != nil {
			result := *pendingResult
			pendingResult = nil
			clearSpinnerLine(out, lastWidth)
			return result.rpc, result.conn, result.err
		}
	}
}

func sendStatus(statusCh chan string, status string) {
	status = strings.TrimSpace(status)
	if status == "" {
		return
	}
	statusCh <- status
}

func formatConnectSpinnerMessage(target string, status string) string {
	target = strings.TrimSpace(target)
	status = strings.TrimSpace(status)

	if target == "" {
		if status == "" {
			return "Connecting ..."
		}
		return fmt.Sprintf("Connecting (%s) ...", status)
	}
	if status == "" {
		return fmt.Sprintf("Connecting to %s ...", target)
	}
	return fmt.Sprintf("Connecting to %s (%s) ...", target, status)
}

func writeSpinnerLine(out io.Writer, line string, lastWidth int) int {
	width := utf8.RuneCountInString(line)
	padding := ""
	if lastWidth > width {
		padding = strings.Repeat(" ", lastWidth-width)
	}
	fmt.Fprintf(out, "\r%s%s", line, padding)
	if width > lastWidth {
		return width
	}
	return lastWidth
}

func clearSpinnerLine(out io.Writer, lastWidth int) {
	if lastWidth <= 0 {
		return
	}
	fmt.Fprintf(out, "\r%s\r", strings.Repeat(" ", lastWidth))
}
