package c2

import (
	"io"
	"os"
)

const (
	// socketEnvelopeDiskSpoolThreshold is the point where envelope payloads are
	// streamed to disk first to avoid large attacker-triggered allocations.
	socketEnvelopeDiskSpoolThreshold = 64 * 1024 * 1024

	// socketEnvelopeReadChunkSize bounds peak in-memory usage while reading large
	// envelopes from the network.
	socketEnvelopeReadChunkSize = 1 * 1024 * 1024
)

func readSocketEnvelopeData(reader io.Reader, dataLength int, inMemoryLimit int) ([]byte, error) {
	if dataLength <= inMemoryLimit {
		dataBuf := make([]byte, dataLength)
		if _, err := io.ReadFull(reader, dataBuf); err != nil {
			return nil, err
		}
		return dataBuf, nil
	}

	tmpFile, err := os.CreateTemp("", "sliver-envelope-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	chunkSize := socketEnvelopeReadChunkSize
	if dataLength < chunkSize {
		chunkSize = dataLength
	}
	chunkBuf := make([]byte, chunkSize)

	remaining := dataLength
	for remaining > 0 {
		nextRead := chunkSize
		if remaining < nextRead {
			nextRead = remaining
		}

		if _, err := io.ReadFull(reader, chunkBuf[:nextRead]); err != nil {
			return nil, err
		}
		if _, err := tmpFile.Write(chunkBuf[:nextRead]); err != nil {
			return nil, err
		}

		remaining -= nextRead
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	dataBuf := make([]byte, dataLength)
	if _, err := io.ReadFull(tmpFile, dataBuf); err != nil {
		return nil, err
	}
	return dataBuf, nil
}
