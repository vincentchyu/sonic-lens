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

// releaseTypeSuffixRe 匹配 Apple Music 连字符发行类型后缀，例如 " - EP"、" - Single"、" - LP"。
// Apple Music 规则：纯专辑类型时在专辑名末尾附加 " - <Type>"，其中 Type 为 EP/Single/LP 等。
var releaseTypeSuffixRe = regexp.MustCompile(`(?i)\s+-\s+(EP|Single|LP)\s*$`)

// normalizeReleaseType 将匹配到的后缀规范为小写枚举值。
func normalizeReleaseType(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ParseAlbumTitleMetadata 解析专辑标题
func ParseAlbumTitleMetadata(raw string) AlbumTitleMetadata {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AlbumTitleMetadata{}
	}

	base := raw
	var versions []AlbumTitleVersion

	// 第一步：先提取括号内的版本说明（Remaster/Deluxe 等），与原逻辑保持一致。
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
		{
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
	}

done:
	// 第二步：在剥除括号后缀后，再对 base 检测连字符发行类型后缀（EP/Single/LP）。
	// 顺序在括号剥离之后，以便 "Flowers - EP (Deluxe)" 能先去掉括号再识别 EP 后缀。
	var releaseType string
	if loc := releaseTypeSuffixRe.FindStringIndex(base); loc != nil {
		suffixMatch := releaseTypeSuffixRe.FindStringSubmatch(base)
		if len(suffixMatch) >= 2 {
			releaseType = normalizeReleaseType(suffixMatch[1])
		}
		base = strings.TrimSpace(base[:loc[0]])
	}

	// 兜底：如果没有提取到任何信息，直接返回原标题
	if (len(versions) == 0 || strings.TrimSpace(base) == "") && releaseType == "" {
		return AlbumTitleMetadata{
			SourceDisplayTitle:     raw,
			OfficialTitle:          raw,
			TitleVersions:          nil,
			NormalizedDisplayTitle: raw,
		}
	}

	// base 剥离后为空说明原标题本身就是 release type 后缀（不合理），回退
	officialTitle := base
	if strings.TrimSpace(officialTitle) == "" {
		return AlbumTitleMetadata{
			SourceDisplayTitle:     raw,
			OfficialTitle:          raw,
			TitleVersions:          nil,
			NormalizedDisplayTitle: raw,
		}
	}

	if len(versions) == 1 && versions[0].Type == AlbumTitleVersionTypeOther && releaseType == "" {
		return AlbumTitleMetadata{
			SourceDisplayTitle:     raw,
			OfficialTitle:          raw,
			TitleVersions:          nil,
			NormalizedDisplayTitle: raw,
		}
	}

	return AlbumTitleMetadata{
		SourceDisplayTitle:     raw,
		OfficialTitle:          officialTitle,
		TitleVersions:          versions,
		NormalizedDisplayTitle: buildNormalizedAlbumDisplayTitle(officialTitle, versions),
		ReleaseType:            releaseType,
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

// ParseAlbumTitleAndReleaseType 从专辑原始名称中提取主标题与发行类型枚举。
// 主要用于处理 Apple Music 上报的 " - EP"、" - Single" 等连字符后缀。
// 返回值：
//   - title：剥离后缀后的主标题（如 "In The Sun"）
//   - releaseType：小写发行类型枚举（如 "ep"、"single"、"lp"），无后缀时为空字符串
func ParseAlbumTitleAndReleaseType(raw string) (title string, releaseType string) {
	meta := ParseAlbumTitleMetadata(raw)
	return meta.OfficialTitle, meta.ReleaseType
}
