package common

import (
	"testing"
)

// TestParseAlbumTitleAndReleaseType 验证 Apple Music 连字符发行类型后缀的提取逻辑。
func TestParseAlbumTitleAndReleaseType(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantTitle       string
		wantReleaseType string
	}{
		{
			name:            "EP 后缀",
			input:           "In The Sun - EP",
			wantTitle:       "In The Sun",
			wantReleaseType: "ep",
		},
		{
			name:            "Single 后缀",
			input:           "Silky Spring - Single",
			wantTitle:       "Silky Spring",
			wantReleaseType: "single",
		},
		{
			name:            "LP 后缀",
			input:           "Mt. Sava - LP",
			wantTitle:       "Mt. Sava",
			wantReleaseType: "lp",
		},
		{
			name:            "大写 EP 后缀",
			input:           "Alma's Cove - EP",
			wantTitle:       "Alma's Cove",
			wantReleaseType: "ep",
		},
		{
			name:            "多余空格",
			input:           "Calzaghe  -  Single",
			wantTitle:       "Calzaghe",
			wantReleaseType: "single",
		},
		{
			name:            "普通专辑无后缀",
			input:           "Abbey Road",
			wantTitle:       "Abbey Road",
			wantReleaseType: "",
		},
		{
			name:            "Deluxe Edition 无发行类型后缀",
			input:           "Abbey Road (Deluxe Edition)",
			wantTitle:       "Abbey Road",
			wantReleaseType: "",
		},
		{
			name:            "连字符在标题中间不触发",
			input:           "A-ha - Take On Me",
			wantTitle:       "A-ha - Take On Me",
			wantReleaseType: "",
		},
		{
			name:            "末尾无效连字符不触发",
			input:           "Something - Other",
			wantTitle:       "Something - Other",
			wantReleaseType: "",
		},
		{
			name:            "EP 加括号后缀",
			input:           "Flowers - EP (Deluxe)",
			wantTitle:       "Flowers",
			wantReleaseType: "ep",
		},
		{
			name:            "大小写混合 single",
			input:           "Love Song - SINGLE",
			wantTitle:       "Love Song",
			wantReleaseType: "single",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTitle, gotReleaseType := ParseAlbumTitleAndReleaseType(tc.input)
			if gotTitle != tc.wantTitle {
				t.Errorf("ParseAlbumTitleAndReleaseType(%q).title = %q, want %q",
					tc.input, gotTitle, tc.wantTitle)
			}
			if gotReleaseType != tc.wantReleaseType {
				t.Errorf("ParseAlbumTitleAndReleaseType(%q).releaseType = %q, want %q",
					tc.input, gotReleaseType, tc.wantReleaseType)
			}
		})
	}
}

// TestParseAlbumTitleMetadataReleaseType 验证 ParseAlbumTitleMetadata 在
// AlbumTitleMetadata.ReleaseType 字段上的正确性。
func TestParseAlbumTitleMetadataReleaseType(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantTitle       string
		wantReleaseType string
	}{
		{
			name:            "EP 元数据提取",
			input:           "In The Sun - EP",
			wantTitle:       "In The Sun",
			wantReleaseType: "ep",
		},
		{
			name:            "Single 元数据提取",
			input:           "Silky Spring - Single",
			wantTitle:       "Silky Spring",
			wantReleaseType: "single",
		},
		{
			name:            "普通专辑无发行类型",
			input:           "Kid A",
			wantTitle:       "Kid A",
			wantReleaseType: "",
		},
		{
			name:            "Remaster 版本不影响 ReleaseType",
			input:           "OK Computer (2017 Remaster)",
			wantTitle:       "OK Computer",
			wantReleaseType: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := ParseAlbumTitleMetadata(tc.input)
			if meta.OfficialTitle != tc.wantTitle {
				t.Errorf("ParseAlbumTitleMetadata(%q).OfficialTitle = %q, want %q",
					tc.input, meta.OfficialTitle, tc.wantTitle)
			}
			if meta.ReleaseType != tc.wantReleaseType {
				t.Errorf("ParseAlbumTitleMetadata(%q).ReleaseType = %q, want %q",
					tc.input, meta.ReleaseType, tc.wantReleaseType)
			}
		})
	}
}

// TestParseAlbumTitleAndReleaseTypeDoesNotBreakExistingSubtitle 确保引入
// release type 解析后，原有的括号版本（Remaster/Deluxe等）解析不受影响。
func TestParseAlbumTitleAndReleaseTypeDoesNotBreakExistingSubtitle(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantTitle    string
		wantSubtitle string
	}{
		{
			name:         "remastered version 不受影响",
			input:        "Daydream Nation (2012 Remastered Version)",
			wantTitle:    "Daydream Nation",
			wantSubtitle: "2012 Remastered Version",
		},
		{
			name:         "deluxe edition 不受影响",
			input:        "Abbey Road (super deluxe edition)",
			wantTitle:    "Abbey Road",
			wantSubtitle: "super deluxe edition",
		},
		{
			name:         "live 不受影响",
			input:        "为你盛开-许巍《无尽光芒》巡回演唱会现场纪念 (Live)",
			wantTitle:    "为你盛开-许巍《无尽光芒》巡回演唱会现场纪念",
			wantSubtitle: "Live",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTitle, gotSubtitle := ParseAlbumTitleAndSubtitle(tc.input)
			if gotTitle != tc.wantTitle {
				t.Errorf("ParseAlbumTitleAndSubtitle(%q).title = %q, want %q",
					tc.input, gotTitle, tc.wantTitle)
			}
			if gotSubtitle != tc.wantSubtitle {
				t.Errorf("ParseAlbumTitleAndSubtitle(%q).subtitle = %q, want %q",
					tc.input, gotSubtitle, tc.wantSubtitle)
			}
		})
	}
}
