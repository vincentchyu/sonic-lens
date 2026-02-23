package applemusic

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/applesciprt"
	alog "github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

const appleSciprtGetNowPlayingTrackInfo = `-- 安全 JSON 值转换
on safeJSONValue(v)
	try
		if v is missing value then
			return ""
		else if (class of v is boolean) then
			if v then
				return "true"
			else
				return "false"
			end if
		else if (class of v is integer) or (class of v is real) then
			return (v as string)
		else
			return my escapeJSON(v as string)
		end if
	on error
		return ""
	end try
end safeJSONValue

-- JSON 转义
on escapeJSON(theText)
	try
		set theText to my replaceText(theText, "\\", "\\\\")
		set theText to my replaceText(theText, "\"", "\\\"")
		set theText to my replaceText(theText, return, "\\n")
		set theText to my replaceText(theText, linefeed, "\\n")
		return theText
	on error
		return theText
	end try
end escapeJSON

-- 替换字符串
on replaceText(theText, search, replace)
	set AppleScript's text item delimiters to search
	set theItems to every text item of theText
	set AppleScript's text item delimiters to replace
	set theText to theItems as string
	set AppleScript's text item delimiters to ""
	return theText
end replaceText


tell application "Music"
	try
		if player state is playing then
			if exists current track then
				set t to current track
				
				set json to "{"
				set json to json & "\"name\":\"" & my safeJSONValue(name of t) & "\","
				set json to json & "\"album\":\"" & my safeJSONValue(album of t) & "\","
				set json to json & "\"artist\":\"" & my safeJSONValue(artist of t) & "\","
				set json to json & "\"albumArtist\":\"" & my safeJSONValue(album artist of t) & "\","
				set json to json & "\"duration\":\"" & my safeJSONValue(duration of t) & "\","
				set json to json & "\"playerPosition\":\"" & my safeJSONValue(player position) & "\","
				set json to json & "\"databaseID\":\"" & my safeJSONValue(database ID of t) & "\","
				set json to json & "\"composer\":\"" & my safeJSONValue(composer of t) & "\","
				set json to json & "\"albumDisliked\":\"" & my safeJSONValue(album disliked of t) & "\","
				set json to json & "\"albumFavorited\":\"" & my safeJSONValue(album favorited of t) & "\","
				set json to json & "\"albumRating\":\"" & my safeJSONValue(album rating of t) & "\","
				set json to json & "\"bitRate\":\"" & my safeJSONValue(bit rate of t) & "\","
				set json to json & "\"bookmark\":\"" & my safeJSONValue(bookmark of t) & "\","
				set json to json & "\"bpm\":\"" & my safeJSONValue(bpm of t) & "\","
				set json to json & "\"category\":\"" & my safeJSONValue(category of t) & "\","
				set json to json & "\"comment\":\"" & my safeJSONValue(comment of t) & "\","
				set json to json & "\"compilation\":\"" & my safeJSONValue(compilation of t) & "\","
				set json to json & "\"dateAdded\":\"" & my safeJSONValue(date added of t) & "\","
				set json to json & "\"description\":\"" & my safeJSONValue(description of t) & "\","
				set json to json & "\"discCount\":\"" & my safeJSONValue(disc count of t) & "\","
				set json to json & "\"discNumber\":\"" & my safeJSONValue(disc number of t) & "\","
				set json to json & "\"disliked\":\"" & my safeJSONValue(disliked of t) & "\","
				set json to json & "\"enabled\":\"" & my safeJSONValue(enabled of t) & "\","
				set json to json & "\"EQ\":\"" & my safeJSONValue(EQ of t) & "\","
				set json to json & "\"finish\":\"" & my safeJSONValue(finish of t) & "\","
				set json to json & "\"gapless\":\"" & my safeJSONValue(gapless of t) & "\","
				set json to json & "\"genre\":\"" & my safeJSONValue(genre of t) & "\","
				set json to json & "\"grouping\":\"" & my safeJSONValue(grouping of t) & "\","
				set json to json & "\"kind\":\"" & my safeJSONValue(kind of t) & "\","
				set json to json & "\"longDescription\":\"" & my safeJSONValue(long description of t) & "\","
				set json to json & "\"favorited\":\"" & my safeJSONValue(favorited of t) & "\","
				set json to json & "\"lyrics\":\"" & my safeJSONValue(lyrics of t) & "\","
				set json to json & "\"modificationDate\":\"" & my safeJSONValue(modification date of t) & "\","
				set json to json & "\"movement\":\"" & my safeJSONValue(movement of t) & "\","
				set json to json & "\"movementCount\":\"" & my safeJSONValue(movement count of t) & "\","
				set json to json & "\"movementNumber\":\"" & my safeJSONValue(movement number of t) & "\","
				set json to json & "\"playedCount\":\"" & my safeJSONValue(played count of t) & "\","
				set json to json & "\"playedDate\":\"" & my safeJSONValue(played date of t) & "\","
				set json to json & "\"rating\":\"" & my safeJSONValue(rating of t) & "\","
				set json to json & "\"releaseDate\":\"" & my safeJSONValue(release date of t) & "\","
				set json to json & "\"sampleRate\":\"" & my safeJSONValue(sample rate of t) & "\","
				set json to json & "\"shufflable\":\"" & my safeJSONValue(shufflable of t) & "\","
				set json to json & "\"skippedCount\":\"" & my safeJSONValue(skipped count of t) & "\","
				set json to json & "\"skippedDate\":\"" & my safeJSONValue(skipped date of t) & "\","
				set json to json & "\"sortAlbum\":\"" & my safeJSONValue(sort album of t) & "\","
				set json to json & "\"sortArtist\":\"" & my safeJSONValue(sort artist of t) & "\","
				set json to json & "\"sortAlbumArtist\":\"" & my safeJSONValue(sort album artist of t) & "\","
				set json to json & "\"sortName\":\"" & my safeJSONValue(sort name of t) & "\","
				set json to json & "\"sortComposer\":\"" & my safeJSONValue(sort composer of t) & "\","
				set json to json & "\"size\":\"" & my safeJSONValue(size of t) & "\","
				set json to json & "\"start\":\"" & my safeJSONValue(start of t) & "\","
				set json to json & "\"trackCount\":\"" & my safeJSONValue(track count of t) & "\","
				set json to json & "\"trackNumber\":\"" & my safeJSONValue(track number of t) & "\","
				set json to json & "\"unplayed\":\"" & my safeJSONValue(unplayed of t) & "\","
				set json to json & "\"volumeAdjustment\":\"" & my safeJSONValue(volume adjustment of t) & "\","
				set json to json & "\"work\":\"" & my safeJSONValue(work of t) & "\","
				set json to json & "\"year\":\"" & my safeJSONValue(year of t) & "\""
				
				set json to json & "}"
				return json
			end if
		end if
	on error errMsg
		return "{\"error\":\"" & my escapeJSON(errMsg) & "\"}"
	end try
end tell`

