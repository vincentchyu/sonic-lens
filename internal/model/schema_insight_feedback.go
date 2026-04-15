package model

import "context"

func ensureInsightFeedbackSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if err := ensureSingleInsightFeedbackTableSchema(ctx, migrator, "track_insight_feedbacks"); err != nil {
		return err
	}
	if err := ensureSingleInsightFeedbackTableSchema(ctx, migrator, "album_insight_feedbacks"); err != nil {
		return err
	}

	return nil
}

func ensureSingleInsightFeedbackTableSchema(ctx context.Context, migrator interface {
	HasTable(dst interface{}) bool
	HasColumn(dst interface{}, field string) bool
}, tableName string) error {
	db := GetDB().WithContext(ctx)
	if !migrator.HasTable(tableName) {
		return nil
	}

	if !migrator.HasColumn(tableName, "reason_codes") {
		if err := db.Exec(
			"ALTER TABLE " + tableName + " ADD COLUMN reason_codes LONGTEXT COMMENT '结构化问题标签 JSON 数组' AFTER comment",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(tableName, "section_key") {
		if err := db.Exec(
			"ALTER TABLE " + tableName + " ADD COLUMN section_key VARCHAR(128) NOT NULL DEFAULT '' COMMENT '指向问题分区的标识' AFTER reason_codes",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(tableName, "source_platform") {
		if err := db.Exec(
			"ALTER TABLE " + tableName + " ADD COLUMN source_platform VARCHAR(64) NOT NULL DEFAULT '' COMMENT '反馈来源终端' AFTER section_key",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
