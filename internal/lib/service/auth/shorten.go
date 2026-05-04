package auth

import (
	"fmt"
	"strings"
)

// ShortenName converts "Иванов Иван Иванович" → "И. Иванов".
// Returns input unchanged if fewer than 2 whitespace-separated words.
func ShortenName(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) < 2 {
		return fullName
	}
	firstRune := []rune(parts[1])[0]
	return fmt.Sprintf("%c. %s", firstRune, parts[0])
}
