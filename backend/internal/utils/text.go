package utils

import "unicode/utf8"

func TruncateText(s string, maxLength int) string {
	if utf8.RuneCountInString(s) <= maxLength {
		return s
	}

	if maxLength <= 3 {
		return "..."
	}

	runes := []rune(s)
	return string(runes[:maxLength-3]) + "..."
}
