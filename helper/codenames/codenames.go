package codenames

import (
	_ "embed"
	"fmt"
	insecureRand "math/rand"
	"strings"
)

var (
	//go:embed adjectives.txt
	adjectives []byte

	Adejctives []string

	//go:embed nouns.txt
	nouns []byte

	Nouns []string
)

func SetupCodenames() {
	Adejctives = strings.Split(string(adjectives), "\n")
	Nouns = strings.Split(string(nouns), "\n")
}

// getRandomWord - Get a random word from a file, not cryptographically secure
func getRandomWord(words []string) (string, error) {
	wordsLen := len(words)
	word := words[insecureRand.Intn(wordsLen-1)]
	return strings.TrimSpace(word), nil
}

// RandomAdjective - Get a random noun, not cryptographically secure
func RandomAdjective() (string, error) {
	return getRandomWord(Adejctives)
}

// RandomNoun - Get a random noun, not cryptographically secure
func RandomNoun() (string, error) {
	return getRandomWord(Nouns)
}

// GetCodename - Returns a randomly generated 'codename'
func GetCodename() string {
	adjective, _ := RandomAdjective()
	noun, _ := RandomNoun()
	codename := fmt.Sprintf("%s_%s", strings.ToUpper(adjective), strings.ToUpper(noun))
	return strings.ReplaceAll(codename, " ", "-")
}
