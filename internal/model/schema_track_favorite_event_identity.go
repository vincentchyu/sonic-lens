package model

import "context"

func ensureTrackFavoriteEventIdentitySchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if migrator.HasIndex(&TrackFavoriteEvent{}, "idx_tfe_identity") {
		if err := db.Exec("ALTER TABLE track_favorite_event DROP INDEX idx_tfe_identity").Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&TrackFavoriteEvent{}, "idx_tfe_identity_subtitle") {
		if err := db.Exec(
			"ALTER TABLE track_favorite_event ADD INDEX idx_tfe_identity_subtitle (artist, album, album_subtitle, track, track_number, disc_number)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
