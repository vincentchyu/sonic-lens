package common

import (
	"strings"
	"time"
)

// 中文星期映射到英文简写（Go 可识别）
var weekMap = map[string]string{
	"星期一": "Mon",
	"星期二": "Tue",
	"星期三": "Wed",
	"星期四": "Thu",
	"星期五": "Fri",
	"星期六": "Sat",
	"星期日": "Sun",
}

func ParseChineseTime(s string) (time.Time, error) {
	// 去掉中文星期部分
	for k := range weekMap {
		if strings.Contains(s, k) {
			s = strings.Replace(s, k, "", 1)
			break
		}
	}

	// 去掉多余空格
	s = strings.TrimSpace(s)

	// 对应的 layout
	layout := "2006年1月2日 15:04:05"

	// 解析为本地时间
	t, err := time.ParseInLocation(layout, s, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
