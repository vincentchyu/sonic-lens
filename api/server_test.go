package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/ai"
	"github.com/vincentchyu/sonic-lens/internal/logic/insight"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

type fakeAIRouteService struct {
	platforms        []ai.PlatformOption
	modelsByPlatform map[common.AIModelPlatform][]ai.ModelOption

	trackErr        error
	albumErr        error
	streamErr       error
	jobErr          error
	jobByID         map[string]*model.InsightJob
	callLogsByJobID map[string][]*model.LLMCallLog
	lastListQuery   model.InsightJobListQuery

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

func (f *fakeAIRouteService) GetOrCreateTrackInsight(
	_ context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
	provider, modelName, legacyModelType string,
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
	_ context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
	provider, modelName, legacyModelType string,
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

func (f *fakeAIRouteService) CreateInsightJob(
	_ context.Context, req insight.CreateInsightJobRequest,
) (*model.InsightJob, bool, error) {
	if f.jobErr != nil {
		return nil, false, f.jobErr
	}
	if f.jobByID == nil {
		f.jobByID = map[string]*model.InsightJob{}
	}
	job := &model.InsightJob{
		ID:                 "job-1",
		AnalysisTargetType: req.TargetType,
		Status:             common.InsightJobPhaseQueued,
		AlbumID:            req.AlbumID,
		Artist:             req.Artist,
		Album:              req.Album,
		Track:              req.Track,
		TrackNumber:        req.TrackNumber,
		DiscNumber:         req.DiscNumber,
		Provider:           req.Provider,
		Model:              req.Model,
		ClientPlatform:     req.ClientPlatform,
	}
	f.jobByID[job.ID] = job
	return job, false, nil
}

func (f *fakeAIRouteService) GetInsightJob(_ context.Context, jobID string) (*model.InsightJob, error) {
	if f.jobErr != nil {
		return nil, f.jobErr
	}
	if job, ok := f.jobByID[jobID]; ok {
		return job, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeAIRouteService) GetInsightJobCallLogs(
	_ context.Context, jobID string,
) ([]*model.LLMCallLog, error) {
	if f.jobErr != nil {
		return nil, f.jobErr
	}
	if _, ok := f.jobByID[jobID]; !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return f.callLogsByJobID[jobID], nil
}

func (f *fakeAIRouteService) ListInsightJobs(
	_ context.Context,
	query model.InsightJobListQuery,
) ([]*model.InsightJob, int64, error) {
	if f.jobErr != nil {
		return nil, 0, f.jobErr
	}
	f.lastListQuery = query

	jobs := make([]*model.InsightJob, 0, len(f.jobByID))
	for _, job := range f.jobByID {
		if query.HasStatus && job.Status != query.Status {
			continue
		}
		if query.HasTargetType && job.AnalysisTargetType != query.TargetType {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, int64(len(jobs)), nil
}

func (f *fakeAIRouteService) UpdateInsightJobLiveActivityToken(
	_ context.Context, jobID, token string,
) (*model.InsightJob, error) {
	if f.jobErr != nil {
		return nil, f.jobErr
	}
	job, ok := f.jobByID[jobID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	job.LiveActivityPushToken = token
	return job, nil
}

func (f *fakeAIRouteService) CancelInsightJob(_ context.Context, jobID string) (*model.InsightJob, error) {
	if f.jobErr != nil {
		return nil, f.jobErr
	}
	job, ok := f.jobByID[jobID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	job.Status = common.InsightJobPhaseCanceled
	return job, nil
}

func (f *fakeAIRouteService) RetryInsightJob(_ context.Context, jobID string) (*model.InsightJob, bool, error) {
	if f.jobErr != nil {
		return nil, false, f.jobErr
	}
	job, ok := f.jobByID[jobID]
	if !ok {
		return nil, false, gorm.ErrRecordNotFound
	}
	retried := *job
	retried.ID = "job-retry"
	retried.Status = common.InsightJobPhaseQueued
	f.jobByID[retried.ID] = &retried
	return &retried, false, nil
}

func (f *fakeAIRouteService) DeleteInsightJob(_ context.Context, jobID string) error {
	if f.jobErr != nil {
		return f.jobErr
	}
	job, ok := f.jobByID[jobID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if job.Status != common.InsightJobPhaseFailed && job.Status != common.InsightJobPhaseCanceled {
		return errors.New("仅允许删除失败或已取消的任务")
	}
	delete(f.jobByID, jobID)
	return nil
}

func newAITestRouter(service aiRouteService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerAIRoutes(router, service)
	return router
}

type fakeInsightFeedbackService struct {
	lastTrackFeedback struct {
		insightID      int64
		score          int
		comment        string
		reasonCodes    []string
		sectionKey     string
		sourcePlatform string
	}
	lastAlbumFeedback struct {
		insightID      int64
		score          int
		comment        string
		reasonCodes    []string
		sectionKey     string
		sourcePlatform string
	}
	trackErr   error
	albumErr   error
	summaryErr error
	historyErr error
	trackList  []*model.TrackInsightFeedback
	albumList  []*model.AlbumInsightFeedback
	summary    *insight.InsightFeedbackSummary
	history    []*insight.InsightFeedbackHistoryItem
	versions   []*model.InsightListItem
}

func (f *fakeInsightFeedbackService) RecordFeedback(
	_ context.Context, insightID int64, req insight.FeedbackRecordRequest,
) error {
	f.lastTrackFeedback = struct {
		insightID      int64
		score          int
		comment        string
		reasonCodes    []string
		sectionKey     string
		sourcePlatform string
	}{
		insightID:      insightID,
		score:          req.Score,
		comment:        req.Comment,
		reasonCodes:    req.ReasonCodes,
		sectionKey:     req.SectionKey,
		sourcePlatform: req.SourcePlatform,
	}
	return f.trackErr
}

func (f *fakeInsightFeedbackService) RecordAlbumFeedback(
	_ context.Context, insightID int64, req insight.FeedbackRecordRequest,
) error {
	f.lastAlbumFeedback = struct {
		insightID      int64
		score          int
		comment        string
		reasonCodes    []string
		sectionKey     string
		sourcePlatform string
	}{
		insightID:      insightID,
		score:          req.Score,
		comment:        req.Comment,
		reasonCodes:    req.ReasonCodes,
		sectionKey:     req.SectionKey,
		sourcePlatform: req.SourcePlatform,
	}
	return f.albumErr
}

func (f *fakeInsightFeedbackService) GetTrackInsightFeedbacks(
	_ context.Context, _ int64,
) ([]*model.TrackInsightFeedback, error) {
	if f.trackErr != nil {
		return nil, f.trackErr
	}
	return f.trackList, nil
}

func (f *fakeInsightFeedbackService) GetAlbumInsightFeedbacks(
	_ context.Context, _ int64,
) ([]*model.AlbumInsightFeedback, error) {
	if f.albumErr != nil {
		return nil, f.albumErr
	}
	return f.albumList, nil
}

func (f *fakeInsightFeedbackService) GetInsightFeedbackSummary(
	_ context.Context, _ common.AnalysisTargetType, _ int64,
) (*insight.InsightFeedbackSummary, error) {
	if f.summaryErr != nil {
		return nil, f.summaryErr
	}
	return f.summary, nil
}

func (f *fakeInsightFeedbackService) GetInsightFeedbackHistory(
	_ context.Context, _ common.AnalysisTargetType, _ int64, _ int,
) ([]*insight.InsightFeedbackHistoryItem, error) {
	if f.historyErr != nil {
		return nil, f.historyErr
	}
	return f.history, nil
}

func (f *fakeInsightFeedbackService) GetInsightHistory(
	_ context.Context, _ common.AnalysisTargetType, _ int64, _ int,
) ([]*model.InsightListItem, error) {
	if f.historyErr != nil {
		return nil, f.historyErr
	}
	return f.versions, nil
}

func newInsightFeedbackTestRouter(service insightFeedbackService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerInsightFeedbackRoutes(router, service)
	return router
}

func TestRegisterAIRoutesListPlatforms(t *testing.T) {
	service := &fakeAIRouteService{
		platforms: []ai.PlatformOption{
			{ID: common.AIModelPlatformOllama, DisplayName: "OpenAI"},
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
	if resp.Platforms[0].ID != common.AIModelPlatformOllama {
		t.Fatalf("unexpected first platform: %s", resp.Platforms[0].ID)
	}
}

func TestRegisterAIRoutesListModelsByPlatform(t *testing.T) {
	service := &fakeAIRouteService{
		modelsByPlatform: map[common.AIModelPlatform][]ai.ModelOption{
			common.AIModelPlatformOllama: {
				{ID: "gpt-5", DisplayName: "GPT-5", IsDefault: true},
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/ai-models/ollama/models", nil)
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

func TestRegisterAIRoutesDeprecatedPostEndpointsReturnGone(t *testing.T) {
	service := &fakeAIRouteService{}
	router := newAITestRouter(service)

	// POST /api/track-insight 应当返回 410 Gone
	bodyTrack := `{"artist":"A","album":"B","track":"C","track_number":1,"disc_number":2,"provider":"openai","model":"gpt-5"}`
	reqTrack := httptest.NewRequest(http.MethodPost, "/api/track-insight", strings.NewReader(bodyTrack))
	reqTrack.Header.Set("Content-Type", "application/json")
	recTrack := httptest.NewRecorder()
	router.ServeHTTP(recTrack, reqTrack)

	if recTrack.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for POST /api/track-insight, got %d body=%s", recTrack.Code, recTrack.Body.String())
	}
	if !strings.Contains(recTrack.Body.String(), "DEPRECATED_USE_INSIGHT_JOBS") {
		t.Fatalf("expected error code DEPRECATED_USE_INSIGHT_JOBS, got %s", recTrack.Body.String())
	}

	// POST /api/album-insight 应当返回 410 Gone
	bodyAlbum := `{"album_id":42,"modelType":"gemini"}`
	reqAlbum := httptest.NewRequest(http.MethodPost, "/api/album-insight", strings.NewReader(bodyAlbum))
	reqAlbum.Header.Set("Content-Type", "application/json")
	recAlbum := httptest.NewRecorder()
	router.ServeHTTP(recAlbum, reqAlbum)

	if recAlbum.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for POST /api/album-insight, got %d body=%s", recAlbum.Code, recAlbum.Body.String())
	}
	if !strings.Contains(recAlbum.Body.String(), "DEPRECATED_USE_INSIGHT_JOBS") {
		t.Fatalf("expected error code DEPRECATED_USE_INSIGHT_JOBS, got %s", recAlbum.Body.String())
	}
}

func TestRegisterInsightFeedbackRoutesAlbumFeedbackPassesThrough(t *testing.T) {
	service := &fakeInsightFeedbackService{}
	router := newInsightFeedbackTestRouter(service)

	body := `{"score":-1,"comment":"专辑结构可以更紧凑","reason_codes":["结构混乱"],"section_key":"summary","source_platform":"iphone"}`
	req := httptest.NewRequest(http.MethodPost, "/api/album-insight/88/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.lastAlbumFeedback.insightID != 88 || service.lastAlbumFeedback.score != -1 {
		t.Fatalf("album feedback not passed through: %+v", service.lastAlbumFeedback)
	}
	if len(service.lastAlbumFeedback.reasonCodes) != 1 || service.lastAlbumFeedback.reasonCodes[0] != "结构混乱" {
		t.Fatalf("album feedback reason codes not passed through: %+v", service.lastAlbumFeedback)
	}
	if service.lastAlbumFeedback.sectionKey != "summary" || service.lastAlbumFeedback.sourcePlatform != "iphone" {
		t.Fatalf("album feedback metadata not passed through: %+v", service.lastAlbumFeedback)
	}
}

func TestRegisterInsightFeedbackRoutesAlbumFeedbackListUsesTargetType(t *testing.T) {
	service := &fakeInsightFeedbackService{
		albumList: []*model.AlbumInsightFeedback{
			{ID: 1, InsightID: 88, Score: -1, Comment: "建议补充结构层次"},
		},
	}
	router := newInsightFeedbackTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/insights/88/feedbacks?analysis_target_type=album", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Feedbacks []model.AlbumInsightFeedback `json:"feedbacks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Feedbacks) != 1 || resp.Feedbacks[0].InsightID != 88 {
		t.Fatalf("unexpected feedback response: %+v", resp.Feedbacks)
	}
}

func TestRegisterInsightFeedbackRoutesFeedbackSummary(t *testing.T) {
	service := &fakeInsightFeedbackService{
		summary: &insight.InsightFeedbackSummary{
			InsightID:          88,
			AnalysisTargetType: common.AnalysisTargetTypeTrack,
			LikeCount:          2,
			DislikeCount:       1,
			HasFeedback:        true,
			TopReasonCodes:     []string{"太空泛"},
		},
	}
	router := newInsightFeedbackTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/insights/88/feedback-summary?analysis_target_type=track", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp insight.InsightFeedbackSummary
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode summary response: %v", err)
	}
	if resp.LikeCount != 2 || resp.DislikeCount != 1 || len(resp.TopReasonCodes) != 1 {
		t.Fatalf("unexpected summary response: %+v", resp)
	}
}

func TestRegisterInsightFeedbackRoutesFeedbackHistory(t *testing.T) {
	service := &fakeInsightFeedbackService{
		history: []*insight.InsightFeedbackHistoryItem{
			{ID: 7, InsightID: 88, Score: -1, Comment: "总评太空泛"},
		},
	}
	router := newInsightFeedbackTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/insights/88/feedback-history?analysis_target_type=track&limit=5", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Feedbacks []insight.InsightFeedbackHistoryItem `json:"feedbacks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode history response: %v", err)
	}
	if len(resp.Feedbacks) != 1 || resp.Feedbacks[0].ID != 7 {
		t.Fatalf("unexpected history response: %+v", resp.Feedbacks)
	}
}

func TestRegisterInsightFeedbackRoutesInsightHistory(t *testing.T) {
	service := &fakeInsightFeedbackService{
		versions: []*model.InsightListItem{
			{ID: 100, Artist: "Artist", Album: "Album", Track: "Track v2"},
			{ID: 88, Artist: "Artist", Album: "Album", Track: "Track v1"},
		},
	}
	router := newInsightFeedbackTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/insights/88/history?analysis_target_type=track&limit=5", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Insights []*model.InsightListItem `json:"insights"`
		Total    int                      `json:"total"`
		Limit    int                      `json:"limit"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode history response: %v", err)
	}
	if len(resp.Insights) != 2 || resp.Insights[0].ID != 100 || resp.Total != 2 || resp.Limit != 5 {
		t.Fatalf("unexpected insight history response: %+v", resp)
	}
}

func TestRegisterAIRoutesCreateInsightJob(t *testing.T) {
	service := &fakeAIRouteService{}
	router := newAITestRouter(service)

	body := `{"target_type":"track","artist":"A","album":"B","track":"C","provider":"openai","model":"gpt-5","client_platform":"iphone"}`
	req := httptest.NewRequest(http.MethodPost, "/api/insight-jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Job      model.InsightJob `json:"job"`
		Existing bool             `json:"existing"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Job.ID == "" || resp.Job.Track != "C" || resp.Job.Provider != "openai" {
		t.Fatalf("unexpected job response: %+v", resp.Job)
	}
}

func TestRegisterAIRoutesGetInsightJob(t *testing.T) {
	resultInsightID := int64(88)
	service := &fakeAIRouteService{
		jobByID: map[string]*model.InsightJob{
			"job-1": {
				ID:                 "job-1",
				AnalysisTargetType: common.AnalysisTargetTypeAlbum,
				Status:             common.InsightJobPhaseRunning,
				AlbumID:            42,
				Artist:             "Artist",
				Album:              "Album",
				Provider:           "openai",
				Model:              "gpt-5",
				ResultInsightID:    &resultInsightID,
			},
		},
		callLogsByJobID: map[string][]*model.LLMCallLog{
			"job-1": {
				{
					ID:         101,
					Provider:   "openai",
					Model:      "gpt-5",
					Status:     "success",
					DurationMs: 2345,
				},
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/insight-jobs/job-1", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Job      model.InsightJob   `json:"job"`
		CallLogs []model.LLMCallLog `json:"call_logs"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Job.ResultInsightID == nil || *resp.Job.ResultInsightID != resultInsightID {
		t.Fatalf("unexpected result insight id: %+v", resp.Job.ResultInsightID)
	}
	if len(resp.CallLogs) != 1 || resp.CallLogs[0].ID != 101 {
		t.Fatalf("unexpected call logs: %+v", resp.CallLogs)
	}
}

func TestRegisterAIRoutesListInsightJobs(t *testing.T) {
	service := &fakeAIRouteService{
		jobByID: map[string]*model.InsightJob{
			"job-1": {
				ID:                 "job-1",
				AnalysisTargetType: common.AnalysisTargetTypeTrack,
				Status:             common.InsightJobPhaseRunning,
			},
			"job-2": {
				ID:                 "job-2",
				AnalysisTargetType: common.AnalysisTargetTypeAlbum,
				Status:             common.InsightJobPhaseFailed,
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/insight-jobs?status=running&analysis_target_type=track", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !service.lastListQuery.HasStatus || service.lastListQuery.Status != common.InsightJobPhaseRunning {
		t.Fatalf("unexpected list status query: %+v", service.lastListQuery)
	}
	if !service.lastListQuery.HasTargetType || service.lastListQuery.TargetType != common.AnalysisTargetTypeTrack {
		t.Fatalf("unexpected target type query: %+v", service.lastListQuery)
	}

	var resp struct {
		Jobs  []model.InsightJob `json:"jobs"`
		Total int64              `json:"total"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Jobs) != 1 || resp.Jobs[0].ID != "job-1" || resp.Total != 1 {
		t.Fatalf("unexpected list response: %+v", resp)
	}
}

func TestRegisterAIRoutesCancelInsightJob(t *testing.T) {
	service := &fakeAIRouteService{
		jobByID: map[string]*model.InsightJob{
			"job-1": {
				ID:     "job-1",
				Status: common.InsightJobPhaseRunning,
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/api/insight-jobs/job-1/cancel", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.jobByID["job-1"].Status != common.InsightJobPhaseCanceled {
		t.Fatalf("job status not canceled: %+v", service.jobByID["job-1"])
	}
}

func TestRegisterAIRoutesRetryInsightJob(t *testing.T) {
	service := &fakeAIRouteService{
		jobByID: map[string]*model.InsightJob{
			"job-1": {
				ID:                 "job-1",
				AnalysisTargetType: common.AnalysisTargetTypeTrack,
				Status:             common.InsightJobPhaseFailed,
				Track:              "Track",
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/api/insight-jobs/job-1/retry", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Job      model.InsightJob `json:"job"`
		Existing bool             `json:"existing"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Job.ID != "job-retry" || resp.Job.Status != common.InsightJobPhaseQueued {
		t.Fatalf("unexpected retry response: %+v", resp)
	}
}

func TestRegisterAIRoutesDeleteInsightJob(t *testing.T) {
	service := &fakeAIRouteService{
		jobByID: map[string]*model.InsightJob{
			"job-1": {
				ID:     "job-1",
				Status: common.InsightJobPhaseFailed,
			},
		},
	}
	router := newAITestRouter(service)

	req := httptest.NewRequest(http.MethodDelete, "/api/insight-jobs/job-1", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if _, ok := service.jobByID["job-1"]; ok {
		t.Fatalf("expected job to be deleted")
	}
}

func TestRegisterAIRoutesDeprecatedStreamReturnsGone(t *testing.T) {
	service := &fakeAIRouteService{}
	router := newAITestRouter(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/track-insight-stream?artist=A&track=C&provider=openai&model=gpt-5&force=true",
		nil,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for GET /api/track-insight-stream, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "DEPRECATED_USE_INSIGHT_JOBS") {
		t.Fatalf("expected error code DEPRECATED_USE_INSIGHT_JOBS, got %s", recorder.Body.String())
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

type fakeGenreRouteService struct {
	lastGenreQueried string
	albumsToReturn   []*model.Album
	totalToReturn    int64
	genreInfo        *model.Genre
}

func (f *fakeGenreRouteService) GetAlbumsByGenre(ctx context.Context, genre string, limit, offset int, sortBy string) ([]*model.Album, error) {
	f.lastGenreQueried = genre
	return f.albumsToReturn, nil
}

func (f *fakeGenreRouteService) GetAlbumsByGenreCount(ctx context.Context, genre string) (int64, error) {
	return f.totalToReturn, nil
}

func (f *fakeGenreRouteService) GetGenreByName(ctx context.Context, name string) (*model.Genre, error) {
	return f.genreInfo, nil
}

func (f *fakeGenreRouteService) CreateGenre(ctx context.Context, genre *model.Genre) error {
	return nil
}
func (f *fakeGenreRouteService) GetGenreByID(ctx context.Context, id uint) (*model.Genre, error) {
	return nil, nil
}
func (f *fakeGenreRouteService) GetAllGenres(ctx context.Context, limit, offset int) ([]*model.Genre, error) {
	return nil, nil
}
func (f *fakeGenreRouteService) UpdateGenre(ctx context.Context, genre *model.Genre) error {
	return nil
}
func (f *fakeGenreRouteService) DeleteGenre(ctx context.Context, id uint) error { return nil }
func (f *fakeGenreRouteService) IncrementGenrePlayCount(ctx context.Context, name string) error {
	return nil
}
func (f *fakeGenreRouteService) GetGenreCount(ctx context.Context) (int64, error) { return 0, nil }
func (f *fakeGenreRouteService) GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*model.Genre, error) {
	return nil, nil
}
func (f *fakeGenreRouteService) GetTopGenresWithDetails(ctx context.Context, limit int) ([]*model.TopGenre, error) {
	return nil, nil
}

func TestGenreAlbumsRouteUnescaping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fakeService := &fakeGenreRouteService{
		albumsToReturn: []*model.Album{
			{ID: 47, Name: "万能青年旅店", Artist: "万能青年旅店", Genre: "Adult Alternative"},
		},
		totalToReturn: 1,
		genreInfo:     &model.Genre{Name: "Adult Alternative", NameZh: "成人另类"},
	}

	router := gin.New()
	router.GET("/api/genres/:name/albums", func(c *gin.Context) {
		ctx := c.Request.Context()
		genreName := c.Param("name")
		if unescaped, err := url.PathUnescape(genreName); err == nil && unescaped != "" {
			genreName = unescaped
		} else if unescaped, err := url.QueryUnescape(genreName); err == nil && unescaped != "" {
			genreName = unescaped
		}
		if strings.Contains(genreName, "%20") || strings.Contains(genreName, "%2B") || strings.Contains(genreName, "%2F") {
			if unescaped, err := url.QueryUnescape(genreName); err == nil && unescaped != "" {
				genreName = unescaped
			}
		}
		genreName = strings.TrimSpace(genreName)
		albums, err := fakeService.GetAlbumsByGenre(ctx, genreName, 20, 0, "play_count")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, _ := fakeService.GetAlbumsByGenreCount(ctx, genreName)
		c.JSON(http.StatusOK, gin.H{
			"genre":    genreName,
			"genre_zh": "成人另类",
			"albums":   albums,
			"total":    total,
		})
	})

	// 1. 标准 URL 编码: GET /api/genres/Adult%20Alternative/albums
	req1 := httptest.NewRequest(http.MethodGet, "/api/genres/Adult%20Alternative/albums", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	if fakeService.lastGenreQueried != "Adult Alternative" {
		t.Fatalf("expected decoded 'Adult Alternative', got '%s'", fakeService.lastGenreQueried)
	}

	// 2. 二次转义防御: GET /api/genres/Adult%2520Alternative/albums
	req2 := httptest.NewRequest(http.MethodGet, "/api/genres/Adult%2520Alternative/albums", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	if fakeService.lastGenreQueried != "Adult Alternative" {
		t.Fatalf("expected defense unescaped 'Adult Alternative', got '%s'", fakeService.lastGenreQueried)
	}
}
