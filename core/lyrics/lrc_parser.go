package lyrics

import (
	"regexp"
	"strconv"
	"strings"
)

var timeTagRegex = regexp.MustCompile(`\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?\]`)

// ParsedLine 表示一行 LRC 展开的同步歌词行。
type ParsedLine struct {
	TimeMs int64
	Text   string
}

// ParseLRC 解析 LRC 文本，展开一行内的多个时间标签。
func ParseLRC(text string) []ParsedLine {
	lines := strings.Split(text, "\n")
	result := make([]ParsedLine, 0, len(lines))

	for _, rawLine := range lines {
		matches := timeTagRegex.FindAllStringSubmatch(rawLine, -1)
		if len(matches) == 0 {
			continue
		}

		cleanText := strings.TrimSpace(timeTagRegex.ReplaceAllString(rawLine, ""))
		if cleanText == "" {
			continue
		}

		for _, match := range matches {
			timeMs, ok := parseTimeTagParts(match[1], match[2], match[3])
			if !ok {
				continue
			}
			result = append(result, ParsedLine{
				TimeMs: timeMs,
				Text:   cleanText,
			})
		}
	}

	return result
}

// IsSyncedLRC 判断文本是否包含至少一个合法时间标签。
func IsSyncedLRC(text string) bool {
	return len(ParseLRC(text)) > 0
}

func parseTimeTagParts(minuteText, secondText, fractionText string) (int64, bool) {
	minutes, err := strconv.Atoi(minuteText)
	if err != nil {
		return 0, false
	}
	seconds, err := strconv.Atoi(secondText)
	if err != nil {
		return 0, false
	}
	if seconds < 0 || seconds >= 60 {
		return 0, false
	}

	milliseconds := 0
	if fractionText != "" {
		msText := fractionText
		for len(msText) < 3 {
			msText += "0"
		}
		milliseconds, err = strconv.Atoi(msText)
		if err != nil {
			return 0, false
		}
	}

	return int64(minutes)*60*1000 + int64(seconds)*1000 + int64(milliseconds), true
}
