package analysis

import (
	"context"

	"github.com/vincentchyu/sonic-lens/internal/logic/analysis"
)

// GenerateMusicRecommendations 基于历史播放记录生成音乐推荐
func GenerateMusicRecommendations(ctx context.Context, limit int) ([]analysis.MusicRecommendation, error) {
	// 直接调用逻辑层的函数
	return analysis.GenerateMusicRecommendations(ctx, limit)
}

// PrintRecommendations 打印音乐推荐
func PrintRecommendations(recommendations []analysis.MusicRecommendation) {
	analysis.PrintRecommendations(recommendations)
}

// GetArtistRecommendations 获取特定艺术家的推荐曲目
func GetArtistRecommendations(ctx context.Context, artist string, limit int) ([]analysis.MusicRecommendation, error) {
	// 直接调用逻辑层的函数
	return analysis.GetArtistRecommendations(ctx, artist, limit)
}