type (
	TrackBase struct {
		// media-control get -h  --now
		TrackID    string
		Title      string `json:"title"`
		Album      string `json:"album"`
		Artist     string `json:"artist"`
		Duration   int64  `json:"duration"`
		Position   float64
		Url        string
		AirfoiLogo string

		ElapsedTimeNow float64 `json:"elapsedTimeNow"` // 当前播放时间
		Genre          string  `json:"genre"`
		TrackNumber    int     `json:"trackNumber"`
		IsMusicApp     bool    `json:"isMusicApp"`
		Playing        bool    `json:"playing"` // 是否正在播放

		Composer         string `json:"composer"`         // 作曲家
		BundleIdentifier string `json:"bundleIdentifier"` // 软件标识

		ElapsedTime           float64   `json:"elapsedTime"`
		ArtworkData           string    `json:"artworkData"`
		UniqueIdentifier      int64     `json:"uniqueIdentifier"`
		ContentItemIdentifier string    `json:"contentItemIdentifier"` // 疑似歌曲id
		RepeatMode            int       `json:"repeatMode"`
		QueueIndex            int       `json:"queueIndex"`
		ArtworkMimeType       string    `json:"artworkMimeType"`
		MediaType             string    `json:"mediaType"`
		Timestamp             time.Time `json:"timestamp"`
		ShuffleMode           int       `json:"shuffleMode"`
		TotalTrackCount       int       `json:"totalTrackCount"`
		ProcessIdentifier     int       `json:"processIdentifier"`
		TotalQueueCount       int       `json:"totalQueueCount"`
		PlaybackRate          int       `json:"playbackRate"`

		// Apple Music specific fields
		AlbumArtist      string    `json:"album_artist"`      // 专辑艺术家
		AlbumDisliked    bool      `json:"album_disliked"`    // 专辑是否被讨厌
		AlbumFavorited   bool      `json:"album_favorited"`   // 专辑是否被收藏
		AlbumRating      int       `json:"album_rating"`      // 专辑评分
		BitRate          int       `json:"bit_rate"`          // 比特率
		Bookmark         float64   `json:"bookmark"`          // 书签时间
		BPM              int       `json:"bpm"`               // 每分钟节拍数
		Category         string    `json:"category"`          // 类别
		Comment          string    `json:"comment"`           // 备注
		Compilation      bool      `json:"compilation"`       // 是否为合辑
		DatabaseID       int       `json:"database_id"`       // 数据库ID
		DateAdded        time.Time `json:"date_added"`        // 添加日期
		Description      string    `json:"description"`       // 描述
		DiscCount        int       `json:"disc_count"`        // 光盘总数
		DiscNumber       int       `json:"disc_number"`       // 光盘编号
		Disliked         bool      `json:"disliked"`          // 是否被讨厌
		DurationString   string    `json:"duration_string"`   // 持续时间字符串格式
		Enabled          bool      `json:"enabled"`           // 是否启用播放
		EQ               string    `json:"eq"`                // 均衡器预设
		Finish           float64   `json:"finish"`            // 结束时间
		Gapless          bool      `json:"gapless"`           // 是否为无缝专辑
		Grouping         string    `json:"grouping"`          // 分组
		Kind             string    `json:"kind"`              // 类型描述
		LongDescription  string    `json:"long_description"`  // 长描述
		Favorited        bool      `json:"favorited"`         // 是否被收藏
		Lyrics           string    `json:"lyrics"`            // 歌词
		ModificationDate time.Time `json:"modification_date"` // 修改日期
		Movement         string    `json:"movement"`          // 运动名称
		MovementCount    int       `json:"movement_count"`    // 运动总数
		MovementNumber   int       `json:"movement_number"`   // 运动编号
		PlayedCount      int       `json:"played_count"`      // 播放次数
		PlayedDate       time.Time `json:"played_date"`       // 最后播放日期
		Rating           int       `json:"rating"`            // 评分
		ReleaseDate      time.Time `json:"release_date"`      // 发布日期
		SampleRate       int       `json:"sample_rate"`       // 采样率
		Shufflable       bool      `json:"shufflable"`        // 是否可随机播放
		SkippedCount     int       `json:"skipped_count"`     // 跳过次数
		SkippedDate      time.Time `json:"skipped_date"`      // 最后跳过日期
		SortAlbum        string    `json:"sort_album"`        // 排序专辑名
		SortArtist       string    `json:"sort_artist"`       // 排序艺术家名
		SortAlbumArtist  string    `json:"sort_album_artist"` // 排序专辑艺术家名
		SortName         string    `json:"sort_name"`         // 排序名称
		SortComposer     string    `json:"sort_composer"`     // 排序作曲家名
		Size             int64     `json:"size"`              // 大小(字节)
		Start            float64   `json:"start"`             // 开始时间
		TrackCount       int       `json:"track_count"`       // 音轨总数
		Unplayed         bool      `json:"unplayed"`          // 是否未播放
		VolumeAdjustment int       `json:"volume_adjustment"` // 音量调整
		Work             string    `json:"work"`              // 作品名
		Year             int       `json:"year"`              // 年份
	}

	tmp struct {
		Name             string `json:"name"`
		Album            string `json:"album"`
		Artist           string `json:"artist"`
		AlbumArtist      string `json:"albumArtist"`
		Duration         string `json:"duration"`
		PlayerPosition   string `json:"playerPosition"`
		DatabaseID       string `json:"databaseID"`
		Composer         string `json:"composer"`
		AlbumDisliked    string `json:"albumDisliked"`
		AlbumFavorited   string `json:"albumFavorited"`
		AlbumRating      string `json:"albumRating"`
		BitRate          string `json:"bitRate"`
		Bookmark         string `json:"bookmark"`
		Bpm              string `json:"bpm"`
		Category         string `json:"category"`
		Comment          string `json:"comment"`
		Compilation      string `json:"compilation"`
		DateAdded        string `json:"dateAdded"`
		Description      string `json:"description"`
		DiscCount        string `json:"discCount"`
		DiscNumber       string `json:"discNumber"`
		Disliked         string `json:"disliked"`
		Enabled          string `json:"enabled"`
		EQ               string `json:"EQ"`
		Finish           string `json:"finish"`
		Gapless          string `json:"gapless"`
		Genre            string `json:"genre"`
		Grouping         string `json:"grouping"`
		Kind             string `json:"kind"`
		LongDescription  string `json:"longDescription"`
		Favorited        string `json:"favorited"`
		Lyrics           string `json:"lyrics"`
		ModificationDate string `json:"modificationDate"`
		Movement         string `json:"movement"`
		MovementCount    string `json:"movementCount"`
		MovementNumber   string `json:"movementNumber"`
		PlayedCount      string `json:"playedCount"`
		PlayedDate       string `json:"playedDate"`
		Rating           string `json:"rating"`
		ReleaseDate      string `json:"releaseDate"`
		SampleRate       string `json:"sampleRate"`
		Shufflable       string `json:"shufflable"`
		SkippedCount     string `json:"skippedCount"`
		SkippedDate      string `json:"skippedDate"`
		SortAlbum        string `json:"sortAlbum"`
		SortArtist       string `json:"sortArtist"`
		SortAlbumArtist  string `json:"sortAlbumArtist"`
		SortName         string `json:"sortName"`
		SortComposer     string `json:"sortComposer"`
		Size             string `json:"size"`
		Start            string `json:"start"`
		TrackCount       string `json:"trackCount"`
		TrackNumber      string `json:"trackNumber"`
		Unplayed         string `json:"unplayed"`
		VolumeAdjustment string `json:"volumeAdjustment"`
		Work             string `json:"work"`
		Year             string `json:"year"`
	}

	TrackInfo struct {
		TrackBase
	}
)

