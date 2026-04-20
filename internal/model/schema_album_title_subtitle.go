package model

import "context"

func ensureAlbumTitleSubtitleSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&Album{}, "name_subtitle") {
		if err := db.Exec(
			"ALTER TABLE album ADD COLUMN name_subtitle VARCHAR(255) NULL COMMENT '专辑补充说明' AFTER name",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&TrackPlayRecord{}, "album_subtitle") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD COLUMN album_subtitle VARCHAR(255) NULL COMMENT '专辑补充说明' AFTER album",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&Track{}, "album_subtitle") {
		if err := db.Exec(
			"ALTER TABLE track ADD COLUMN album_subtitle VARCHAR(255) NULL COMMENT '专辑补充说明' AFTER album",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&TrackFavoriteEvent{}, "album_subtitle") {
		if err := db.Exec(
			"ALTER TABLE track_favorite_event ADD COLUMN album_subtitle VARCHAR(255) NULL COMMENT '专辑补充说明' AFTER album",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&PendingAlbumWorkItem{}, "album_subtitle") {
		if err := db.Exec(
			"ALTER TABLE pending_album_work_item ADD COLUMN album_subtitle VARCHAR(255) NULL COMMENT '专辑补充说明' AFTER album",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&TopAlbumStat{}, "album_subtitle") {
		if err := db.Exec(
			"ALTER TABLE top_album_stat ADD COLUMN album_subtitle VARCHAR(255) NULL COMMENT '专辑补充说明' AFTER album",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
