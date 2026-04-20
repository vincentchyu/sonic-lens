package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/vincentchyu/sonic-lens/common"
)

// AlbumTitleMetadataJSON 用结构化 JSON 承载专辑标题元数据，避免把 JSON 当字符串在业务层来回拼接。
type AlbumTitleMetadataJSON common.AlbumTitleMetadata

// AlbumTitleMetadataJSONFromCommon 将公共结构转换为可持久化的模型结构。
func AlbumTitleMetadataJSONFromCommon(metadata *common.AlbumTitleMetadata) *AlbumTitleMetadataJSON {
	if metadata == nil {
		return nil
	}
	value := AlbumTitleMetadataJSON(*metadata)
	return &value
}

// ToCommon 将模型结构还原为公共结构，方便上层业务复用解析结果。
func (m *AlbumTitleMetadataJSON) ToCommon() *common.AlbumTitleMetadata {
	if m == nil {
		return nil
	}
	value := common.AlbumTitleMetadata(*m)
	return &value
}

// Value 将结构化元数据序列化为数据库可存储的 JSON 文本。
func (m AlbumTitleMetadataJSON) Value() (driver.Value, error) {
	raw, err := json.Marshal(common.AlbumTitleMetadata(m))
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

// Scan 将数据库中的 JSON 文本反序列化回结构化元数据。
func (m *AlbumTitleMetadataJSON) Scan(value interface{}) error {
	if m == nil {
		return fmt.Errorf("album title metadata scan on nil receiver")
	}
	if value == nil {
		*m = AlbumTitleMetadataJSON{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*m = AlbumTitleMetadataJSON{}
			return nil
		}
		return json.Unmarshal(v, (*common.AlbumTitleMetadata)(m))
	case string:
		if v == "" {
			*m = AlbumTitleMetadataJSON{}
			return nil
		}
		return json.Unmarshal([]byte(v), (*common.AlbumTitleMetadata)(m))
	default:
		return fmt.Errorf("unsupported AlbumTitleMetadataJSON scan type %T", value)
	}
}
