package crack

import "github.com/bishopfox/sliver/client/command/crack/internal/hashcatdisplay"

func hashcatHashTypeName(mode int32) (string, bool) {
	return hashcatdisplay.TypeName(mode)
}

func humanizeHashRate(rate uint64) string {
	return hashcatdisplay.HumanizeRate(rate)
}
