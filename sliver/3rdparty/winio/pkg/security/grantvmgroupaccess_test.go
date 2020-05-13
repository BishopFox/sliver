package security

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGrantVmGroupAccess verifies for the three case of a file, a directory,
// and a file in a directory that the appropriate ACEs are set, including
// inheritance in the second two examples. These are the expected ACES. Is
// verified by running icacls and comparing output.
//
// File:
// S-1-15-3-1024-2268835264-3721307629-241982045-173645152-1490879176-104643441-2915960892-1612460704:(R,W)
// S-1-5-83-1-3166535780-1122986932-343720105-43916321:(R,W)
//
// Directory:
// S-1-15-3-1024-2268835264-3721307629-241982045-173645152-1490879176-104643441-2915960892-1612460704:(OI)(CI)(R,W)
// S-1-5-83-1-3166535780-1122986932-343720105-43916321:(OI)(CI)(R,W)
//
// File in directory (inherited):
// S-1-15-3-1024-2268835264-3721307629-241982045-173645152-1490879176-104643441-2915960892-1612460704:(I)(R,W)
// S-1-5-83-1-3166535780-1122986932-343720105-43916321:(I)(R,W)

func TestGrantVmGroupAccess(t *testing.T) {
	f, err := ioutil.TempFile("", "gvmgafile")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	d, err := ioutil.TempDir("", "gvmgadir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)

	find, err := os.Create(filepath.Join(d, "find.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if err := GrantVmGroupAccess(f.Name()); err != nil {
		t.Fatal(err)
	}

	if err := GrantVmGroupAccess(d); err != nil {
		t.Fatal(err)
	}

	verifyicacls(t,
		f.Name(),
		[]string{`NT VIRTUAL MACHINE\\Virtual Machines:(R)`},
	)

	// Two items here:
	//  - One explicit read only.
	//  - Other applies to this folder, subfolders and files
	//      (OI): object inherit
	//      (CI): container inherit
	//      (IO): inherit only
	//      (GR): generic read
	//
	// In properties for the directory, advanced security settings, this will
	// show as a single line "Allow/Virtual Machines/Read/Inherited from none/This folder, subfolder and files
	verifyicacls(t,
		d,
		[]string{`NT VIRTUAL MACHINE\\Virtual Machines:(R)`, `NT VIRTUAL MACHINE\\Virtual Machines:(OI)(CI)(IO)(GR)`},
	)

	verifyicacls(t,
		find.Name(),
		[]string{`NT VIRTUAL MACHINE\\Virtual Machines:(I)(R)`},
	)

}

func verifyicacls(t *testing.T, name string, aces []string) {
	cmd := exec.Command("icacls", name)
	outb, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	out := string(outb)

	for _, ace := range aces {
		// Avoid '(' and ')' being part of match groups
		ace = strings.Replace(ace, "(", "\\(", -1)
		ace = strings.Replace(ace, ")", "\\)", -1)

		rx := regexp.MustCompile(ace)
		matches := rx.FindAllStringIndex(out, -1)
		if len(matches) != 1 {
			t.Fatalf("expected one match for %s got %d\n%s", ace, len(matches), out)
		}
	}
}
