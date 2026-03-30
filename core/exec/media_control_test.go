package exec

import (
	"testing"

	"encoding/json/v2"
)

func TestMediaControlArtworkDataUnmarshalBase64(t *testing.T) {
	var info MediaControlNowPlayingInfo
	payload := []byte(`{"artworkData":"aGVsbG8="}`)
	if err := json.Unmarshal(payload, &info); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if got := string(info.ArtworkData); got != "hello" {
		t.Fatalf("expected decoded artwork data, got %q", got)
	}
}

func TestMediaControlArtworkDataUnmarshalPlainString(t *testing.T) {
	var info MediaControlNowPlayingInfo
	payload := []byte(`{"artworkData":"not-base64-artwork"}`)
	if err := json.Unmarshal(payload, &info); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if got := string(info.ArtworkData); got != "not-base64-artwork" {
		t.Fatalf("expected raw artwork data fallback, got %q", got)
	}
}
