package gobfuscate

import (
	"errors"
	"fmt"
	"go/build"
	"log"
	"os"
	gogo "sliver/server/gogo"
	"strings"
)

// Gobfuscate - Obfuscate Go code
func Gobfuscate(config gogo.GoConfig, encKey string, pkgName string, outPath string) (string, error) {

	newGopath := outPath
	if err := os.Mkdir(newGopath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create destination:", err)
		return "", err
	}

	ctx := build.Default
	ctx.GOOS = config.GOOS
	ctx.GOARCH = config.GOARCH
	ctx.GOROOT = config.GOROOT
	ctx.GOPATH = config.GOPATH

	log.Printf("Copying GOPATH (%s) ...\n", ctx.GOPATH)

	if !CopyGopath(ctx, pkgName, newGopath, false) {
		return "", errors.New("Failed to copy GOPATH")
	}

	enc := &Encrypter{Key: encKey}
	log.Println("Obfuscating package names...")
	if err := ObfuscatePackageNames(newGopath, enc); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to obfuscate package names:", err)
		return "", err
	}
	log.Println("Obfuscating strings...")
	if err := ObfuscateStrings(newGopath); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to obfuscate strings:", err)
		return "", err
	}
	log.Println("Obfuscating symbols...")
	if err := ObfuscateSymbols(newGopath, enc); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to obfuscate symbols:", err)
		return "", err
	}

	newPkg := encryptComponents(pkgName, enc)

	return newPkg, nil
}

func encryptComponents(pkgName string, enc *Encrypter) string {
	comps := strings.Split(pkgName, "/")
	for i, comp := range comps {
		comps[i] = enc.Encrypt(comp)
	}
	return strings.Join(comps, "/")
}
