package opcodescsv

import (
	"os"

	"golang.org/x/arch/x86/x86csv"
)

// ReadFile reads the given x86 CSV file.
func ReadFile(filename string) ([]*x86csv.Inst, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := x86csv.NewReader(f)
	return r.ReadAll()
}
