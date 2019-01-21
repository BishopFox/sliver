package gobfuscate

import (
	"errors"
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
		log.Println("Failed to create destination:", err)
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
	log.Println("Obfuscating package names ...")
	if err := ObfuscatePackageNames(ctx, newGopath, enc); err != nil {
		log.Println("Failed to obfuscate package names:", err)
		return "", err
	}
	log.Println("Obfuscating strings ...")
	if err := ObfuscateStrings(newGopath); err != nil {
		log.Println("Failed to obfuscate strings:", err)
		return "", err
	}
	log.Println("Obfuscating symbols ...")
	if err := ObfuscateSymbols(ctx, newGopath, enc); err != nil {
		log.Println("Failed to obfuscate symbols:", err)
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
