package model

import "context"

func ensureTrackIdentitySchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&Track{}, "disc_number") {
		if err := db.Exec("ALTER TABLE track ADD COLUMN disc_number TINYINT DEFAULT 1 COMMENT '碟号' AFTER track_number").Error; err != nil {
			return err
		}
	}
	if migrator.HasIndex(&Track{}, "uidx_artist_album_track") {
		if err := db.Exec("ALTER TABLE track DROP INDEX uidx_artist_album_track").Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&Track{}, "uidx_t_aatdntn") {
		if err := db.Exec(
			"ALTER TABLE track ADD UNIQUE KEY uidx_t_aatdntn (artist, album, track, disc_number, track_number)",
		).Error; err != nil {
			return err
		}
	}

	if !migrator.HasColumn(&TrackPlayRecord{}, "disc_number") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD COLUMN disc_number TINYINT DEFAULT 1 COMMENT '碟号' AFTER track_number",
		).Error; err != nil {
			return err
		}
	}

	if !migrator.HasColumn(&TrackRankStat{}, "track_number") {
		if err := db.Exec("ALTER TABLE track_rank_stat ADD COLUMN track_number TINYINT NULL AFTER track").Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&TrackRankStat{}, "disc_number") {
		if err := db.Exec("ALTER TABLE track_rank_stat ADD COLUMN disc_number TINYINT NULL AFTER track_number").Error; err != nil {
			return err
		}
	}
	if migrator.HasIndex(&TrackRankStat{}, "uk_track_rank_period_track") {
		if err := db.Exec("ALTER TABLE track_rank_stat DROP INDEX uk_track_rank_period_track").Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&TrackRankStat{}, "uk_track_rank_period_track") {
		if err := db.Exec(
			"ALTER TABLE track_rank_stat ADD UNIQUE KEY uk_track_rank_period_track (period_type, artist(191), album(191), track(191), disc_number, track_number)",
		).Error; err != nil {
			return err
		}
	}

	if !migrator.HasIndex(&TrackAlbum{}, "idx_ta_album_disc_track") {
		if err := db.Exec(
			"ALTER TABLE track_album ADD INDEX idx_ta_album_disc_track (album_id, disc_number, track_number)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
