package clean

import (
	"regexp"
	"strings"
	"unicode"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var multiSpaceRe = regexp.MustCompile(`\s+`)

// Text normalizes and cleans text content for dataset samples.
func Text(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// IsQuality checks if a sample meets minimum quality requirements.
func IsQuality(instruction, context, solution string, minLen, maxLen int) bool {
	if len(strings.TrimSpace(instruction)) < minLen {
		return false
	}
	total := len(instruction) + len(context) + len(solution)
	if maxLen > 0 && total > maxLen {
		return false
	}
	if !hasAlpha(instruction) {
		return false
	}
	return true
}

func hasAlpha(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
