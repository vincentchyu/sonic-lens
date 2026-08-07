package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanLRCTimestampsAndHeaders(t *testing.T) {
	t.Parallel()

	input := `[ti:晴天]
[ar:周杰伦]
[al:叶惠美]
[00:28.91]故事的小黄花
[00:32.45]从出生那年就飘着
[00:36.12]童年的荡秋千
[00:40.01]随记忆一直晃到现在`

	expected := `故事的小黄花
从出生那年就飘着
童年的荡秋千
随记忆一直晃到现在`

	actual := CleanLRCTimestampsAndHeaders(input)
	assert.Equal(t, expected, actual)
}

func TestIsChineseText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"纯中文", "故事的小黄花 从出生那年就飘着", true},
		{"中文带少量英文衬词", "草莓 就在一瞬间 划过天际 Oh yeah baby", true},
		{"中文带英文标点数字", "2026年 这是一个测试 123", true},
		{"中英混排大幅英文", "This is an English song translated: 这是中文翻译内容 for testing", false},
		{"纯英文", "On a dark desert highway cool wind in my hair", false},
		{"日歌词含有假名", "さくら さくら 会いたいよ", false},
		{"韩文歌词", "안녕하세요 감사해요 잘있어요", false},
		{"空字符串", "  \n\t ", false},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, IsChineseText(tt.text))
			},
		)
	}
}

func TestIsChineseLyrics(t *testing.T) {
	t.Parallel()

	lrcChinese := `[00:12.34]故事的小黄花
[00:15.67]从出生那年就飘着`
	assert.True(t, IsChineseLyrics(lrcChinese))

	lrcEnglish := `[00:12.34]On a dark desert highway
[00:15.67]Cool wind in my hair`
	assert.False(t, IsChineseLyrics(lrcEnglish))
}

func TestFormatChineseLyricsTranslation(t *testing.T) {
	t.Parallel()

	input := `[00:12.34]故事的小黄花
[00:15.67]从出生那年就飘着`

	expected := `<original>故事的小黄花</original>
<translation></translation>
<original>从出生那年就飘着</original>
<translation></translation>`

	actual := FormatChineseLyricsTranslation(input)
	assert.Equal(t, expected, actual)
}
func TestFormatChineseLyricsTranslation2(t *testing.T) {
	t.Parallel()

	input := `故事的小黄花
从出生那年就飘着`

	expected := `<original>故事的小黄花</original>
<translation></translation>
<original>从出生那年就飘着</original>
<translation></translation>`

	actual := FormatChineseLyricsTranslation(input)
	assert.Equal(t, expected, actual)
}
