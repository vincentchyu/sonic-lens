package model

import "context"

func ensureTrackPlayRecordTraceSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&TrackPlayRecord{}, "trace_id") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD COLUMN trace_id VARCHAR(32) NULL COMMENT '播放链路 TraceID' AFTER source",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&TrackPlayRecord{}, "root_span_id") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD COLUMN root_span_id VARCHAR(16) NULL COMMENT '当前播放根 SpanID' AFTER trace_id",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&TrackPlayRecord{}, "trace_sampled") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD COLUMN trace_sampled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '链路是否命中采样' AFTER root_span_id",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&TrackPlayRecord{}, "idx_track_play_records_trace_id") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD INDEX idx_track_play_records_trace_id (trace_id)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
