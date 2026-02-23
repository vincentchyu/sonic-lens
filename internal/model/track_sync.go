package model

import (
	"context"
	"time"
)

// GetTracksUpdatedSince 获取自指定时间后更新的曲目
func GetTracksUpdatedSince(ctx context.Context, since time.Time) ([]*Track, error) {
	var tracks []*Track
	err := GetDB().WithContext(ctx).
		Where("updated_at >= ?", since).
		Order("updated_at ASC").
		Find(&tracks).Error
	if err != nil {
		return nil, err
	}
	return tracks, nil
}

// GetPlayRecordsUpdatedSince 获取自指定时间后更新的播放记录
func GetPlayRecordsUpdatedSince(ctx context.Context, since time.Time) ([]*TrackPlayRecord, error) {
	var records []*TrackPlayRecord
	err := GetDB().WithContext(ctx).
		Where("updated_at >= ?", since).
		Order("updated_at ASC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetGenresUpdatedSince 获取自指定时间后更新的流派
func GetGenresUpdatedSince(ctx context.Context, since time.Time) ([]*Genre, error) {
	var genres []*Genre
	err := GetDB().WithContext(ctx).
		Where("updated_at >= ?", since).
		Order("updated_at ASC").
		Find(&genres).Error
	if err != nil {
		return nil, err
	}
	return genres, nil
}
