package model

import "context"

func ensureAlbumIdentitySchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if migrator.HasIndex(&Album{}, "uidx_album_artist_name_release_date") {
		if err := db.Exec("ALTER TABLE album DROP INDEX uidx_album_artist_name_release_date").Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&Album{}, "uidx_album_artist_name_subtitle_release_date") {
		if err := db.Exec(
			"ALTER TABLE album ADD UNIQUE KEY uidx_album_artist_name_subtitle_release_date (artist, name, name_subtitle, release_date)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
