package codenames

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
	"bufio"
	"fmt"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"

	"github.com/sirupsen/logrus"
)

var (
	codenameLog = log.RootLogger.WithFields(logrus.Fields{
		"pkg":    "generate",
		"stream": "codenames",
	})
)

// readLines - Read lines of a text file into a slice
func readLines(txtFilePath string) ([]string, error) {
	file, err := os.Open(txtFilePath)
	if err != nil {
		codenameLog.Errorf("Error opening %s: %v", txtFilePath, err)
		return nil, err
	}
	defer file.Close()

	words := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		codenameLog.Errorf("Error scanning: %v", err)
		return nil, err
	}

	return words, nil
}

// getRandomWord - Get a random word from a file, not cryptographically secure
func getRandomWord(txtFilePath string) (string, error) {
	appDir := assets.GetRootAppDir()
	words, err := readLines(filepath.Join(appDir, txtFilePath))
	if err != nil {
		return "", err
	}
	wordsLen := len(words)
	if wordsLen == 0 {
		return "", fmt.Errorf("no words found in %s", txtFilePath)
	}
	word := words[insecureRand.Intn(wordsLen-1)]
	return strings.TrimSpace(word), nil
}

// RandomAdjective - Get a random noun, not cryptographically secure
func RandomAdjective() (string, error) {
	return getRandomWord("adjectives.txt")
}

// RandomNoun - Get a random noun, not cryptographically secure
func RandomNoun() (string, error) {
	return getRandomWord("nouns.txt")
}

// GetCodename - Returns a randomly generated 'codename'
func GetCodename() (string, error) {
	adjective, err := RandomAdjective()
	if err != nil {
		return "", err
	}
	noun, err := RandomNoun()
	if err != nil {
		return "", err
	}
	codename := fmt.Sprintf("%s_%s", strings.ToUpper(adjective), strings.ToUpper(noun))
	return strings.ReplaceAll(codename, " ", "-"), nil
}
