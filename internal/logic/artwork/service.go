package artwork

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	artworkcore "github.com/vincentchyu/sonic-lens/core/artwork"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/objectstorage"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var (
	modelGetAlbum                = model.GetAlbum
	modelGetAlbumByArtistAndName = model.GetAlbumByArtistAndName
	modelUpsertAlbumCoverByID    = model.UpsertAlbumCoverByID
	objectStorageGet             = objectstorage.Get
)

// ResolveArtworkInput 描述封面解析请求参数。
type ResolveArtworkInput struct {
	AlbumID     int64
	AlbumArtist string
	Artist      string
	Album       string
	ArtworkKey  string
}

// ResolveArtworkResult 描述封面解析输出。
type ResolveArtworkResult struct {
	Exists            bool   `json:"exists"`
	CoverArtURL       string `json:"cover_art_url"`
	CoverArtObjectKey string `json:"cover_art_object_key"`
}

// EnsureAlbumCoverInput 描述专辑封面补齐参数。
type EnsureAlbumCoverInput struct {
	AlbumID           int64
	AlbumArtist       string
	Artist            string
	Album             string
	ArtworkKey        string
	CoverArtURL       string
	CoverArtMime      string
	CoverArtObjectKey string
}

// Service 定义封面解析业务能力。
type Service interface {
	Resolve(ctx context.Context, input ResolveArtworkInput) (ResolveArtworkResult, error)
	EnsureAlbumCover(ctx context.Context, input EnsureAlbumCoverInput) error
}

type service struct{}

// NewService 创建封面解析服务。
func NewService() Service {
	return &service{}
}

// EnsureAlbumCover 统一补齐专辑封面；优先写入调用方已拿到的封面，其次回退对象存储解析。
func EnsureAlbumCover(ctx context.Context, input EnsureAlbumCoverInput) error {
	return NewService().EnsureAlbumCover(ctx, input)
}

// Resolve 按专辑身份优先 + artwork key 兜底顺序解析封面对象。
func (s *service) Resolve(ctx context.Context, input ResolveArtworkInput) (ResolveArtworkResult, error) {
	return s.resolve(ctx, input, true)
}

func (s *service) resolve(
	ctx context.Context,
	input ResolveArtworkInput,
	enableAsyncBackfill bool,
) (ResolveArtworkResult, error) {
	result := ResolveArtworkResult{}
	normalized := normalizeResolveInput(input)
	log.Debug(
		ctx,
		"开始解析专辑封面",
		zap.Int64("album_id", normalized.AlbumID),
		zap.String("album_artist", normalized.AlbumArtist),
		zap.String("artist", normalized.Artist),
		zap.String("album", normalized.Album),
		zap.String("artwork_key", normalized.ArtworkKey),
	)
	if normalized.AlbumID <= 0 && normalized.Album == "" && normalized.ArtworkKey == "" {
		return result, nil
	}

	albumRecord, err := findCandidateAlbum(ctx, normalized)
	if err != nil {
		return result, err
	}

	if albumRecord != nil && strings.TrimSpace(albumRecord.CoverArtURL) != "" {
		log.Info(
			ctx,
			"从专辑记录命中封面",
			zap.Int64("album_id", albumRecord.ID),
			zap.String("cover_art_object_key", strings.TrimSpace(albumRecord.CoverArtObjectKey)),
		)
		return ResolveArtworkResult{
			Exists:            true,
			CoverArtURL:       strings.TrimSpace(albumRecord.CoverArtURL),
			CoverArtObjectKey: strings.TrimSpace(albumRecord.CoverArtObjectKey),
		}, nil
	}

	obj := objectStorageGet()
	if obj == nil {
		return result, nil
	}

	if candidate, ok, checkErr := resolveByAlbumSeed(ctx, obj, normalized); checkErr != nil {
		return result, checkErr
	} else if ok {
		log.Info(
			ctx,
			"从对象存储命中封面",
			zap.Int64("album_id", normalized.AlbumID),
			zap.String("cover_art_object_key", candidate.CoverArtObjectKey),
		)
		if enableAsyncBackfill {
			asyncBackfillAlbumCover(ctx, normalized.AlbumID, candidate)
		}
		return candidate, nil
	}

	if normalized.ArtworkKey == "" {
		return result, nil
	}
	if candidate, ok, checkErr := resolveByArtworkKey(ctx, obj, normalized.ArtworkKey); checkErr != nil {
		return result, checkErr
	} else if ok {
		log.Info(
			ctx,
			"从 artwork key 命中封面",
			zap.Int64("album_id", normalized.AlbumID),
			zap.String("cover_art_object_key", candidate.CoverArtObjectKey),
		)
		if enableAsyncBackfill {
			asyncBackfillAlbumCover(ctx, normalized.AlbumID, candidate)
		}
		return candidate, nil
	}

	return result, nil
}

