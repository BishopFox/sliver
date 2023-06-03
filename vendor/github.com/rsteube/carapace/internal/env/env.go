package env

import "os"

func ColorDisabled() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0"
}

func Lenient() bool {
	return os.Getenv("CARAPACE_LENIENT") != ""
}

func Hashdirs() string {
	return os.Getenv("CARAPACE_ZSH_HASH_DIRS")
}

func Sandbox() string {
	return os.Getenv("CARAPACE_SANDBOX")
}

func Log() bool {
	return os.Getenv("CARAPACE_LOG") != ""
}
