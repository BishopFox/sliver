package apt

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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

var chinese = []string{
	"隐蔽", "中华人民共和国", "义勇军进行曲", "上海市", "贝壳", "码", "鼠", "隐蔽", "中华人民共和国", "义勇军进行曲", "上海市",
	"贝壳", "码", "鼠", "总参", "总参三部二局", "61398部队", "注入", "脚本", "漏洞", "木马", "攻击", "提权", "监控", "執行",
	"窃听", "监听", "肉鸡", "后门", "嗅探", "伪装", "渗透", "代理",
}
var iranian = []string{"اسلامی ایران‎", "استقلال، آزادی، جمهوری اسلامی", "سرود ملی جمهوری اسلامی ایران"}
var korean = []string{"조선민주주의인민공화국", "애국가", "조선", "평양직할시", "량강도", "김정일"}
var russian = []string{"Росси́йская", "Федера́ция", "Государственный", "гимн", "Российской", "Федерации", "подробнее", "рождений"}
var vietnamese = []string{"nhớp", "nhác", "nương", "đâu", "Chuột", "Cửa hậu"}