// ToTrackMetadata converts TrackInfo to TrackMetadata for database storage
func (ti *TrackInfo) ToTrackMetadata() model.TrackMetadata {
	return model.TrackMetadata{
		AlbumArtist:   ti.AlbumArtist,
		TrackNumber:   int8(ti.TrackNumber),
		Duration:      ti.Duration,
		Genre:         ti.Genre,
		Composer:      ti.Composer,
		ReleaseDate:   ti.ReleaseDate.Format("2006-01-02"),
		MusicBrainzID: "", // Apple Music doesn't provide this directly
		Source:        "Apple Music",
		BundleID:      ti.BundleIdentifier,
		UniqueID:      fmt.Sprintf("%d", ti.UniqueIdentifier),
	}
}

func IsRunning(ctx context.Context) bool {
	tell, err := applesciprt.Tell(
		common.AppSystemEvents, fmt.Sprintf(
			`set listApplicationProcessNames to name of every application process
			if listApplicationProcessNames contains "%s" then
				set APPLE_MUSIC_RUNNING_STATE to "true"
			else
				set APPLE_MUSIC_RUNNING_STATE to "false"
			end if`, "Music",
		),
	)
	if err != nil {
		return false
	}

	parseBool, err := strconv.ParseBool(tell)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return false
	}
	return parseBool
}

