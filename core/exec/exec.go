package exec

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-audio/wav"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	alog "github.com/vincentchyu/sonic-lens/core/log"
)

const (
	MRMediaNowPlayingGet                   = "get"
	MRMediaNowPlayingAppRoon               = "com.roon.Roon"
	MRMediaNowPlayingAppMusic              = "com.apple.Music"
	MRMediaNowPlaying163                   = "com.netease.163music"
	MRMediaNowPlayingBundleIdentifier      = "bundleIdentifier"
	MRMediaNowPlayingContentItemIdentifier = "contentItemIdentifier"

	MRMediaNowPlayingAlbum  = "album"
	MRMediaNowPlayingTitle  = "title"
	MRMediaNowPlayingArtist = "artist"

	MRMediaNowPlayingIsPlaying = "isPlaying"
	MRMediaNowPlayingPlaying   = "playing"

	MRMediaNowPlayingDuration         = "duration"
	MRMediaNowPlayingElapsedTime      = "elapsedTime"
	MRMediaNowPlayingElapsedTimeNow   = "elapsedTimeNow"
	MRMediaNowPlayingTimestamp        = "timestamp"
	MRMediaNowPlayingMediaType        = "mediaType"
	MRMediaNowPlayingIsMusicApp       = "isMusicApp"
	MRMediaNowPlayingUniqueIdentifier = "uniqueIdentifier"

	// MediaControlCmd MediaControl commands
	MediaControlCmd      = "media-control"
	MediaControlGet      = "get"
	MediaControlNowFlag  = "--now"
	MediaControlHelpFlag = "-h"
)

type (
	MataDataHandle interface {
		GetTitle() string
		GetArtists() string
		GetArtist() string
		GetAlbum() string
		GetTrackNumber() int64
		GetMusicBrainzTrackId() string
		GetGenre() string
		GetComposer() string
		GetDuration() int64
		GetReleaseDate() string
		GetSource() string
		GetBundleID() string
		GetUniqueID() string
	}

	ExiftoolInfo map[string]any
	WavInfo      struct {
		wav.Metadata
	}
	MRMediaNowPlaying struct {
		Title            string  `json:"title"`
		Artist           string  `json:"artist"`
		Album            string  `json:"album"`
		IsPlaying        bool    `json:"isPlaying"`
		Duration         float64 `json:"duration"`
		ElapsedTime      float64 `json:"elapsed_time"`
		BundleIdentifier string  `json:"bundleIdentifier"`
	}
	MediaControlNowPlayingInfo struct {
		// media-control get -h --now
		TrackID         string
		Title           string `json:"title"`           // 标题
		Album           string `json:"album"`           // 专辑
		Artist          string `json:"artist"`          // 艺术家
		Genre           string `json:"genre"`           // 流派
		TrackNumber     int    `json:"trackNumber"`     // 曲目编号
		TotalTrackCount int    `json:"totalTrackCount"` // 总曲目数

		Playing        bool    `json:"playing"`        // 是否正在播放
		Duration       int64   `json:"duration"`       // 持续时间
		ElapsedTime    float64 `json:"elapsedTime"`    // 播放时间（在暂时无用）
		ElapsedTimeNow float64 `json:"elapsedTimeNow"` // 当前播放时间
		IsMusicApp     bool    `json:"isMusicApp"`     // 是否是音乐应用

		Composer              string `json:"composer"`              // 作曲家
		BundleIdentifier      string `json:"bundleIdentifier"`      // 软件标识
		ContentItemIdentifier string `json:"contentItemIdentifier"` // 疑似歌曲id

		ArtworkData       string    `json:"artworkData"`       // 封面数据
		UniqueIdentifier  int64     `json:"uniqueIdentifier"`  // 唯一标识符
		RepeatMode        int       `json:"repeatMode"`        // 重复模式
		QueueIndex        int       `json:"queueIndex"`        // 队列索引
		ArtworkMimeType   string    `json:"artworkMimeType"`   // 封面类型
		MediaType         string    `json:"mediaType"`         // 媒体类型
		Timestamp         time.Time `json:"timestamp"`         // 时间戳
		ShuffleMode       int       `json:"shuffleMode"`       //	洗牌模式
		ProcessIdentifier int       `json:"processIdentifier"` // 进程标识符
		TotalQueueCount   int       `json:"totalQueueCount"`   // 总队列数
		PlaybackRate      int       `json:"playbackRate"`      // 播放速率

		Position   float64 // 播放位置
		Url        string  // 歌曲链接
		AirfoiLogo string  // 封面
	}
)

