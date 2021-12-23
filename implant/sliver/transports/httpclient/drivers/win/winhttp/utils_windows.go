package winhttp

import "fmt"

func convertFail(str string, e error) error {
	return fmt.Errorf(
		"failed to convert %s to Windows type: %w",
		str,
		e,
	)
}
