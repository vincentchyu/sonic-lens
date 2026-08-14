package applemusic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrackInfoToTrackMetadata(t *testing.T) {
	relDate := time.Date(1973, 3, 1, 0, 0, 0, 0, time.UTC)
	info := &TrackInfo{
		TrackBase: TrackBase{
			Title:            "Time",
			Album:            "The Dark Side of the Moon",
			Artist:           "Pink Floyd",
			AlbumArtist:      "Pink Floyd",
			TrackNumber:      4,
			DiscNumber:       1,
			Duration:         425,
			Genre:            "Progressive Rock",
			Composer:         "David Gilmour, Nick Mason, Roger Waters, Richard Wright",
			ReleaseDate:      relDate,
			BundleIdentifier: "com.apple.Music",
			UniqueIdentifier: 987654321,
		},
	}

	metadata := info.ToTrackMetadata()
	assert.Equal(t, "Pink Floyd", metadata.AlbumArtist)
	assert.Equal(t, int8(4), metadata.TrackNumber)
	assert.Equal(t, int64(425), metadata.Duration)
	assert.Equal(t, "Progressive Rock", metadata.Genre)
	assert.Equal(t, "David Gilmour, Nick Mason, Roger Waters, Richard Wright", metadata.Composer)
	assert.Equal(t, "1973-03-01", metadata.ReleaseDate)
	assert.Equal(t, "Apple Music", metadata.Source)
	assert.Equal(t, "com.apple.Music", metadata.BundleID)
	assert.Equal(t, "987654321", metadata.UniqueID)
}

func TestTrackInfoFields(t *testing.T) {
	relDate := time.Date(2000, 10, 2, 0, 0, 0, 0, time.UTC)

	info := &TrackInfo{
		TrackBase: TrackBase{
			Title:            "Everything in Its Right Place",
			Album:            "Kid A",
			Artist:           "Radiohead",
			AlbumArtist:      "Radiohead",
			TrackNumber:      1,
			DiscNumber:       1,
			Duration:         251,
			Position:         12.5,
			SampleRate:       44100,
			Genre:            "Electronic / Experimental",
			Composer:         "Thom Yorke",
			ReleaseDate:      relDate,
			BundleIdentifier: "com.apple.Music",
			UniqueIdentifier: 123456,
		},
	}

	assert.Equal(t, "Everything in Its Right Place", info.Title)
	assert.Equal(t, "Kid A", info.Album)
	assert.Equal(t, "Radiohead", info.Artist)
	assert.Equal(t, "Radiohead", info.AlbumArtist)
	assert.Equal(t, 1, info.TrackNumber)
	assert.Equal(t, 1, info.DiscNumber)
	assert.Equal(t, int64(251), info.Duration)
	assert.Equal(t, 12.5, info.Position)
	assert.Equal(t, 44100, info.SampleRate)
	assert.Equal(t, "Electronic / Experimental", info.Genre)
	assert.Equal(t, "Thom Yorke", info.Composer)
	assert.Equal(t, "com.apple.Music", info.BundleIdentifier)
	assert.Equal(t, int64(123456), info.UniqueIdentifier)
}
