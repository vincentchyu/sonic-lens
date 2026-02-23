package common

import (
	"strings"
	"unicode"

	"github.com/longbridgeapp/opencc"
)

var (
	s2t *opencc.OpenCC
	t2s *opencc.OpenCC
)

func init() {
	var err error
	s2t, err = opencc.New("s2t")
	if err != nil {
		panic(err)
	}
	t2s, err = opencc.New("t2s")
	if err != nil {
		panic(err)
	}
}

func ConversionSimplifiedFx(target string) string {
	// 检查字符串是否为空或只包含空白字符
	if strings.TrimSpace(target) == "" {
		return target
	}

	// 使用unicode包判断字符串是否包含中文字符
	hasChinese := false
	for _, r := range target {
		if unicode.Is(unicode.Han, r) {
			hasChinese = true
			break
		}
	}

	if !hasChinese {
		return target
	}

	// 检查是否已经是简体字，如果是则直接返回
	/*if isChineseSimplified(target) {
		return target
	}*/

	// 先尝试繁体转简体
	simplified, err := t2s.Convert(target)
	if err != nil {
		// 转换失败则返回原字符串
		return target
	}

	return simplified
}

// IsExistsChineseSimplified 检查字符串是否存在简体字
func IsExistsChineseSimplified(s string) bool {
	// 常见的简体字Unicode范围
	simplifiedRanges := [][2]rune{
		{0x4E00, 0x9FA5}, // 基本汉字
		{0x9FA6, 0x9FCB}, // 基本汉字补充
		{0x3400, 0x4DB5}, // 扩展A
	}

	for _, r := range s {
		// 如果字符是中文字符
		if unicode.Is(unicode.Han, r) {
			// 检查是否在简体字范围内
			inSimplifiedRange := false
			for _, rng := range simplifiedRanges {
				if r >= rng[0] && r <= rng[1] {
					inSimplifiedRange = true
					break
				}
			}
			// 如果不在简体字范围内，则不是简体字符串
			if inSimplifiedRange {
				return true
			}
		}
	}
	return false
}
