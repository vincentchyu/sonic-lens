package lyrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

type LrcAPIProvider struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewLrcAPIProvider(baseURL, token string) *LrcAPIProvider {
	if baseURL == "" {
		baseURL = "https://api.lrc.cx/lyrics"
	}
	return &LrcAPIProvider{
		baseURL: baseURL,
		token:   token,
		client: telemetry.WrapHTTPClient(&http.Client{
			Timeout: 20 * time.Second,
		}),
	}
}

func (p *LrcAPIProvider) GetName() string {
	return "LrcAPI"
}

func (p *LrcAPIProvider) GetLyrics(ctx context.Context, artist, album, track string) (string, error) {
	log.Info(
		ctx, "LrcAPI 开始获取歌词",
		zap.String("artist", artist),
		zap.String("album", album),
		zap.String("track", track),
		zap.String("baseURL", p.baseURL),
	)

	u, err := url.Parse(p.baseURL)
	if err != nil {
		log.Error(ctx, "LrcAPI 解析 baseURL 失败", zap.Error(err), zap.String("baseURL", p.baseURL))
		return "", err
	}

	params := url.Values{}
	params.Add("title", track)
	params.Add("artist", artist)
	if album != "" {
		params.Add("album", album)
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		log.Error(ctx, "LrcAPI 创建请求失败", zap.Error(err), zap.String("url", u.String()))
		return "", err
	}
	if p.token != "" {
		req.Header.Set("Authorization", p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.Error(ctx, "LrcAPI 发起请求失败", zap.Error(err), zap.String("url", u.String()))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("lrcapi returned status: %d", resp.StatusCode)
		log.Error(ctx, "LrcAPI 请求状态码异常", zap.Int("status", resp.StatusCode), zap.String("url", u.String()))
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "LrcAPI 读取响应体失败", zap.Error(err))
		return "", err
	}

	log.Info(
		ctx, "LrcAPI 成功获取歌词",
		zap.String("artist", artist),
		zap.String("track", track),
		zap.Int("length", len(body)),
	)

	return string(body), nil
}
