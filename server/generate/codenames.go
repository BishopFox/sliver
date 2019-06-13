package generate

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
	"path"
	"strings"
	"time"

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

// readlines - Read lines of a text file into a slice
func readlines(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		codenameLog.Fatal(err)
	}
	defer file.Close()

	words := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		codenameLog.Fatal(err)
	}

	return words
}

// getRandomAdjective - Get a random noun, not cryptographically secure
func getRandomAdjective() string {
	insecureRand.Seed(time.Now().UnixNano())
	appDir := assets.GetRootAppDir()
	words := readlines(path.Join(appDir, "adjectives.txt"))
	word := words[insecureRand.Intn(len(words)-1)]
	return strings.TrimSpace(word)
}

// getRandomNoun - Get a random noun, not cryptographically secure
func getRandomNoun() string {
	insecureRand.Seed(time.Now().UnixNano())
	appDir := assets.GetRootAppDir()
	words := readlines(path.Join(appDir, "nouns.txt"))
	word := words[insecureRand.Intn(len(words)-1)]
	return strings.TrimSpace(word)
}

// GetCodename - Returns a randomly generated 'codename'
func GetCodename() string {
	adjective := strings.ToUpper(getRandomAdjective())
	noun := strings.ToUpper(getRandomNoun())
	codename := fmt.Sprintf("%s_%s", adjective, noun)
	return strings.ReplaceAll(codename, " ", "-")
}
