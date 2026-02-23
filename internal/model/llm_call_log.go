package model

import (
	"context"
	"time"
)

// LLMCallLog 大模型调用流水表，记录每次请求/响应的完整 JSON 数据，用于排查和恢复现场
type LLMCallLog struct {
	ID           int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	Provider     string    `gorm:"column:provider;type:varchar(64);index" json:"provider"`      // 提供方：doubao/ollama/openai
	Model        string    `gorm:"column:model;type:varchar(128)" json:"model"`                 // 模型名称
	RequestJSON  string    `gorm:"column:request_json;type:text" json:"request_json"`           // 请求体全文 JSON
	ResponseJSON string    `gorm:"column:response_json;type:text" json:"response_json"`         // 响应体全文 JSON
	Status       string    `gorm:"column:status;type:varchar(32);index" json:"status"`          // 调用状态：success/error
	ErrorMsg     string    `gorm:"column:error_msg;type:text" json:"error_msg"`                 // 错误信息
	DurationMs   int64     `gorm:"column:duration_ms;type:bigint" json:"duration_ms"`           // 调用耗时（毫秒）
	TrackInfo    string    `gorm:"column:track_info;type:varchar(512);index" json:"track_info"` // 关联曲目信息（artist - track）
	CallType     string    `gorm:"column:call_type;type:varchar(32)" json:"call_type"`          // 调用类型：sync/stream
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 自定义表名
func (LLMCallLog) TableName() string {
	return "llm_call_logs"
}

// CreateLLMCallLog 插入一条调用流水记录
func CreateLLMCallLog(ctx context.Context, log *LLMCallLog) error {
	return GetDB().WithContext(ctx).Create(log).Error
}
