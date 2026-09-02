package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	arkmgmt "github.com/volcengine/volcengine-go-sdk/service/ark"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// --- Doubao Provider ---

// DoubaoProvider 使用本地 Doubao 服务实现 LLMProvider
type DoubaoProvider struct {
	BaseProvider
	host  string
	model string

	temperature       float32
	topP              float32
	topK              float32
	repetitionPenalty float32

	client *arkruntime.Client
}

type doubaoProviderFactory struct {
	cfg               config.DoubaoConfig
	runtimeBaseURL    string
	defaultModel      string
	runtimeClient     *arkruntime.Client
	managementClient  *arkmgmt.ARK
	initErr           error
	managementInitErr error
}

// newDoubaoProvider 从 config 创建 Doubao Provider
func newDoubaoProvider(cfg config.DoubaoConfig) (LLMProvider, error) {
	return newDoubaoProviderFactory(cfg).Create("")
}

func newDoubaoProviderFactory(cfg config.DoubaoConfig) providerFactory {
	factory := &doubaoProviderFactory{
		cfg:            cfg,
		runtimeBaseURL: resolveDoubaoRuntimeBaseURL(cfg),
		defaultModel:   resolveDoubaoRuntimeModel(cfg),
	}
	if !factory.hasRuntimeConfig() {
		return factory
	}

	factory.runtimeClient = arkruntime.NewClientWithApiKey(
		resolveDoubaoRuntimeAPIKey(cfg),
		arkruntime.WithBaseUrl(factory.runtimeBaseURL),
	)

	if hasDoubaoManagementConfig(cfg) {
		client, err := newDoubaoManagementClient(cfg)
		if err != nil {
			factory.managementInitErr = err
			return factory
		}
		factory.managementClient = client
	}

	return factory
}

func (f *doubaoProviderFactory) Platform() common.AIModelPlatform {
	return common.AIModelPlatformDoubao
}

func (f *doubaoProviderFactory) DisplayName() string {
	return "Doubao"
}

func (f *doubaoProviderFactory) Configured() bool {
	return f.hasRuntimeConfig() && f.initErr == nil
}

func (f *doubaoProviderFactory) DefaultModel() string {
	return f.defaultModel
}

func (f *doubaoProviderFactory) Create(model string) (LLMProvider, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}

	modelName := strings.TrimSpace(model)
	if modelName == "" {
		modelName = f.defaultModel
	}

	return &DoubaoProvider{
		BaseProvider: BaseProvider{
			ProviderName: "doubao",
			ModelName:    modelName,
		},
		host:   f.runtimeBaseURL,
		model:  modelName,
		client: f.runtimeClient,
		/*
			用于 JSON 输出：
			* 0.0 = 最稳定（强推荐）
			* 0.1 = 工程常用
			* 0.2 = 还能接受
			* 0.3 = 已经开始偶尔漂
		*/
		temperature: DefaultInsightTemperature,
		/*
			场景
			推荐 topP
			JSON / function calling
			1.0（或 0.9~1.0）
			一般文本生成
			0.7~0.9
			创意写作
			0.9~1.0
		*/
		topP: 1.0,
		/*
			topK 太小：
			* 限制 token 候选
			* JSON 符号（{ } : , “）可能被“挤掉”
			* 结构完整性下降
			场景
			topK
			JSON / API 输出
			0（关闭）或 50~100
			普通生成
			20~50
			创意
			50~200
		*/
		topK: 0,
		/*
			在 JSON 场景：
			* penalty > 1.0 可能：
			    * 抑制重复 key pattern
			    * 导致字段缺失（严重问题）
			场景
			值
			JSON 输出
			1.0（推荐）
			普通文本
			1.0 ~ 1.1
		*/
		repetitionPenalty: 1.0,
	}, nil
}

func (f *doubaoProviderFactory) ListModels(ctx context.Context) ([]ModelOption, error) {
	if !hasDoubaoManagementConfig(f.cfg) {
		return fallbackSingleModelCatalog(f.DefaultModel(), "Doubao Endpoint"), nil
	}
	if f.managementInitErr != nil {
		log.Warn(ctx, "Doubao 管理端客户端初始化失败，回退到默认 Endpoint 目录", zap.Error(f.managementInitErr))
		return fallbackSingleModelCatalog(f.DefaultModel(), "Doubao Endpoint"), nil
	}
	if f.managementClient == nil {
		return fallbackSingleModelCatalog(f.DefaultModel(), "Doubao Endpoint"), nil
	}
	// ListFoundationModels
	resp, err := f.managementClient.ListEndpointsWithContext(
		ctx, (&arkmgmt.ListEndpointsInput{}).
			SetProjectName(resolveDoubaoProjectName(f.cfg)).
			SetPageNumber(1).
			SetPageSize(100),
	)
	if err != nil {
		return nil, err
	}

	defaultModel := f.DefaultModel()
	models := make([]ModelOption, 0, len(resp.Items))
	seen := make(map[string]struct{}, len(resp.Items))
	for _, item := range resp.Items {
		if item == nil || item.Id == nil {
			continue
		}
		id := strings.TrimSpace(volcengine.StringValue(item.Id))
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		displayName := buildDoubaoDisplayName(item)
		models = append(
			models, ModelOption{
				ID:          id,
				DisplayName: displayName,
				IsDefault:   id == defaultModel,
			},
		)
	}

	if len(models) == 0 {
		return fallbackSingleModelCatalog(defaultModel, defaultModel), nil
	}
	if defaultModel != "" {
		if _, exists := seen[defaultModel]; !exists {
			models = append(
				models, ModelOption{
					ID:          defaultModel,
					DisplayName: defaultModel,
					IsDefault:   true,
				},
			)
		}
	}
	slices.SortFunc(
		models, func(a, b ModelOption) int {
			if a.IsDefault != b.IsDefault {
				if a.IsDefault {
					return -1
				}
				return 1
			}
			return cmp.Compare(a.DisplayName, b.DisplayName)
		},
	)
	return models, nil
}

