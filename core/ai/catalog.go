package ai

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	coreredis "github.com/vincentchyu/sonic-lens/core/redis"
)

const (
	modelCatalogCacheTTL      = 10 * time.Minute
	modelCatalogEmptyCacheTTL = 30 * time.Second
)

// ModelOption 表示某个平台上的可选模型。
type ModelOption struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	IsDefault   bool   `json:"is_default"`
}

// PlatformOption 表示当前服务支持的模型平台。
type PlatformOption struct {
	ID           common.AIModelPlatform `json:"id"`
	DisplayName  string                 `json:"display_name"`
	DefaultModel string                 `json:"default_model,omitempty"`
}

// ProviderSelectionInput 表示接口层提交的模型选择输入。
type ProviderSelectionInput struct {
	Provider        string
	Model           string
	LegacyModelType string
}

// ResolvedProviderSelection 表示归一后的模型选择结果。
type ResolvedProviderSelection struct {
	Platform          common.AIModelPlatform
	Model             string
	RequestedProvider string
	RequestedModel    string
	UsedLegacyField   bool
}

type providerFactory interface {
	Platform() common.AIModelPlatform
	DisplayName() string
	Configured() bool
	DefaultModel() string
	Create(model string) (LLMProvider, error)
	ListModels(ctx context.Context) ([]ModelOption, error)
	CacheFingerprint() string
	InitErr() error
}

type modelCatalogCacheEntry struct {
	Models []ModelOption `json:"models"`
}

type providerFactoryRegistry struct {
	mu          sync.RWMutex
	fingerprint string
	factories   map[common.AIModelPlatform]providerFactory
}

var globalProviderFactoryRegistry providerFactoryRegistry

// ResolveProviderSelection 统一解析 provider/model 输入，并兼容旧的 modelType。
func ResolveProviderSelection(input ProviderSelectionInput) (ResolvedProviderSelection, error) {
	requestedProvider := strings.TrimSpace(input.Provider)
	requestedModel := strings.TrimSpace(input.Model)
	legacyModelType := strings.TrimSpace(input.LegacyModelType)

	selection := ResolvedProviderSelection{
		RequestedProvider: requestedProvider,
		RequestedModel:    requestedModel,
		UsedLegacyField:   requestedProvider == "" && legacyModelType != "",
	}

	var platform common.AIModelPlatform
	switch {
	case requestedProvider != "":
		platform = common.ParseAIModelPlatform(requestedProvider)
		if !platform.IsValid() {
			return selection, fmt.Errorf("不支持的 AI 平台: %s", requestedProvider)
		}
	case legacyModelType != "":
		platform = common.ParseAIModelPlatform(legacyModelType)
		if !platform.IsValid() {
			return selection, fmt.Errorf("不支持的旧模型平台参数: %s", legacyModelType)
		}
		selection.RequestedProvider = string(platform)
	default:
		platform = defaultConfiguredPlatform()
		selection.RequestedProvider = string(platform)
	}

	factory, err := getProviderFactory(platform)
	if err != nil {
		return selection, err
	}
	if !factory.Configured() {
		if factory.InitErr() != nil {
			return selection, fmt.Errorf("AI 平台初始化失败: %w", factory.InitErr())
		}
		return selection, fmt.Errorf("AI 平台未配置: %s", platform)
	}

	model := requestedModel
	if model == "" {
		model = strings.TrimSpace(factory.DefaultModel())
	}
	if model == "" {
		return selection, fmt.Errorf("AI 平台 %s 未配置默认模型，请显式传入 model", platform)
	}

	selection.Platform = platform
	selection.Model = model
	return selection, nil
}

// GetConfiguredPlatforms 返回当前服务已配置的平台列表。
func GetConfiguredPlatforms() []PlatformOption {
	factories := getConfiguredFactories()
	platforms := make([]PlatformOption, 0, len(factories))
	for _, factory := range factories {
		platforms = append(
			platforms, PlatformOption{
				ID:           factory.Platform(),
				DisplayName:  factory.DisplayName(),
				DefaultModel: factory.DefaultModel(),
			},
		)
	}
	return platforms
}

