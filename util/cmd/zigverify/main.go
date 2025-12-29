// Command zigverify validates a Zig archive against a minisign signature.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	minisign "github.com/bishopfox/sliver/util/minisign"
)

const defaultZigPublicKey = "RWSGOq2NVecA2UPNdBUZykf1CCb147pkmdtYxgb3Ti+JO/wCYvhbAb/U"

func loadPublicKey() (minisign.PublicKey, error) {
	key := os.Getenv("ZIG_PUBLIC_KEY")
	if key == "" {
		key = defaultZigPublicKey
	}

	var publicKey minisign.PublicKey
	if err := publicKey.UnmarshalText([]byte(key)); err != nil {
		return minisign.PublicKey{}, fmt.Errorf("invalid minisign public key: %w", err)
	}

	return publicKey, nil
}

func verifySignature(artifactPath, signaturePath, expectedFile string) error {
	artifact, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("open artifact: %w", err)
	}
	defer artifact.Close()

	sigData, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("read signature: %w", err)
	}

	pubKey, err := loadPublicKey()
	if err != nil {
		return err
	}

	reader := minisign.NewReader(artifact)
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("stream artifact: %w", err)
	}

	if !reader.Verify(pubKey, sigData) {
		return errors.New("signature verification failed")
	}

	if err := verifyTrustedComment(sigData, expectedFile); err != nil {
		return err
	}

	return nil
}

func verifyTrustedComment(signature []byte, expectedFile string) error {
	var sig minisign.Signature
	if err := sig.UnmarshalText(signature); err != nil {
		return fmt.Errorf("parse signature: %w", err)
	}

	fileField := trustedCommentFile(sig.TrustedComment)
	if fileField == "" {
		return fmt.Errorf("trusted comment missing file field: %q", sig.TrustedComment)
	}
	if fileField != expectedFile {
		return fmt.Errorf("trusted comment file mismatch: expected %q, got %q", expectedFile, fileField)
	}

	return nil
}

func trustedCommentFile(trustedComment string) string {
	for field := range strings.FieldsSeq(trustedComment) {
		if after, ok := strings.CutPrefix(field, "file:"); ok {
			return after
		}
	}
	return ""
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s <artifact-path> <signature-path> <expected-filename>\n", os.Args[0])
		os.Exit(1)
	}

	if err := verifySignature(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