func (f *doubaoProviderFactory) CacheFingerprint() string {
	values := []string{
		string(f.Platform()),
		resolveDoubaoRuntimeBaseURL(f.cfg),
		f.DefaultModel(),
		resolveDoubaoRuntimeAPIKey(f.cfg),
		resolveDoubaoManagementAccessKey(f.cfg),
		resolveDoubaoManagementSecretKey(f.cfg),
		resolveDoubaoManagementRegion(f.cfg),
		resolveDoubaoProjectName(f.cfg),
	}
	return strings.Join(values, "|")
}

func (f *doubaoProviderFactory) InitErr() error {
	return f.initErr
}

func (f *doubaoProviderFactory) hasRuntimeConfig() bool {
	return resolveDoubaoRuntimeAPIKey(f.cfg) != ""
}

func resolveDoubaoRuntimeAPIKey(cfg config.DoubaoConfig) string {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("DOUBAO_API_KEY"))
	}
	return apiKey
}

func resolveDoubaoRuntimeBaseURL(cfg config.DoubaoConfig) string {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("DOUBAO_BASE_URL"))
	}
	return strings.TrimRight(baseURL, "/")
}

func resolveDoubaoRuntimeModel(cfg config.DoubaoConfig) string {
	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(os.Getenv("DOUBAO_MODEL"))
	}
	return modelName
}

func resolveDoubaoManagementAccessKey(cfg config.DoubaoConfig) string {
	value := strings.TrimSpace(cfg.ManagementAccessKey)
	if value == "" {
		value = strings.TrimSpace(os.Getenv("DOUBAO_MANAGEMENT_ACCESS_KEY"))
	}
	return value
}

func resolveDoubaoManagementSecretKey(cfg config.DoubaoConfig) string {
	value := strings.TrimSpace(cfg.ManagementSecretKey)
	if value == "" {
		value = strings.TrimSpace(os.Getenv("DOUBAO_MANAGEMENT_SECRET_KEY"))
	}
	return value
}

func resolveDoubaoManagementRegion(cfg config.DoubaoConfig) string {
	value := strings.TrimSpace(cfg.ManagementRegion)
	if value == "" {
		value = strings.TrimSpace(os.Getenv("DOUBAO_MANAGEMENT_REGION"))
	}
	if value == "" {
		value = "cn-beijing"
	}
	return value
}

func resolveDoubaoProjectName(cfg config.DoubaoConfig) string {
	value := strings.TrimSpace(cfg.ProjectName)
	if value == "" {
		value = strings.TrimSpace(os.Getenv("DOUBAO_PROJECT_NAME"))
	}
	return value
}

func hasDoubaoManagementConfig(cfg config.DoubaoConfig) bool {
	return resolveDoubaoManagementAccessKey(cfg) != "" &&
		resolveDoubaoManagementSecretKey(cfg) != "" &&
		resolveDoubaoProjectName(cfg) != ""
}

func newDoubaoManagementClient(cfg config.DoubaoConfig) (*arkmgmt.ARK, error) {
	managerConfig := volcengine.NewConfig().
		WithRegion(resolveDoubaoManagementRegion(cfg)).
		WithCredentials(
			credentials.NewStaticCredentials(
				resolveDoubaoManagementAccessKey(cfg),
				resolveDoubaoManagementSecretKey(cfg),
				"",
			),
		)
	sess, err := session.NewSession(managerConfig)
	if err != nil {
		return nil, err
	}
	return arkmgmt.New(sess), nil
}

func fallbackSingleModelCatalog(id string, displayName string) []ModelOption {
	id = strings.TrimSpace(id)
	if id == "" {
		return []ModelOption{}
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = id
	}
	return []ModelOption{
		{
			ID:          id,
			DisplayName: displayName,
			IsDefault:   true,
		},
	}
}

