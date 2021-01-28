package buffer

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

func BenchmarkMemory(b *testing.B) {
	buf := New(32 * 1024)
	for i := 0; i < b.N; i++ {
		io.Copy(buf, io.LimitReader(rand.Reader, 32*1024))
		io.Copy(ioutil.Discard, buf)
	}
}

func TestPool(t *testing.T) {
	pool := NewPool(func() Buffer { return New(10) })
	buf, err := pool.Get()
	if err != nil {
		t.Error(err)
	}
	buf.Write([]byte("hello world"))
	pool.Put(buf)
}

func TestMemPool(t *testing.T) {
	p := NewMemPool(10)
	poolTest(p, t)

	b := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(b)
	if err := enc.Encode(&p); err != nil {
		t.Error(err)
	}

	var pool Pool
	dec := gob.NewDecoder(b)
	if err := dec.Decode(&pool); err != nil {
		t.Error(err)
	}
	poolTest(pool, t)
}

func poolTest(pool Pool, t *testing.T) {
	buf, err := pool.Get()
	if err != nil {
		t.Error(err)
	}
	if n, err := buf.Write([]byte("hello world")); n != 10 {
		t.Errorf("wrote incorrect amount")
	} else if err == nil {
		t.Errorf("should have been a shortwrite error here")
	}
	pool.Put(buf)
	if buf.Len() > 0 {
		t.Errorf("should have emptied the buffer")
	}
}

func TestFilePool(t *testing.T) {
	pool := NewFilePool(1024, "::~_bad_dir_~::")
	buf := NewPartition(pool)
	_, err := buf.Write([]byte("hello"))
	if err == nil {
		t.Error("an error was expected here")
	}
}

func TestFilePool2(t *testing.T) {
	pool := NewFilePool(1024, "::~_bad_dir_~::")
	buf := NewPartition(pool, New(0))
	_, err := buf.Write([]byte("hello"))
	if err == nil {
		t.Error("an error was expected here")
	}
}

func TestOverflow(t *testing.T) {
	buf := NewMulti(New(5), Discard)
	buf.Write([]byte("Hello World"))

	b := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(b)
	if err := enc.Encode(&buf); err != nil {
		t.Error(err)
	}

	var buf2 Buffer
	dec := gob.NewDecoder(b)
	if err := dec.Decode(&buf2); err != nil {
		t.Error(err)
	}

	buf = buf2

	data, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(data, []byte("Hello")) {
		t.Errorf("Expected Hello got %s", string(data))
	}

}

func TestWriteAt(t *testing.T) {
	var b BufferAt

	b = New(5)
	BufferAtTester(t, b)

	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	b = NewFile(5, file)
	BufferAtTester(t, b)

	b = NewMultiAt(New(1), NewFile(4, file))
	BufferAtTester(t, b)

	b = NewMultiAt(New(2), NewFile(3, file))
	BufferAtTester(t, b)

	b = NewMultiAt(New(3), NewFile(2, file))
	BufferAtTester(t, b)

	b = NewMultiAt(New(4), NewFile(1, file))
	BufferAtTester(t, b)

	b = NewSwapAt(New(1), New(5))
	BufferAtTester(t, b)

	b = NewSwapAt(New(2), New(5))
	BufferAtTester(t, b)

	b = NewSwapAt(New(3), New(5))
	BufferAtTester(t, b)

	b = NewSwapAt(New(4), New(5))
	BufferAtTester(t, b)

	b = NewPartitionAt(NewMemPoolAt(3))
	BufferAtTester(t, b)

	b = NewPartitionAt(NewFilePoolAt(3, os.TempDir()))
	BufferAtTester(t, b)

	b = NewPartitionAt(NewMemPoolAt(1))
	BufferAtTester(t, b)

	b = NewPartitionAt(NewFilePoolAt(1, os.TempDir()))
	BufferAtTester(t, b)

	b = NewPartitionAt(NewMemPoolAt(5))
	BufferAtTester(t, b)

	b = NewPartitionAt(NewFilePoolAt(5, os.TempDir()))
	BufferAtTester(t, b)
}

