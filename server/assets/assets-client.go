//go:build client

package assets

import "embed"

var (
	//go:embed fs/english.txt fs/sliver.asc
	assetsFs embed.FS
)

// Blank for client, no need for traffic encoders
func setupTrafficEncoders(appDir string) error {
	return nil
}