func buildDoubaoDisplayName(item *arkmgmt.ItemForListEndpointsOutput) string {
	name := strings.TrimSpace(volcengine.StringValue(item.Name))
	if name == "" {
		name = strings.TrimSpace(volcengine.StringValue(item.Id))
	}
	if item.ModelReference == nil || item.ModelReference.FoundationModel == nil {
		return name
	}
	foundation := item.ModelReference.FoundationModel
	foundationName := strings.TrimSpace(volcengine.StringValue(foundation.Name))
	modelVersion := strings.TrimSpace(volcengine.StringValue(foundation.ModelVersion))
	switch {
	case foundationName != "" && modelVersion != "":
		return fmt.Sprintf("%s (%s %s)", name, foundationName, modelVersion)
	case foundationName != "":
		return fmt.Sprintf("%s (%s)", name, foundationName)
	default:
		return name
	}
}

// SendChatRequest 实现 RawChatClient 接口
func (p *DoubaoProvider) SendChatRequest(
	ctx context.Context, req TrackAnalysisRequest, systemPrompt, userPrompt string, schema map[string]any, step string,
) (string, error) {
	startTime := time.Now()

	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(systemPrompt),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(userPrompt),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeEnabled,
		},
		Temperature:       &p.temperature,
		TopP:              &p.topP,
		MaxTokens:         volcengine.Int(4096),
		RepetitionPenalty: &p.repetitionPenalty,
		ResponseFormat: &model.ResponseFormat{
			Type: model.ResponseFormatJsonObject,
		},
	}

	reqJSON, _ := json.Marshal(dReq)
	requestJSON := string(reqJSON)

	resp, err := p.client.CreateChatCompletion(ctx, dReq)
	var respJSON string
	if err == nil {
		rb, _ := json.Marshal(resp)
		respJSON = string(rb)
	}
	if err != nil {
		p.SaveCallLog(ctx, req, requestJSON, respJSON, err, startTime, step)
		return "", err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == nil || resp.Choices[0].Message.Content.StringValue == nil {
		err = errors.New("豆包 API 返回内容为空")
		p.SaveCallLog(ctx, req, requestJSON, respJSON, err, startTime, step)
		return "", err
	}

	p.SaveCallLog(ctx, req, requestJSON, respJSON, nil, startTime, step)
	return *resp.Choices[0].Message.Content.StringValue, nil
}

// AnalyzeTrack 调用豆包 API，对歌词进行翻译和深度解析
func (p *DoubaoProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	if config.ConfigObj.AI.MultiStep {
		return multiStepAnalyzeTrack(ctx, p, req)
	}
	return p.analyzeTrackSingleStep(ctx, req)
}

func (p *DoubaoProvider) analyzeTrackSingleStep(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	respStr, err := p.SendChatRequest(
		ctx, req, buildTrackInsightSystemPromptWithSchema(req), buildTrackInsightUserPrompt(req), nil, "sync",
	)
	if err != nil {
		return nil, err
	}

	result, raw, err := ParseTrackResult(respStr)
	if err != nil {
		log.Error(ctx, "解析豆包响应失败", zap.Error(err), zap.String("raw", raw))
		return nil, err
	}

	result.LLMProvider = "doubao:" + p.model
	return result, nil
}

// AnalyzeAlbum 调用豆包 API，对专辑聚合上下文进行深度解析。
func (p *DoubaoProvider) AnalyzeAlbum(
	ctx context.Context, req AlbumAnalysisRequest,
) (*AlbumAnalysisResult, error) {
	startTime := time.Now()
	var err error

	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildAlbumInsightSystemPromptWithSchema(req)),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildAlbumInsightUserPrompt(req)),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeEnabled,
		},
		Temperature:       &p.temperature,
		TopP:              &p.topP,
		MaxTokens:         volcengine.Int(4096),
		RepetitionPenalty: &p.repetitionPenalty,
		ResponseFormat: &model.ResponseFormat{
			Type: model.ResponseFormatJsonObject,
		},
	}

	reqJSON, _ := json.Marshal(dReq)
	requestJSON := string(reqJSON)
	log.Info(ctx, "豆包专辑分析请求体", zap.String("body", requestJSON))

	resp, err := p.client.CreateChatCompletion(ctx, dReq)
	var respJSON string
	if err == nil {
		rb, _ := json.Marshal(resp)
		respJSON = string(rb)
	}
	if err != nil {
		log.Error(ctx, "调用豆包专辑分析失败", zap.Error(err))
		p.SaveAlbumCallLog(ctx, req, requestJSON, "", err, startTime, "sync")
		return nil, err
	}

	log.Debug(ctx, "豆包专辑分析响应体", zap.String("body", respJSON))

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == nil || resp.Choices[0].Message.Content.StringValue == nil {
		err = errors.New("豆包 API 返回内容为空")
		p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

	result, raw, err := ParseAlbumResult(*resp.Choices[0].Message.Content.StringValue)
	if err != nil {
		log.Error(ctx, "解析豆包专辑分析响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

	p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, nil, startTime, "sync")
	result.LLMProvider = "doubao:" + p.model
	return result, nil
}

// AnalyzeTrackStream 实现流式输出
// Deprecated: 流式接口已废弃
func (p *DoubaoProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	return nil, errors.New("流式接口已废弃")
}
