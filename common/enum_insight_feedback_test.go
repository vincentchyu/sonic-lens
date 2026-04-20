package common

import (
	"reflect"
	"testing"
)

func TestParseInsightFeedbackReason(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected InsightFeedbackReason
	}{
		{name: "inaccurate", input: "不准确", expected: InsightFeedbackReasonInaccurate},
		{name: "trimmed", input: " 太空泛 ", expected: InsightFeedbackReasonVague},
		{name: "invalid", input: "跑题", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseInsightFeedbackReason(tt.input); got != tt.expected {
				t.Fatalf("ParseInsightFeedbackReason() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalizeInsightFeedbackReasons(t *testing.T) {
	got := NormalizeInsightFeedbackReasons(
		[]string{"结构混乱", "太空泛", " 跑题 ", "太空泛", "不准确", "其他"},
	)
	want := []string{"不准确", "太空泛", "结构混乱", "其他"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeInsightFeedbackReasons() = %#v, want %#v", got, want)
	}
}
