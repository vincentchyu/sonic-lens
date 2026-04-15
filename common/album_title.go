package common

import (
	"regexp"
	"strings"
	"unicode"
)

var albumTitleSubtitlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bremaster(?:ed)?(?:\s+version)?\b`),
	regexp.MustCompile(`(?i)\b\d{4}\s+remaster(?:ed)?(?:\s+version)?\b`),
	regexp.MustCompile(`(?i)\b(?:expanded|deluxe|super deluxe|bonus|collector'?s|ultimate|anniversary)\b.*\bedition\b`),
	regexp.MustCompile(`(?i)\b\d{1,3}(?:st|nd|rd|th)\s+anniversary\b`),
	regexp.MustCompile(`(?i)^bonus\s+(?:track|video)\s+version$`),
	regexp.MustCompile(`(?i)^audio\s+version$`),
	regexp.MustCompile(`(?i)^original\s+motion\s+picture\s+soundtrack$`),
	regexp.MustCompile(`(?i)\bremix\b`),
	regexp.MustCompile(`(?i)^(?:dj|mono|stereo|original album|original|new|\d{4})\s+mix$`),
	regexp.MustCompile(`(?i)^(?:.+\s+)?version$`),
	regexp.MustCompile(`(?i)^live(?:\s+at\b.*)?$`),
}

// ParseAlbumTitleAndSubtitle 将展示专辑名拆成主名和补充说明。
func ParseAlbumTitleAndSubtitle_Tmp(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	base := raw
	segments := make([]string, 0, 2)

	for {
		trimmed := strings.TrimRightFunc(base, unicode.IsSpace)
		if trimmed == "" {
			break
		}

		end := len(trimmed) - 1
		closing := trimmed[end]
		var opening byte
		switch closing {
		case ')':
			opening = '('
		case ']':
			opening = '['
		default:
			goto done
		}

		start := strings.LastIndexByte(trimmed[:end], opening)
		if start < 0 {
			goto done
		}
		if start == 0 || !unicode.IsSpace(rune(trimmed[start-1])) {
			goto done
		}

		content := strings.TrimSpace(trimmed[start+1 : end])
		if !isAlbumSubtitleToken(content) {
			goto done
		}

		segment := content
		if opening == '[' {
			segment = "[" + segment + "]"
		}
		segments = append([]string{segment}, segments...)
		base = strings.TrimSpace(trimmed[:start])
	}

done:
	if len(segments) == 0 || strings.TrimSpace(base) == "" {
		return raw, ""
	}
	return base, strings.Join(segments, " ")
}

func isAlbumSubtitleToken(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}

	for _, pattern := range albumTitleSubtitlePatterns {
		if pattern.MatchString(normalized) {
			return true
		}
	}
	return false
}
