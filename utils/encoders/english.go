package encoders

import (
	insecureRand "math/rand"
	"strings"
)

var dictionary map[int][]string

// English Encoder - An ASCIIEncoder for binary to english text
type English struct{}

// Encode - Binary => English
func (e English) Encode(data []byte) ([]byte, error) {
	if dictionary == nil {
		buildDictionary()
	}
	words := []string{}
	for _, b := range data {
		possibleWords := dictionary[int(b)]
		index := insecureRand.Intn(len(possibleWords))
		words = append(words, possibleWords[index])
	}
	return []byte(strings.Join(words, " ")), nil
}

// Decode - English => Binary
func (e English) Decode(words []byte) ([]byte, error) {
	wordList := strings.Split(string(words), " ")
	data := []byte{}
	for _, word := range wordList {
		word = strings.TrimSpace(word)
		if len(word) == 0 {
			continue
		}
		byteValue := SumWord(word)
		data = append(data, byte(byteValue))
	}
	return data, nil
}

var rawEnglishDictionary []string

func SetEnglishDictionary(dictionary []string) {
	rawEnglishDictionary = dictionary
}

func getEnglishDictionary() []string {
	return rawEnglishDictionary
}

func buildDictionary() {
	dictionary = map[int][]string{}
	for _, word := range getEnglishDictionary() {
		word = strings.TrimSpace(word)
		sum := SumWord(word)
		dictionary[sum] = append(dictionary[sum], word)
	}
}

// SumWord - Sum the ASCII values of a word
func SumWord(word string) int {
	sum := 0
	for _, char := range word {
		sum += int(char)
	}
	return sum % 256
}
