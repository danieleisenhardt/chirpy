package main

import (
	"slices"
	"strings"
)

func replaceBadWords(s string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(s, " ")
	for i, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}
