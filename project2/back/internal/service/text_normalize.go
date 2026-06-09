package service

import "regexp"

var (
	excessiveNewLinesRe = regexp.MustCompile(`\n{3,}`)
	lineEndingsRe       = regexp.MustCompile(`\r\n?`)
)

func normalizeUserText(in string) string {
	if in == "" {
		return ""
	}
	s := normalizeLineEndings(in)
	s = trimEdgeNewLines(s)
	return excessiveNewLinesRe.ReplaceAllString(s, "\n\n")
}

func normalizeLineEndings(in string) string {
	return lineEndingsRe.ReplaceAllString(in, "\n")
}

func trimEdgeNewLines(in string) string {
	start := 0
	for start < len(in) && in[start] == '\n' {
		start++
	}
	end := len(in)
	for end > start && in[end-1] == '\n' {
		end--
	}
	return in[start:end]
}
