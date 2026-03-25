package feishu

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// mentionPlaceholderRegex matches @_user_N placeholders inserted by Feishu for mentions.
var mentionPlaceholderRegex = regexp.MustCompile(`@_user_\d+`)
var markdownLinkRegex = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

// stringValue safely dereferences a *string pointer.
func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// buildMarkdownCard builds a Feishu Interactive Card JSON 2.0 string with markdown content.
// JSON 2.0 cards support full CommonMark standard markdown syntax.
func buildMarkdownCard(content string) (string, error) {
	card := map[string]any{
		"schema": "2.0",
		"body": map[string]any{
			"elements": []map[string]any{
				{
					"tag":     "markdown",
					"content": content,
				},
			},
		},
	}
	data, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type repoRow struct {
	Name        string
	URL         string
	Description string
	Why         string
}

func buildRepoRecommendationCard(content string) (string, bool, error) {
	rows, ok := parseRepoRowsFromMarkdownTable(content)
	if !ok {
		return "", false, nil
	}

	elements := make([]map[string]any, 0, len(rows)+1)
	elements = append(elements, map[string]any{
		"tag":     "markdown",
		"content": "## Repository Recommendations",
	})

	for _, row := range rows {
		var parts []string
		if row.URL != "" {
			parts = append(parts, fmt.Sprintf("**[%s](%s)**", row.Name, row.URL))
		} else {
			parts = append(parts, fmt.Sprintf("**%s**", row.Name))
		}
		if row.Description != "" {
			parts = append(parts, row.Description)
		}
		if row.Why != "" {
			parts = append(parts, "Why: "+row.Why)
		}

		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": strings.Join(parts, "\n"),
		})
	}

	card := map[string]any{
		"schema": "2.0",
		"body": map[string]any{
			"elements": elements,
		},
	}
	data, err := json.Marshal(card)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

func parseRepoRowsFromMarkdownTable(content string) ([]repoRow, bool) {
	lines := strings.Split(content, "\n")
	tableLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			tableLines = append(tableLines, trimmed)
		}
	}
	if len(tableLines) < 3 {
		return nil, false
	}

	header := parseTableRow(tableLines[0])
	if len(header) < 2 || !isTableDivider(tableLines[1]) {
		return nil, false
	}

	repoIdx := findColumnIndex(header, "repository", "repo", "project", "仓库", "项目")
	descIdx := findColumnIndex(header, "description", "summary", "desc", "简介", "说明")
	whyIdx := findColumnIndex(header, "why", "reason", "use case", "notes", "推荐理由", "适用场景")
	if repoIdx < 0 || descIdx < 0 {
		return nil, false
	}

	rows := make([]repoRow, 0, len(tableLines)-2)
	for _, line := range tableLines[2:] {
		cols := parseTableRow(line)
		if repoIdx >= len(cols) || descIdx >= len(cols) {
			continue
		}

		name, url := parseMarkdownLink(cols[repoIdx])
		if name == "" {
			name = stripMd(cols[repoIdx])
		}
		description := stripMd(cols[descIdx])
		why := ""
		if whyIdx >= 0 && whyIdx < len(cols) {
			why = stripMd(cols[whyIdx])
		}
		if name == "" || description == "" {
			continue
		}

		rows = append(rows, repoRow{
			Name:        name,
			URL:         url,
			Description: description,
			Why:         why,
		})
	}

	if len(rows) == 0 {
		return nil, false
	}
	return rows, true
}

func parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	parts := strings.Split(trimmed, "|")
	row := make([]string, 0, len(parts))
	for _, part := range parts {
		row = append(row, strings.TrimSpace(part))
	}
	return row
}

func isTableDivider(line string) bool {
	for _, col := range parseTableRow(line) {
		if col == "" {
			return false
		}
		for _, r := range col {
			if r != '-' && r != ':' && r != ' ' {
				return false
			}
		}
	}
	return true
}

func findColumnIndex(header []string, keywords ...string) int {
	for i, col := range header {
		normalized := strings.ToLower(stripMd(col))
		for _, keyword := range keywords {
			if strings.Contains(normalized, keyword) {
				return i
			}
		}
	}
	return -1
}

