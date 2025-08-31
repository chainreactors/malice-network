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

	Adjectives []string

	//go:embed nouns.txt
	nouns []byte

	Nouns []string
)

func SetupCodenames() {
	Adjectives = strings.Split(string(adjectives), "\n")
	Nouns = strings.Split(string(nouns), "\n")
}

// getRandomWord - Get a random word from a file, not cryptographically secure
func getRandomWord(words []string) (string, error) {
	wordsLen := len(words)
	word := words[insecureRand.Intn(wordsLen-1)]
	return strings.TrimSpace(word), nil
}

// RandomAdjective - Get a random adjective, not cryptographically secure
func RandomAdjective() (string, error) {
	return getRandomWord(Adjectives)
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

// GetCodenameWithMaxLength - Returns a randomly generated 'codename' with maximum length limit
func GetCodenameWithMaxLength(maxLength int) string {
	if maxLength <= 0 {
		return GetCodename()
	}

	// Try to generate a codename within the length limit (max 100 attempts)
	for attempts := 0; attempts < 20; attempts++ {
		adjective, _ := RandomAdjective()
		noun, _ := RandomNoun()
		codename := fmt.Sprintf("%s_%s", strings.ToUpper(adjective), strings.ToUpper(noun))
		codename = strings.ReplaceAll(codename, " ", "-")

		if len(codename) <= maxLength {
			return codename
		}
	}

	// If we can't generate a short enough codename, truncate it
	codename := GetCodename()
	if len(codename) > maxLength {
		return codename[:maxLength]
	}
	return codename
}
