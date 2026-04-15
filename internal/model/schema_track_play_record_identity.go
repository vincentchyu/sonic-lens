package model

import "context"

func ensureTrackPlayRecordIdentitySchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasIndex(&TrackPlayRecord{}, "idx_track_play_records_identity_subtitle") {
		if err := db.Exec(
			"ALTER TABLE track_play_records ADD INDEX idx_track_play_records_identity_subtitle (artist, album, album_subtitle, track, track_number, disc_number, source, resolution_status, library_applied, play_time)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
