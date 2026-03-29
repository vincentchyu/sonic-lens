package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
