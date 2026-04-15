package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringArray 用 JSON 数组的形式持久化字符串切片，便于在反馈等场景复用。
type StringArray []string

// Value 将字符串切片序列化为数据库可存储的 JSON 字符串。
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}

	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan 将数据库字段反序列化为字符串切片。
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*s = nil
			return nil
		}
		return json.Unmarshal(v, s)
	case string:
		if v == "" {
			*s = nil
			return nil
		}
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported StringArray scan type %T", value)
	}
}
