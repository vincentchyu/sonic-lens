package ai

import (
	"encoding/json"
	"strings"
)

// ParseTrackResult 处理模型返回的内容，提取并解析为 TrackAnalysisResult。
// 包含了多级容错：
// 1. 去除代码块包裹
// 2. 修复可能存在的“顶层对象提前闭合”
// 3. 反序列化
// 4. 若失败则尝试提取 JSON 块再解析
// 5. 将解析后的换行符字面量进行替换
func ParseTrackResult(raw string) (*TrackAnalysisResult, string, error) {
	trimmed := TrimCodeFence(raw)
	normalized := repairPrematureTopLevelObjectClosure(trimmed)

	result := new(TrackAnalysisResult)
	var err error

	if err = json.Unmarshal([]byte(normalized), &result); err != nil {
		if extracted := extractJSON(normalized); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		return nil, normalized, err
	}

SUCCESS:
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")
	return result, normalized, nil
}

// ParseAlbumResult 处理模型返回的内容，提取并解析为 AlbumAnalysisResult。
func ParseAlbumResult(raw string) (*AlbumAnalysisResult, string, error) {
	trimmed := TrimCodeFence(raw)
	normalized := repairPrematureTopLevelObjectClosure(trimmed)

	result := new(AlbumAnalysisResult)

	var err error

	if err = json.Unmarshal([]byte(normalized), &result); err != nil {
		if extracted := extractJSON(normalized); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		return nil, normalized, err
	}

SUCCESS:
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	return result, normalized, nil
}
