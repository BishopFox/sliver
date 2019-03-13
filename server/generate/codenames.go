package generate

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path"
	"sliver/server/assets"
	"sliver/server/log"
	"strings"
	"time"

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
	rand.Seed(time.Now().UnixNano())
	appDir := assets.GetRootAppDir()
	words := readlines(path.Join(appDir, "adjectives.txt"))
	word := words[rand.Intn(len(words)-1)]
	return strings.TrimSpace(word)
}

// getRandomNoun - Get a random noun, not cryptographically secure
func getRandomNoun() string {
	rand.Seed(time.Now().Unix())
	appDir := assets.GetRootAppDir()
	words := readlines(path.Join(appDir, "nouns.txt"))
	word := words[rand.Intn(len(words)-1)]
	return strings.TrimSpace(word)
}

// GetCodename - Returns a randomly generated 'codename'
func GetCodename() string {
	adjective := strings.ToUpper(getRandomAdjective())
	noun := strings.ToUpper(getRandomNoun())
	return fmt.Sprintf("%s_%s", adjective, noun)
}
