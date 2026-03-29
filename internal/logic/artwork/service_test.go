package artwork

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/core/objectstorage"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

type fakeObjectStorage struct {
	exists map[string]bool
}

func (f *fakeObjectStorage) CheckObjectExists(
	ctx context.Context, objectKey string,
) (exists bool, contentType string, err error) {
	return f.exists[objectKey], "image/jpeg", nil
}

func (f *fakeObjectStorage) UploadFileToObject(ctx context.Context, objectKey, filePath, contentType string) error {
	return nil
}

func (f *fakeObjectStorage) UploadBytesToObject(ctx context.Context, objectKey string, payload []byte, contentType string) error {
	return nil
}

func (f *fakeObjectStorage) DeleteObject(ctx context.Context, objectKey string) error {
	return nil
}

func (f *fakeObjectStorage) DeleteObjects(ctx context.Context, objectKeys []string) error {
	return nil
}

func (f *fakeObjectStorage) GetObjectCDNURL(objectKey string) string {
	return "/bucket/" + objectKey
}

func (f *fakeObjectStorage) BuildOriginalObjectKey(seed string) string {
	return "orig/" + seed
}

func TestResolvePrefersAlbumCoverURL(t *testing.T) {
	origGetAlbum := modelGetAlbum
	origGetAlbumByArtistAndName := modelGetAlbumByArtistAndName
	origGetProvider := objectStorageGet
	origUpsert := modelUpsertAlbumCoverByID
	t.Cleanup(func() {
		modelGetAlbum = origGetAlbum
		modelGetAlbumByArtistAndName = origGetAlbumByArtistAndName
		objectStorageGet = origGetProvider
		modelUpsertAlbumCoverByID = origUpsert
	})

	modelGetAlbum = func(ctx context.Context, id int64) (*model.Album, error) {
		require.Equal(t, int64(42), id)
		return &model.Album{
			ID:                id,
			CoverArtURL:       "/bucket/orig/found",
			CoverArtObjectKey: "orig/found",
		}, nil
	}
	modelGetAlbumByArtistAndName = func(ctx context.Context, artist, albumName string) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	objectStorageGet = func() objectstorage.Provider {
		return &fakeObjectStorage{}
	}
	modelUpsertAlbumCoverByID = func(ctx context.Context, albumID int64, update model.AlbumCoverUpdate) error {
		return nil
	}

	svc := NewService()
	result, err := svc.Resolve(
		context.Background(),
		ResolveArtworkInput{
			AlbumID: 42,
			Album:   "The Album",
		},
	)
	require.NoError(t, err)
	require.True(t, result.Exists)
	require.Equal(t, "/bucket/orig/found", result.CoverArtURL)
	require.Equal(t, "orig/found", result.CoverArtObjectKey)
}

func TestResolveFallsBackToAlbumSeedThenArtworkKey(t *testing.T) {
	origGetAlbum := modelGetAlbum
	origGetAlbumByArtistAndName := modelGetAlbumByArtistAndName
	origGetProvider := objectStorageGet
	origUpsert := modelUpsertAlbumCoverByID
	t.Cleanup(func() {
		modelGetAlbum = origGetAlbum
		modelGetAlbumByArtistAndName = origGetAlbumByArtistAndName
		objectStorageGet = origGetProvider
		modelUpsertAlbumCoverByID = origUpsert
	})

	modelGetAlbum = func(ctx context.Context, id int64) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	modelGetAlbumByArtistAndName = func(ctx context.Context, artist, albumName string) (*model.Album, error) {
		if artist == "Album Artist" && albumName == "The Album" {
			return &model.Album{ID: 100}, nil
		}
		return nil, gorm.ErrRecordNotFound
	}
	fake := &fakeObjectStorage{
		exists: map[string]bool{
			"orig/album artist|the album": true,
		},
	}
	objectStorageGet = func() objectstorage.Provider {
		return fake
	}
	modelUpsertAlbumCoverByID = func(ctx context.Context, albumID int64, update model.AlbumCoverUpdate) error {
		return nil
	}

	svc := NewService()
	result, err := svc.Resolve(
		context.Background(),
		ResolveArtworkInput{
			AlbumArtist: "Album Artist",
			Artist:      "Track Artist",
			Album:       "The Album",
			ArtworkKey:  "legacy-key",
		},
	)
	require.NoError(t, err)
	require.True(t, result.Exists)
	require.Equal(t, "/bucket/orig/album artist|the album", result.CoverArtURL)
	require.Equal(t, "orig/album artist|the album", result.CoverArtObjectKey)
}

func TestResolveFallbackByArtworkKey(t *testing.T) {
	origGetAlbum := modelGetAlbum
	origGetAlbumByArtistAndName := modelGetAlbumByArtistAndName
	origGetProvider := objectStorageGet
	origUpsert := modelUpsertAlbumCoverByID
	t.Cleanup(func() {
		modelGetAlbum = origGetAlbum
		modelGetAlbumByArtistAndName = origGetAlbumByArtistAndName
		objectStorageGet = origGetProvider
		modelUpsertAlbumCoverByID = origUpsert
	})

	modelGetAlbum = func(ctx context.Context, id int64) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	modelGetAlbumByArtistAndName = func(ctx context.Context, artist, albumName string) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	fake := &fakeObjectStorage{
		exists: map[string]bool{
			"orig/legacy-key": true,
		},
	}
	objectStorageGet = func() objectstorage.Provider {
		return fake
	}
	modelUpsertAlbumCoverByID = func(ctx context.Context, albumID int64, update model.AlbumCoverUpdate) error {
		return nil
	}

	svc := NewService()
	result, err := svc.Resolve(
		context.Background(),
		ResolveArtworkInput{
			ArtworkKey: "legacy-key",
		},
	)
	require.NoError(t, err)
	require.True(t, result.Exists)
	require.Equal(t, "/bucket/orig/legacy-key", result.CoverArtURL)
	require.Equal(t, "orig/legacy-key", result.CoverArtObjectKey)
}