func BufferAtTester(t *testing.T, b BufferAt) {
	t.Helper()
	b.WriteAt([]byte("abc"), 0)
	Compare(t, b, "abc")

	b.WriteAt([]byte("abc"), 1)
	Compare(t, b, "aabc")

	b.WriteAt([]byte("abc"), 2)
	Compare(t, b, "aaabc")

	b.WriteAt([]byte("def"), 3)
	switch {
	case b.Cap() > 5:
		Compare(t, b, "aaadef")
	default:
		Compare(t, b, "aaade")
	}

	b.Read(make([]byte, 2))
	switch {
	case b.Cap() > 5:
		Compare(t, b, "adef")
	default:
		Compare(t, b, "ade")
	}

	b.WriteAt([]byte("ab"), 3)
	Compare(t, b, "adeab")

	b.Reset()
}

func Compare(t *testing.T, b BufferAt, s string) {
	t.Helper()
	data := make([]byte, b.Len())
	n, _ := b.ReadAt(data, 0)
	if string(data[:n]) != s {
		t.Errorf("Mismatch: got %q want %q", string(data[:n]), s)
	}
	off := int64(len(s) / 2)
	n, _ = b.ReadAt(data, off)
	if string(data[:n]) != s[off:] {
		t.Errorf("Mismatch: got %q want %q", string(data[:n]), s[off:])
	}
}

func TestRingComplex(t *testing.T) {
	ringSize := 10
	data := []byte("hello!")
	ring := NewRing(New(int64(ringSize)))
	buf := make([]byte, ringSize)

	for i := 0; i < 10; i++ {
		ring.Write(data)
		n, err := ring.Read(buf)
		if err != nil {
			t.Errorf("unexpected error %s", err)
		} else if !bytes.Equal(buf[:n], data) {
			t.Errorf("expected %s got %s", data, buf[:n])
		}
	}

	// we are going to overflow the buffer here, we expect the last 10 bytes to remain
	repData := bytes.Repeat(data, 3)
	leftover := repData[len(repData)-ringSize:]
	for i := 0; i < 10; i++ {
		ring.Write(repData)
		n, err := ring.Read(buf)
		if err != nil {
			t.Errorf("unexpected error %s", err)
		} else if !bytes.Equal(buf[:n], leftover) {
			t.Errorf("expected %s got %s", leftover, buf[:n])
		}
	}
}

func TestRing(t *testing.T) {
	ring := NewRing(New(3))
	if ring.Len() != 0 {
		t.Errorf("Ring non-empty start!")
	}

	if ring.Cap() != math.MaxInt64 {
		t.Errorf("Ring has < max capacity")
	}

	ring.Write([]byte("abc"))
	if ring.Len() != 3 {
		t.Errorf("expected ring len == 3")
	}

	ring.Write([]byte("de"))
	if ring.Len() != 3 {
		t.Errorf("expected ring len == 3")
	}

	data := make([]byte, 12)
	if n, err := ring.Read(data); err != nil {
		t.Error(err.Error())
	} else {
		if !bytes.Equal(data[:n], []byte("cde")) {
			t.Errorf("expected cde, got %s", data[:n])
		}
	}

	if ring.Len() != 0 {
		t.Errorf("ring should now be empty")
	}

	ring.Write([]byte("hello"))
	ring.Reset()

	if ring.Len() != 0 {
		t.Errorf("ring should still be empty")
	}
}

func TestGob(t *testing.T) {
	str := "HelloWorld"

	buf := NewUnboundedBuffer(2, 2)
	buf.Write([]byte(str))
	b := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(b).Encode(&buf); err != nil {
		t.Error(err.Error())
		return
	}
	var buffer Buffer
	if err := gob.NewDecoder(b).Decode(&buffer); err != nil {
		t.Error(err.Error())
		return
	}
	data := make([]byte, len(str))
	if _, err := buffer.Read(data); err != nil && err != io.EOF {
		t.Error(err.Error())
	}
	if !bytes.Equal(data, []byte(str)) {
		t.Error("Gob Recover Failed... " + string(data))
	}
	buffer.Reset()
}

