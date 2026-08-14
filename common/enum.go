package common

import (
	"strings"
)

var (
	AppSystemEvents    = "System Events"
	AppAudirvanaOrigin = "Audirvana Origin"
	FileExtWav1        = ".wav"
	FileExtWav2        = strings.ToUpper(FileExtWav1)
)

type PlayerState string

const (
	PlayerStateDefault = ""
	PlayerStateStopped = "Stopped"
	PlayerStatePlaying = "Playing"
	PlayerStatePaused  = "Paused"
)

// AnalysisTargetType 定义音眸分析对象类型。
type AnalysisTargetType string

const (
	AnalysisTargetTypeTrack AnalysisTargetType = "track"
	AnalysisTargetTypeAlbum AnalysisTargetType = "album"
)

// AIModelPlatform 定义 AI 模型平台类型。
type AIModelPlatform string

const (
	AIModelPlatformGemini AIModelPlatform = "gemini"
	AIModelPlatformOllama AIModelPlatform = "ollama"
	AIModelPlatformDoubao AIModelPlatform = "doubao"
	AIModelPlatformOMLX   AIModelPlatform = "omlx"
)

// ParseAnalysisTargetType 将外部输入归一到已知分析对象类型。
func ParseAnalysisTargetType(value string) AnalysisTargetType {
	switch AnalysisTargetType(strings.ToLower(strings.TrimSpace(value))) {
	case AnalysisTargetTypeAlbum:
		return AnalysisTargetTypeAlbum
	default:
		return AnalysisTargetTypeTrack
	}
}

// IsTrack 判断是否为曲目分析类型。
func (t AnalysisTargetType) IsTrack() bool {
	return t == AnalysisTargetTypeTrack
}

// IsAlbum 判断是否为专辑分析类型。
func (t AnalysisTargetType) IsAlbum() bool {
	return t == AnalysisTargetTypeAlbum
}

func ParseAIModelPlatform(value string) AIModelPlatform {
	switch AIModelPlatform(strings.ToLower(strings.TrimSpace(value))) {
	case AIModelPlatformGemini:
		return AIModelPlatformGemini
	case AIModelPlatformOllama:
		return AIModelPlatformOllama
	case AIModelPlatformDoubao:
		return AIModelPlatformDoubao
	case AIModelPlatformOMLX:
		return AIModelPlatformOMLX
	default:
		return ""
	}
}

func (p AIModelPlatform) IsValid() bool {
	switch p {
	case AIModelPlatformGemini, AIModelPlatformOllama, AIModelPlatformDoubao, AIModelPlatformOMLX:
		return true
	default:
		return false
	}
}

// TrackMetadataConfidence 定义曲目元数据置信度。
type TrackMetadataConfidence int8

const (
	TrackMetadataConfidenceLow           TrackMetadataConfidence = 1
	TrackMetadataConfidenceMedium        TrackMetadataConfidence = 2
	TrackMetadataConfidenceHigh          TrackMetadataConfidence = 3
	TrackMetadataConfidenceAuthoritative TrackMetadataConfidence = 4
)

// TrackFavoriteState 定义收藏态投影视图的统一枚举。
type TrackFavoriteState string

const (
	TrackFavoriteStateNotFavorited      TrackFavoriteState = "not_favorited"
	TrackFavoriteStateFavorited         TrackFavoriteState = "favorited"
	TrackFavoriteStateFavoritePending   TrackFavoriteState = "favorite_pending"
	TrackFavoriteStateUnfavoritePending TrackFavoriteState = "unfavorite_pending"
)

// IsFavoritedEffective 判断该状态在 UI 上是否应视为“已收藏”。
func (s TrackFavoriteState) IsFavoritedEffective() bool {
	switch s {
	case TrackFavoriteStateFavorited, TrackFavoriteStateFavoritePending:
		return true
	default:
		return false
	}
}

// PlayerType 定义播放器类型
type PlayerType string

const (
	PlayerAudirvana  PlayerType = "Audirvana"
	PlayerFoobar2000 PlayerType = "Foobar2000"
	PlayerNetEase    PlayerType = "NetEase Music"
	PlayerRoon       PlayerType = "Roon"
	PlayerAppleMusic PlayerType = "Apple Music"
)

// DatabaseType 定义数据库类型
type DatabaseType string

const (
	DatabaseTypeSQLite DatabaseType = "sqlite"
	DatabaseTypeMySQL  DatabaseType = "mysql"
)