func BuildExiftoolHandle(ctx context.Context, file string) (MataDataHandle, error) {
	infos := make([]*ExiftoolInfo, 0)
	res := new(ExiftoolInfo)
	command, err := runCommand(ctx, "exiftool", "-json", file)
	if err != nil {
		alog.Warn(ctx, "fail to get exiftool info", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal([]byte(command), &infos)
	if err != nil {
		return nil, err
	}
	if len(infos) > 0 {
		res = infos[0]
	}
	return res, nil
}

func BuildWavInfoHandle(file string) (MataDataHandle, error) {
	wavInfo := new(WavInfo)
	in, err := os.Open(file)
	defer func(in *os.File) {
		err := in.Close()
		if err != nil {
			alog.Error(context.Background(), "Failed to close file", zap.String("file", file), zap.Error(err))
		}
	}(in)
	if err != nil {
		return nil, err
	}
	if mwav := wav.NewDecoder(in); mwav.IsValidFile() {
		mwav.ReadMetadata()
		wavInfo.Metadata = *mwav.Metadata
	}
	return wavInfo, nil
}

func (receiver ExiftoolInfo) GetTitle() string {
	key1, key2 := "Title", "title"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

func (receiver ExiftoolInfo) GetArtists() string {
	key1, key2 := "Artists", "artists"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	return ""
}
func (receiver ExiftoolInfo) GetArtist() string {
	key1, key2 := "Artist", "artist"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	return ""
}
func (receiver ExiftoolInfo) GetAlbum() string {
	key1, key2, key3 := "Album", "album", "album_"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetTrackNumber GetTrackNumber
func (receiver ExiftoolInfo) GetTrackNumber() int64 {
	key1, key2, key3 := "TrackNumber", "Tracknumber", "tracknumber"
	//  "TrackNumber": "1 of 12",
	//   "TrackNumber": 1,
	var val any
	val, ok := receiver[key1]
	if ok {
		toString := cast.ToString(val)
		if ok := strings.Contains(toString, "of"); ok {
			split := strings.Split(toString, "of")
			return cast.ToInt64(strings.TrimSpace(split[0]))
		}
		return cast.ToInt64(toString)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToInt64(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToInt64(val)
	}
	return 0
}

// GetMusicBrainzTrackID GetMusicBrainzTrackID
func (receiver ExiftoolInfo) GetMusicBrainzTrackId() string {
	key1, key2 := "MusicbrainzTrackid", "MusicBrainzReleaseTrackId"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetGenre returns the genre of the track
func (receiver ExiftoolInfo) GetGenre() string {
	key1, key2, key3 := "Genre", "genre", "GenreName"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetComposer returns the composer of the track
func (receiver ExiftoolInfo) GetComposer() string {
	key1, key2, key3 := "Composer", "composer", "ComposerName"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetDuration returns the duration of the track in seconds
func (receiver ExiftoolInfo) GetDuration() int64 {
	key1, key2, key3 := "Duration", "duration", "TrackDuration"
	var val any
	val, ok := receiver[key1]
	if ok {
		// "Duration": "0:05:48" 是这个格式
		// 分钟加秒的字符串转换成秒
		if strVal, ok := val.(string); ok {
			parts := strings.Split(strVal, ":")
			if len(parts) == 3 {
				hours := cast.ToInt64(parts[0])
				minutes := cast.ToInt64(parts[1])
				seconds := cast.ToInt64(parts[2])
				return hours*3600 + minutes*60 + seconds
			} else if len(parts) == 2 {
				minutes := cast.ToInt64(parts[0])
				seconds := cast.ToInt64(parts[1])
				return minutes*60 + seconds
			} else if len(parts) == 1 {
				return cast.ToInt64(parts[0])
			}
		}
		return cast.ToInt64(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToInt64(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToInt64(val)
	}
	return 0
}

// GetReleaseDate returns the release date of the track
func (receiver ExiftoolInfo) GetReleaseDate() string {
	key1, key2, key3, key4 := "Originaldate", "ReleaseDate", "RELEASETIME", "OriginalDate"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key4]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetSource returns the source of the track metadata
func (receiver ExiftoolInfo) GetSource() string {
	key1, key2 := "ISRC", "ISRCNumber"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetBundleID returns the bundle identifier of the application
func (receiver ExiftoolInfo) GetBundleID() string {
	key1, key2, key3, key4, key5 := "BundleID", "Comment", "BundleIdentifier", "ISRC", "ISRCNumber"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key4]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key5]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetUniqueID returns the unique identifier of the track
func (receiver ExiftoolInfo) GetUniqueID() string {
	key1, key2, key3 := "MD5Signature", "AppleStoreCatalogID", "UniqueIdentifier"
	var val any
	val, ok := receiver[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key2]
	if ok {
		return cast.ToString(val)
	}
	val, ok = receiver[key3]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// GetTitle returns the title of the track
func (receiver *WavInfo) GetTitle() string {
	return receiver.Title
}

// GetArtists returns the artists of the track
func (receiver *WavInfo) GetArtists() string {
	return receiver.Artist
}

// GetArtist returns the primary artist of the track
func (receiver *WavInfo) GetArtist() string {
	return receiver.Artist
}

// GetAlbumartist returns the album artist
func (receiver *WavInfo) GetAlbum() string {
	return receiver.Product
}

// GetTrackNumber returns the track number
func (receiver *WavInfo) GetTrackNumber() int64 {
	return castToInt64(receiver.TrackNbr)
}

// GetMusicBrainzTrackId returns the MusicBrainz track ID
func (receiver *WavInfo) GetMusicBrainzTrackId() string {
	// WAV files don't have a standard MusicBrainz track ID field
	return ""
}

// GetGenre returns the genre of the track
func (receiver *WavInfo) GetGenre() string {
	// WAV files have a Genre field in metadata
	return receiver.Genre
}

// GetComposer returns the composer of the track
func (receiver *WavInfo) GetComposer() string {
	// WAV files don't have a standard composer field in the Metadata struct
	return ""
}

// GetDuration returns the duration of the track in seconds
func (receiver *WavInfo) GetDuration() int64 {
	// WAV files don't have a standard duration field in metadata
	// Duration would typically come from the file itself, not metadata
	return 0
}

// GetReleaseDate returns the release date of the track
func (receiver *WavInfo) GetReleaseDate() string {
	// WAV files have a CreationDate field in metadata
	return receiver.CreationDate
}

// GetSource returns the source of the track metadata
func (receiver *WavInfo) GetSource() string {
	// WAV files have a Source field in metadata
	return receiver.Source
}

// GetBundleID returns the bundle identifier of the application
func (receiver *WavInfo) GetBundleID() string {
	// WAV files don't have a bundle identifier in metadata
	return ""
}

// GetUniqueID returns the unique identifier of the track
func (receiver *WavInfo) GetUniqueID() string {
	// WAV files don't have a standard unique identifier field in metadata
	return ""
}

func GetMRMediaNowPlayingCli(ctx context.Context) (*MRMediaNowPlaying, error) {
	// nowplaying-cli  get album title artist duration elapsedTime timestamp mediaType isMusicApp  uniqueIdentifier
	args := []string{
		MRMediaNowPlayingGet,
		MRMediaNowPlayingAlbum,
		MRMediaNowPlayingTitle,
		MRMediaNowPlayingArtist,
		MRMediaNowPlayingDuration,
		MRMediaNowPlayingElapsedTime,
		MRMediaNowPlayingTimestamp,
		MRMediaNowPlayingMediaType,
		MRMediaNowPlayingIsMusicApp,
		MRMediaNowPlayingUniqueIdentifier,
	}
	curList := map[string]int{
		MRMediaNowPlayingBundleIdentifier: 0,
		MRMediaNowPlayingIsPlaying:        1,
		MRMediaNowPlayingAlbum:            2,
		MRMediaNowPlayingTitle:            3,
		MRMediaNowPlayingArtist:           4,
		MRMediaNowPlayingDuration:         5,
		MRMediaNowPlayingElapsedTime:      6,
		MRMediaNowPlayingTimestamp:        7,
		MRMediaNowPlayingMediaType:        8,
		MRMediaNowPlayingIsMusicApp:       9,
		MRMediaNowPlayingUniqueIdentifier: 10,
	}
	output, err := runCommand(ctx, "nowplaying-cli-mac", args...)
	if err != nil {
		return nil, err
	}
	MRMediaNowPlayingList := strings.Split(output, "\n")
	var np MRMediaNowPlaying
	if len(MRMediaNowPlayingList) > 10 {
		artists := cast.ToString(MRMediaNowPlayingList[curList[MRMediaNowPlayingArtist]])
		artist := artists
		if artistList := strings.Split(artists, ","); len(artistList) > 0 {
			artist = artistList[0]
		}
		np = MRMediaNowPlaying{
			Title:            cast.ToString(MRMediaNowPlayingList[curList[MRMediaNowPlayingTitle]]),
			Artist:           artist,
			Album:            cast.ToString(MRMediaNowPlayingList[curList[MRMediaNowPlayingAlbum]]),
			IsPlaying:        cast.ToString(MRMediaNowPlayingList[curList[MRMediaNowPlayingIsPlaying]]) == "YES",
			Duration:         cast.ToFloat64(MRMediaNowPlayingList[curList[MRMediaNowPlayingDuration]]),
			ElapsedTime:      cast.ToFloat64(MRMediaNowPlayingList[curList[MRMediaNowPlayingElapsedTime]]),
			BundleIdentifier: cast.ToString(MRMediaNowPlayingList[curList[MRMediaNowPlayingBundleIdentifier]]),
		}
	}
	return &np, nil
}

// GetMediaControlNowPlaying executes media-control command to get current playing track info
func GetMediaControlNowPlaying(ctx context.Context) (*MediaControlNowPlayingInfo, error) {
	// Execute: media-control get -h --now
	output, err := runCommand(ctx, MediaControlCmd, MediaControlGet, MediaControlHelpFlag, MediaControlNowFlag)
	if err != nil {
		return nil, err
	}

	// Parse JSON output
	var info MediaControlNowPlayingInfo
	err = json.Unmarshal([]byte(output), &info)
	if err != nil {
		return nil, fmt.Errorf("error parsing media-control output: %v", err)
	}

	return &info, nil
}

func castToInt64(val any) int64 {
	switch v := val.(type) {
	case string:
		var toInt64 int64 = 0
		if strings.Contains(v, "/") {
			tmp := strings.Split(v, "/")
			if len(tmp) > 0 {
				toInt64 = cast.ToInt64(strings.TrimSpace(tmp[0]))
			}
		} else if strings.Contains(v, "-") {
			tmp := strings.Split(v, "-")
			if len(tmp) > 0 {
				toInt64 = cast.ToInt64(strings.TrimSpace(tmp[0]))
			}
		} else if strings.Contains(v, "of") {
			tmp := strings.Split(v, "of")
			if len(tmp) > 0 {
				toInt64 = cast.ToInt64(strings.TrimSpace(tmp[0]))
			}
		} else {
			toInt64 = cast.ToInt64(strings.TrimSpace(v))
		}
		return toInt64
	case int, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
		return cast.ToInt64(v)
	}
	return 0
}

func runCommand(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		alog.Warn(ctx, "error executing command", zap.Error(err))
		return "", errors.New(string(output))
	}
	return string(output), nil
}

func IsValidPath(ctx context.Context, path string) (bool, string, error) {
	// 确保路径不是空字符串
	if path == "" {
		return false, "", fmt.Errorf("empty or undefined path")
	}
	path, _ = strings.CutPrefix(path, "file://")
	// 使用 filepath.Clean() 处理任何多余的斜杠或其他非法字符，确保路径整洁。
	path = filepath.Clean(path)
	// 解析符号链接和相对路径到绝对路径
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		alog.Warn(ctx, "Cannot resolve symlinks:", zap.Error(err))
		// 检查是否因为文件不存在而失败，如果是则返回。
		var pathError *os.PathError
		isNotExist := errors.As(err, &pathError)
		if isNotExist || os.IsNotExist(err) {
			return false, "", fmt.Errorf("path does not exist: %s", path)
		}
		// 如果不是因为路径不存在导致的错误，则记录并返回
		alog.Warn(ctx, "Unknown error occurred while resolving symlinks:", zap.Error(err))

		return false, "", err
	}
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		alog.Info(
			ctx,
			"Cannot stat the path: %s - Error: ", zap.String("resolvedPath", resolvedPath), zap.Error(err),
		)
		return false, "", fmt.Errorf("error while checking file existence: %s", err)
	}
	// 根据fileInfo.IsDir()判断是文件还是目录
	isDirectory := fileInfo.IsDir()
	alog.Info(
		ctx,
		fmt.Sprintf("checkValidPath:The path [%s] exists and [%v] a directory", resolvedPath, isDirectory),
	)
	return true, resolvedPath, nil
}
func GetFilePathExt(path string) string {
	return filepath.Ext(path)
}
