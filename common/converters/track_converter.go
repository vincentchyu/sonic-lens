package converters

import (
	"strings"

	"github.com/spf13/cast"

	"github.com/vincentchyu/sonic-lens/core/exec"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// ConvertExiftoolInfoToTrack converts ExiftoolInfo to Track model
func ConvertExiftoolInfoToTrack(exiftoolInfo *exec.ExiftoolInfo, source string) *model.Track {
	if exiftoolInfo == nil {
		return nil
	}

	track := &model.Track{
		Artist:        exiftoolInfo.GetArtist(),
		AlbumArtist:   exiftoolInfo.GetArtist(),
		Album:         exiftoolInfo.GetAlbum(),
		Track:         exiftoolInfo.GetTitle(),
		TrackNumber:   int8(exiftoolInfo.GetTrackNumber()),
		Genre:         exiftoolInfo.GetGenre(),
		Composer:      exiftoolInfo.GetComposer(),
		ReleaseDate:   exiftoolInfo.GetReleaseDate(),
		MusicBrainzID: exiftoolInfo.GetMusicBrainzTrackId(),
		Source:        source,
	}

	track.Duration = exiftoolInfo.GetDuration()
	return track
}

// ConvertMediaControlInfoToTrack converts MediaControlNowPlayingInfo to Track model
func ConvertMediaControlInfoToTrack(mediaInfo *exec.MediaControlNowPlayingInfo, source string) *model.Track {
	if mediaInfo == nil {
		return nil
	}

	track := &model.Track{
		Artist:      mediaInfo.Artist,
		Album:       mediaInfo.Album,
		Track:       mediaInfo.Title,
		TrackNumber: int8(mediaInfo.TrackNumber),
		Duration:    mediaInfo.Duration,
		Genre:       mediaInfo.Genre,
		Composer:    mediaInfo.Composer,
		Source:      source,
		// ReleaseDate: mediaInfo.releaseDate,

		BundleID: mediaInfo.BundleIdentifier,
		UniqueID: cast.ToString(mediaInfo.UniqueIdentifier),
	}

	// Format timestamp as release date string
	if !mediaInfo.Timestamp.IsZero() {
		track.ReleaseDate = mediaInfo.Timestamp.Format("2006-01-02")
	}

	return track
}

// getStringValue retrieves a string value from ExiftoolInfo with multiple possible keys
func getStringValue(info exec.ExiftoolInfo, key1, key2 string) string {
	var val any
	val, ok := info[key1]
	if ok {
		return cast.ToString(val)
	}
	val, ok = info[key2]
	if ok {
		return cast.ToString(val)
	}
	return ""
}

// parseDurationString converts duration string like "0:05:48" to seconds
func parseDurationString(durationStr string) int64 {
	// Handle format like "0:05:48" (hours:minutes:seconds)
	parts := strings.Split(durationStr, ":")
	if len(parts) == 3 {
		hours := cast.ToInt64(strings.TrimSpace(parts[0]))
		minutes := cast.ToInt64(strings.TrimSpace(parts[1]))
		seconds := cast.ToInt64(strings.TrimSpace(parts[2]))
		return hours*3600 + minutes*60 + seconds
	}

	// Handle format like "5:48" (minutes:seconds)
	if len(parts) == 2 {
		minutes := cast.ToInt64(strings.TrimSpace(parts[0]))
		seconds := cast.ToInt64(strings.TrimSpace(parts[1]))
		return minutes*60 + seconds
	}

	// Handle format like "288" (seconds)
	if len(parts) == 1 {
		return cast.ToInt64(strings.TrimSpace(parts[0]))
	}

	return 0
}

// getSourceFromBundleID maps bundle identifier to source name
func getSourceFromBundleID(bundleID string) string {
	switch bundleID {
	case "com.apple.Music":
		return "Apple Music"
	case "com.roon.Roon":
		return "Roon"
	case "com.netease.163music":
		return "NetEase Music"
	default:
		return bundleID
	}
}
