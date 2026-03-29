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
	AIModelPlatformOpenAI AIModelPlatform = "openai"
	AIModelPlatformGemini AIModelPlatform = "gemini"
	AIModelPlatformOllama AIModelPlatform = "ollama"
	AIModelPlatformDoubao AIModelPlatform = "doubao"
	AIModelPlatformCustom AIModelPlatform = "custom"
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

// ParseAIModelPlatform 将外部输入归一到已知 AI 模型平台。
func ParseAIModelPlatform(value string) AIModelPlatform {
	switch AIModelPlatform(strings.ToLower(strings.TrimSpace(value))) {
	case AIModelPlatformOpenAI:
		return AIModelPlatformOpenAI
	case AIModelPlatformGemini:
		return AIModelPlatformGemini
	case AIModelPlatformOllama:
		return AIModelPlatformOllama
	case AIModelPlatformDoubao:
		return AIModelPlatformDoubao
	case AIModelPlatformCustom:
		return AIModelPlatformCustom
	default:
		return ""
	}
}

// IsValid 判断是否为已知 AI 模型平台。
func (p AIModelPlatform) IsValid() bool {
	switch p {
	case AIModelPlatformOpenAI, AIModelPlatformGemini, AIModelPlatformOllama, AIModelPlatformDoubao, AIModelPlatformCustom:
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
	PlayerRoon       PlayerType = "Roon"
	PlayerAppleMusic PlayerType = "Apple Music"
)

// DatabaseType 定义数据库类型
type DatabaseType string

const (
	DatabaseTypeSQLite DatabaseType = "sqlite"
	DatabaseTypeMySQL  DatabaseType = "mysql"
)

/*
var GenreMap = map[string]string{
	// 流行
	"流行":   "Pop",
	"独立流行": "Indie Pop",
	"国语流行": "C-Pop",
	"日语流行": "J-Pop",
	"韩语流行": "K-Pop",
	"合成流行": "Synth Pop",
	"电音流行": "Electropop",
	"成人当代": "Adult Contemporary",
	"舞曲":   "Dance",
	"流行摇滚": "Pop Rock",

	// 另类
	"另类音乐": "Alternative",

	// 摇滚
	"摇滚":    "Rock",
	"经典摇滚":  "Classic Rock",
	"硬摇滚":   "Hard Rock",
	"另类摇滚":  "Alternative Rock",
	"独立摇滚":  "Indie Rock",
	"进步摇滚":  "Progressive Rock",
	"前卫摇滚":  "Progressive Rock",
	"前进摇滚":  "Progressive Rock",
	"重金属":   "Metal",
	"前卫金属":  "Progressive Metal",
	"金属核":   "Metalcore",
	"车库摇滚":  "Garage Rock",
	"布鲁斯摇滚": "Blues Rock",
	"朋克":    "Punk",
	"流行朋克":  "Pop Punk",
	"后朋克":   "Post-Punk",
	"硬核朋克":  "Hardcore Punk",
	"哥特摇滚":  "Gothic Rock",

	// 爵士
	"爵士":      "Jazz",
	"光辉爵士":    "Smooth Jazz",
	"摇摆爵士":    "Swing",
	"拉丁爵士":    "Latin Jazz",
	"融合爵士":    "Jazz Fusion",
	"自由爵士":    "Free Jazz",
	"酸爵士":     "Acid Jazz",
	"前卫爵士":    "Avant-Garde Jazz",
	"传统爵士":    "Traditional Jazz",
	"灵魂爵士":    "Soul Jazz",
	"当代爵士乐大赏": "Contemporary Jazz",

	// 蓝调
	"蓝调":    "Blues",
	"城市蓝调":  "Urban Blues",
	"芝加哥蓝调": "Chicago Blues",
	"德州蓝调":  "Texas Blues",
	"摇滚蓝调":  "Blues Rock",

	// 嘻哈 / 说唱
	"嘻哈":   "Hip-Hop",
	"说唱":   "Rap",
	"陷阱":   "Trap",
	"老派嘻哈": "Old School Hip-Hop",
	"新派嘻哈": "New School Hip-Hop",
	"地下嘻哈": "Underground Hip-Hop",
	"爵士说唱": "Jazz Rap",

	// R&B / 灵魂
	"R&B":  "R&B",
	"灵魂":   "Soul",
	"新灵魂":  "Neo Soul",
	"放克":   "Funk",
	"节奏蓝调": "Rhythm and Blues",

	// 电子
	"电子":   "Electronic",
	"环境音乐": "Ambient",
	"舞曲电子": "EDM",
	"浩室":   "House",
	"浩室深":  "Deep House",
	"浩室进":  "Progressive House",
	"浩室浩":  "Tech House",
	"电子合成": "Synthwave",
	"氛围电子": "Chillwave",
	"鼓贝斯":  "Drum and Bass",
	"工业":   "Industrial",
	"电子实验": "Experimental Electronic",
	"电子流行": "Electropop",

	// 民谣 / 乡村
	"民谣":   "Folk",
	"独立民谣": "Indie Folk",
	"乡村":   "Country",
	"现代乡村": "Contemporary Country",
	"蓝草":   "Bluegrass",
	"民谣摇滚": "Folk Rock",

	// 拉丁 / 世界音乐
	"拉丁":   "Latin",
	"萨尔萨":  "Salsa",
	"探戈":   "Tango",
	"巴西":   "Brazilian",
	"弗拉明戈": "Flamenco",
	"世界音乐": "World",
	"非洲":   "African",
	"凯尔特":  "Celtic",
	"中东":   "Middle Eastern",
	"亚洲":   "Asian",

	// 新世纪 / 其他
	"新世纪":  "New Age",
	"原声":   "Acoustic",
	"声乐":   "Vocal",
	"电影原声": "Soundtrack",
	"音乐剧":  "Musical",
	"实验":   "Experimental",
}*/
