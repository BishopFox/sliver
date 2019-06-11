package encoders

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
	"strings"
	"time"

	"github.com/bishopfox/sliver/server/assets"
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
