package analysis

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// MusicAnalysisService 定义音乐分析服务的接口
type MusicAnalysisService interface {
	// GenerateMusicPreferenceReport 生成音乐偏好分析报告
	GenerateMusicPreferenceReport(ctx context.Context) (*ReportData, error)

	// GenerateRecommendations 生成音乐推荐
	GenerateRecommendations(ctx context.Context) ([]MusicRecommendation, error)

	// ScheduleReport 定时生成报告
	ScheduleReport(ctx context.Context, interval time.Duration)
}

// MusicAnalysisServiceImpl 实现音乐分析服务接口
type MusicAnalysisServiceImpl struct{}

// NewMusicAnalysisService 创建音乐分析服务实例
func NewMusicAnalysisService() MusicAnalysisService {
	return &MusicAnalysisServiceImpl{}
}

// ReportData 音乐偏好分析报告数据
type ReportData struct {
	TotalTracks   int64
	TopTracks     []*model.Track
	RecentRecords []*model.TrackPlayRecord
}

// GenerateMusicPreferenceReport 生成音乐偏好分析报告
func (s *MusicAnalysisServiceImpl) GenerateMusicPreferenceReport(ctx context.Context) (*ReportData, error) {
	// 获取播放统计总数
	totalTracks, err := model.GetTrackCounts(ctx)
	if err != nil {
		log.Error(ctx, "Failed to get track counts", zap.Error(err))
		return nil, err
	}

	// 获取播放次数最多的曲目 (30条数据)
	topTracks, err := model.GetTracks(ctx, 30, 0)
	if err != nil {
		log.Error(ctx, "Failed to get top tracks", zap.Error(err))
		return nil, err
	}

	// 获取最近播放的曲目 (30条数据)
	recentRecords, err := getRecentPlayRecords(ctx, 30)
	if err != nil {
		log.Error(ctx, "Failed to get recent play records", zap.Error(err))
		return nil, err
	}
	data := &ReportData{
		TotalTracks:   totalTracks,
		TopTracks:     topTracks,
		RecentRecords: recentRecords,
	}
	PrintReportData(data)

	return data, nil
}

// GenerateRecommendations 生成音乐推荐
func (s *MusicAnalysisServiceImpl) GenerateRecommendations(ctx context.Context) ([]MusicRecommendation, error) {
	// 生成推荐
	recommendations, err := GenerateMusicRecommendations(ctx, 10)
	if err != nil {
		log.Error(ctx, "Failed to generate music recommendations", zap.Error(err))
		return nil, err
	}

	// 打印推荐
	PrintRecommendations(recommendations)
	return recommendations, nil
}

// getRecentPlayRecords 获取最近播放的记录
func getRecentPlayRecords(ctx context.Context, limit int) ([]*model.TrackPlayRecord, error) {
	return model.GetRecentPlayRecords(ctx, limit)
}

