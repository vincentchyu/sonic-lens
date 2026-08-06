package common

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/mitchellh/mapstructure"
)

func Decode(input interface{}, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			ZeroFields: true,
			TagName:    "json",
			Result:     output,
		},
	)
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}

// ValidateTrackInfo validates the artist, album, and track names
// Returns an error if any of them are empty or contain only whitespace
func ValidateTrackInfo(ctx context.Context, artist, album, track string) error {
	// Trim whitespace from all fields
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	track = strings.TrimSpace(track)

	// Check if any field is empty after trimming
	if artist == "" {
		return errors.New("艺术家名称不能为空")
	}
	if album == "" {
		return errors.New("专辑名称不能为空")
	}
	if track == "" {
		return errors.New("歌曲名称不能为空")
	}

	return nil
}

// 去掉末尾“乐”字
func NormalizeChineseGenre(genre string) string {
	if strings.HasSuffix(genre, "音乐") {
		return genre
	}
	if strings.HasSuffix(genre, "乐") {
		return strings.TrimSuffix(genre, "乐")
	}
	return genre
}

// GenreCustomFit 自定义适配
func GenreCustomFit(genre string) string {
	switch genre {
	case "Rock & Roll":
		return "Rock"
	case "韩国流行乐":
		return "韩语流行乐"
	case "中國搖滾":
		return "Rock"
	case "Singer/Songwriter":
		return "Singer-Songwriter"
	case "R&B/Soul":
		return "R&B-Soul"
	case "Folk-Rock":
		return "Folk Rock"
	case "Rock Music", "Rock Musical":
		return "Rock"
	case "R&B/骚灵乐":
		return "R&B-Soul"
	case "Prog-Rock/Art Rock":
		return "Progressive Rock-Art Rock"
	case "迷幻":
		return "迷幻音乐"
		// todo add
	}
	return genre
}

// ArtistCustomFit 艺术家自定义适配
func ArtistCustomFit(artist string) string {
	switch artist {
	case "Omnipotent Youth Society", "萬能青年旅店":
		return "万能青年旅店"
	case "腰乐队", "腰樂隊":
		return "腰"
	case "诺拉·琼斯", "諾拉·瓊斯":
		return "Norah Jones"
	case "重塑雕像的权利":
		return "Re-TROS"
		// todo add
	}
	return artist
}

// TrackCustomFit 歌曲自定义适配
func TrackCustomFit(name string) string {
	// strings.HasSuffix(name,"Pt. 1")
	switch name {
	case "Another Brick In the Wall, Pt. 1":
		return "Another Brick in the Wall, Part 1"
	case "Another Brick In the Wall, Pt. 2":
		return "Another Brick in the Wall, Part 2"
	case "Another Brick In the Wall, Pt. 3":
		return "Another Brick in the Wall, Part 3"
	case "蚂蚁蚂蚁":
		return "蚂蚁 蚂蚁"
	case "What God Wants, Pt. I":
		return "What God Wants, Part I"
	case "What God Wants, Pt. II":
		return "What God Wants, Part II"
	case "What God Wants, Pt. III":
		return "What God Wants, Part III"
	case "Leave It (A Capella Version)":
		return "Leave It (\"a cappella\" version)"
	case "Leave It (Single Remix)":
		return "Leave It (single remix)"
		// todo add
	}
	return name
}

func UnityFixAll(str string) string {
	// 1  符号统一
	str = UnityPunctuationMarksFix(str)
	// 2 feat统一
	str = UnityFeatFix(str)
	return str
}

// UnityPunctuationMarksFix 替换字符串函数
// ’ => '
// ，=> ,
// （=> (
// ）=> )
// 替换为英文引号
func UnityPunctuationMarksFix(target string) string {
	if strings.Contains(target, "\\'") {
		target = strings.ReplaceAll(target, "\\'", "'")
	}
	if strings.ContainsAny(target, "’‘") {
		target = strings.NewReplacer("’", "'", "‘", "'").Replace(target)
	}
	if strings.ContainsAny(target, "“”") {
		target = strings.NewReplacer("“", "\"", "”", "\"").Replace(target)
	}
	if strings.ContainsAny(target, "，") {
		target = strings.ReplaceAll(target, "，", ",")
	}
	if strings.ContainsAny(target, "（）") {
		target = strings.NewReplacer("（", "(", "）", ")").Replace(target)
	}
	return target
}

// NormalizeForIdentity 提供一种极其松散的归一化结果，用于在身份比对时忽略大小写、空格、引号及常见干扰符。
func NormalizeForIdentity(str string) string {
	str = UnityFixAll(str)
	str = strings.ToLower(str)
	// 移除引号
	str = strings.ReplaceAll(str, "\"", "")
	str = strings.ReplaceAll(str, "'", "")
	// 移除常见干扰标点
	str = strings.NewReplacer(
		"(", " ", ")", " ",
		"[", " ", "]", " ",
		"{", " ", "}", " ",
		".", " ", ",", " ",
		"-", " ", "_", " ",
	).Replace(str)

	// 合并连续空格并 trim
	fields := strings.Fields(str)
	return strings.Join(fields, " ")
}

//	UnityFeatFix
//
// Hikky Burr (feat. Bill Cosby) => Hikky Burr (feat. Bill Cosby)
// 太阳 (feat. Jukka Ahonen) => 太阳 (feat. Jukka Ahonen)
// 太阳(feat.Jukka Ahonen) => 太阳 (feat. Jukka Ahonen)
func UnityFeatFix(title string) string {
	lower := strings.ToLower(title)

	featIdx := strings.Index(lower, "feat")
	if featIdx == -1 {
		return title
	}

	startIdx := strings.LastIndex(title[:featIdx], "(")
	if startIdx == -1 {
		return title
	}

	prefix := title[startIdx+1 : featIdx]
	if strings.TrimSpace(prefix) != "" {
		return title
	}

	endIdx := strings.Index(title[featIdx:], ")")
	if endIdx == -1 {
		return title
	}
	endIdx += featIdx

	content := title[featIdx+4 : endIdx]
	afterFeat := strings.TrimLeft(content, ". ")

	baseTitle := strings.TrimSpace(title[:startIdx])
	remainder := title[endIdx+1:]

	return baseTitle + " (feat. " + afterFeat + ")" + remainder
}

// CapitalizeWords 将字符串按空格分割，并将每个单词的首字母转为大写，其余部分转为小写。
// 例如: "indie pop" -> "Indie Pop", "ALTERNATIVE" -> "Alternative"
func CapitalizeWords(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(strings.ToLower(word))
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
