package traverse

import "path/filepath"

// NixProfile returns the location of the ~/.nix-profile folder.
func NixProfile(tc Context) (string, error) {
	home, err := UserHomeDir(tc)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(home, ".nix-profile")), nil
}