func TestDiscard(t *testing.T) {
	buf := Discard
	if buf.Cap() != math.MaxInt64 {
		t.Errorf("cap isn't infinite")
	}
	buf.Write([]byte("hello"))
	if buf.Len() != 0 {
		t.Errorf("buf should always be empty")
	}
}

func TestList(t *testing.T) {
	mem := New(10)
	ory := New(10)
	mem.Write([]byte("Hello"))
	ory.Write([]byte("world"))

	buf := List([]Buffer{mem, ory, Discard})
	if buf.Len() != 10 {
		t.Errorf("incorrect sum of lengths")
	}
	if buf.Cap() != math.MaxInt64 {
		t.Errorf("incorrect sum of caps")
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty")
	}
}

func TestList2(t *testing.T) {
	mem := New(10)
	ory := New(10)
	mem.Write([]byte("Hello"))
	ory.Write([]byte("world"))

	buf := List([]Buffer{mem, ory})
	if buf.Len() != 10 {
		t.Errorf("incorrect sum of lengths")
	}
	if buf.Cap() != 20 {
		t.Errorf("incorrect sum of caps")
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty")
	}
}

func TestSpill(t *testing.T) {
	buf := NewSpill(New(5), Discard)
	buf.Write([]byte("Hello World"))
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello")) {
		t.Error("ReadAt Failed. " + string(data[:n]))
	}
}

func TestSpill2(t *testing.T) {
	buf := NewSpill(New(5), nil)

	if buf.Cap() != math.MaxInt64 {
		t.Errorf("cap isn't infinite")
	}

	towrite := []byte("Hello World")
	m, _ := buf.Write(towrite)
	if m != len(towrite) {
		t.Errorf("failed to write all data: %d != %d", m, len(towrite))
	}
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
}

func TestNoSpill(t *testing.T) {
	buf := NewSpill(New(1024), nil)
	buf.Write([]byte("Hello World"))
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello World")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
}

func TestFile(t *testing.T) {
	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	buf := NewFile(1024, file)
	checkCap(t, buf, 1024)
	runPerfectSeries(t, buf)
	buf.Reset()

	buf = NewFile(3, file)
	buf.Write([]byte("abc"))
	buf.Read(make([]byte, 1))
	buf.Write([]byte("a"))
	d, _ := ioutil.ReadAll(buf)
	if !bytes.Equal(d, []byte("bca")) {
		t.Error("back and forth error!")
	}
}

func TestMem(t *testing.T) {
	buf := New(1024)
	checkCap(t, buf, 1024)
	runPerfectSeries(t, buf)
	buf.Reset()
	if n, err := buf.WriteAt([]byte("hello"), 1); err == nil || n != 0 {
		t.Errorf("write should have failed")
	}
}

func TestFilePartition(t *testing.T) {
	buf := NewPartition(NewFilePool(1024, ""))
	checkCap(t, buf, math.MaxInt64)
	runPerfectSeries(t, buf)
	buf.Reset()
}

func TestSmallMulti(t *testing.T) {
	if empty := NewMulti(); empty != nil {
		t.Errorf("the empty buffer should return nil")
	}

	one := NewMulti(New(10))
	if one.Len() != 0 {
		t.Errorf("singleton multi doesn't match inner buffer len")
	}
	if one.Cap() != 10 {
		t.Errorf("singleton multi doesn't match inner buffer cap")
	}
}

func TestMulti(t *testing.T) {
	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	buf := NewMulti(New(5), New(5), NewFile(500, file), NewPartition(NewFilePool(1024, "")))
	checkCap(t, buf, math.MaxInt64)
	runPerfectSeries(t, buf)
	isPerfectMatch(t, buf, 1024*1024)
	buf.Reset()
}

func TestMulti2(t *testing.T) {
	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	buf := NewMulti(New(5), New(5), NewFile(500, file))
	checkCap(t, buf, 510)
	runPerfectSeries(t, buf)
	isPerfectMatch(t, buf, 1024*1024)
	buf.Reset()
}

func runPerfectSeries(t *testing.T, buf Buffer) {
	checkEmpty(t, buf)
	simple(t, buf)

	max := int64(1024)
	isPerfectMatch(t, buf, 0)
	for i := int64(1); i < max; i *= 2 {
		isPerfectMatch(t, buf, i)
	}
	isPerfectMatch(t, buf, max)
}