func TestResolveReturnsErrorWhenAlbumQueryFails(t *testing.T) {
	origGetAlbum := modelGetAlbum
	origGetAlbumByArtistAndName := modelGetAlbumByArtistAndName
	origGetProvider := objectStorageGet
	origUpsert := modelUpsertAlbumCoverByID
	t.Cleanup(func() {
		modelGetAlbum = origGetAlbum
		modelGetAlbumByArtistAndName = origGetAlbumByArtistAndName
		objectStorageGet = origGetProvider
		modelUpsertAlbumCoverByID = origUpsert
	})

	modelGetAlbum = func(ctx context.Context, id int64) (*model.Album, error) {
		return nil, errors.New("db down")
	}
	modelGetAlbumByArtistAndName = func(ctx context.Context, artist, albumName string) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	objectStorageGet = func() objectstorage.Provider {
		return &fakeObjectStorage{}
	}
	modelUpsertAlbumCoverByID = func(ctx context.Context, albumID int64, update model.AlbumCoverUpdate) error {
		return nil
	}

	svc := NewService()
	_, err := svc.Resolve(
		context.Background(),
		ResolveArtworkInput{
			AlbumID: 1,
			Album:   "The Album",
		},
	)
	require.Error(t, err)
}

func TestEnsureAlbumCoverUsesDirectPayload(t *testing.T) {
	origGetAlbum := modelGetAlbum
	origGetAlbumByArtistAndName := modelGetAlbumByArtistAndName
	origGetProvider := objectStorageGet
	origUpsert := modelUpsertAlbumCoverByID
	t.Cleanup(func() {
		modelGetAlbum = origGetAlbum
		modelGetAlbumByArtistAndName = origGetAlbumByArtistAndName
		objectStorageGet = origGetProvider
		modelUpsertAlbumCoverByID = origUpsert
	})

	modelGetAlbum = func(ctx context.Context, id int64) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	modelGetAlbumByArtistAndName = func(ctx context.Context, artist, albumName string) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	objectStorageGet = func() objectstorage.Provider {
		return &fakeObjectStorage{}
	}

	called := false
	modelUpsertAlbumCoverByID = func(ctx context.Context, albumID int64, update model.AlbumCoverUpdate) error {
		called = true
		require.Equal(t, int64(99), albumID)
		require.Equal(t, "/bucket/orig/direct", update.CoverArtURL)
		require.Equal(t, "image/webp", update.CoverArtMime)
		require.Equal(t, "orig/direct", update.CoverArtObjectKey)
		return nil
	}

	svc := NewService()
	err := svc.EnsureAlbumCover(
		context.Background(),
		EnsureAlbumCoverInput{
			AlbumID:           99,
			CoverArtURL:       "/bucket/orig/direct",
			CoverArtMime:      "image/webp",
			CoverArtObjectKey: "orig/direct",
		},
	)
	require.NoError(t, err)
	require.True(t, called)
}

func TestEnsureAlbumCoverResolvesFromObjectStorage(t *testing.T) {
	origGetAlbum := modelGetAlbum
	origGetAlbumByArtistAndName := modelGetAlbumByArtistAndName
	origGetProvider := objectStorageGet
	origUpsert := modelUpsertAlbumCoverByID
	t.Cleanup(func() {
		modelGetAlbum = origGetAlbum
		modelGetAlbumByArtistAndName = origGetAlbumByArtistAndName
		objectStorageGet = origGetProvider
		modelUpsertAlbumCoverByID = origUpsert
	})

	modelGetAlbum = func(ctx context.Context, id int64) (*model.Album, error) {
		return nil, gorm.ErrRecordNotFound
	}
	modelGetAlbumByArtistAndName = func(ctx context.Context, artist, albumName string) (*model.Album, error) {
		require.Equal(t, "Album Artist", artist)
		require.Equal(t, "The Album", albumName)
		return &model.Album{ID: 100}, nil
	}
	objectStorageGet = func() objectstorage.Provider {
		return &fakeObjectStorage{
			exists: map[string]bool{
				"orig/album artist|the album": true,
			},
		}
	}

	called := false
	modelUpsertAlbumCoverByID = func(ctx context.Context, albumID int64, update model.AlbumCoverUpdate) error {
		called = true
		require.Equal(t, int64(100), albumID)
		require.Equal(t, "/bucket/orig/album artist|the album", update.CoverArtURL)
		require.Equal(t, "orig/album artist|the album", update.CoverArtObjectKey)
		return nil
	}

	svc := NewService()
	err := svc.EnsureAlbumCover(
		context.Background(),
		EnsureAlbumCoverInput{
			AlbumArtist: "Album Artist",
			Artist:      "Track Artist",
			Album:       "The Album",
		},
	)
	require.NoError(t, err)
	require.True(t, called)
}
