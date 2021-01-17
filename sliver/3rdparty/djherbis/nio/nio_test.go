package nio

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"

	"io/ioutil"

	"time"

	"github.com/djherbis/buffer"
)

func TestCopy(t *testing.T) {
	buf := buffer.New(1024)
	input := bytes.NewBuffer(nil)
	output := bytes.NewBuffer(nil)

	input.Write([]byte("hello world"))
	n, err := Copy(output, input, buf)
	if err != nil {
		t.Errorf(err.Error())
	}
	if int(n) != len("hello world") {
		t.Errorf("wrote wrong # of bytes")
	}

	if !bytes.Equal(output.Bytes(), []byte("hello world")) {
		t.Errorf("output didn't match: %s", output.Bytes())
	}
}

func TestBigWriteSmallBuf(t *testing.T) {
	buf := buffer.New(5)
	r, w := Pipe(buf)
	defer r.Close()

	go func() {
		defer w.Close()
		n, err := w.Write([]byte("hello world"))
		if err != nil {
			t.Error(err.Error())
		}
		if int(n) != len("hello world") {
			t.Errorf("wrote wrong # of bytes")
		}
	}()

	output := bytes.NewBuffer(nil)
	_, err := io.Copy(output, r)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(output.Bytes(), []byte("hello world")) {
		t.Errorf("unexpected output %s", output.Bytes())
	}
}

func TestPipeCloseEarly(t *testing.T) {
	buf := buffer.New(1024)
	r, w := Pipe(buf)
	r.Close()

	_, err := w.Write([]byte("hello world"))
	if err != io.ErrClosedPipe {
		t.Errorf("expected closed pipe")
	}

	_, err = io.Copy(ioutil.Discard, r)
	if err != io.ErrClosedPipe {
		t.Errorf("expected closed pipe")
	}
}

func TestPipe(t *testing.T) {
	buf := buffer.New(1024)
	r, w := Pipe(buf)
	defer r.Close()

	data := []byte("the quick brown fox jumps over the lazy dog")
	if _, err := w.Write(data); err != nil {
		t.Error(err.Error())
		return
	}
	w.Close()

	result := make([]byte, 1024)
	n, err := r.Read(result)
	if err != nil && err != io.EOF {
		t.Error(err.Error())
		return
	}
	result = result[:n]

	if !bytes.Equal(data, result) {
		t.Errorf("exp [%s]\ngot[%s]", string(data), string(result))
	}
}

func TestEarlyCloseWrite(t *testing.T) {
	buf := buffer.New(1)
	r, w := Pipe(buf)

	testerr := errors.New("test err")

	w.CloseWithError(testerr)

	_, err := w.Write([]byte("ab")) // too big for buffer

	if err != io.ErrClosedPipe {
		t.Errorf("expected %s but got %s.", testerr, err)
	}

	_, err = io.Copy(ioutil.Discard, r)
	if err != testerr {
		t.Errorf("expected %s but got %s.", testerr, err)
	}
}

func TestUnblockWrite(t *testing.T) {
	buf := buffer.New(1)
	r, w := Pipe(buf)

	testerr := errors.New("test err")

	go func() {
		<-time.After(100 * time.Millisecond)
		if er := w.CloseWithError(testerr); er != nil {
			t.Error(er)
		}
	}()

	_, err := w.Write([]byte("ab")) // too big for buffer

	if err != io.ErrClosedPipe {
		t.Errorf("expected %s but got %s.", testerr, err)
	}

	_, err = io.Copy(ioutil.Discard, r)
	if err != testerr {
		t.Errorf("expected %s but got %s.", testerr, err)
	}
}

type badBuffer struct{}

func (badBuffer) Len() int64                  { return 3 }
func (badBuffer) Cap() int64                  { return 6 }
func (badBuffer) Write(p []byte) (int, error) { return len(p), nil }
func (badBuffer) Read(p []byte) (int, error)  { return 0, io.EOF }

func TestEmpty(t *testing.T) {
	r, w := Pipe(badBuffer{})
	n, err := w.Write([]byte("any"))

	if err != nil {
		t.Error(err)
	}

	if n != 3 {
		t.Errorf("wrote wrong # of bytes %d", n)
	}

	n, err = r.Read(nil)

	if err != nil {
		t.Error(err)
	}

	if n != 0 {
		t.Errorf("wrote wrong # of bytes %d", n)
	}
}

func BenchmarkPipe(b *testing.B) {
	r, w := Pipe(buffer.New(1024))
	benchPipe(r, w, b)
}

func BenchmarkIOPipe(b *testing.B) {
	r, w := io.Pipe()
	benchPipe(r, w, b)
}

func benchPipe(r io.Reader, w io.WriteCloser, b *testing.B) {
	b.ReportAllocs()
	f, err := ioutil.TempFile("", "benchPipe")
	if err != nil {
		b.Error(err)
	}

	go func() {
		defer f.Close()
		io.Copy(bufio.NewWriter(f), r)
	}()

	defer w.Close()

	for i := 0; i < b.N; i++ {
		w.Write([]byte("hello world"))
	}
}
