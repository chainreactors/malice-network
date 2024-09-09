package codenames

import (
	"bufio"
	"embed"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"strings"
)

var (
	//go:embed  *.txt
	assetsFs embed.FS
)

func SetupCodenames(appDir string) error {
	nouns, err := assetsFs.ReadFile("nouns.txt")
	if err != nil {
		logs.Log.Errorf("nouns.txt asset not found")
		return err
	}

	adjectives, err := assetsFs.ReadFile("adjectives.txt")
	if err != nil {
		logs.Log.Errorf("adjectives.txt asset not found")
		return err
	}

	err = os.WriteFile(filepath.Join(appDir, "nouns.txt"), nouns, 0600)
	if err != nil {
		logs.Log.Errorf("Failed to write noun data to: %s", appDir)
		return err
	}

	err = os.WriteFile(filepath.Join(appDir, "adjectives.txt"), adjectives, 0600)
	if err != nil {
		logs.Log.Errorf("Failed to write adjective data to: %s", appDir)
		return err
	}
	return nil
}

// readLines - Read lines of a text file into a slice
func readLines(txtFilePath string) ([]string, error) {
	file, err := os.Open(txtFilePath)
	if err != nil {
		// TODO - log error opening
		//codenameLog.Errorf("Error opening %s: %v", txtFilePath, err)
		return nil, err
	}
	defer file.Close()

	words := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		// TODO - log error scanning
		//codenameLog.Errorf("Error scanning: %v", err)
		return nil, err
	}

	return words, nil
}

// getRandomWord - Get a random word from a file, not cryptographically secure
func getRandomWord(txtFilePath string) (string, error) {
	txtDir := files.GetExcPath()
	txtPath := filepath.Join(txtDir, ".malice", txtFilePath)
	words, err := readLines(txtPath)
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
