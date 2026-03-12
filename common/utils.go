package common

import (
	"context"
	"errors"
	"strings"

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
	case "R&B/骚灵乐":
		return "R&B-Soul"
	case "Prog-Rock/Art Rock":
		return "Progressive Rock-Art Rock"
		// todo add
	}
	return genre
}

// ArtistCustomFit 艺术家自定义适配
func ArtistCustomFit(artist string) string {
	switch artist {
	case "Omnipotent Youth Society":
		return "万能青年旅店"
		// todo add
	}
	return artist
}

// TrackCustomFit 歌曲自定义适配
func TrackCustomFit(artist string) string {
	switch artist {
	case "Another Brick In the Wall, Pt. 1":
		return "Another Brick in the Wall, Part 1"
	case "Another Brick In the Wall, Pt. 2":
		return "Another Brick in the Wall, Part 2"
	case "Another Brick In the Wall, Pt. 3":
		return "Another Brick in the Wall, Part 3"
		// todo add
	}
	return artist
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
// 替换为英文引号
func UnityPunctuationMarksFix(target string) string {
	if strings.ContainsAny(target, "’") {
		target = strings.ReplaceAll(target, "’", "'")
	}
	if strings.ContainsAny(target, "，") {
		target = strings.ReplaceAll(target, "，", ",")
	}
	return target
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
