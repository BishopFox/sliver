// +build !darwin !windows !linux

package hostuuid

// GetUUID - Function implementation for unsupported platforms
func GetUUID() string {
	return ""
}
