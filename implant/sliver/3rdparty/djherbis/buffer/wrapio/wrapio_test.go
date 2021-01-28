package wrapio

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestWrapper(t *testing.T) {
	if file, err := ioutil.TempFile("", "wrap.test"); err == nil {
		defer os.Remove(file.Name())
		defer file.Close()
		file.Write([]byte("abcdef"))

		data := make([]byte, 12)

		w := NewWrapper(file, 6, 1, 6)
		if n, err := w.ReadAt(data, 1); err == nil || err == io.EOF {
			if !bytes.Equal(data[:n], []byte("cdefa")) {
				t.Errorf("Exp cdefa, Got %s", data[:n])
			}
		} else {
			t.Error(err.Error())
		}
	}
}

func TestWrap(t *testing.T) {
	if file, err := ioutil.TempFile("", "wrap.test"); err == nil {
		w := NewWrapWriter(file, 0, 3)
		w.Write([]byte("abcdef"))

		r := NewWrapReader(file, 0, 2)
		data := make([]byte, 6)
		r.Read(data)
		if !bytes.Equal(data, []byte("dedede")) {
			t.Error("Wrapper error!")
		}
		file.Close()
		os.Remove(file.Name())
	}
}