// GetState 使用 AppleScript 从 Apple Music 应用获取当前播放器状态。
// 返回播放器状态（common.PlayerState）以及过程中遇到的任何错误。
func GetState(ctx context.Context) (playerState common.PlayerState, err error) {
	result, err := applesciprt.Tell("Music", `set musicState to get player state`)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return "", err
	}
	switch result {
	case "playing":
		return common.PlayerStatePlaying, nil
	case "paused":
		return common.PlayerStatePaused, nil
	default:
		return common.PlayerState(result), nil
	}
}

// GetNowPlayingTrackInfo 使用 AppleScript 从 Apple Music 获取当前正在播放的曲目信息。
// 它返回一个指向 TrackInfo 结构体的指针，包含曲目的详细信息。
// 如果在执行 AppleScript 或解析数据时发生错误，函数会记录警告并返回 nil。
func GetNowPlayingTrackInfo(ctx context.Context) *TrackInfo {
	// 首先检查是否正在播放
	state, err := GetState(ctx)
	if err != nil || state != common.PlayerStatePlaying {
		// 如果没有播放或获取状态出错，返回nil
		return nil
	}

	// 使用更简洁的AppleScript代码获取所有相关信息
	tell, err := applesciprt.Tell(
		"Music",
		`try
			if player state is playing then
				if exists current track then
					set t to current track
					set trackInfo to {name:(name of t), album:(album of t), artist:(artist of t), albumArtist:(album artist of t), duration:(duration of t), playerPosition:(player position), databaseID:(database ID of t), composer:(composer of t), albumDisliked:(album disliked of t), albumFavorited:(album favorited of t), albumRating:(album rating of t), bitRate:(bit rate of t), bookmark:(bookmark of t), bpm:(bpm of t), category:(category of t), comment:(comment of t), compilation:(compilation of t), dateAdded:((date added of t) as string), description:(description of t), discCount:(disc count of t), discNumber:(disc number of t), disliked:(disliked of t), enabled:(enabled of t), eq:(EQ of t), finish:(finish of t), gapless:(gapless of t), genre:(genre of t), grouping:(grouping of t), kind:(kind of t), longDescription:(long description of t), favorited:(favorited of t), lyrics:(lyrics of t), modificationDate:((modification date of t) as string), movement:(movement of t), movementCount:(movement count of t), movementNumber:(movement number of t), playedCount:(played count of t), playedDate:((played date of t) as string), rating:(rating of t), releaseDate:((release date of t) as string), sampleRate:(sample rate of t), shufflable:(shufflable of t), skippedCount:(skipped count of t), skippedDate:((skipped date of t) as string), sortAlbum:(sort album of t), sortArtist:(sort artist of t), sortAlbumArtist:(sort album artist of t), sortName:(sort name of t), sortComposer:(sort composer of t), size:(size of t), start:(start of t), trackCount:(track count of t), trackNumber:(track number of t), unplayed:(unplayed of t), volumeAdjustment:(volume adjustment of t), work:(work of t), year:(year of t)}
					return trackInfo
				end if
			end if
		on error errMsg
			return "error:" & errMsg
		end try`,
	)
	if err != nil {
		alog.Warn(ctx, "AppleScript execution error:", zap.Error(err))
		return nil
	}

	// 检查是否有错误
	if strings.HasPrefix(tell, "error:") {
		alog.Warn(ctx, "AppleScript runtime error:", zap.String("response", tell))
		return nil
	}

	// 如果没有返回数据，说明没有播放曲目
	if tell == "" {
		return nil
	}

	// 解析AppleScript返回的记录格式数据
	// 格式类似于: {name:Song Title, album:Album Name, artist:Artist Name, ...}
	info := &TrackInfo{
		TrackBase: TrackBase{
			IsMusicApp: true,
		},
	}

	// 移除首尾的大括号
	tell = strings.TrimPrefix(tell, "{")
	tell = strings.TrimSuffix(tell, "}")

	// 按逗号分割各个字段
	fields := strings.Split(tell, ", ")

	// 解析每个字段
	for _, field := range fields {
		// 查找冒号位置
		colonIndex := strings.Index(field, ":")
		if colonIndex <= 0 || colonIndex >= len(field)-1 {
			continue
		}

		// 提取键和值
		key := strings.TrimSpace(field[:colonIndex])
		value := strings.TrimSpace(field[colonIndex+1:])

		// 处理"missing value"的特殊情况
		if value == "missing value" {
			continue
		}

		// 根据键设置相应的字段
		switch key {
		case "name":
			info.Title = strings.Trim(value, "\"")
		case "album":
			info.Album = strings.Trim(value, "\"")
		case "artist":
			info.Artist = strings.Trim(value, "\"")
		case "albumArtist":
			info.AlbumArtist = strings.Trim(value, "\"")
		case "duration":
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				info.Duration = int64(num)
			}
		case "playerPosition":
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				info.Position = num
			}
		case "databaseID":
			if num, err := strconv.ParseInt(value, 10, 64); err == nil {
				info.Url = fmt.Sprintf("%d", num)
				info.DatabaseID = int(num)
			}
		case "composer":
			info.Composer = strings.Trim(value, "\"")
		case "albumDisliked":
			if b, err := strconv.ParseBool(value); err == nil {
				info.AlbumDisliked = b
			}
		case "albumFavorited":
			if b, err := strconv.ParseBool(value); err == nil {
				info.AlbumFavorited = b
			}
		case "albumRating":
			if num, err := strconv.Atoi(value); err == nil {
				info.AlbumRating = num
			}
		case "bitRate":
			if num, err := strconv.Atoi(value); err == nil {
				info.BitRate = num
			}
		case "bookmark":
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				info.Bookmark = num
			}
		case "bpm":
			if num, err := strconv.Atoi(value); err == nil {
				info.BPM = num
			}
		case "category":
			info.Category = strings.Trim(value, "\"")
		case "comment":
			info.Comment = strings.Trim(value, "\"")
		case "compilation":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Compilation = b
			}
		case "dateAdded":
			// 尝试不同的日期格式
			if t, err := common.ParseChineseTime(value); err == nil {
				info.DateAdded = t
			} else if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", strings.Trim(value, "\"")); err == nil {
				info.DateAdded = t
			}
		case "description":
			info.Description = strings.Trim(value, "\"")
		case "discCount":
			if num, err := strconv.Atoi(value); err == nil {
				info.DiscCount = num
			}
		case "discNumber":
			if num, err := strconv.Atoi(value); err == nil {
				info.DiscNumber = num
			}
		case "disliked":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Disliked = b
			}
		case "enabled":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Enabled = b
			}
		case "eq":
			info.EQ = strings.Trim(value, "\"")
		case "finish":
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				info.Finish = num
			}
		case "gapless":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Gapless = b
			}
		case "genre":
			info.Genre = common.ConversionSimplifiedFx(strings.Trim(value, "\""))
		case "grouping":
			info.Grouping = strings.Trim(value, "\"")
		case "kind":
			info.Kind = strings.Trim(value, "\"")
		case "longDescription":
			info.LongDescription = strings.Trim(value, "\"")
		case "favorited":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Favorited = b
			}
		case "lyrics":
			info.Lyrics = strings.Trim(value, "\"")
		case "modificationDate":
			// 尝试不同的日期格式
			if t, err := common.ParseChineseTime(value); err == nil {
				info.ModificationDate = t
			} else if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", strings.Trim(value, "\"")); err == nil {
				info.ModificationDate = t
			}
		case "movement":
			info.Movement = strings.Trim(value, "\"")
		case "movementCount":
			if num, err := strconv.Atoi(value); err == nil {
				info.MovementCount = num
			}
		case "movementNumber":
			if num, err := strconv.Atoi(value); err == nil {
				info.MovementNumber = num
			}
		case "playedCount":
			if num, err := strconv.Atoi(value); err == nil {
				info.PlayedCount = num
			}
		case "playedDate":
			// 尝试不同的日期格式
			if t, err := common.ParseChineseTime(value); err == nil {
				info.PlayedDate = t
			} else if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", strings.Trim(value, "\"")); err == nil {
				info.PlayedDate = t
			}
		case "rating":
			if num, err := strconv.Atoi(value); err == nil {
				info.Rating = num
			}
		case "releaseDate":
			// 尝试不同的日期格式
			if t, err := common.ParseChineseTime(value); err == nil {
				info.ReleaseDate = t
			} else if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", strings.Trim(value, "\"")); err == nil {
				info.ReleaseDate = t
			}
		case "sampleRate":
			if num, err := strconv.Atoi(value); err == nil {
				info.SampleRate = num
			}
		case "shufflable":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Shufflable = b
			}
		case "skippedCount":
			if num, err := strconv.Atoi(value); err == nil {
				info.SkippedCount = num
			}
		case "skippedDate":
			// 尝试不同的日期格式
			if t, err := common.ParseChineseTime(value); err == nil {
				info.SkippedDate = t
			} else if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", strings.Trim(value, "\"")); err == nil {
				info.SkippedDate = t
			}
		case "sortAlbum":
			info.SortAlbum = strings.Trim(value, "\"")
		case "sortArtist":
			info.SortArtist = strings.Trim(value, "\"")
		case "sortAlbumArtist":
			info.SortAlbumArtist = strings.Trim(value, "\"")
		case "sortName":
			info.SortName = strings.Trim(value, "\"")
		case "sortComposer":
			info.SortComposer = strings.Trim(value, "\"")
		case "size":
			if num, err := strconv.ParseInt(value, 10, 64); err == nil {
				info.Size = num
			}
		case "start":
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				info.Start = num
			}
		case "trackCount":
			if num, err := strconv.Atoi(value); err == nil {
				info.TrackCount = num
			}
		case "trackNumber":
			if num, err := strconv.Atoi(value); err == nil {
				info.TrackNumber = num
			}
		case "unplayed":
			if b, err := strconv.ParseBool(value); err == nil {
				info.Unplayed = b
			}
		case "volumeAdjustment":
			if num, err := strconv.Atoi(value); err == nil {
				info.VolumeAdjustment = num
			}
		case "work":
			info.Work = strings.Trim(value, "\"")
		case "year":
			if num, err := strconv.Atoi(value); err == nil {
				info.Year = num
			}
		}
	}

	return info
}

