package model

import "context"

func ensureLLMCallLogSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&LLMCallLog{}, "analysis_target_type") {
		if err := db.Exec(
			"ALTER TABLE llm_call_logs ADD COLUMN analysis_target_type VARCHAR(32) NOT NULL DEFAULT 'track' COMMENT '分析对象类型: track/album' AFTER duration_ms",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&LLMCallLog{}, "target_key") {
		if err := db.Exec(
			"ALTER TABLE llm_call_logs ADD COLUMN target_key VARCHAR(512) NOT NULL DEFAULT '' COMMENT '对象唯一标识' AFTER analysis_target_type",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&LLMCallLog{}, "target_metadata") {
		if err := db.Exec(
			"ALTER TABLE llm_call_logs ADD COLUMN target_metadata LONGTEXT COMMENT '对象元数据 JSON' AFTER target_key",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&LLMCallLog{}, "idx_llm_logs_target_type") {
		if err := db.Exec(
			"ALTER TABLE llm_call_logs ADD INDEX idx_llm_logs_target_type (analysis_target_type)",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&LLMCallLog{}, "idx_llm_logs_target_key") {
		if err := db.Exec(
			"ALTER TABLE llm_call_logs ADD INDEX idx_llm_logs_target_key (target_key)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
