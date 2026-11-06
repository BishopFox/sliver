package util

func ValidPageSize(s int) bool {
	return s&(s-1) == 0 && 512 <= s && s <= 65536
}
