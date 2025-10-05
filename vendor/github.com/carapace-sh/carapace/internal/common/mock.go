package common

import (
	"fmt"
	"os"
)

type Mock struct {
	Dir     string
	Replies map[string]string
}

func (m Mock) CacheDir() string {
	return m.Dir + "/cache/"
}

func (m Mock) WorkDir() string {
	return m.Dir + "/work/"
}

type t interface {
	Name() string
	Fatal(...interface{})
}

func NewMock(t t) *Mock {
	tempDir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("carapace-sandbox_%v_", t.Name()))
	if err != nil {
		t.Fatal("failed to create sandbox dir: " + err.Error())
	}

	m := &Mock{
		Dir:     tempDir,
		Replies: make(map[string]string),
	}
	if err := os.Mkdir(m.CacheDir(), os.ModePerm); err != nil {
		t.Fatal("failed to create sandbox cache dir: " + err.Error())
	}
	if err := os.Mkdir(m.WorkDir(), os.ModePerm); err != nil {
		t.Fatal("failed to create sandbox work dir: " + err.Error())
	}
	return m
}