func GetNowPlayingTrackInfoV2(ctx context.Context) *TrackInfo {
	// 首先检查是否正在播放
	state, err := GetState(ctx)
	if err != nil || state != common.PlayerStatePlaying {
		// 如果没有播放或获取状态出错，返回nil
		return nil
	}

	// 使用更简洁的AppleScript代码获取所有相关信息
	tell, err := run(appleSciprtGetNowPlayingTrackInfo)
	if err != nil {
		alog.Warn(ctx, "AppleScript runtime error:", zap.String("response", tell))
		return nil
	}

	// 检查是否有错误
	if strings.HasPrefix(tell, "error:") {
		alog.Warn(ctx, "AppleScript runtime error:", zap.String("response", tell))
		return nil
	}

	// 如果没有返回数据，说明没有播放曲目
	if tell == "" {
		return nil
	}

	// 解析AppleScript返回的记录格式数据
	// 格式类似于: {name:Song Title, album:Album Name, artist:Artist Name, ...}
	tmp, err := parseTrackInfo(tell)
	if err != nil {
		alog.Warn(ctx, "AppleScript parse error:", zap.String("response", tell))
		return nil
	}
	trackInfo := convertTmpToTrackBase(tmp)
	trackInfo.IsMusicApp = true
	return trackInfo
}

