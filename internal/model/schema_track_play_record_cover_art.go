package model

import "context"

func ensureTrackPlayRecordCoverArtSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&TrackPlayRecord{}, "cover_art_path") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD COLUMN cover_art_path VARCHAR(1024) NULL COMMENT '最近播放封面路径' AFTER source",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
