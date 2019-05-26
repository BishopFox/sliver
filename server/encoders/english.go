package encoders

import (
	insecureRand "math/rand"
	"github.com/bishopfox/sliver/server/assets"
	"strings"
	"time"
)

var dictionary *map[int][]string

func sumWord(word string) int {
	sum := 0
	for _, char := range word {
		sum += int(char)
	}
	return sum % 256
}

func buildDictionary() {
	dictionary = &map[int][]string{}
	for _, word := range assets.English() {
		word = strings.TrimSpace(word)
		sum := sumWord(word)
		(*dictionary)[sum] = append((*dictionary)[sum], word)
	}
}

// English Encoder - An ASCIIEncoder for binary to english text
type English struct{}

// Encode - Binary => English
func (e English) Encode(data []byte) string {
	if dictionary == nil {
		buildDictionary()
	}
	insecureRand.Seed(time.Now().Unix())
	words := []string{}
	for _, b := range data {
		possibleWords := (*dictionary)[int(b)]
		index := insecureRand.Intn(len(possibleWords))
		words = append(words, possibleWords[index])
	}
	return strings.Join(words, " ")
}

// Decode - English => Binary
func (e English) Decode(words string) ([]byte, error) {
	wordList := strings.Split(words, " ")
	data := []byte{}
	for _, word := range wordList {
		word = strings.TrimSpace(word)
		if len(word) == 0 {
			continue
		}
		byteValue := sumWord(word)
		data = append(data, byte(byteValue))
	}
	return data, nil
}