func parseMarkdownLink(s string) (string, string) {
	matches := markdownLinkRegex.FindStringSubmatch(strings.TrimSpace(s))
	if len(matches) != 3 {
		return "", ""
	}
	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
}

func stripMd(s string) string {
	s = strings.TrimSpace(s)
	s = markdownLinkRegex.ReplaceAllString(s, "$1")
	replacer := strings.NewReplacer(
		"**", "",
		"__", "",
		"`", "",
		"*", "",
		"_", "",
	)
	s = replacer.Replace(s)
	return strings.TrimSpace(s)
}

// extractJSONStringField unmarshals content as JSON and returns the value of the given string field.
// Returns "" if the content is invalid JSON or the field is missing/empty.
func extractJSONStringField(content, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return ""
	}
	raw, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// extractImageKey extracts the image_key from a Feishu image message content JSON.
// Format: {"image_key": "img_xxx"}
func extractImageKey(content string) string { return extractJSONStringField(content, "image_key") }

// extractFileKey extracts the file_key from a Feishu file/audio message content JSON.
// Format: {"file_key": "file_xxx", "file_name": "...", ...}
func extractFileKey(content string) string { return extractJSONStringField(content, "file_key") }

// extractFileName extracts the file_name from a Feishu file message content JSON.
func extractFileName(content string) string { return extractJSONStringField(content, "file_name") }

// stripMentionPlaceholders removes @_user_N placeholders from the text content.
// These are inserted by Feishu when users @mention someone in a message.
func stripMentionPlaceholders(content string, mentions []*larkim.MentionEvent) string {
	if len(mentions) == 0 {
		return content
	}
	for _, m := range mentions {
		if m.Key != nil && *m.Key != "" {
			content = strings.ReplaceAll(content, *m.Key, "")
		}
	}
	// Also clean up any remaining @_user_N patterns
	content = mentionPlaceholderRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

// extractCardImageKeys recursively extracts all image keys from a Feishu interactive card.
// Image keys are used to download images from Feishu API.
// Returns two slices: Feishu-hosted keys and external URLs.
func extractCardImageKeys(rawContent string) (feishuKeys []string, externalURLs []string) {
	if rawContent == "" {
		return nil, nil
	}

	var card map[string]any
	if err := json.Unmarshal([]byte(rawContent), &card); err != nil {
		return nil, nil
	}

	extractImageKeysRecursive(card, &feishuKeys, &externalURLs)
	return feishuKeys, externalURLs
}

// isExternalURL returns true if the string is an external HTTP/HTTPS URL.
func isExternalURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// extractImageKeysRecursive traverses card structure to find all image keys.
// Collects both Feishu-hosted keys and external URLs separately.
func extractImageKeysRecursive(v any, feishuKeys, externalURLs *[]string) {
	switch val := v.(type) {
	case map[string]any:
		// Check if this is an img element
		if tag, ok := val["tag"].(string); ok {
			switch tag {
			case "img":
				// Try img_key first (always Feishu-hosted)
				if imgKey, ok := val["img_key"].(string); ok && imgKey != "" {
					*feishuKeys = append(*feishuKeys, imgKey)
				}
				// Check src - could be Feishu key or external URL
				if src, ok := val["src"].(string); ok && src != "" {
					if isExternalURL(src) {
						*externalURLs = append(*externalURLs, src)
					} else {
						*feishuKeys = append(*feishuKeys, src)
					}
				}
			case "icon":
				// Icon elements use icon_key
				if iconKey, ok := val["icon_key"].(string); ok && iconKey != "" {
					*feishuKeys = append(*feishuKeys, iconKey)
				}
			}
		}
		// Recurse into all nested structures
		for _, child := range val {
			extractImageKeysRecursive(child, feishuKeys, externalURLs)
		}
	case []any:
		for _, item := range val {
			extractImageKeysRecursive(item, feishuKeys, externalURLs)
		}
	}
}
