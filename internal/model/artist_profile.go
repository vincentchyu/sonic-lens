package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
)

// ArtistProfile 维护艺术家轻量资料，供首页热门艺术家在响应层补充头像信息。
type ArtistProfile struct {
	ID                  int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	ArtistName          string    `gorm:"column:artist_name;type:varchar(255);not null;uniqueIndex:uk_artist_profile_artist_name" json:"artist_name"`
	NormalizedArtistKey string    `gorm:"column:normalized_artist_key;type:varchar(255);not null;uniqueIndex:uk_artist_profile_normalized_key" json:"normalized_artist_key"`
	AvatarURL           string    `gorm:"column:avatar_url;type:varchar(1024)" json:"avatar_url"`
	AvatarMime          string    `gorm:"column:avatar_mime;type:varchar(128)" json:"avatar_mime"`
	AvatarObjectKey     string    `gorm:"column:avatar_object_key;type:varchar(512);index:idx_artist_profile_avatar_object_key" json:"avatar_object_key"`
	CreatedAt           time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (ArtistProfile) TableName() string {
	return "artist_profile"
}

// ListArtistProfiles 返回艺术家资料分页结果。
func ListArtistProfiles(ctx context.Context, limit, offset int, keyword string) ([]ArtistProfile, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	db := GetDB().WithContext(ctx).Model(&ArtistProfile{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		normalized := NormalizeArtistProfileKey(keyword)
		if normalized != "" {
			db = db.Where("artist_name LIKE ? OR normalized_artist_key LIKE ?", like, "%"+normalized+"%")
		} else {
			db = db.Where("artist_name LIKE ?", like)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []ArtistProfile
	if err := db.Order("updated_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetArtistProfilesByNames 按艺术家名批量查询资料。
func GetArtistProfilesByNames(ctx context.Context, artistNames []string) (map[string]ArtistProfile, error) {
	result := make(map[string]ArtistProfile, len(artistNames))
	if len(artistNames) == 0 {
		return result, nil
	}

	normalizedKeys := make([]string, 0, len(artistNames))
	for _, artistName := range artistNames {
		if key := NormalizeArtistProfileKey(artistName); key != "" {
			normalizedKeys = append(normalizedKeys, key)
		}
	}
	if len(normalizedKeys) == 0 {
		return result, nil
	}

	var rows []ArtistProfile
	if err := GetDB().WithContext(ctx).Where("normalized_artist_key IN ?", normalizedKeys).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.NormalizedArtistKey] = row
	}
	return result, nil
}

// UpsertArtistProfile 按 normalized key 写入或更新艺术家资料。
func UpsertArtistProfile(ctx context.Context, profile *ArtistProfile) error {
	return upsertArtistProfileTx(GetDB().WithContext(ctx), profile)
}

// NormalizeArtistProfileKey 统一艺术家资料索引键，避免上层重复散落归一规则。
func NormalizeArtistProfileKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return strings.ToLower(common.ConversionSimplifiedFx(common.UnityFixAll(raw)))
}

func ensureArtistProfileIndexes(ctx context.Context) error {
	if config.ConfigObj.Database.Type != string(common.DatabaseTypeMySQL) {
		return nil
	}

	db := GetDB().WithContext(ctx)
	migrator := db.Migrator()

	if !migrator.HasIndex(&ArtistProfile{}, "uk_artist_profile_normalized_key") {
		if err := db.Exec(
			"ALTER TABLE artist_profile ADD UNIQUE KEY uk_artist_profile_normalized_key (normalized_artist_key)",
		).Error; err != nil {
			return err
		}
	}

	return nil
}

func ensureDefaultArtistProfiles(ctx context.Context) error {
	defaultProfiles := []ArtistProfile{
		{
			ArtistName:          "Pink Floyd",
			NormalizedArtistKey: NormalizeArtistProfileKey("Pink Floyd"),
			AvatarMime:          "image/jpeg",
			AvatarObjectKey:     "artist/v1/originals/0081fdc20d001db4cab309e315b54888d8f17d68",
		},
	}

	for _, profile := range defaultProfiles {
		if err := GetDB().WithContext(ctx).Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "normalized_artist_key"}},
				DoNothing: true,
			},
		).Create(&profile).Error; err != nil {
			if isLegacyDashboardSchemaError(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

func upsertArtistProfileTx(tx *gorm.DB, profile *ArtistProfile) error {
	if tx == nil || profile == nil {
		return nil
	}
	profile.ArtistName = strings.TrimSpace(profile.ArtistName)
	profile.NormalizedArtistKey = NormalizeArtistProfileKey(profile.ArtistName)
	profile.AvatarURL = strings.TrimSpace(profile.AvatarURL)
	profile.AvatarMime = strings.TrimSpace(profile.AvatarMime)
	profile.AvatarObjectKey = strings.TrimSpace(profile.AvatarObjectKey)
	if profile.ArtistName == "" || profile.NormalizedArtistKey == "" {
		return fmt.Errorf("artist name is required")
	}

	now := tx.NowFunc()
	profile.UpdatedAt = now
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "normalized_artist_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"artist_name":           profile.ArtistName,
			"normalized_artist_key": profile.NormalizedArtistKey,
			"avatar_url":            profile.AvatarURL,
			"avatar_mime":           profile.AvatarMime,
			"avatar_object_key":     profile.AvatarObjectKey,
			"updated_at":            now,
		}),
	}).Create(profile).Error; err != nil {
		return err
	}

	stored, err := getArtistProfileByNormalizedKeyTx(tx, profile.NormalizedArtistKey)
	if err != nil {
		return err
	}
	*profile = *stored
	return nil
}

func getArtistProfileByNormalizedKeyTx(tx *gorm.DB, normalizedKey string) (*ArtistProfile, error) {
	var profile ArtistProfile
	err := tx.Where("normalized_artist_key = ?", strings.TrimSpace(normalizedKey)).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
