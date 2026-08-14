package model

import (
	"context"

	"gorm.io/gorm"
)

// BackfillGenresResult 表示流水流派回补与对账结果指标
type BackfillGenresResult struct {
	BackfilledRecords int64 `json:"backfilled_records"` // 成功补全/纠偏流派的播放流水行数
	ReconciledGenres  int   `json:"reconciled_genres"`  // 参与对账修正的流派数量
}

// BackfillTrackPlayRecordGenresTx 在事务中将 album / track 的已知流派全量对齐回补到 track_play_records 听歌流水表，并补全 missing 的 resolved_track_id
func BackfillTrackPlayRecordGenresTx(tx *gorm.DB) (*BackfillGenresResult, error) {
	if tx == nil {
		return nil, gorm.ErrInvalidTransaction
	}

	var totalBackfilled int64

	// 0. 通过 track_album 映射补全听歌流水缺失的 resolved_track_id
	sqlTrackIDRepair := `
		UPDATE track_play_records
		SET resolved_track_id = (
			SELECT ta.track_id 
			FROM track_album ta 
			WHERE ta.album_id = track_play_records.album_id 
			  AND ta.disc_number = track_play_records.disc_number 
			  AND ta.track_number = track_play_records.track_number
			  AND ta.track_id > 0
			LIMIT 1
		)
		WHERE (resolved_track_id IS NULL OR resolved_track_id = 0)
		  AND album_id > 0
		  AND EXISTS (
			SELECT 1 FROM track_album ta 
			WHERE ta.album_id = track_play_records.album_id 
			  AND ta.disc_number = track_play_records.disc_number 
			  AND ta.track_number = track_play_records.track_number
			  AND ta.track_id > 0
		  )
	`
	_ = tx.Exec(sqlTrackIDRepair)

	// 1. 优先从关联单曲 track 补写缺失的 track_play_records.genre（单曲流派粒度更精细）
	sqlTrackBackfill := `
		UPDATE track_play_records
		SET genre = (SELECT t.genre FROM track t WHERE t.id = track_play_records.resolved_track_id)
		WHERE (genre IS NULL OR TRIM(genre) = '') 
		  AND resolved_track_id > 0
		  AND EXISTS (
			SELECT 1 FROM track t 
			WHERE t.id = track_play_records.resolved_track_id 
			  AND t.genre IS NOT NULL 
			  AND TRIM(t.genre) != ''
		  )
	`
	resTrack := tx.Exec(sqlTrackBackfill)
	if resTrack.Error != nil {
		return nil, resTrack.Error
	}
	totalBackfilled += resTrack.RowsAffected

	// 2. 剩余缺失的流派从关联的 album 补写保底
	sqlAlbumBackfill := `
		UPDATE track_play_records
		SET genre = (SELECT a.genre FROM album a WHERE a.id = track_play_records.album_id)
		WHERE (genre IS NULL OR TRIM(genre) = '') 
		  AND album_id > 0
		  AND EXISTS (
			SELECT 1 FROM album a 
			WHERE a.id = track_play_records.album_id 
			  AND a.genre IS NOT NULL 
			  AND TRIM(a.genre) != ''
		  )
	`
	resAlbum := tx.Exec(sqlAlbumBackfill)
	if resAlbum.Error != nil {
		return nil, resAlbum.Error
	}
	totalBackfilled += resAlbum.RowsAffected

	// 3. 执行单曲播放数全量对账
	if err := ReconcileTrackPlayCountsTx(tx); err != nil {
		return nil, err
	}

	// 4. 执行专辑播放数全量对账
	if err := ReconcileAlbumPlayCountsTx(tx); err != nil {
		return nil, err
	}

	// 5. 执行流派播放数全量对账（计算 genre.play_count，并刷新未匹配流派暴露列表）
	if err := ReconcileGenrePlayCountsTx(tx); err != nil {
		return nil, err
	}

	// 6. 获取参与对账修正的有效流派总数
	var genreCount int64
	if err := tx.Model(&Genre{}).Where("play_count > 0").Count(&genreCount).Error; err != nil {
		return nil, err
	}

	return &BackfillGenresResult{
		BackfilledRecords: totalBackfilled,
		ReconciledGenres:  int(genreCount),
	}, nil
}

// BackfillTrackPlayRecordGenres 将 album / track 的已知流派全量对齐回补到 track_play_records，并刷新全局统计
func BackfillTrackPlayRecordGenres(ctx context.Context) (*BackfillGenresResult, error) {
	var result *BackfillGenresResult
	err := InTx(ctx, func(tx *gorm.DB) error {
		res, err := BackfillTrackPlayRecordGenresTx(tx)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 触发全局仪表盘统计刷新
	_ = RefreshDashboardStats(ctx)

	return result, nil
}
