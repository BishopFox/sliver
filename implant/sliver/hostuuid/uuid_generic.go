//go:build !(linux || darwin || windows)

package hostuuid

// GetUUID - Function implementation for unsupported platforms
func GetUUID() string {
	return UUIDFromMAC()
}
