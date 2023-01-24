package encoders

var rawEnglishDictionary []string

func SetEnglishDictionary(dictionary []string) {
	rawEnglishDictionary = dictionary
}

func getEnglishDictionary() []string {
	return rawEnglishDictionary
}
