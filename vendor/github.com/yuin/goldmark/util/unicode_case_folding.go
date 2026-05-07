package util

//go:generate go run ../_tools unicode-case-folding-map -o ../_tools/unicode-case-folding-map.json
//go:generate go run ../_tools emb-structs -i ../_tools/unicode-case-folding-map.json -o ./unicode_case_folding.gen.go

var unicodeCaseFoldings map[rune][]rune

func init() {
	unicodeCaseFoldings = make(map[rune][]rune, _unicodeCaseFoldingLength)
	cTo := 0
	for i := 0; i < _unicodeCaseFoldingLength; i++ {
		tTo := cTo + int(_unicodeCaseFoldingToIndex[i])
		to := _unicodeCaseFoldingTo[cTo:tTo]
		unicodeCaseFoldings[_unicodeCaseFoldingFrom[i]] = to
		cTo = tTo
	}
}