// EnsureAlbumCover 统一补齐专辑封面；优先写入调用方已拿到的封面，其次回退对象存储解析。
func (s *service) EnsureAlbumCover(ctx context.Context, input EnsureAlbumCoverInput) error {
	normalized := normalizeEnsureInput(input)
	albumID, err := resolveAlbumIDForEnsure(ctx, normalized)
	if err != nil || albumID <= 0 {
		if err != nil {
			log.Warn(
				ctx,
				"解析专辑封面目标专辑失败",
				zap.Int64("album_id", normalized.AlbumID),
				zap.String("album_artist", normalized.AlbumArtist),
				zap.String("album", normalized.Album),
				zap.Error(err),
			)
		}
		return err
	}

	directUpdate := model.AlbumCoverUpdate{
		CoverArtURL:       normalized.CoverArtURL,
		CoverArtMime:      normalized.CoverArtMime,
		CoverArtObjectKey: normalized.CoverArtObjectKey,
	}
	if strings.TrimSpace(directUpdate.CoverArtURL) != "" || strings.TrimSpace(directUpdate.CoverArtObjectKey) != "" {
		log.Info(
			ctx,
			"直接写入专辑封面信息",
			zap.Int64("album_id", albumID),
			zap.String("cover_art_object_key", directUpdate.CoverArtObjectKey),
		)
		return modelUpsertAlbumCoverByID(ctx, albumID, directUpdate)
	}

	resolved, err := s.ensureResolveWithoutBackfill(ctx, albumID, normalized)
	if err != nil || !resolved.Exists || strings.TrimSpace(resolved.CoverArtObjectKey) == "" {
		if err != nil {
			log.Warn(
				ctx,
				"回源解析专辑封面失败",
				zap.Int64("album_id", albumID),
				zap.Error(err),
			)
		}
		return err
	}
	log.Info(
		ctx,
		"回源解析到专辑封面",
		zap.Int64("album_id", albumID),
		zap.String("cover_art_object_key", resolved.CoverArtObjectKey),
	)
	return modelUpsertAlbumCoverByID(
		ctx,
		albumID,
		model.AlbumCoverUpdate{
			CoverArtURL: resolved.CoverArtURL,
			// CoverArtMime:      normalized.CoverArtMime,
			CoverArtObjectKey: resolved.CoverArtObjectKey,
		},
	)
}

func (s *service) ensureResolveWithoutBackfill(
	ctx context.Context,
	albumID int64,
	input EnsureAlbumCoverInput,
) (ResolveArtworkResult, error) {
	return s.resolve(
		ctx,
		ResolveArtworkInput{
			AlbumID:     albumID,
			AlbumArtist: input.AlbumArtist,
			Artist:      input.Artist,
			Album:       input.Album,
			ArtworkKey:  input.ArtworkKey,
		},
		false,
	)
}

func normalizeResolveInput(input ResolveArtworkInput) ResolveArtworkInput {
	input.AlbumArtist = strings.TrimSpace(input.AlbumArtist)
	input.Artist = strings.TrimSpace(input.Artist)
	input.Album = strings.TrimSpace(input.Album)
	input.ArtworkKey = strings.TrimSpace(input.ArtworkKey)
	return input
}

func normalizeEnsureInput(input EnsureAlbumCoverInput) EnsureAlbumCoverInput {
	input.AlbumArtist = strings.TrimSpace(input.AlbumArtist)
	input.Artist = strings.TrimSpace(input.Artist)
	input.Album = strings.TrimSpace(input.Album)
	input.ArtworkKey = strings.TrimSpace(input.ArtworkKey)
	input.CoverArtURL = strings.TrimSpace(input.CoverArtURL)
	input.CoverArtMime = strings.TrimSpace(input.CoverArtMime)
	input.CoverArtObjectKey = strings.TrimSpace(input.CoverArtObjectKey)

	if len(input.CoverArtMime) == 0 {
		input.CoverArtMime = "image/jpeg" // 处理为空的情况
	}
	return input
}

