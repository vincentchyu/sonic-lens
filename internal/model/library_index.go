package model

import (
	"context"
	"time"
)

// AlbumIndexRow 表示专辑列表页使用的轻量索引行
type AlbumIndexRow struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Artist            string    `json:"artist"`
	ReleaseDate       string    `json:"release_date"`
	CoverArtURL       string    `json:"cover_art_url"`
	CoverArtMime      string    `json:"cover_art_mime"`
	CoverArtObjectKey string    `json:"cover_art_object_key"`
	HasInsight        bool      `json:"has_insight"`
	PlayCount         int       `json:"play_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TrackIndexRow 表示曲目列表页使用的轻量索引行
type TrackIndexRow struct {
	ID              int64     `json:"id"`
	Artist          string    `json:"artist"`
	Album           string    `json:"album"`
	Track           string    `json:"track"`
	PlayCount       int       `json:"play_count"`
	TrackNumber     int8      `json:"track_number"`
	DiscNumber      int8      `json:"disc_number"`
	Duration        int64     `json:"duration"`
	IsAppleMusicFav bool      `json:"is_apple_music_fav"`
	IsLastFmFav     bool      `json:"is_last_fm_fav"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// LibrarySyncDelta 表示资料库索引同步增量
type LibrarySyncDelta struct {
	Albums          []*AlbumIndexRow
	Tracks          []*TrackIndexRow
	DeletedAlbumIDs []int64
	DeletedTrackIDs []int64
	Version         int64
}

// GetAlbumIndexRows 获取专辑索引快照；当 since 非零时，仅返回自身或关联曲目有更新的专辑
func GetAlbumIndexRows(ctx context.Context, since time.Time) ([]*AlbumIndexRow, error) {
	var rows []*AlbumIndexRow

	query := GetDB().WithContext(ctx).
		Table("album AS a").
		Select(
			`a.id, a.name, a.artist, a.release_date,
a.cover_art_url, a.cover_art_mime, a.cover_art_object_key,
EXISTS(
    SELECT 1
    FROM album_insight AS ai
    WHERE ai.is_disabled = false
      AND (ai.album_id = a.id OR ((ai.album_id = 0 OR ai.album_id IS NULL) AND ai.artist = a.artist AND ai.album = a.name))
) AS has_insight,
COALESCE(SUM(t.play_count), 0) AS play_count,
a.created_at, MAX(COALESCE(t.updated_at, a.updated_at, a.created_at)) AS updated_at`,
		).
		Joins("LEFT JOIN track AS t ON t.album = a.name AND t.artist = a.artist").
		Group("a.id, a.name, a.artist, a.release_date, a.cover_art_url, a.cover_art_mime, a.cover_art_object_key, a.created_at")

	if !since.IsZero() {
		query = query.Where("a.updated_at >= ? OR t.updated_at >= ?", since, since)
	}

	err := query.Order("a.id ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetAlbumIndexRowsByIDs 根据专辑 ID 批量获取索引行
func GetAlbumIndexRowsByIDs(ctx context.Context, ids []int64) ([]*AlbumIndexRow, error) {
	var rows []*AlbumIndexRow
	if len(ids) == 0 {
		return rows, nil
	}

	err := GetDB().WithContext(ctx).
		Table("album AS a").
		Select(
			`a.id, a.name, a.artist, a.release_date,
a.cover_art_url, a.cover_art_mime, a.cover_art_object_key,
EXISTS(
    SELECT 1
    FROM album_insight AS ai
    WHERE ai.is_disabled = false
      AND (ai.album_id = a.id OR ((ai.album_id = 0 OR ai.album_id IS NULL) AND ai.artist = a.artist AND ai.album = a.name))
) AS has_insight,
COALESCE(SUM(t.play_count), 0) AS play_count,
a.created_at, MAX(COALESCE(t.updated_at, a.updated_at, a.created_at)) AS updated_at`,
		).
		Joins("LEFT JOIN track AS t ON t.album = a.name AND t.artist = a.artist").
		Where("a.id IN ?", ids).
		Group("a.id, a.name, a.artist, a.release_date, a.cover_art_url, a.cover_art_mime, a.cover_art_object_key, a.created_at").
		Order("a.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetTrackIndexRows 获取曲目索引快照
func GetTrackIndexRows(ctx context.Context, since time.Time) ([]*TrackIndexRow, error) {
	var rows []*TrackIndexRow

	query := GetDB().WithContext(ctx).Model(&Track{})
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}

	err := query.Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetTrackIndexRowsByIDs 根据曲目 ID 批量获取索引行
func GetTrackIndexRowsByIDs(ctx context.Context, ids []int64) ([]*TrackIndexRow, error) {
	var rows []*TrackIndexRow
	if len(ids) == 0 {
		return rows, nil
	}

	err := GetDB().WithContext(ctx).
		Model(&Track{}).
		Where("id IN ?", ids).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetLibrarySyncVersion 返回资料库同步游标，当前使用专辑/曲目最新 updated_at 的 Unix 秒值
func GetLibrarySyncDelta(ctx context.Context, sinceVersion int64) (*LibrarySyncDelta, error) {
	version, err := GetLatestLibraryChangeVersion(ctx)
	if err != nil {
		return nil, err
	}

	delta := &LibrarySyncDelta{
		Albums:          []*AlbumIndexRow{},
		Tracks:          []*TrackIndexRow{},
		DeletedAlbumIDs: []int64{},
		DeletedTrackIDs: []int64{},
		Version:         version,
	}

	if sinceVersion <= 0 {
		delta.Albums, err = GetAlbumIndexRows(ctx, time.Time{})
		if err != nil {
			return nil, err
		}
		delta.Tracks, err = GetTrackIndexRows(ctx, time.Time{})
		if err != nil {
			return nil, err
		}
		return delta, nil
	}

	changes, err := GetLibraryChangesSince(ctx, sinceVersion)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return delta, nil
	}

	albumOps := make(map[int64]string)
	trackOps := make(map[int64]string)
	for _, change := range changes {
		switch change.EntityType {
		case LibraryEntityAlbum:
			albumOps[change.EntityID] = change.Operation
		case LibraryEntityTrack:
			trackOps[change.EntityID] = change.Operation
		}
	}

	var upsertAlbumIDs []int64
	for id, op := range albumOps {
		if op == LibraryOpDelete {
			delta.DeletedAlbumIDs = append(delta.DeletedAlbumIDs, id)
			continue
		}
		upsertAlbumIDs = append(upsertAlbumIDs, id)
	}

	var upsertTrackIDs []int64
	for id, op := range trackOps {
		if op == LibraryOpDelete {
			delta.DeletedTrackIDs = append(delta.DeletedTrackIDs, id)
			continue
		}
		upsertTrackIDs = append(upsertTrackIDs, id)
	}

	if len(upsertAlbumIDs) > 0 {
		delta.Albums, err = GetAlbumIndexRowsByIDs(ctx, upsertAlbumIDs)
		if err != nil {
			return nil, err
		}
	}
	if len(upsertTrackIDs) > 0 {
		delta.Tracks, err = GetTrackIndexRowsByIDs(ctx, upsertTrackIDs)
		if err != nil {
			return nil, err
		}
	}

	return delta, nil
}
