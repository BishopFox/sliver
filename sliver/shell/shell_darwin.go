package shell

const (
	// Shell constants
	bash = "/bin/bash"
	sh   = "/bin/sh"
)

// GetSystemShellPath - Find bash or sh
func GetSystemShellPath() string {
	if exists(bash) {
		return bash
	}
	return sh
}
