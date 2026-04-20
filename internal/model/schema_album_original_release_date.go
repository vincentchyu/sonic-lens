package model

import "context"

func ensureAlbumOriginalReleaseDateSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&Album{}, "original_release_date") {
		if err := db.Exec(
			"ALTER TABLE album ADD COLUMN original_release_date VARCHAR(50) NULL COMMENT '专辑首发日期' AFTER release_date",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
