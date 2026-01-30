package bar

import "foo/bar/baz"

const Name = "bar"

func Use() string {
	return baz.Name
}
