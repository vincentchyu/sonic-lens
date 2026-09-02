package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

type openAICompatibleModelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func listOpenAICompatibleModels(
	ctx context.Context,
	baseURL string,
	apiKey string,
	defaultModel string,
	timeout time.Duration,
) ([]ModelOption, error) {
	client := telemetry.WrapHTTPClient(&http.Client{Timeout: timeout})
	return listOpenAICompatibleModelsWithClient(ctx, client, baseURL, apiKey, defaultModel)
}

func listOpenAICompatibleModelsWithClient(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	apiKey string,
	defaultModel string,
) ([]ModelOption, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询模型目录失败，状态码: %s", resp.Status)
	}

	var payload openAICompatibleModelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]ModelOption, 0, len(payload.Data))
	seen := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		models = append(
			models, ModelOption{
				ID:          id,
				DisplayName: id,
				IsDefault:   id == defaultModel,
			},
		)
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
			return cmp.Compare(a.ID, b.ID)
		},
	)
	return models, nil
}
