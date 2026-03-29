package model

import "context"

func ensureAlbumCoverSchema(ctx context.Context) error {
	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasColumn(&Album{}, "cover_art_url") {
		if err := db.Exec(
			"ALTER TABLE album ADD COLUMN cover_art_url VARCHAR(1024) NULL COMMENT '专辑封面访问地址' AFTER sync_status",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&Album{}, "cover_art_mime") {
		if err := db.Exec(
			"ALTER TABLE album ADD COLUMN cover_art_mime VARCHAR(128) NULL COMMENT '专辑封面 MIME' AFTER cover_art_url",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasColumn(&Album{}, "cover_art_object_key") {
		if err := db.Exec(
			"ALTER TABLE album ADD COLUMN cover_art_object_key VARCHAR(512) NULL COMMENT '专辑封面对象存储键' AFTER cover_art_mime",
		).Error; err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&Album{}, "idx_album_cover_art_object_key") {
		if err := db.Exec(
			"ALTER TABLE album ADD INDEX idx_album_cover_art_object_key (cover_art_object_key)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}
