package artistprofile

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/vincentchyu/sonic-lens/core/objectstorage"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

var objectStorageGet = objectstorage.Get

// Service 定义艺术家资料服务接口。
type Service interface {
	ListProfiles(ctx context.Context, limit, offset int, keyword string) (ListResult, error)
	ListTopArtistSources(ctx context.Context, limit int) ([]string, error)
	UploadAvatar(ctx context.Context, artistName, imageData string) (*model.ArtistProfile, error)
	MergeProfilesIntoTopArtists(ctx context.Context, items []map[string]any) ([]map[string]any, error)
}

type service struct{}

// ListItem 描述艺术家资料列表项。
type ListItem struct {
	ID                  int64  `json:"id"`
	ArtistName          string `json:"artist_name"`
	NormalizedArtistKey string `json:"normalized_artist_key"`
	AvatarURL           string `json:"avatar_url"`
	AvatarMime          string `json:"avatar_mime"`
	AvatarObjectKey     string `json:"avatar_object_key"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

// ListResult 描述艺术家资料分页结果。
type ListResult struct {
	Items []ListItem `json:"items"`
	Total int64      `json:"total"`
}

// NewService 创建艺术家资料服务。
func NewService() Service {
	return &service{}
}

func (s *service) ListProfiles(ctx context.Context, limit, offset int, keyword string) (ListResult, error) {
	rows, total, err := model.ListArtistProfiles(ctx, limit, offset, keyword)
	if err != nil {
		return ListResult{}, err
	}
	items := make([]ListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildListItem(row))
	}
	return ListResult{Items: items, Total: total}, nil
}

func (s *service) ListTopArtistSources(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	candidates := make([]string, 0, limit)
	seen := map[string]struct{}{}
	appendNames := func(items []map[string]any) {
		for _, item := range items {
			artistName := strings.TrimSpace(topArtistResponseArtistName(item["artist"]))
			key := model.NormalizeArtistProfileKey(artistName)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, artistName)
			if len(candidates) >= limit {
				return
			}
		}
	}

	plays, err := model.GetTopArtistsByPlayCount(ctx, limit)
	if err != nil && !isLegacyDashboardSchemaError(err) {
		return nil, err
	}
	appendNames(plays)
	if len(candidates) < limit {
		tracks, trackErr := model.GetTopArtistsByTrackCount(ctx, limit)
		if trackErr != nil && !isLegacyDashboardSchemaError(trackErr) {
			return nil, trackErr
		}
		appendNames(tracks)
	}
	slices.Sort(candidates)
	return candidates, nil
}

func (s *service) UploadAvatar(ctx context.Context, artistName, imageData string) (*model.ArtistProfile, error) {
	artistName = strings.TrimSpace(artistName)
	if artistName == "" {
		return nil, fmt.Errorf("artist name is required")
	}
	provider := objectStorageGet()
	if provider == nil {
		return nil, fmt.Errorf("object storage is not enabled")
	}
	payload, mimeType, err := decodeImageData(imageData)
	if err != nil {
		return nil, err
	}
	objectKey := buildAvatarObjectKey(artistName, payload)
	if objectKey == "" {
		return nil, fmt.Errorf("build artist avatar object key failed")
	}
	if err := provider.UploadBytesToObject(ctx, objectKey, payload, mimeType); err != nil {
		return nil, err
	}
	profile := &model.ArtistProfile{
		ArtistName:          artistName,
		NormalizedArtistKey: model.NormalizeArtistProfileKey(artistName),
		AvatarMime:          mimeType,
		AvatarObjectKey:     objectKey,
		AvatarURL:           strings.TrimSpace(provider.GetObjectCDNURL(objectKey)),
	}
	if err := model.UpsertArtistProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *service) MergeProfilesIntoTopArtists(
	ctx context.Context, items []map[string]any,
) ([]map[string]any, error) {
	if len(items) == 0 {
		return items, nil
	}
	profiles, err := model.GetArtistProfilesByNames(ctx, collectTopArtistNames(items))
	if err != nil {
		if isLegacyDashboardSchemaError(err) {
			return items, nil
		}
		return nil, err
	}
	for _, item := range items {
		item["avatar_url"] = ""
		item["avatar_mime"] = ""
		item["avatar_object_key"] = ""
		artistName := topArtistResponseArtistName(item["artist"])
		profile, ok := profiles[model.NormalizeArtistProfileKey(artistName)]
		if !ok {
			continue
		}
		avatarURL := strings.TrimSpace(profile.AvatarURL)
		if avatarURL == "" && strings.TrimSpace(profile.AvatarObjectKey) != "" {
			if provider := objectStorageGet(); provider != nil {
				avatarURL = provider.GetObjectCDNURL(profile.AvatarObjectKey)
			}
		}
		item["avatar_url"] = avatarURL
		item["avatar_mime"] = profile.AvatarMime
		item["avatar_object_key"] = profile.AvatarObjectKey
	}
	return items, nil
}

func buildListItem(profile model.ArtistProfile) ListItem {
	avatarURL := strings.TrimSpace(profile.AvatarURL)
	if avatarURL == "" && strings.TrimSpace(profile.AvatarObjectKey) != "" {
		if provider := objectStorageGet(); provider != nil {
			avatarURL = strings.TrimSpace(provider.GetObjectCDNURL(profile.AvatarObjectKey))
		}
	}
	// /album/artist/v1/originals/e03ba558251e6b77ec0781d62bf18e6c9e17d134
	// artist/v1/originals/e03ba558251e6b77ec0781d62bf18e6c9e17d134
	return ListItem{
		ID:                  profile.ID,
		ArtistName:          profile.ArtistName,
		NormalizedArtistKey: profile.NormalizedArtistKey,
		AvatarURL:           fmt.Sprintf("http://localhost:9000%s", avatarURL),
		AvatarMime:          profile.AvatarMime,
		AvatarObjectKey:     profile.AvatarObjectKey,
		CreatedAt:           profile.CreatedAt.Format(timeLayout),
		UpdatedAt:           profile.UpdatedAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02 15:04:05"

func collectTopArtistNames(items []map[string]any) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		artistName := topArtistResponseArtistName(item["artist"])
		key := model.NormalizeArtistProfileKey(artistName)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, artistName)
	}
	return result
}

func topArtistResponseArtistName(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func decodeImageData(raw string) ([]byte, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", fmt.Errorf("image data is required")
	}
	payload := raw
	mimeType := ""
	if strings.HasPrefix(raw, "data:") {
		commaIdx := strings.Index(raw, ",")
		if commaIdx <= 5 || commaIdx >= len(raw)-1 {
			return nil, "", fmt.Errorf("invalid image data url")
		}
		header := raw[5:commaIdx]
		payload = raw[commaIdx+1:]
		parts := strings.Split(header, ";")
		mimeType = strings.TrimSpace(parts[0])
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", fmt.Errorf("decode image data failed: %w", err)
		}
	}
	if len(decoded) == 0 {
		return nil, "", fmt.Errorf("image data is empty")
	}
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = strings.TrimSpace(http.DetectContentType(decoded))
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("unsupported image mime type: %s", mimeType)
	}
	return decoded, mimeType, nil
}

func buildAvatarObjectKey(artistName string, payload []byte) string {
	normalizedKey := model.NormalizeArtistProfileKey(artistName)
	if normalizedKey == "" || len(payload) == 0 {
		return ""
	}
	sum := sha1.Sum(append([]byte(normalizedKey+":"), payload...))
	return objectstorage.JoinObjectKey("artist", "v1", "originals", hex.EncodeToString(sum[:]))
}

func isLegacyDashboardSchemaError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "artist_profile") && (strings.Contains(msg, "no such table") || strings.Contains(
		msg, "doesn't exist",
	) || strings.Contains(msg, "unknown column"))
}
