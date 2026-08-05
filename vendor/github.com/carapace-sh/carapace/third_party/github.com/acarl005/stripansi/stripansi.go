package stripansi

import (
	"regexp"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re *regexp.Regexp

func Strip(str string) string {
	if re == nil {
		re = regexp.MustCompile(ansi)
	}
	return re.ReplaceAllString(str, "")
}
