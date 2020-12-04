package bj

import (
	"io/ioutil"
	"os"

	"github.com/Binject/shellcode"
)

// Injection Methods
const (
	PtNoteInject int = iota
	SilvioInject     = iota
)

// BinjectConfig - Configuration Settings for the Binject modules
type BinjectConfig struct {
	CodeCaveMode    bool
	InjectionMethod int

	Repo *shellcode.Repo
}

// BinjectFile - Inject shellcode into a binary file
func BinjectFile(sourceFile string, destFile string, shellcodeFile string, config *BinjectConfig) error {

	shellcodeBytes, err := ioutil.ReadFile(shellcodeFile)
	if err != nil {
		return err
	}

	sourceBytes, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return err
	}

	destBytes, err := Binject(sourceBytes, shellcodeBytes, config)
	if err != nil {
		return err
	}

	f, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(destBytes)
	return err
}

// Binject - Inject shellcode into a byte array
func Binject(sourceBytes []byte, shellcodeBytes []byte, config *BinjectConfig) ([]byte, error) {

	binType, err := BinaryMagic(sourceBytes)
	var binject func([]byte, []byte, *BinjectConfig) ([]byte, error)
	switch binType {
	case ELF:
		binject = ElfBinject
	case MACHO:
		binject = MachoBinject
	case PE:
		binject = PeBinject
	case ERROR:
		return nil, err
	}
	return binject(sourceBytes, shellcodeBytes, config)
}
