package model

import "context"

func ensureAlbumTitleMetadataSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&Album{}, "title_metadata") {
		if err := db.Exec(
			"ALTER TABLE album ADD COLUMN title_metadata LONGTEXT NULL COMMENT '专辑标题详细元数据(JSON)' AFTER name_subtitle",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
