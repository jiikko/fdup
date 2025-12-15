package code

import (
	"regexp"
	"strings"
	"unicode"
)

// Normalize normalizes a code by uppercasing and removing hyphens.
// Example: "prj-001" -> "PRJ001"
func Normalize(code string) string {
	upper := strings.ToUpper(code)
	return strings.ReplaceAll(upper, "-", "")
}

// Format formats a normalized code for display by inserting a hyphen
// between the letter and number parts.
// Example: "PRJ001" -> "PRJ-001"
func Format(normalized string) string {
	// Find boundary between letters and digits
	for i, r := range normalized {
		if unicode.IsDigit(r) {
			if i > 0 {
				return normalized[:i] + "-" + normalized[i:]
			}
			break
		}
	}
	return normalized
}

// Extractor extracts codes from filenames using patterns.
type Extractor struct {
	patterns []*regexp.Regexp
}

// NewExtractor creates a new code extractor with the given regex patterns.
func NewExtractor(patterns []string) (*Extractor, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return &Extractor{patterns: compiled}, nil
}

// Extract extracts a code from a filename using the configured patterns.
// Returns the normalized code and true if found, or empty string and false if not.
func (e *Extractor) Extract(filename string) (string, bool) {
	for _, re := range e.patterns {
		matches := re.FindStringSubmatch(filename)
		if len(matches) > 1 {
			// Combine all capture groups
			var code string
			for i := 1; i < len(matches); i++ {
				code += matches[i]
			}
			return Normalize(code), true
		}
	}
	return "", false
}
