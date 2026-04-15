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

func TestExiftoolInfoIgnoresBinaryPlaceholderValues(t *testing.T) {
	info := ExiftoolInfo{
		"Artist": "(Binary data 6 bytes, use -b option to extract)",
		"Band":   "赵雷",
		"Album":  "(Binary data 6 bytes, use -b option to extract)",
		"title":  "署前街少年",
	}

	if got := info.GetArtist(); got != "赵雷" {
		t.Fatalf("GetArtist() = %q, want %q", got, "赵雷")
	}

	if got := info.GetAlbum(); got != "" {
		t.Fatalf("GetAlbum() = %q, want empty string", got)
	}

	if got := info.GetTitle(); got != "署前街少年" {
		t.Fatalf("GetTitle() = %q, want %q", got, "署前街少年")
	}
}
