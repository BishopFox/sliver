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
	obfuscateLog = log.NamedLogger("gobfuscate", "obfuscator")
)

// Gobfuscate - Obfuscate Go code
func Gobfuscate(config gogo.GoConfig, encKey string, pkgName string, outPath string, symbols bool) (string, error) {

	newGopath := outPath
	if err := os.Mkdir(newGopath, 0755); err != nil {
		obfuscateLog.Errorf("Failed to create destination: %v", err)
		return "", err
	}

	ctx := build.Default
	ctx.GOOS = config.GOOS
	ctx.GOARCH = config.GOARCH
	ctx.GOROOT = config.GOROOT
	ctx.GOPATH = config.GOPATH

	obfuscateLog.Infof("Copying GOPATH (%s) ...\n", ctx.GOPATH)

	newPkgName := "github.com/bishopfox/sliver"
	enc := &Encrypter{Key: encKey}

	if !CopyGopath(ctx, pkgName, newGopath, false) {
		return "", errors.New("Failed to copy GOPATH")
	}

	obfuscateLog.Info("Obfuscating strings ...")
	if err := ObfuscateStrings(newGopath); err != nil {
		obfuscateLog.Errorf("Failed to obfuscate strings: %v", err)
		return "", err
	}

	if symbols {
		obfuscateLog.Info("Obfuscating package names ...")
		if err := ObfuscatePackageNames(ctx, newGopath, enc); err != nil {
			obfuscateLog.Errorf("Failed to obfuscate package names: %s", err)
			return "", err
		}

		obfuscateLog.Info("Obfuscating symbols ...")
		if err := ObfuscateSymbols(ctx, newGopath, enc); err != nil {
			obfuscateLog.Errorf("Failed to obfuscate symbols: %s", err)
			return "", err
		}
		newPkgName = encryptComponents(pkgName, enc)
	}

	return newPkgName, nil
}

func encryptComponents(pkgName string, enc *Encrypter) string {
	comps := strings.Split(pkgName, "/")
	for i, comp := range comps {
		comps[i] = enc.Encrypt(comp)
	}
	return strings.Join(comps, "/")
}