func simple(t *testing.T, buf Buffer) {
	buf.Write([]byte("hello world"))
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal([]byte("hello world"), data) {
		t.Error("Hello world failed.")
	}

	buf.Write([]byte("hello world"))
	data = make([]byte, 3)
	buf.Read(data)
	buf.Write([]byte(" yolo"))
	data, err = ioutil.ReadAll(buf)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal([]byte("lo world yolo"), data) {
		t.Error("Buffer crossing error :(", string(data))
	}
}

func buildOutputs(t *testing.T, buf Buffer, size int64) (wrote []byte, read []byte) {
	r := io.LimitReader(rand.Reader, size)
	tee := io.TeeReader(r, buf)

	wrote, _ = ioutil.ReadAll(tee)
	read, _ = ioutil.ReadAll(buf)

	return wrote, read
}

func isPerfectMatch(t *testing.T, buf Buffer, size int64) {
	wrote, read := buildOutputs(t, buf, size)
	if !bytes.Equal(wrote, read) {
		t.Error("Buffer should have matched")
	}
}

func checkEmpty(t *testing.T, buf Buffer) {
	if !Empty(buf) {
		t.Error("Buffer should start empty!")
	}
}

func checkCap(t *testing.T, buf Buffer, correctCap int64) {
	if buf.Cap() != correctCap {
		t.Error("Buffer cap is incorrect", buf.Cap(), correctCap)
	}
}

type bigBuffer struct{}

func (b bigBuffer) Len() int64 { return math.MaxInt64 }

func (b bigBuffer) Cap() int64 { return math.MaxInt64 }

func (b bigBuffer) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func (b bigBuffer) Write(p []byte) (int, error) { return 0, io.ErrShortBuffer }

func (b bigBuffer) Reset() {}

func TestBigList(t *testing.T) {
	l := List([]Buffer{bigBuffer{}, bigBuffer{}})
	if l.Len() != math.MaxInt64 {
		t.Errorf("expected list to be max-size: %d", l.Len())
	}
}

func TestBigMulti(t *testing.T) {
	single := NewMulti(bigBuffer{})
	if single.Len() != math.MaxInt64 {
		t.Errorf("expected len to be max-size: %d", single.Len())
	}
	if single.Cap() != math.MaxInt64 {
		t.Errorf("expected cap to be max-size: %d", single.Cap())
	}

	buf := NewMulti(bigBuffer{}, bigBuffer{})
	if buf.Len() != math.MaxInt64 {
		t.Errorf("expected len to be max-size: %d", buf.Len())
	}
	if buf.Cap() != math.MaxInt64 {
		t.Errorf("expected cap to be max-size: %d", buf.Cap())
	}

	// test defrag failure
	NewMulti(badBuffer{}, badBuffer{})
}

type badBuffer struct{}

func (b badBuffer) Len() int64 { return 1024 }

func (b badBuffer) Cap() int64 { return 10 * 1024 }

func (b badBuffer) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func (b badBuffer) Write(p []byte) (int, error) { return 0, io.ErrShortBuffer }

func (b badBuffer) Reset() {}

func (b badBuffer) MarshalBinary() ([]byte, error) {
	return nil, errors.New("no data")
}

func TestBadPartition(t *testing.T) {
	p := NewPool(func() Buffer { return badBuffer{} })
	buf := NewPartition(p)
	_, err := buf.Write([]byte("bad write"))
	if err != io.ErrShortBuffer {
		t.Errorf("wrong read error was returned! %v", err)
	}

	b := make([]byte, 1024)
	_, err = buf.Read(b)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("wrong write error was returned! %v", err)
	}
}

type fakeFile struct {
	name string
}

func (f *fakeFile) Name() string {
	return f.name
}

func (f *fakeFile) Stat() (fi os.FileInfo, err error) {
	return nil, nil
}

func (f *fakeFile) ReadAt(p []byte, off int64) (int, error) {
	return 0, io.EOF
}

func (f *fakeFile) WriteAt(p []byte, off int64) (int, error) {
	return 0, io.ErrShortWrite
}

