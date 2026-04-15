package common

/*
import (
	"regexp"
	"strings"
	"unicode"
)

type AlbumTitleVersionType string

const (
	AlbumTitleVersionTypeEdition    AlbumTitleVersionType = "edition"
	AlbumTitleVersionTypeRemaster   AlbumTitleVersionType = "remaster"
	AlbumTitleVersionTypeSoundtrack AlbumTitleVersionType = "soundtrack"
	AlbumTitleVersionTypeMix        AlbumTitleVersionType = "mix"
	AlbumTitleVersionTypeLive       AlbumTitleVersionType = "live"
	AlbumTitleVersionTypeVersion    AlbumTitleVersionType = "version"
	AlbumTitleVersionTypeOther      AlbumTitleVersionType = "other"
)

type AlbumTitleVersion struct {
	Text          string                `json:"text"`
	Type          AlbumTitleVersionType `json:"type"`
	Bracketed     bool                  `json:"bracketed"`
	Parenthesized bool                  `json:"parenthesized"`
}

type AlbumTitleMetadata struct {
	SourceDisplayTitle     string              `json:"source_display_title"`
	OfficialTitle          string              `json:"official_title"`
	TitleVersions          []AlbumTitleVersion `json:"title_versions"`
	NormalizedDisplayTitle string              `json:"normalized_display_title"`
}

type subtitleRule struct {
	Pattern *regexp.Regexp
	Type    AlbumTitleVersionType
}

// 宽松正则：只匹配标准版本关键词（修复你最早的解析失败问题）
var albumTitleSubtitleRules = []subtitleRule{
	{Pattern: regexp.MustCompile(`(?i)\bremaster(?:ed|s)?\b`), Type: AlbumTitleVersionTypeRemaster},
	{
		Pattern: regexp.MustCompile(`(?i)\b(?:anniversary|deluxe|edition|expanded|bonus)\b`),
		Type:    AlbumTitleVersionTypeEdition,
	},
	{Pattern: regexp.MustCompile(`(?i)\b(?:mix|remix)(?:es)?\b`), Type: AlbumTitleVersionTypeMix},
	{Pattern: regexp.MustCompile(`(?i)\blive\b`), Type: AlbumTitleVersionTypeLive},
	{Pattern: regexp.MustCompile(`(?i)\bsoundtrack\b`), Type: AlbumTitleVersionTypeSoundtrack},
	{Pattern: regexp.MustCompile(`(?i)\bversion\b`), Type: AlbumTitleVersionTypeVersion},
}

func ParseAlbumTitleMetadata(raw string) AlbumTitleMetadata {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AlbumTitleMetadata{}
	}

	base := raw
	var versions []AlbumTitleVersion

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
		// 核心：只有匹配到规则，才拆分；不匹配则直接保留原括号
		version, ok := classifyAlbumTitleVersion(content, opening, closing)
		if !ok {
			goto done
		}

		versions = append([]AlbumTitleVersion{version}, versions...)
		base = strings.TrimSpace(trimmed[:start])
	}

done:
	if len(versions) == 0 || strings.TrimSpace(base) == "" {
		return AlbumTitleMetadata{
			SourceDisplayTitle:     raw,
			OfficialTitle:          raw,
			TitleVersions:          nil,
			NormalizedDisplayTitle: raw,
		}
	}

	return AlbumTitleMetadata{
		SourceDisplayTitle:     raw,
		OfficialTitle:          base,
		TitleVersions:          versions,
		NormalizedDisplayTitle: buildNormalizedAlbumDisplayTitle(base, versions),
	}
}

// 恢复：不匹配规则返回 false，不拆分标题
func classifyAlbumTitleVersion(value string, opening, closing byte) (AlbumTitleVersion, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return AlbumTitleVersion{}, false
	}

	for _, rule := range albumTitleSubtitleRules {
		if rule.Pattern.MatchString(normalized) {
			return AlbumTitleVersion{
				Text:          normalized,
				Type:          rule.Type,
				Bracketed:     opening == '[' && closing == ']',
				Parenthesized: opening == '(' && closing == ')',
			}, true
		}
	}

	// 无匹配关键词 → 不识别为副标题
	return AlbumTitleVersion{}, false
}

func buildNormalizedAlbumDisplayTitle(base string, versions []AlbumTitleVersion) string {
	if len(versions) == 0 {
		return base
	}
	var b strings.Builder
	b.WriteString(base)
	for _, v := range versions {
		b.WriteByte(' ')
		if v.Parenthesized {
			b.WriteByte('(')
			b.WriteString(v.Text)
			b.WriteByte(')')
		} else {
			b.WriteByte('[')
			b.WriteString(v.Text)
			b.WriteByte(']')
		}
	}
	return b.String()
}

// 兼容旧接口
func ParseAlbumTitleAndSubtitle(raw string) (string, string) {
	meta := ParseAlbumTitleMetadata(raw)
	if len(meta.TitleVersions) == 0 {
		return meta.OfficialTitle, ""
	}
	var parts []string
	for _, v := range meta.TitleVersions {
		parts = append(parts, v.Text)
	}
	return meta.OfficialTitle, strings.Join(parts, " ")
}
*/
