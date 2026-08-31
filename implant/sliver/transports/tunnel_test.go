package transports

import (
	"io"
	"sort"
	"sync"
	"testing"
)

func TestTunnelNextWriteSequenceConcurrent(t *testing.T) {
	const writers = 128

	tunnel := NewTunnel(1, nopWriteCloser{Writer: io.Discard})
	sequences := make([]uint64, writers)
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(writers)
	done.Add(writers)

	for index := range sequences {
		go func(index int) {
			defer done.Done()
			ready.Done()
			<-start
			sequences[index] = tunnel.NextWriteSequence()
		}(index)
	}

	ready.Wait()
	close(start)
	done.Wait()

	sort.Slice(sequences, func(i, j int) bool { return sequences[i] < sequences[j] })
	for index, sequence := range sequences {
		if want := uint64(index); sequence != want {
			t.Fatalf("sequence[%d] = %d, want %d", index, sequence, want)
		}
	}
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
