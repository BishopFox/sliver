package assets

import (
	"os"
	"path/filepath"
)

const (
	// AIDirName stores server-side AI helper data under the Sliver root.
	AIDirName = "ai"

	// AIAliasesDirName stores aliases made available to the server-side AI agent.
	AIAliasesDirName = "aliases"

	// AIExtensionsDirName stores extensions made available to the server-side AI agent.
	AIExtensionsDirName = "extensions"
)

// GetAIDir returns the server-side AI data directory: ~/.sliver/ai
func GetAIDir() string {
	return ensureAIDir(filepath.Join(GetRootAppDir(), AIDirName))
}

// GetAIAliasesDir returns the server-side AI alias directory: ~/.sliver/ai/aliases
func GetAIAliasesDir() string {
	return ensureAIDir(filepath.Join(GetAIDir(), AIAliasesDirName))
}

// GetAIExtensionsDir returns the server-side AI extension directory: ~/.sliver/ai/extensions
func GetAIExtensionsDir() string {
	return ensureAIDir(filepath.Join(GetAIDir(), AIExtensionsDirName))
}

func ensureAIDir(dir string) string {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			setupLog.Fatalf("Cannot create AI data dir %s", err)
		}
	}
	return dir
}
