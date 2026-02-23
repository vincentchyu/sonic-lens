package lyrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type LrcAPIProvider struct {
	baseURL string
}

func NewLrcAPIProvider() *LrcAPIProvider {
	return &LrcAPIProvider{
		baseURL: "https://api.lrc.cx/lyrics",
	}
}

func (p *LrcAPIProvider) GetName() string {
	return "LrcAPI"
}

type lrcResponse struct {
	Lyrics string `json:"lyrics"`
}

func (p *LrcAPIProvider) GetLyrics(ctx context.Context, artist, album, track string) (string, error) {
	u, err := url.Parse(p.baseURL)
	if err != nil {
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
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lrcapi returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
