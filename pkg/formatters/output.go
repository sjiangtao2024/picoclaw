package formatters

import (
	"regexp"
	"strings"
)

var (
	reFenceStart = regexp.MustCompile("(?m)^```[a-zA-Z0-9_-]*\\s*$")
	reFenceEnd   = regexp.MustCompile("(?m)^```\\s*$")
	reHeading    = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`)
	reBold       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reItalicA    = regexp.MustCompile(`\*([^*]+)\*`)
	reItalicB    = regexp.MustCompile(`_([^_]+)_`)
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reListA      = regexp.MustCompile(`(?m)^\s*[-*]\s+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
)

// Format normalizes outbound content for channels with weak Markdown support.
func Format(channel, content string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "dingtalk", "discord":
		return toPlainText(content)
	default:
		return strings.TrimSpace(content)
	}
}

func toPlainText(input string) string {
	s := strings.ReplaceAll(input, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	s = reFenceStart.ReplaceAllString(s, "")
	s = reFenceEnd.ReplaceAllString(s, "")
	s = reHeading.ReplaceAllString(s, "")
	s = reBold.ReplaceAllString(s, "$1")
	s = reItalicA.ReplaceAllString(s, "$1")
	s = reItalicB.ReplaceAllString(s, "$1")
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reListA.ReplaceAllString(s, "• ")
	s = reBlankLines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}