// ScheduleReport 定时生成报告
func (s *MusicAnalysisServiceImpl) ScheduleReport(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := s.GenerateMusicPreferenceReport(ctx); err != nil {
				log.Error(ctx, "Failed to generate music preference report", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

// MusicRecommendation 音乐推荐结构
type MusicRecommendation struct {
	Artist string
	Album  string
	Track  string
	Score  float64
}

// GenerateMusicRecommendations 基于历史播放记录生成音乐推荐
func GenerateMusicRecommendations(ctx context.Context, limit int) ([]MusicRecommendation, error) {

	// 获取所有播放统计记录
	allTracks, err := getAllTrackPlayCounts(ctx)
	if err != nil {
		log.Error(ctx, "Failed to get track play counts", zap.Error(err))
		return nil, err
	}

	// 获取最近播放的记录
	recentRecords, err := getRecentPlayRecords(ctx, 50)
	if err != nil {
		log.Error(ctx, "Failed to get recent play records", zap.Error(err))
		return nil, err
	}

	// 基于播放统计和最近播放记录生成推荐
	recommendations := calculateRecommendations(allTracks, recentRecords, limit)

	return recommendations, nil
}

// getAllTrackPlayCounts 获取所有播放统计记录
func getAllTrackPlayCounts(ctx context.Context) ([]*model.Track, error) {
	return model.GetAllTrackPlayCounts(ctx)
}

// calculateRecommendations 计算推荐分数并生成推荐列表
func calculateRecommendations(
	allTracks []*model.Track, recentRecords []*model.TrackPlayRecord, limit int,
) []MusicRecommendation {
	// 创建艺术家和专辑的播放频率映射
	artistFrequency := make(map[string]int)
	albumFrequency := make(map[string]int)

	// 统计最近播放记录中的艺术家和专辑频率
	for _, record := range recentRecords {
		artistFrequency[record.Artist]++
		albumFrequency[record.Album]++
	}

	// 计算推荐分数
	var recommendations []MusicRecommendation
	for _, track := range allTracks {
		score := calculateScore(track, artistFrequency, albumFrequency)
		recommendations = append(
			recommendations, MusicRecommendation{
				Artist: track.Artist,
				Album:  track.Album,
				Track:  track.Track,
				Score:  score,
			},
		)
	}

	// 按分数排序并返回前limit个推荐
	sortRecommendations(recommendations)

	if len(recommendations) > limit {
		return recommendations[:limit]
	}

	return recommendations
}

// calculateScore 计算单个曲目的推荐分数
func calculateScore(track *model.Track, artistFrequency, albumFrequency map[string]int) float64 {
	// 基础分数基于播放次数
	baseScore := float64(track.PlayCount)

	// 艺术家频率加权
	artistWeight := 1.0
	if freq, ok := artistFrequency[track.Artist]; ok {
		artistWeight = 1.0 + math.Log(float64(freq)+1)
	}

	// 专辑频率加权
	albumWeight := 1.0
	if freq, ok := albumFrequency[track.Album]; ok {
		albumWeight = 1.0 + math.Log(float64(freq)+1)
	}

	// 计算最终分数
	finalScore := baseScore * artistWeight * albumWeight

	return finalScore
}

// sortRecommendations 按分数降序排序推荐列表
func sortRecommendations(recommendations []MusicRecommendation) {
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[i].Score < recommendations[j].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
}

// PrintRecommendations 打印音乐推荐
func PrintRecommendations(recommendations []MusicRecommendation) {
	fmt.Println("=== 音乐推荐 ===")
	for i, rec := range recommendations {
		fmt.Printf("%d. %s - %s - %s (推荐分数: %.2f)\n", i+1, rec.Artist, rec.Album, rec.Track, rec.Score)
	}
}
func PrintReportData(reportData *ReportData) {
	// 打印报告
	fmt.Println("=== 音乐偏好分析报告 ===")
	fmt.Printf("总曲目数: %d\n", reportData.TotalTracks)
	fmt.Println("\n播放次数最多的曲目:")
	for i, track := range reportData.TopTracks {
		fmt.Printf("%d. %s - %s - %s (播放次数: %d)\n", i+1, track.Artist, track.Album, track.Track, track.PlayCount)
	}

	fmt.Println("\n最近播放的曲目:")
	for i, record := range reportData.RecentRecords {
		fmt.Printf(
			"%d. %s - %s - %s (%s)\n", i+1, record.Artist, record.Album, record.Track,
			record.PlayTime.Format("2006-01-02 15:04:05"),
		)
	}
}

// GetArtistRecommendations 获取特定艺术家的推荐曲目
func GetArtistRecommendations(ctx context.Context, artist string, limit int) ([]MusicRecommendation, error) {

	// 获取该艺术家的所有曲目
	tracks, err := getTracksByArtist(ctx, artist)
	if err != nil {
		log.Error(ctx, "Failed to get tracks by artist", zap.String("artist", artist), zap.Error(err))
		return nil, err
	}

	// 计算推荐分数
	var recommendations []MusicRecommendation
	for _, track := range tracks {
		recommendations = append(
			recommendations, MusicRecommendation{
				Artist: track.Artist,
				Album:  track.Album,
				Track:  track.Track,
				Score:  float64(track.PlayCount),
			},
		)
	}

	// 按分数排序
	sortRecommendations(recommendations)

	if len(recommendations) > limit {
		return recommendations[:limit], nil
	}

	return recommendations, nil
}

// getTracksByArtist 获取特定艺术家的所有曲目
func getTracksByArtist(ctx context.Context, artist string) ([]*model.Track, error) {
	return model.GetTracksByArtist(ctx, artist)
}