// IsFavorite checks if the current track is favorited in Apple Music
func IsFavorite(ctx context.Context) (bool, error) {
	tell, err := applesciprt.Tell(
		"Music",
		`if exists current track then
		return favorited of current track
	end if`,
	)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return false, err
	}

	parseBool, err := strconv.ParseBool(tell)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return false, err
	}
	return parseBool, nil
}

// SetFavorite sets the favorited status of the current track in Apple Music
func SetFavorite(ctx context.Context, favorited bool) error {
	alog.Debug(ctx, "apple music. Track love status:", zap.Bool("favorited", favorited))
	_, err := applesciprt.Tell(
		"Music",
		fmt.Sprintf(`set favorited of current track to %s`, strconv.FormatBool(favorited)),
	)
	if err != nil {
		alog.Warn(ctx, "err:", zap.Error(err))
		return err
	}
	alog.Info(ctx, "apple music. Track loved successfully")
	return nil
}
func parseTrackInfo(output string) (*tmp, error) {
	var info tmp
	err := json.Unmarshal([]byte(output), &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// Build the AppleScript command from a set of optional parameters, return the output
func run(command string) (string, error) {
	cmd := exec.Command("osascript", "-e", command)
	output, err := cmd.CombinedOutput()
	prettyOutput := strings.Replace(string(output), "\n", "", -1)

	// Ignore errors from the user hitting the cancel button
	if err != nil && strings.Index(string(output), "User canceled.") < 0 {
		return "", errors.New(err.Error() + ": " + prettyOutput + " (" + command + ")")
	}

	return prettyOutput, nil
}

func convertTmpToTrackBase(t *tmp) *TrackInfo {
	result := new(TrackInfo)

	// 字符串直接赋值字段
	result.Title = t.Name // tmp的Name对应TrackBase的Title
	result.Album = t.Album
	result.Artist = t.Artist
	result.AlbumArtist = t.AlbumArtist
	result.Composer = t.Composer
	result.Genre = common.ConversionSimplifiedFx(t.Genre)
	result.DurationString = t.Duration // 保留原始字符串格式的时长
	result.EQ = t.EQ
	result.Category = t.Category
	result.Comment = t.Comment
	result.Description = t.Description
	result.Grouping = t.Kind
	result.Kind = t.Kind
	result.LongDescription = t.LongDescription
	result.Lyrics = t.Lyrics
	result.Movement = t.Movement
	result.SortAlbum = t.SortAlbum
	result.SortArtist = t.SortArtist
	result.SortAlbumArtist = t.SortAlbumArtist
	result.SortName = t.SortName
	result.SortComposer = t.SortComposer
	result.Work = t.Work

	// 整数类型转换
	result.Duration = int64(parseFloat(t.Duration))
	result.TrackNumber = parseInt(t.TrackNumber)
	result.DatabaseID = parseInt(t.DatabaseID)
	result.AlbumRating = parseInt(t.AlbumRating)
	result.BitRate = parseInt(t.BitRate)
	result.BPM = parseInt(t.Bpm)
	result.DiscCount = parseInt(t.DiscCount)
	result.DiscNumber = parseInt(t.DiscNumber)
	result.MovementCount = parseInt(t.MovementCount)
	result.MovementNumber = parseInt(t.MovementNumber)
	result.PlayedCount = parseInt(t.PlayedCount)
	result.Rating = parseInt(t.Rating)
	result.SampleRate = parseInt(t.SampleRate)
	result.SkippedCount = parseInt(t.SkippedCount)
	result.Size = parseInt64(t.Size)
	result.TrackCount = parseInt(t.TrackCount)
	result.VolumeAdjustment = parseInt(t.VolumeAdjustment)
	result.Year = parseInt(t.Year)

	// 浮点类型转换
	result.Position = parseFloat(t.PlayerPosition)
	result.Bookmark = parseFloat(t.Bookmark)
	result.Finish = parseFloat(t.Finish)
	result.Start = parseFloat(t.Start)

	// 布尔类型转换
	result.AlbumDisliked = parseBool(t.AlbumDisliked)
	result.AlbumFavorited = parseBool(t.AlbumFavorited)
	result.Compilation = parseBool(t.Compilation)
	result.Disliked = parseBool(t.Disliked)
	result.Enabled = parseBool(t.Enabled)
	result.Gapless = parseBool(t.Gapless)
	result.Favorited = parseBool(t.Favorited)
	result.Shufflable = parseBool(t.Shufflable)
	result.Unplayed = parseBool(t.Unplayed)

	// 时间类型转换（尝试多种常见格式）
	result.DateAdded = parseTime(t.DateAdded)
	result.ModificationDate = parseTime(t.ModificationDate)
	result.PlayedDate = parseTime(t.PlayedDate)
	result.ReleaseDate = parseTime(t.ReleaseDate)
	result.SkippedDate = parseTime(t.SkippedDate)

	return result
}

// 辅助函数：将字符串转换为int
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// 辅助函数：将字符串转换为int64
func parseInt64(s string) int64 {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// 辅助函数：将字符串转换为float64
func parseFloat(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// 辅助函数：将字符串转换为bool
func parseBool(s string) bool {
	// 处理常见的布尔值表示形式
	switch s {
	case "true", "1", "yes":
		return true
	case "false", "0", "no", "":
		return false
	default:
		// 尝试直接解析
		val, _ := strconv.ParseBool(s)
		return val
	}
}

// 辅助函数：将字符串转换为time.Time
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// 尝试不同的日期格式
	if t, err := common.ParseChineseTime(s); err == nil {
		return t
	} else {
		// 尝试多种常见的时间格式
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"01/02/2006",
		}

		for _, format := range formats {
			t, err := time.Parse(format, s)
			if err == nil {
				return t
			}
		}

		// 尝试Unix时间戳（秒）
		if unix, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(unix, 0)
		}

		// 尝试Unix时间戳（毫秒）
		if unixMs, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(unixMs/1000, (unixMs%1000)*1e6)
		}
	}

	return time.Time{}
}
