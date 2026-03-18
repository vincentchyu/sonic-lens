package musicbrainz

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uploadedlobster.com/mbtypes"
	"go.uploadedlobster.com/musicbrainzws2"

	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestNormalizeMBTrackLookupKey(t *testing.T) {
	t.Run("繁简和括号归一", func(t *testing.T) {
		require.Equal(
			t,
			"我讲给你一个笑话(南京场)",
			normalizeMBTrackLookupKey("我講給你一個笑話（南京場）"),
		)
	})

	t.Run("英文大小写归一", func(t *testing.T) {
		require.Equal(t, "alive in china 2017-2023", normalizeMBTrackLookupKey("aLIVE IN CHINA 2017-2023"))
	})
}

func TestFindMBTrackForHeardTrackPrefersTrackAlbumPosition(t *testing.T) {
	discOneTrack := mbTrackInfo{
		Track: musicbrainzws2.Track{
			Title:    "我講給你一個笑話",
			Position: 3,
			Recording: musicbrainzws2.Recording{
				ID:    mbtypes.MBID("75abcdbc-d0c5-4907-9209-18f24cc80228"),
				Title: "我講給你一個笑話",
			},
		},
		DiscNumber: 1,
	}
	discTwoTrack := mbTrackInfo{
		Track: musicbrainzws2.Track{
			Title:    "我講給你一個笑話（南京場）",
			Position: 3,
			Recording: musicbrainzws2.Recording{
				ID:    mbtypes.MBID("64bb0ec1-1a74-440b-9a87-74329a2a16a4"),
				Title: "我講給你一個笑話（南京場）",
			},
		},
		DiscNumber: 2,
	}

	mbTrackMapByPos := map[string]mbTrackInfo{
		buildMBTrackPositionKey(1, 3): discOneTrack,
		buildMBTrackPositionKey(2, 3): discTwoTrack,
	}
	mbTrackMapByName := map[string][]mbTrackInfo{
		normalizeMBTrackLookupKey(discOneTrack.Title): {discOneTrack},
		normalizeMBTrackLookupKey(discTwoTrack.Title): {discTwoTrack},
	}

	ta := &model.TrackAlbum{
		AlbumID:     4129,
		TrackID:     5050,
		Track:       "我讲给你一个笑话",
		TrackNumber: 3,
		DiscNumber:  2,
	}
	trackObj := &model.Track{
		ID:          5050,
		Track:       "我讲给你一个笑话",
		TrackNumber: 3,
		DiscNumber:  1,
	}
	processedRecordingIDs := map[string]bool{
		string(discOneTrack.Recording.ID): true,
	}

	matched, found, source := findMBTrackForHeardTrack(
		ta,
		trackObj,
		mbTrackMapByPos,
		mbTrackMapByName,
		processedRecordingIDs,
	)

	require.True(t, found)
	require.Equal(t, "track_album_position", source)
	require.Equal(t, string(discTwoTrack.Recording.ID), string(matched.Recording.ID))
	require.Equal(t, int8(2), matched.DiscNumber)
	require.Equal(t, 3, matched.Position)
}

func TestFindMBTrackForHeardTrackTitleCanRepairWrongPosition(t *testing.T) {
	discOneTrack := mbTrackInfo{
		Track: musicbrainzws2.Track{
			Title:    "我講給你一個笑話",
			Position: 3,
			Recording: musicbrainzws2.Recording{
				ID:    mbtypes.MBID("75abcdbc-d0c5-4907-9209-18f24cc80228"),
				Title: "我講給你一個笑話",
			},
		},
		DiscNumber: 1,
	}
	discTwoTrack := mbTrackInfo{
		Track: musicbrainzws2.Track{
			Title:    "我講給你一個笑話（南京場）",
			Position: 3,
			Recording: musicbrainzws2.Recording{
				ID:    mbtypes.MBID("64bb0ec1-1a74-440b-9a87-74329a2a16a4"),
				Title: "我講給你一個笑話（南京場）",
			},
		},
		DiscNumber: 2,
	}

	mbTrackMapByPos := map[string]mbTrackInfo{
		buildMBTrackPositionKey(1, 3): discOneTrack,
		buildMBTrackPositionKey(2, 3): discTwoTrack,
	}
	mbTrackMapByName := map[string][]mbTrackInfo{
		normalizeMBTrackLookupKey(discOneTrack.Title): {discOneTrack},
		normalizeMBTrackLookupKey(discTwoTrack.Title): {discTwoTrack},
	}

	ta := &model.TrackAlbum{
		AlbumID:     4129,
		TrackID:     5050,
		Track:       "我講給你一個笑話",
		TrackNumber: 3,
		DiscNumber:  1,
	}
	trackObj := &model.Track{
		ID:          5050,
		Track:       "我讲给你一个笑话(南京场)",
		TrackNumber: 3,
		DiscNumber:  1,
	}

	matched, found, source := findMBTrackForHeardTrack(
		ta,
		trackObj,
		mbTrackMapByPos,
		mbTrackMapByName,
		map[string]bool{},
	)

	require.True(t, found)
	require.Equal(t, "track_title_override_position", source)
	require.Equal(t, string(discTwoTrack.Recording.ID), string(matched.Recording.ID))
	require.Equal(t, int8(2), matched.DiscNumber)
	require.Equal(t, 3, matched.Position)
}