// GetModelsByPlatform 返回某个平台可用的模型目录。
func GetModelsByPlatform(ctx context.Context, platform common.AIModelPlatform) ([]ModelOption, error) {
	factory, err := getProviderFactory(platform)
	if err != nil {
		return nil, err
	}
	if !factory.Configured() {
		if factory.InitErr() != nil {
			return nil, fmt.Errorf("AI 平台初始化失败: %w", factory.InitErr())
		}
		return nil, fmt.Errorf("AI 平台未配置: %s", platform)
	}

	models, err := getCachedModels(ctx, factory)
	if err != nil {
		return nil, err
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

// ValidateModelAvailability 校验显式传入的模型是否在当前平台目录中。
func ValidateModelAvailability(ctx context.Context, platform common.AIModelPlatform, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}

	models, err := GetModelsByPlatform(ctx, platform)
	if err != nil {
		return err
	}
	for _, item := range models {
		if item.ID == model {
			return nil
		}
	}
	return fmt.Errorf("AI 平台 %s 不支持模型 %s", platform, model)
}

func getConfiguredFactories() []providerFactory {
	all := getOrBuildProviderFactories()
	factories := make([]providerFactory, 0, len(all))
	for _, factory := range all {
		if factory.Configured() {
			factories = append(factories, factory)
		}
	}
	return factories
}

func getProviderFactory(platform common.AIModelPlatform) (providerFactory, error) {
	factories := getOrBuildProviderFactories()
	factory, ok := factories[platform]
	if !ok {
		return nil, fmt.Errorf("不支持的 AI 平台: %s", platform)
	}
	return factory, nil
}

func getOrBuildProviderFactories() map[common.AIModelPlatform]providerFactory {
	fingerprint := currentProviderFactoryFingerprint()

	globalProviderFactoryRegistry.mu.RLock()
	if globalProviderFactoryRegistry.fingerprint == fingerprint && globalProviderFactoryRegistry.factories != nil {
		factories := globalProviderFactoryRegistry.factories
		globalProviderFactoryRegistry.mu.RUnlock()
		return factories
	}
	globalProviderFactoryRegistry.mu.RUnlock()

	globalProviderFactoryRegistry.mu.Lock()
	defer globalProviderFactoryRegistry.mu.Unlock()

	if globalProviderFactoryRegistry.fingerprint == fingerprint && globalProviderFactoryRegistry.factories != nil {
		return globalProviderFactoryRegistry.factories
	}

	factories := buildProviderFactories()
	globalProviderFactoryRegistry.fingerprint = fingerprint
	globalProviderFactoryRegistry.factories = factories
	return factories
}

func buildProviderFactories() map[common.AIModelPlatform]providerFactory {
	all := []providerFactory{
		newOpenAIProviderFactory(config.ConfigObj.AI.OpenAI),
		newGeminiProviderFactory(config.ConfigObj.AI.Gemini),
		newOllamaProviderFactory(config.ConfigObj.AI.Ollama),
		newDoubaoProviderFactory(config.ConfigObj.AI.Doubao),
		newCustomProviderFactory(config.ConfigObj.AI.Custom),
	}
	factories := make(map[common.AIModelPlatform]providerFactory, len(all))
	for _, factory := range all {
		if factory.InitErr() != nil {
			log.Warn(
				context.Background(),
				"初始化 AI 平台工厂失败，该平台将暂时不可用",
				zap.String("platform", string(factory.Platform())),
				zap.Error(factory.InitErr()),
			)
		}
		factories[factory.Platform()] = factory
	}
	return factories
}

func currentProviderFactoryFingerprint() string {
	payload, err := json.Marshal(config.ConfigObj.AI)
	if err != nil {
		return fmt.Sprintf("%+v", config.ConfigObj.AI)
	}
	return string(payload)
}

func defaultConfiguredPlatform() common.AIModelPlatform {
	platform := common.ParseAIModelPlatform(config.ConfigObj.AI.Provider)
	if platform.IsValid() {
		return platform
	}
	return common.AIModelPlatformOpenAI
}

func getCachedModels(ctx context.Context, factory providerFactory) ([]ModelOption, error) {
	cacheKey := buildModelCatalogCacheKey(factory)
	client := coreredis.GetRedisClient()
	if client != nil {
		cached, err := readModelCatalogCache(ctx, client, cacheKey)
		if err == nil {
			return cached, nil
		}
		if !errors.Is(err, goredis.Nil) {
			log.Warn(ctx, "读取 AI 模型目录缓存失败，将回源查询", zap.Error(err), zap.String("cache_key", cacheKey))
		}
	}

	models, err := factory.ListModels(ctx)
	if err != nil {
		if fallbackModels, fallback := fallbackModelCatalog(factory, err); fallback {
			log.Warn(
				ctx,
				"模型目录回源失败，已降级为默认模型",
				zap.Error(err),
				zap.String("cache_key", cacheKey),
				zap.String("platform", string(factory.Platform())),
				zap.String("default_model", fallbackModels[0].ID),
			)
			models = fallbackModels
		} else {
			log.Warn(ctx, "回源查询失败", zap.Error(err), zap.String("cache_key", cacheKey))
			return nil, err
		}
	}

	if client != nil {
		cacheTTL := modelCatalogCacheTTL
		if len(models) == 0 {
			cacheTTL = modelCatalogEmptyCacheTTL
		}
		if err := writeModelCatalogCache(ctx, client, cacheKey, models, cacheTTL); err != nil {
			log.Warn(ctx, "写入 AI 模型目录缓存失败", zap.Error(err), zap.String("cache_key", cacheKey))
		}
	}

	return models, nil
}

func buildModelCatalogCacheKey(factory providerFactory) string {
	hash := sha1.Sum([]byte(factory.CacheFingerprint()))
	return fmt.Sprintf("ai:model_catalog:%s:%s", factory.Platform(), hex.EncodeToString(hash[:]))
}

func readModelCatalogCache(ctx context.Context, client *goredis.Client, key string) ([]ModelOption, error) {
	value, err := client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, goredis.Nil) {
			log.Warn(ctx, "读取 AI 模型目录缓存失败", zap.Error(err), zap.String("cache_key", key))
		}
		return nil, err
	}

	var entry modelCatalogCacheEntry
	if err := json.Unmarshal([]byte(value), &entry); err != nil {
		log.Warn(ctx, "解析 AI 模型目录缓存失败", zap.Error(err), zap.String("cache_key", key))
		return nil, err
	}
	return entry.Models, nil
}

func writeModelCatalogCache(
	ctx context.Context, client *goredis.Client, key string, models []ModelOption, ttl time.Duration,
) error {
	payload, err := json.Marshal(modelCatalogCacheEntry{Models: models})
	if err != nil {
		return err
	}
	return client.Set(ctx, key, payload, ttl).Err()
}

func fallbackModelCatalog(factory providerFactory, err error) ([]ModelOption, bool) {
	if factory == nil || err == nil {
		return nil, false
	}
	if factory.Platform() != common.AIModelPlatformGemini {
		return nil, false
	}
	if !isGeminiModelCatalogUnavailableError(err) {
		return nil, false
	}

	defaultModel := strings.TrimSpace(factory.DefaultModel())
	if defaultModel == "" {
		return nil, false
	}

	return []ModelOption{
		{
			ID:          defaultModel,
			DisplayName: defaultModel,
			IsDefault:   true,
		},
	}, true
}

func isGeminiModelCatalogUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "user location is not supported") ||
		(strings.Contains(msg, "failed_precondition") && strings.Contains(msg, "location"))
}
