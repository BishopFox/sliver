package gobfuscate

import (
	"errors"
	"go/build"
	"os"
	"strings"

	gogo "github.com/bishopfox/sliver/server/gogo"
	"github.com/bishopfox/sliver/server/log"
)

var (
	obfLog = log.NamedLogger("gobfuscate", "obfuscator")
)

// Gobfuscate - Obfuscate Go code
func Gobfuscate(config gogo.GoConfig, encKey string, pkgName string, outPath string) (string, error) {

	newGopath := outPath
	if err := os.Mkdir(newGopath, 0755); err != nil {
		obfLog.Errorf("Failed to create destination: %v", err)
		return "", err
	}

	ctx := build.Default
	ctx.GOOS = config.GOOS
	ctx.GOARCH = config.GOARCH
	ctx.GOROOT = config.GOROOT
	ctx.GOPATH = config.GOPATH

	obfLog.Infof("Copying GOPATH (%s) ...\n", ctx.GOPATH)

	if !CopyGopath(ctx, pkgName, newGopath, false) {
		return "", errors.New("Failed to copy GOPATH")
	}

	// enc := &Encrypter{Key: encKey}
	// log.Println("Obfuscating package names ...")
	// if err := ObfuscatePackageNames(ctx, newGopath, enc); err != nil {
	// 	log.Println("Failed to obfuscate package names:", err)
	// 	return "", err
	// }

	obfLog.Info("Obfuscating strings ...")
	if err := ObfuscateStrings(newGopath); err != nil {
		obfLog.Errorf("Failed to obfuscate strings: %v", err)
		return "", err
	}

	// log.Println("Obfuscating symbols ...")
	// if err := ObfuscateSymbols(ctx, newGopath, enc); err != nil {
	// 	log.Println("Failed to obfuscate symbols:", err)
	// 	return "", err
	// }

	// newPkg := encryptComponents(pkgName, enc)
	// Hard code this until we get the pkg obfuscator working correctly
	return "github.com/bishopfox/sliver", nil
}

func encryptComponents(pkgName string, enc *Encrypter) string {
	comps := strings.Split(pkgName, "/")
	for i, comp := range comps {
		comps[i] = enc.Encrypt(comp)
	}
	return strings.Join(comps, "/")
}
