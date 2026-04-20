package common

import (
	"regexp"
	"strings"
	"unicode"
)

type subtitleRule struct {
	Pattern *regexp.Regexp
	Type    AlbumTitleVersionType
}

// ====================== 核心修改1：重构正则规则（宽松匹配，覆盖所有变体） ======================
var albumTitleSubtitleRules = []subtitleRule{
	// 重制版：匹配 remaster/remastered/remasters（任意位置、任意语序、复数）
	{
		Pattern: regexp.MustCompile(`(?i)\bremaster(?:ed|s)?\b`),
		Type:    AlbumTitleVersionTypeRemaster,
	},
	// 版本/周年版：匹配 anniversary/deluxe/edition/expanded/bonus
	{
		Pattern: regexp.MustCompile(`(?i)\b(?:anniversary|deluxe|edition|expanded|bonus)\b`),
		Type:    AlbumTitleVersionTypeEdition,
	},
	// 混音版：匹配 mix/mixes/remix/remixes
	{
		Pattern: regexp.MustCompile(`(?i)\b(?:mix|remix)(?:es)?\b`),
		Type:    AlbumTitleVersionTypeMix,
	},
	// 现场版
	{
		Pattern: regexp.MustCompile(`(?i)\blive\b`),
		Type:    AlbumTitleVersionTypeLive,
	},
	// 原声带
	{
		Pattern: regexp.MustCompile(`(?i)\bsoundtrack\b`),
		Type:    AlbumTitleVersionTypeSoundtrack,
	},
	// 版本：匹配 version（含法语变体）
	{
		Pattern: regexp.MustCompile(`(?i)\bversion\b`),
		Type:    AlbumTitleVersionTypeVersion,
	},
}

// ParseAlbumTitleMetadata 解析专辑标题
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

		// 查找最后一个匹配的左括号
		start := strings.LastIndexByte(trimmed[:end], opening)
		if start < 0 {
			goto done
		}

		// 必须是「空格+括号」结构，避免误拆主标题
		if start == 0 || !unicode.IsSpace(rune(trimmed[start-1])) {
			goto done
		}

		content := strings.TrimSpace(trimmed[start+1 : end])
		// ====================== 核心修改2：永远提取有效括号内容（不再丢弃） ======================
		version := classifyAlbumTitleVersion(content, opening, closing)
		versions = append([]AlbumTitleVersion{version}, versions...)
		base = strings.TrimSpace(trimmed[:start])
	}

done:
	// 兜底：如果没有提取到版本，直接返回原标题
	if len(versions) == 0 || strings.TrimSpace(base) == "" {
		return AlbumTitleMetadata{
			SourceDisplayTitle:     raw,
			OfficialTitle:          raw,
			TitleVersions:          nil,
			NormalizedDisplayTitle: raw,
		}
	}

	if len(versions) == 1 && versions[0].Type == AlbumTitleVersionTypeOther {
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

// ====================== 核心修改3：永远返回有效版本，兜底为Other类型 ======================
func classifyAlbumTitleVersion(value string, opening, closing byte) AlbumTitleVersion {
	normalized := strings.TrimSpace(value)
	version := AlbumTitleVersion{
		Text:          normalized,
		Bracketed:     opening == '[' && closing == ']',
		Parenthesized: opening == '(' && closing == ')',
		Type:          AlbumTitleVersionTypeOther, // 默认兜底类型
	}

	// 匹配预定义规则
	for _, rule := range albumTitleSubtitleRules {
		if rule.Pattern.MatchString(normalized) {
			version.Type = rule.Type
			break
		}
	}

	return version
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

// ParseAlbumTitleAndSubtitle 兼容旧接口
func ParseAlbumTitleAndSubtitle(raw string) (string, string) {
	meta := ParseAlbumTitleMetadata(raw)
	if len(meta.TitleVersions) == 0 {
		return meta.OfficialTitle, ""
	}
	if len(meta.TitleVersions) == 1 {
		return meta.OfficialTitle, meta.TitleVersions[0].Text
	}

	parts := make([]string, 0, len(meta.TitleVersions))
	for _, v := range meta.TitleVersions {
		switch {
		case v.Parenthesized:
			parts = append(parts, "("+v.Text+")")
		default:
			parts = append(parts, "["+v.Text+"]")
		}
	}

	return meta.OfficialTitle, strings.Join(parts, " ")
}
