package apt

import (
	insecureRand "math/rand"
	"time"
)

// RandomString - Returns a random string for a particular APT group
func RandomString(group int) string {
	if aptStrings, ok := APTGroups[group]; ok {
		insecureRand.Seed(time.Now().UnixNano())
		return aptStrings[insecureRand.Intn(len(aptStrings))]
	}
	return ""
}

// APTGroups - Map of groups to character sets
var APTGroups = map[int][]string{
	1:  chinese,
	3:  chinese,
	5:  undisclosed,
	10: chinese,
	12: chinese,
	16: chinese,
	17: chinese,
	18: chinese,
	19: chinese,
	28: russian,
	29: russian,
	30: chinese,
	32: vietnamese,
	33: iranian,
	34: iranian,
	37: korean,
	38: korean,
	39: iranian,
	40: chinese,
}

var undisclosed = []string{}
var chinese = []string{"隐蔽"}
var iranian = []string{}
var korean = []string{}
var russian = []string{}
var vietnamese = []string{}
