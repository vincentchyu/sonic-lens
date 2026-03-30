package exec

import "testing"

func TestExiftoolInfoGetArtistPrefersID3ArtistFields(t *testing.T) {
	info := ExiftoolInfo{
		"Artist":  "æ\x9d\x8eå¿\u2014",
		"Artists": "李志",
		"Band":    "李志",
	}

	if got := info.GetArtist(); got != "李志" {
		t.Fatalf("GetArtist() = %q, want %q", got, "李志")
	}

	if got := info.GetArtists(); got != "李志" {
		t.Fatalf("GetArtists() = %q, want %q", got, "李志")
	}
}