func resolveAlbumIDForEnsure(ctx context.Context, input EnsureAlbumCoverInput) (int64, error) {
	if input.AlbumID > 0 {
		return input.AlbumID, nil
	}
	albumRecord, err := findCandidateAlbum(
		ctx,
		ResolveArtworkInput{
			AlbumArtist: input.AlbumArtist,
			Artist:      input.Artist,
			Album:       input.Album,
		},
	)
	if err != nil || albumRecord == nil {
		return 0, err
	}
	return albumRecord.ID, nil
}

func findCandidateAlbum(ctx context.Context, input ResolveArtworkInput) (*model.Album, error) {
	if input.AlbumID > 0 {
		albumRecord, err := modelGetAlbum(ctx, input.AlbumID)
		if err == nil {
			return albumRecord, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if input.Album == "" {
		return nil, nil
	}
	candidates := buildArtistCandidates(input.AlbumArtist, input.Artist)
	for _, artist := range candidates {
		albumRecord, err := modelGetAlbumByArtistAndName(ctx, artist, input.Album)
		if err == nil {
			return albumRecord, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

func buildArtistCandidates(albumArtist, artist string) []string {
	values := []string{strings.TrimSpace(albumArtist), strings.TrimSpace(artist)}
	candidates := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}
	return candidates
}

func resolveByAlbumSeed(
	ctx context.Context, obj objectstorage.Provider, input ResolveArtworkInput,
) (ResolveArtworkResult, bool, error) {
	if input.Album == "" {
		return ResolveArtworkResult{}, false, nil
	}
	seed := artworkcore.BuildAlbumArtworkSeed(input.AlbumArtist, input.Artist, input.Album)
	return resolveBySeed(ctx, obj, seed)
}

func resolveByArtworkKey(
	ctx context.Context, obj objectstorage.Provider, artworkKey string,
) (ResolveArtworkResult, bool, error) {
	if artworkKey == "" {
		return ResolveArtworkResult{}, false, nil
	}
	return resolveBySeed(ctx, obj, artworkKey)
}

func resolveBySeed(
	ctx context.Context, obj objectstorage.Provider, seed string,
) (ResolveArtworkResult, bool, error) {
	objectKey := obj.BuildOriginalObjectKey(seed)
	if objectKey == "" {
		return ResolveArtworkResult{}, false, nil
	}

	exists, _, err := obj.CheckObjectExists(ctx, objectKey)
	if err != nil {
		log.Warn(
			ctx,
			"ResolveArtwork check object err",
			zap.Error(err),
			zap.String("cover_art_object_key", objectKey),
		)
		return ResolveArtworkResult{}, false, err
	}
	if !exists {
		return ResolveArtworkResult{}, false, nil
	}

	return ResolveArtworkResult{
		Exists:            true,
		CoverArtURL:       obj.GetObjectCDNURL(objectKey),
		CoverArtObjectKey: objectKey,
	}, true, nil
}

func asyncBackfillAlbumCover(ctx context.Context, albumID int64, resolved ResolveArtworkResult) {
	if albumID <= 0 || !resolved.Exists || resolved.CoverArtURL == "" || resolved.CoverArtObjectKey == "" {
		return
	}

	telemetry.GoSafeDetached(
		ctx, "artwork.async_backfill_album_cover", func(asyncCtx context.Context) {
			dbCtx, cancel := context.WithTimeout(asyncCtx, 5*time.Second)
			defer cancel()

			if err := modelUpsertAlbumCoverByID(
				dbCtx,
				albumID,
				model.AlbumCoverUpdate{
					CoverArtURL:       resolved.CoverArtURL,
					CoverArtObjectKey: resolved.CoverArtObjectKey,
				},
			); err != nil {
				log.Warn(
					dbCtx,
					"ResolveArtwork async backfill album cover err",
					zap.Int64("album_id", albumID),
					zap.Error(err),
				)
			}
		},
	)
}
