package limio

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestLimits(t *testing.T) {
	LimitTestN(0, t)
	LimitTestN(1, t)
	LimitTestN(2, t)
	LimitTestN(1024, t)

	OverLimitTestN(0, t)
	OverLimitTestN(1, t)
	OverLimitTestN(2, t)
	OverLimitTestN(1024, t)
}

func OverLimitTestN(N int64, t *testing.T) {
	got := bytes.NewBuffer(nil)
	exp := bytes.NewBuffer(nil)

	lw := LimitWriter(got, N)
	w := io.MultiWriter(exp, lw)

	_, err := io.CopyN(w, rand.Reader, 2*N)
	if err != io.ErrShortWrite && N != 2*N {
		t.Error("did not throw io.ErrShortWrite!")
		if err != nil {
			t.Error(err.Error())
		}
	}

	if !bytes.Equal(got.Bytes(), exp.Bytes()[:N]) {
		t.Errorf("Exp: %s\n Got: %s\n",
			string(exp.Bytes()[:N]),
			string(got.Bytes()))
	}

	if int64(got.Len()) != N {
		t.Errorf("LimitWriter did not cap at %d", N)
	}
}

func LimitTestN(N int64, t *testing.T) {
	got := bytes.NewBuffer(nil)
	exp := bytes.NewBuffer(nil)

	lw := LimitWriter(got, N)
	w := io.MultiWriter(lw, exp)

	io.CopyN(w, rand.Reader, N)

	if !bytes.Equal(got.Bytes(), exp.Bytes()) {
		t.Errorf("Exp: %s\n Got: %s\n",
			string(exp.Bytes()[:N]),
			string(got.Bytes()))
	}
}