func (f *fakeFile) Close() error { return nil }

func TestBadGobFile(t *testing.T) {
	b := NewFile(10, &fakeFile{name: "test"})
	b2 := NewFile(10, &fakeFile{name: "test2"})
	b3 := NewMulti(b, b2)
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&b3); err != nil {
		t.Error(err)
	}
	var buffer Buffer
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&buffer); err == nil {
		t.Error("expected an error here, file does not exist")
	}
}

func TestBadGobFile2(t *testing.T) {
	b := New(10)
	b2 := NewFile(10, &fakeFile{name: "test2"})
	b3 := NewMulti(b, b2)
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&b3); err != nil {
		t.Error(err)
	}
	var buffer Buffer
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&buffer); err == nil {
		t.Error("expected an error here, file does not exist")
	}
}

func TestBadMultiGob(t *testing.T) {
	var b Buffer = badBuffer{}
	var b2 Buffer = badBuffer{}
	var b3 Buffer = New(10)
	b4 := NewMulti(b, b3)
	b5 := NewMulti(b3, b2)

	enc := gob.NewEncoder(ioutil.Discard)
	if err := enc.Encode(&b4); err == nil {
		t.Error("expected an error here, bad buffer can't be gobbed")
	}
	if err := enc.Encode(&b5); err == nil {
		t.Error("expected an error here, bad buffer can't be gobbed")
	}
}

func TestSwapAt(t *testing.T) {
	buf := NewSwapAt(New(5), New(10))
	buf.WriteAt([]byte("hey"), 0)
	data := make([]byte, 10)
	n, _ := buf.ReadAt(data, 0)
	if string(data[:n]) != "hey" {
		t.Error("expected hey got", string(data[:n]))
	}
	buf.WriteAt([]byte("hey"), 3)
	n, _ = buf.ReadAt(data, 1)
	if string(data[:n]) != "eyhey" {
		t.Error("expected eyhey got", string(data[:n]))
	}
	buf.WriteAt([]byte("hey"), 5)
	n, _ = buf.ReadAt(data, 1)
	if string(data[:n]) != "eyhehey" {
		t.Error("expected eyhehey got", string(data[:n]))
	}
}

func TestSwap(t *testing.T) {
	defer func() {
		recover()
	}()
	buf := NewSwap(New(5), New(10))
	if buf.Len() != 0 {
		t.Error("buffer should start empty got", buf.Len())
	}
	if buf.Cap() != 10 {
		t.Error("cap should be second buffer size (10) got", buf.Cap())
	}
	io.WriteString(buf, "hey")
	data, _ := ioutil.ReadAll(buf)
	if string(data) != "hey" {
		t.Error("expected hey got", string(data))
	}
	io.WriteString(buf, "hey")
	io.WriteString(buf, "hey")
	io.WriteString(buf, "hey")
	data, _ = ioutil.ReadAll(buf)
	if string(data) != "heyheyhey" {
		t.Error("expected heyheyhey got", string(data))
	}
	io.WriteString(buf, "hey")
	if buf.Len() != 3 {
		t.Error("should have data")
	}
	buf.Reset()
	if buf.Len() != 0 {
		t.Error("should be empty")
	}

	runPerfectSeries(t, NewSwap(New(512), New(1024)))

	NewSwap(New(1), New(0))
	t.Error("expected panic")
}

func TestPanicReadAt(t *testing.T) {
	defer func() {
		recover()
	}()
	buf := toBufferAt(New(5))
	buf.ReadAt(nil, 0)
	t.Error("expected a panic!")
}

func TestPanicWriteAt(t *testing.T) {
	defer func() {
		recover()
	}()
	buf := toBufferAt(New(5))
	buf.WriteAt(nil, 0)
	t.Error("expected a panic!")
}

type badFile struct{}

func (b badFile) Name() string                             { return "" }
func (b badFile) Stat() (os.FileInfo, error)               { return nil, errors.New("unsupported") }
func (b badFile) ReadAt(p []byte, off int64) (int, error)  { return 0, nil }
func (b badFile) WriteAt(p []byte, off int64) (int, error) { return len(p), nil }
func (b badFile) Close() error                             { return nil }

