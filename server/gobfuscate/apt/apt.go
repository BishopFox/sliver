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
	1: chinese,
	3: chinese,
	// 5:  undisclosed,
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

var chinese = []string{"隐蔽", "中华人民共和国", "义勇军进行曲", "上海市", "贝壳", "码", "鼠"}
var iranian = []string{"اسلامی ایران‎", "استقلال، آزادی، جمهوری اسلامی", "سرود ملی جمهوری اسلامی ایران"}
var korean = []string{"조선민주주의인민공화국", "애국가", "조선", "평양직할시", "량강도", "김정일"}
var russian = []string{"Росси́йская", "Федера́ция", "Государственный", "гимн", "Российской", "Федерации", "подробнее", "рождений"}
var vietnamese = []string{"nhớp", "nhác", "nương", "đâu", "Chuột", "Cửa hậu"}
