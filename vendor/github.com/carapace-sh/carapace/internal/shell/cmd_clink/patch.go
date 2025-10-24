package cmd_clink

import (
	"os"

	shlex "github.com/carapace-sh/carapace-shlex"
)

func Patch(args []string) ([]string, error) {
	compline, ok := os.LookupEnv("CARAPACE_COMPLINE")
	if !ok {
		return args, nil
	}
	os.Unsetenv("CARAPACE_COMPLINE")

	if compline == "" {
		return args, nil
	}

	tokens, err := shlex.Split(compline)
	if err != nil {
		return nil, err
	}
	args = append(args[:1], tokens.CurrentPipeline().FilterRedirects().Words().Strings()...)
	return args, nil
}
