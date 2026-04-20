package common

import "strings"

// InsightFeedbackReason 定义音眸反馈原因枚举，需与 Bridge 端保持一致。
type InsightFeedbackReason string

const (
	InsightFeedbackReasonInaccurate     InsightFeedbackReason = "不准确"
	InsightFeedbackReasonVague          InsightFeedbackReason = "太空泛"
	InsightFeedbackReasonNotRelevant    InsightFeedbackReason = "不贴合歌曲/专辑"
	InsightFeedbackReasonMissingInfo    InsightFeedbackReason = "缺少关键信息"
	InsightFeedbackReasonMessyStructure InsightFeedbackReason = "结构混乱"
	InsightFeedbackReasonOther          InsightFeedbackReason = "其他"
)

var insightFeedbackReasonOrder = []InsightFeedbackReason{
	InsightFeedbackReasonInaccurate,
	InsightFeedbackReasonVague,
	InsightFeedbackReasonNotRelevant,
	InsightFeedbackReasonMissingInfo,
	InsightFeedbackReasonMessyStructure,
	InsightFeedbackReasonOther,
}

// AllInsightFeedbackReasons 返回当前系统支持的反馈原因，顺序需与客户端一致。
func AllInsightFeedbackReasons() []InsightFeedbackReason {
	result := make([]InsightFeedbackReason, len(insightFeedbackReasonOrder))
	copy(result, insightFeedbackReasonOrder)
	return result
}

// ParseInsightFeedbackReason 将外部输入归一为合法反馈原因。
func ParseInsightFeedbackReason(value string) InsightFeedbackReason {
	trimmed := strings.TrimSpace(value)
	switch InsightFeedbackReason(trimmed) {
	case InsightFeedbackReasonInaccurate:
		return InsightFeedbackReasonInaccurate
	case InsightFeedbackReasonVague:
		return InsightFeedbackReasonVague
	case InsightFeedbackReasonNotRelevant:
		return InsightFeedbackReasonNotRelevant
	case InsightFeedbackReasonMissingInfo:
		return InsightFeedbackReasonMissingInfo
	case InsightFeedbackReasonMessyStructure:
		return InsightFeedbackReasonMessyStructure
	case InsightFeedbackReasonOther:
		return InsightFeedbackReasonOther
	default:
		return ""
	}
}

// IsValid 判断是否为合法反馈原因。
func (r InsightFeedbackReason) IsValid() bool {
	return ParseInsightFeedbackReason(string(r)) != ""
}

// NormalizeInsightFeedbackReasons 过滤非法值、去重，并按枚举定义顺序返回。
func NormalizeInsightFeedbackReasons(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[InsightFeedbackReason]struct{}, len(values))
	for _, value := range values {
		reason := ParseInsightFeedbackReason(value)
		if !reason.IsValid() {
			continue
		}
		seen[reason] = struct{}{}
	}

	result := make([]string, 0, len(seen))
	for _, reason := range insightFeedbackReasonOrder {
		if _, ok := seen[reason]; ok {
			result = append(result, string(reason))
		}
	}
	return result
}
