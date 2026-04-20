package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
)

func TestResolveProviderSelectionUsesExplicitProviderAndModel(t *testing.T) {
	original := *config.ConfigObj
	t.Cleanup(func() {
		*config.ConfigObj = original
	})

	config.ConfigObj.AI = config.AIConfig{
		Provider: "openai",
		OpenAI: config.OpenAIConfig{
			APIKey: "test-key",
			Model:  "gpt-4.1-mini",
		},
	}

	selection, err := ResolveProviderSelection(
		ProviderSelectionInput{
			Provider: "openai",
			Model:    "gpt-5-mini",
		},
	)
	if err != nil {
		t.Fatalf("ResolveProviderSelection() error = %v", err)
	}
	if selection.Platform != "openai" {
		t.Fatalf("Platform = %q, want openai", selection.Platform)
	}
	if selection.Model != "gpt-5-mini" {
		t.Fatalf("Model = %q, want gpt-5-mini", selection.Model)
	}
}

func TestResolveProviderSelectionFallsBackToLegacyProvider(t *testing.T) {
	original := *config.ConfigObj
	t.Cleanup(func() {
		*config.ConfigObj = original
	})

	config.ConfigObj.AI = config.AIConfig{
		Provider: "openai",
		Ollama: config.OllamaConfig{
			Host:  "http://127.0.0.1:11434",
			Model: "qwen3:latest",
		},
	}

	selection, err := ResolveProviderSelection(
		ProviderSelectionInput{
			LegacyModelType: "ollama",
		},
	)
	if err != nil {
		t.Fatalf("ResolveProviderSelection() error = %v", err)
	}
	if selection.Platform != "ollama" {
		t.Fatalf("Platform = %q, want ollama", selection.Platform)
	}
	if selection.Model != "qwen3:latest" {
		t.Fatalf("Model = %q, want qwen3:latest", selection.Model)
	}
	if !selection.UsedLegacyField {
		t.Fatal("UsedLegacyField = false, want true")
	}
}

func TestListOpenAICompatibleModels(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/models" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"},{"id":"gpt-5-mini"}]}`))
		}),
	)
	defer server.Close()

	models, err := listOpenAICompatibleModels(context.Background(), server.URL, "key", "gpt-5-mini", 0)
	if err != nil {
		t.Fatalf("listOpenAICompatibleModels() error = %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	if models[0].ID != "gpt-5-mini" || !models[0].IsDefault {
		t.Fatalf("first model = %#v, want default gpt-5-mini first", models[0])
	}
}

func TestProviderFactoryRegistryReusesSingletonUntilConfigChanges(t *testing.T) {
	original := *config.ConfigObj
	t.Cleanup(func() {
		*config.ConfigObj = original
	})

	config.ConfigObj.AI = config.AIConfig{
		OpenAI: config.OpenAIConfig{
			APIKey: "test-key",
			Model:  "gpt-4.1-mini",
		},
	}

	first, err := getProviderFactory("openai")
	if err != nil {
		t.Fatalf("getProviderFactory() error = %v", err)
	}
	second, err := getProviderFactory("openai")
	if err != nil {
		t.Fatalf("getProviderFactory() error = %v", err)
	}
	if first != second {
		t.Fatal("expected factory singleton to be reused for identical config")
	}

	config.ConfigObj.AI.OpenAI.Model = "gpt-5-mini"

	third, err := getProviderFactory("openai")
	if err != nil {
		t.Fatalf("getProviderFactory() error = %v", err)
	}
	if first == third {
		t.Fatal("expected factory to rebuild after config change")
	}
}

func TestGetCachedModelsFallsBackToGeminiDefaultModelWhenCatalogUnavailable(t *testing.T) {
	factory := &stubProviderFactory{
		platform:     common.AIModelPlatformGemini,
		defaultModel: "gemini-3-flash-preview",
		listErr:      errors.New("Error 400, Message: User location is not supported for the API use., Status: FAILED_PRECONDITION, Details: []"),
	}

	models, err := getCachedModels(context.Background(), factory)
	if err != nil {
		t.Fatalf("getCachedModels() error = %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(models))
	}
	if !models[0].IsDefault {
		t.Fatal("fallback model should be marked as default")
	}
	if models[0].ID != factory.defaultModel {
		t.Fatalf("model ID = %q, want %q", models[0].ID, factory.defaultModel)
	}
}

type stubProviderFactory struct {
	platform     common.AIModelPlatform
	defaultModel string
	listModels   []ModelOption
	listErr      error
}

func (f *stubProviderFactory) Platform() common.AIModelPlatform {
	return f.platform
}

func (f *stubProviderFactory) DisplayName() string {
	return string(f.platform)
}

func (f *stubProviderFactory) Configured() bool {
	return true
}

func (f *stubProviderFactory) DefaultModel() string {
	return f.defaultModel
}

func (f *stubProviderFactory) Create(model string) (LLMProvider, error) {
	return nil, nil
}

func (f *stubProviderFactory) ListModels(ctx context.Context) ([]ModelOption, error) {
	return f.listModels, f.listErr
}

func (f *stubProviderFactory) CacheFingerprint() string {
	return "stub"
}

func (f *stubProviderFactory) InitErr() error {
	return nil
}
