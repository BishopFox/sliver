package util

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func randomData() []byte {
	buf := make([]byte, 128)
	rand.Read(buf)
	return buf
}

func TestCopyFileContents(t *testing.T) {
	sample := randomData()
	tmpfile, err := ioutil.TempFile("", "sliver-unit-test")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up
	if _, err := tmpfile.Write(sample); err != nil {
		t.Error(err)
	}
	err = tmpfile.Close()
	if err != nil {
		t.Error(err)
	}

	dst := fmt.Sprintf("%s.2", tmpfile.Name())
	CopyFileContents(tmpfile.Name(), dst)

	srcData, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Error(err)
	}
	dstData, err := ioutil.ReadFile(dst)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(sample, srcData) {
		t.Error(fmt.Errorf("Sample and src do not match:\nsample: %v\n src: %v", sample, srcData))
	}
	if !bytes.Equal(dstData, srcData) {
		t.Error(fmt.Errorf("dst and src do not match:\ndst: %v\nsrc: %v", dstData, srcData))
	}

}
