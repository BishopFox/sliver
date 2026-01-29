// Copyright 2021 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	logLevelName = map[int64]string{
		0: "DEBUG",
		1: "INFO",
		2: "WARNING",
		3: "ERROR",
		4: "CRITICAL",
	}
	traceContextRe = regexp.MustCompile(`^(\w+)/(\d+)(?:;o=[01])?$`)

	// maxLogMessage is the largest message that will be logged without chunking, reserving room for prefixes.
	// See http://cloud/logging/quotas#log-limits
	maxLogMessage = 255000
)

func logf(c *aeContext, level int64, format string, args ...interface{}) {
	if c == nil {
		panic("not an App Engine aeContext")
	}

	if !IsStandard() {
		s := strings.TrimRight(fmt.Sprintf(format, args...), "\n")
		now := timeNow().UTC()
		timestamp := fmt.Sprintf("%d/%02d/%02d %02d:%02d:%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
		fmt.Fprintf(logStream, "%s %s: %s\n", timestamp, logLevelName[level], s)
		return
	}

	eol := func(s string) string {
		if strings.HasSuffix(s, "\n") {
			return ""
		}
		return "\n"
	}

	msg := fmt.Sprintf(format, args...)

	if strings.HasPrefix(msg, "{") {
		// Assume the message is already structured, leave as-is unless it is too long.
		// Note: chunking destroys the structure; developers will have to ensure their structured log
		// is small enough to fit in a single message.
		for _, m := range chunkLog(msg) {
			fmt.Fprint(logStream, m, eol(m))
		}
		return
	}

	// First chunk the message, then structure each chunk.
	traceID, spanID := traceAndSpan(c)
	for _, m := range chunkLog(msg) {
		sl := structuredLog{
			Message:  m,
			Severity: logLevelName[level],
			TraceID:  traceID,
			SpanID:   spanID,
		}
		if b, err := json.Marshal(sl); err != nil {
			// Write raw message if error.
			fmt.Fprint(logStream, m, eol(m))
		} else {
			s := string(b)
			fmt.Fprint(logStream, s, eol(s))
		}
	}
}

type structuredLog struct {
	Message  string `json:"message"`
	Severity string `json:"severity"`
	TraceID  string `json:"logging.googleapis.com/trace,omitempty"`
	SpanID   string `json:"logging.googleapis.com/spanId,omitempty"`
}

func chunkLog(msg string) []string {
	if len(msg) <= maxLogMessage {
		return []string{msg}
	}
	var chunks []string
	i := 0
	for {
		if i == len(msg) {
			break
		}
		if i+maxLogMessage > len(msg) {
			chunks = append(chunks, msg[i:])
			break
		}
		chunks = append(chunks, msg[i:i+maxLogMessage])
		i += maxLogMessage
	}
	for i, c := range chunks {
		chunks[i] = fmt.Sprintf("Part %d/%d: ", i+1, len(chunks)) + c
	}
	return chunks
}

func traceAndSpan(c *aeContext) (string, string) {
	headers := c.req.Header["X-Cloud-Trace-Context"]
	if len(headers) < 1 {
		return "", ""
	}
	matches := traceContextRe.FindAllStringSubmatch(headers[0], -1)
	if len(matches) < 1 || len(matches[0]) < 3 {
		return "", ""
	}
	traceID := matches[0][1]
	spanID := matches[0][2]
	projectID := projectID()
	return fmt.Sprintf("projects/%s/traces/%s", projectID, traceID), spanID
}