func TestWrapioBreakout(t *testing.T) {
	buf := NewFile(10, badFile{})
	io.WriteString(buf, "hello world")
	if _, err := ioutil.ReadAll(buf); err != io.ErrNoProgress {
		t.Error("expected no progress to be made")
		t.FailNow()
	}
}

func TestListAt(t *testing.T) {
	mem := New(10)
	ory := New(10)
	mem.Write([]byte("Hello"))
	ory.Write([]byte("world"))

	buf := ListAt([]BufferAt{mem, ory})
	if buf.Len() != 10 {
		t.Errorf("incorrect sum of lengths")
	}
	if buf.Cap() != 20 {
		t.Errorf("incorrect sum of caps")
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty")
	}
}

func TestPoolAt(t *testing.T) {
	pool := NewPoolAt(func() BufferAt { return New(10) })
	buf, err := pool.Get()
	if err != nil {
		t.Error(err)
	}
	buf.Write([]byte("hello world"))
	pool.Put(buf)
}

func TestMemPoolAt(t *testing.T) {
	p := NewMemPoolAt(10)
	poolAtTest(p, t)

	b := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(b)
	if err := enc.Encode(&p); err != nil {
		t.Error(err)
	}

	var pool PoolAt
	dec := gob.NewDecoder(b)
	if err := dec.Decode(&pool); err != nil {
		t.Error(err)
	}
	poolAtTest(pool, t)
}

func poolAtTest(pool PoolAt, t *testing.T) {
	buf, err := pool.Get()
	if err != nil {
		t.Error(err)
	}
	if n, err := buf.Write([]byte("hello world")); n != 10 {
		t.Errorf("wrote incorrect amount")
	} else if err == nil {
		t.Errorf("should have been a shortwrite error here")
	}
	pool.Put(buf)
	if buf.Len() > 0 {
		t.Errorf("should have emptied the buffer")
	}
}

func TestPartitionAt(t *testing.T) {
	buf := NewPartitionAt(NewMemPoolAt(5))
	buf.Write([]byte("Hello World"))
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello World")) {
		t.Error("Read Failed. " + string(data[:n]))
	}

	checkCap(t, buf, math.MaxInt64)
	runPerfectSeries(t, buf)
}

// TestPartitionAt2 tests ability to read at various offsets from buffer previously written.
func TestPartitionAt2(t *testing.T) {
	buf := NewPartitionAt(NewMemPoolAt(5))
	buf.Write([]byte("Hello World"))
	data := make([]byte, 2)
	n, _ := buf.ReadAt(data, 2)
	if !bytes.Equal(data[:n], []byte("ll")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
	n, _ = buf.ReadAt(data, 4)
	if !bytes.Equal(data[:n], []byte("o ")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
	n, _ = buf.ReadAt(data, 10)
	if !bytes.Equal(data[:n], []byte("d")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
	n, _ = buf.ReadAt(data, 100)
	if !bytes.Equal(data[:n], []byte{}) {
		t.Error("Read Failed. " + string(data[:n]))
	}

	buf.Reset()
	checkCap(t, buf, math.MaxInt64)
	runPerfectSeries(t, buf)
}

// TestPartitionAt3 tests ability to overwrite buffer previously written.
func TestPartitionAt3(t *testing.T) {
	buf := NewPartitionAt(NewMemPoolAt(5))
	buf.Write(make([]byte, 15, 15)) // allocates 3 membuffers
	buf.WriteAt([]byte("hey"), 0)
	data := make([]byte, 10)
	data = data[:3]
	buf.ReadAt(data, 0)
	if string(data) != "hey" {
		t.Error("expected hey got", string(data))
	}
	buf.WriteAt([]byte("hey"), 3)
	data = data[:5]
	buf.ReadAt(data, 1)
	if string(data) != "eyhey" {
		t.Error("expected eyhey got", string(data))
	}
	buf.WriteAt([]byte("hey"), 5)
	data = data[:7]
	buf.ReadAt(data, 1)
	if string(data) != "eyhehey" {
		t.Error("expected eyhehey got", string(data))
	}
}
