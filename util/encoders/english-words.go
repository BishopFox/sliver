package encoders

var rawEnglishDictionary []string

func InitEnglishDictionary(initDictionary []string) {
	rawEnglishDictionary = initDictionary
}

func getEnglishDictionary() []string {
	return rawEnglishDictionary
}
