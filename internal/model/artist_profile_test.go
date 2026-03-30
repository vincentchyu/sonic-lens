package model

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetArtistProfilesByNamesReturnsProfilesByNormalizedKey(t *testing.T) {
	t.Parallel()

	_, mock := newModelTestDB(t)
	mock.ExpectQuery(`SELECT \* FROM ` + "`artist_profile`" + ` WHERE normalized_artist_key IN \(\?\)`).
		WithArgs("pink floyd").
		WillReturnRows(
			sqlmock.NewRows(
				[]string{"id", "artist_name", "normalized_artist_key", "avatar_url", "avatar_object_key", "avatar_mime", "created_at", "updated_at"},
			).AddRow(
				1,
				"Pink Floyd",
				"pink floyd",
				"/artist/pink-floyd.jpg",
				"artist/v1/originals/abc",
				"image/jpeg",
				modelTestNow,
				modelTestNow,
			),
		)

	items, err := GetArtistProfilesByNames(context.Background(), []string{"Pink Floyd"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "Pink Floyd", items["pink floyd"].ArtistName)
	assert.Equal(t, "artist/v1/originals/abc", items["pink floyd"].AvatarObjectKey)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNormalizeArtistProfileKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "pink floyd", NormalizeArtistProfileKey("  Pink Floyd  "))
	assert.Equal(t, "", NormalizeArtistProfileKey("   "))
}
