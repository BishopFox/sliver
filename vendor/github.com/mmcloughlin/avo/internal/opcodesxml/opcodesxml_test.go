package opcodesxml

import "testing"

func TestReadFile(t *testing.T) {
	_, err := ReadFile("testdata/x86_64.xml")
	if err != nil {
		t.Fatal(err)
	}
}
