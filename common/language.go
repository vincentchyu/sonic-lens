package common

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	// lrcTimestampRegex 用于匹配 LRC 时间轴标记，如 [01:23.45], [00:12]
	lrcTimestampRegex = regexp.MustCompile(`\[\d{2,}:\d{2}(?:\.\d+)?\]`)
	// lrcHeaderRegex 用于匹配 LRC 头部元数据行，如 [ar:歌手], [ti:歌名], [by:作词]
	lrcHeaderRegex = regexp.MustCompile(`^\[[a-zA-Z]+:[^\]]*\]$`)
)

// CleanLRCTimestampsAndHeaders 清洗歌词中的 LRC 时间轴与元数据标示
func CleanLRCTimestampsAndHeaders(lyrics string) string {
	lines := strings.Split(lyrics, "\n")
	var cleanedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 过滤元数据头如 [ar:xxx], [ti:xxx]
		if lrcHeaderRegex.MatchString(trimmed) {
			continue
		}
		// 移除时间轴标记 [00:12.34]
		cleanLine := lrcTimestampRegex.ReplaceAllString(trimmed, "")
		cleanLine = strings.TrimSpace(cleanLine)
		if cleanLine != "" {
			cleanedLines = append(cleanedLines, cleanLine)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

// IsChineseText 判断一段普通文本是否主要为中文
func IsChineseText(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	var totalHan int
	var totalLatin int

	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			totalHan++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			totalLatin++
		} else if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			// 含有日文假名或韩文，直接一票否决
			return false
		}
	}

	if totalHan == 0 {
		return false
	}

	totalAlpha := totalHan + totalLatin
	if totalAlpha == 0 {
		return false
	}

	// 汉字在所有文字/字母字符中的占比超过 80%，或者拉英文总数 <= 20 (通常是少数衬词或标点缩写)
	ratio := float64(totalHan) / float64(totalAlpha)
	return ratio >= 0.80 || totalLatin <= 20
}

// IsChineseLyrics 判断歌词是否为纯中文歌词（在清洗 LRC 标记后进行判定）
func IsChineseLyrics(lyrics string) bool {
	cleaned := CleanLRCTimestampsAndHeaders(lyrics)
	return IsChineseText(cleaned)
}

// FormatChineseLyricsTranslation 将纯中文歌词格式化为带有 <original> 与 <translation> 标签的 Step 1 标准格式
func FormatChineseLyricsTranslation(lyrics string) string {
	cleaned := CleanLRCTimestampsAndHeaders(lyrics)
	if cleaned == "" {
		return ""
	}

	lines := strings.Split(cleaned, "\n")
	var builder strings.Builder

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i > 0 && builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("<original>%s</original>\n<translation></translation>", line))
	}

	return builder.String()
}

// ToASCIISlug 将字符串转换为纯 ASCII 英文 Slug，绝不包含任何汉字或非法字符
func ToASCIISlug(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "unknown"
	}
	var builder strings.Builder
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	res := strings.ToLower(strings.TrimSpace(builder.String()))
	if res == "" {
		// 无 ASCII 字符时使用稳定 Hash 生成 Slug
		var h uint32 = 2166136261
		for _, b := range []byte(trimmed) {
			h ^= uint32(b)
			h *= 16777619
		}
		return fmt.Sprintf("slug-%x", h)
	}
	return res
}
