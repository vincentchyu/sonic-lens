package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/ai"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

type fakeAIRouteService struct {
	platforms        []ai.PlatformOption
	modelsByPlatform map[common.AIModelPlatform][]ai.ModelOption

	trackErr  error
	albumErr  error
	streamErr error

	lastTrackInput struct {
		provider  string
		model     string
		legacy    string
		artist    string
		track     string
		trackNo   int8
		discNo    int8
		forceFlag bool
	}
	lastAlbumInput struct {
		albumID   int64
		provider  string
		model     string
		legacy    string
		forceFlag bool
	}
	lastStreamInput struct {
		provider  string
		model     string
		legacy    string
		artist    string
		track     string
		trackNo   int8
		discNo    int8
		forceFlag bool
	}
}

func (f *fakeAIRouteService) GetAvailableAIPlatforms() []ai.PlatformOption {
	return f.platforms
}

func (f *fakeAIRouteService) GetPlatformModels(
	_ context.Context, platform common.AIModelPlatform,
) ([]ai.ModelOption, error) {
	return f.modelsByPlatform[platform], nil
}

func (f *fakeAIRouteService) GetOrCreateInsight(
	_ context.Context, artist, album, track string, trackNumber, discNumber int8, force bool, provider, modelName, legacyModelType string,
) ([]*model.TrackInsight, bool, error) {
	f.lastTrackInput = struct {
		provider  string
		model     string
		legacy    string
		artist    string
		track     string
		trackNo   int8
		discNo    int8
		forceFlag bool
	}{
		provider:  provider,
		model:     modelName,
		legacy:    legacyModelType,
		artist:    artist,
		track:     track,
		trackNo:   trackNumber,
		discNo:    discNumber,
		forceFlag: force,
	}
	if f.trackErr != nil {
		return nil, false, f.trackErr
	}
	return []*model.TrackInsight{}, false, nil
}

func (f *fakeAIRouteService) GetOrCreateAlbumInsight(
	_ context.Context, albumID int64, force bool, provider, modelName, legacyModelType string,
) ([]*model.AlbumInsight, bool, error) {
	f.lastAlbumInput = struct {
		albumID   int64
		provider  string
		model     string
		legacy    string
		forceFlag bool
	}{
		albumID:   albumID,
		provider:  provider,
		model:     modelName,
		legacy:    legacyModelType,
		forceFlag: force,
	}
	if f.albumErr != nil {
		return nil, false, f.albumErr
	}
	return []*model.AlbumInsight{}, false, nil
}

func (f *fakeAIRouteService) GetOrCreateInsightStream(
	_ context.Context, artist, album, track string, trackNumber, discNumber int8, force bool, provider, modelName, legacyModelType string,
) (<-chan string, bool, error) {
	f.lastStreamInput = struct {
		provider  string
		model     string
		legacy    string
		artist    string
		track     string
		trackNo   int8
		discNo    int8
		forceFlag bool
	}{
		provider:  provider,
		model:     modelName,
		legacy:    legacyModelType,
		artist:    artist,
		track:     track,
		trackNo:   trackNumber,
		discNo:    discNumber,
		forceFlag: force,
	}
	if f.streamErr != nil {
		return nil, false, f.streamErr
	}
	ch := make(chan string, 1)
	ch <- "ok"
	close(ch)
	return ch, false, nil
}

func newAITestRouter(service aiRouteService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerAIRoutes(router, service)
	return router
}

func TestRegisterAIRoutesListPlatforms(t *testing.T) {
	service := &fakeAIRouteService{
		platforms: []ai.PlatformOption{
			{ID: common.AIModelPlatformOpenAI, DisplayName: "OpenAI"},
			{ID: common.AIModelPlatformGemini, DisplayName: "Gemini"},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/ai-models", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var resp struct {
		Platforms []ai.PlatformOption `json:"platforms"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Platforms) != 2 {
		t.Fatalf("unexpected platform count: %d", len(resp.Platforms))
	}
	if resp.Platforms[0].ID != common.AIModelPlatformOpenAI {
		t.Fatalf("unexpected first platform: %s", resp.Platforms[0].ID)
	}
}

func TestRegisterAIRoutesListModelsByPlatform(t *testing.T) {
	service := &fakeAIRouteService{
		modelsByPlatform: map[common.AIModelPlatform][]ai.ModelOption{
			common.AIModelPlatformOpenAI: {
				{ID: "gpt-5", DisplayName: "GPT-5", IsDefault: true},
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/ai-models/openai/models", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var resp struct {
		Models []ai.ModelOption `json:"models"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Models) != 1 || resp.Models[0].ID != "gpt-5" {
		t.Fatalf("unexpected models response: %+v", resp.Models)
	}
}

func TestRegisterAIRoutesRejectInvalidPlatform(t *testing.T) {
	router := newAITestRouter(&fakeAIRouteService{})

	req := httptest.NewRequest(http.MethodGet, "/api/ai-models/not-real/models", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestRegisterAIRoutesTrackInsightPassesProviderAndModel(t *testing.T) {
	service := &fakeAIRouteService{}
	router := newAITestRouter(service)

	body := `{"artist":"A","album":"B","track":"C","track_number":1,"disc_number":2,"provider":"openai","model":"gpt-5"}`
	req := httptest.NewRequest(http.MethodPost, "/api/track-insight", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.lastTrackInput.provider != "openai" || service.lastTrackInput.model != "gpt-5" {
		t.Fatalf("provider/model not passed through: %+v", service.lastTrackInput)
	}
	if !service.lastTrackInput.forceFlag {
		t.Fatalf("expected force flag to be true")
	}
}

func TestRegisterAIRoutesTrackInsightBadRequestOnUnsupportedModel(t *testing.T) {
	service := &fakeAIRouteService{trackErr: context.Canceled}
	service.trackErr = assertAISelectionError("不支持模型: bad-model")
	router := newAITestRouter(service)

	body := `{"artist":"A","track":"C","provider":"openai","model":"bad-model"}`
	req := httptest.NewRequest(http.MethodPost, "/api/track-insight", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterAIRoutesAlbumInsightSupportsLegacyModelType(t *testing.T) {
	service := &fakeAIRouteService{}
	router := newAITestRouter(service)

	body := `{"album_id":42,"modelType":"gemini"}`
	req := httptest.NewRequest(http.MethodPost, "/api/album-insight", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.lastAlbumInput.albumID != 42 || service.lastAlbumInput.legacy != "gemini" {
		t.Fatalf("legacy modelType not passed through: %+v", service.lastAlbumInput)
	}
}

func TestRegisterAIRoutesStreamSupportsProviderModelAndBadRequest(t *testing.T) {
	service := &fakeAIRouteService{streamErr: assertAISelectionError("不支持模型: bad-model")}
	router := newAITestRouter(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/track-insight-stream?artist=A&track=C&provider=openai&model=bad-model&force=true",
		nil,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.lastStreamInput.provider != "openai" || service.lastStreamInput.model != "bad-model" {
		t.Fatalf("stream params not passed through: %+v", service.lastStreamInput)
	}
	if !service.lastStreamInput.forceFlag {
		t.Fatalf("expected force flag to be true")
	}
}

func assertAISelectionError(message string) error {
	return &aiSelectionError{message: message}
}

type aiSelectionError struct {
	message string
}

func (e *aiSelectionError) Error() string {
	return e.message
}
