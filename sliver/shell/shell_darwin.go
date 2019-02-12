package shell

var (
	// Shell constants
	bash = []string{"/bin/bash"}
	sh   = []string{"/bin/sh"}
)

// GetSystemShellPath - Find bash or sh
func GetSystemShellPath() []string {
	if exists(bash[0]) {
		return bash
	}
	return sh
}
