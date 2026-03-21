package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
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
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

// --- Doubao Provider ---

// DoubaoProvider 使用本地 Doubao 服务实现 LLMProvider
type DoubaoProvider struct {
	BaseProvider
	host   string
	model  string
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
	sort.Slice(
		models, func(i, j int) bool {
			if models[i].IsDefault != models[j].IsDefault {
				return models[i].IsDefault
			}
			return models[i].DisplayName < models[j].DisplayName
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

// AnalyzeTrack 调用豆包 API，对歌词进行翻译和深度解析
func (p *DoubaoProvider) AnalyzeTrack(
	ctx context.Context, req TrackAnalysisRequest,
) (*TrackAnalysisResult, error) {
	// 记录开始时间
	startTime := time.Now()
	var err error
	// 构建请求消息
	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightSystemPromptAll()),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightUserPrompt(req)),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeEnabled,
		},
		/*ResponseFormat: &model.ResponseFormat{
			Type: model.ResponseFormatJsonObject,
			JSONSchema: &model.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        "track_analysis",
				Description: "Deep analysis of a music track including lyrics translation and literary appreciation.",
				Schema:      GetTrackInsightSchema(),
				Strict:      true,
			},
		},*/
	}

	// 打印请求体
	reqJSON, _ := json.Marshal(dReq)
	requestJSON := string(reqJSON)
	log.Info(ctx, "豆包请求体", zap.String("body", requestJSON))

	// 调用豆包 API
	resp, err := p.client.CreateChatCompletion(ctx, dReq)
	// 异步保存调用流水
	var respJSON string
	if err == nil {
		rb, _ := json.Marshal(resp)
		respJSON = string(rb)
	}
	if err != nil {
		log.Error(ctx, "调用豆包 API 失败", zap.Error(err))
		return nil, err
	}

	/*responsesRequest := responses.ResponsesRequest{
		Input:              nil,
		Model:              "",
		MaxOutputTokens:    nil,
		PreviousResponseId: nil,
		Thinking:           nil,
		ServiceTier:        nil,
		Store:              nil,
		Stream:             nil,
		Temperature:        nil,
		Tools:              nil,
		TopP:               nil,
		Instructions:       nil,
		Include:            nil,
		Caching:            nil,
		Text:               nil,
		ExpireAt:           nil,
		ToolChoice:         nil,
		ParallelToolCalls:  nil,
		MaxToolCalls:       nil,
		Reasoning:          nil,
		ContextManagement:  nil,
	}
	createResponses, err := p.client.CreateResponses(ctx, &responsesRequest)
	if err != nil {
		log.Error(ctx, "调用豆包 API 失败", zap.Error(err))
		return nil, err
	}*/

	// 打印响应体
	log.Debug(ctx, "豆包响应体", zap.String("body", respJSON))

	// 检查响应内容
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == nil || resp.Choices[0].Message.Content.StringValue == nil {
		log.Warn(ctx, "豆包 API 返回内容为空")
		err = errors.New("豆包 API 返回内容为空")
		return nil, err
	}

	// 获取响应内容
	raw := TrimCodeFence(*resp.Choices[0].Message.Content.StringValue)
	normalizedRaw := repairPrematureTopLevelObjectClosure(raw)

	// 解析 JSON 响应
	var result TrackAnalysisResult
	if err = json.Unmarshal([]byte(normalizedRaw), &result); err != nil {
		// 如果解析失败，尝试从文本中提取 JSON 块
		if extracted := extractJSON(normalizedRaw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析豆包响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	p.SaveCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
	if normalizedRaw != raw {
		log.Warn(ctx, "豆包响应存在多余顶层闭括号，已自动修复", zap.String("model", p.model))
	}
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	// 修复：将字面量 \n 转换为实际换行符
	result.LyricsTranslation = strings.ReplaceAll(result.LyricsTranslation, "\\n", "\n")

	result.LLMProvider = "doubao:" + p.model
	return &result, nil
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
					StringValue: volcengine.String(buildAlbumInsightSystemPromptAll()),
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

	raw := TrimCodeFence(*resp.Choices[0].Message.Content.StringValue)
	normalizedRaw := repairPrematureTopLevelObjectClosure(raw)

	var result AlbumAnalysisResult
	if err = json.Unmarshal([]byte(normalizedRaw), &result); err != nil {
		if extracted := extractJSON(normalizedRaw); extracted != "" {
			if err = json.Unmarshal([]byte(extracted), &result); err == nil {
				goto SUCCESS
			}
		}
		log.Error(ctx, "解析豆包专辑分析响应失败", zap.Error(err), zap.String("raw", raw))
		p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, err, startTime, "sync")
		return nil, err
	}

SUCCESS:
	p.SaveAlbumCallLog(ctx, req, requestJSON, respJSON, nil, startTime, "sync")
	if normalizedRaw != raw {
		log.Warn(ctx, "豆包专辑响应存在多余顶层闭括号，已自动修复", zap.String("model", p.model))
	}
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.LLMProvider = "doubao:" + p.model
	return &result, nil
}

// AnalyzeTrackStream 实现流式输出
func (p *DoubaoProvider) AnalyzeTrackStream(ctx context.Context, req TrackAnalysisRequest) (<-chan string, error) {
	startTime := time.Now()

	// 构建请求消息
	dReq := model.CreateChatCompletionRequest{
		Model: p.model,
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightSystemPromptAll()),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(buildTrackInsightUserPrompt(req)),
				},
			},
		},
		Thinking: &model.Thinking{
			Type: model.ThinkingTypeDisabled,
		},
		StreamOptions: &model.StreamOptions{
			IncludeUsage: true,
		},
	}

	// 打印请求体
	reqJSON, _ := json.Marshal(dReq)
	requestJSON := string(reqJSON)
	log.Debug(ctx, "豆包流式请求体", zap.String("body", requestJSON))

	// 调用豆包流式 API
	stream, err := p.client.CreateChatCompletionStream(ctx, dReq)
	if err != nil {
		p.SaveCallLog(ctx, req, requestJSON, "", err, startTime, "stream")
		log.Error(ctx, "调用豆包流式 API 失败", zap.Error(err))
		return nil, err
	}

	out := make(chan string, 100)
	telemetry.GoSafe(
		ctx, "ai.doubao.stream_track_analysis", func(asyncCtx context.Context) {
			defer close(out)
			defer stream.Close()

			var fullResponse strings.Builder
			var finalErr error

			for {
				recv, err := stream.Recv()
				if err != nil {
					if err != io.EOF {
						log.Error(asyncCtx, "接收豆包流式响应失败", zap.Error(err))
						finalErr = err
					}
					break
				}

				// 记录全量内容，用于 SaveCallLog
				rb, _ := json.Marshal(recv)
				fullResponse.WriteString(string(rb))
				fullResponse.WriteString("\n")

				// 发送流式内容
				if len(recv.Choices) > 0 {
					content := recv.Choices[0].Delta.Content
					if content != "" {
						out <- content
					}
				}
			}

			// 流结束，保存日志
			p.SaveCallLog(asyncCtx, req, requestJSON, fullResponse.String(), finalErr, startTime, "stream")
		},
	)

	return out, nil
}
