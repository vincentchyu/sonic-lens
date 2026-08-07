package ai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTrackInsightSchema(t *testing.T) {
	t.Parallel()

	schema := GetTrackInsightSchema()
	require.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, properties, "lyrics_translation")
	require.Contains(t, properties, "analysis_summary")
	require.Contains(t, properties, "analysis_by_section")
}

func TestGetAlbumInsightSchema(t *testing.T) {
	t.Parallel()

	schema := GetAlbumInsightSchema()
	require.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, properties, "analysis_summary")
	require.Contains(t, properties, "analysis_by_section")
	require.Contains(t, properties, "background_info")
	require.Contains(t, properties, "era_context")
}

func TestPrintTrackAndAlbumPrompts(t *testing.T) {
	// 1. 模拟单曲数据
	trackReq := TrackAnalysisRequest{
		Title:           "Hotel California",
		Artist:          "Eagles",
		Album:           "Hotel California",
		Lyrics:          "On a dark desert highway, cool wind in my hair...",
		LangSource:      "English",
		LangTarget:      "Chinese",
		FeedbackContext: "之前翻译的修辞不够优雅，请注意信雅达的表现。",
	}

	trackSystemPrompt := buildTrackInsightSystemPromptWithSchema(trackReq)
	trackUserPrompt := buildTrackInsightUserPrompt(trackReq)

	t.Log("==================== MOCK SINGLE TRACK SYSTEM PROMPT ====================")
	t.Log(trackSystemPrompt)
	t.Log("==================== MOCK SINGLE TRACK USER PROMPT ====================")
	t.Log(trackUserPrompt)

	// 2. 模拟专辑数据
	albumReq := AlbumAnalysisRequest{
		AlbumID:        12345,
		Artist:         "Eagles",
		Album:          "Hotel California",
		ReleaseDate:    "1976-12-08",
		Genre:          "Classic Rock",
		TrackCount:     9,
		AnalyzedTracks: 3,
		TrackContexts: []AlbumTrackContext{
			{
				TrackNumber:     1,
				Title:           "Hotel California",
				AnalysisSummary: "分析了意象与吉他独奏",
			},
			{
				TrackNumber:     2,
				Title:           "New Kid in Town",
				AnalysisSummary: "分析了名声带来的压力",
			},
			{
				TrackNumber:     3,
				Title:           "Life in the Fast Lane",
				AnalysisSummary: "分析了堕落与极速生活",
			},
		},
		FeedbackContext: "请在此次分析中特别突出 70 年代中产阶级美国梦的破灭这一哲学命题。",
	}

	albumSystemPrompt := buildAlbumInsightSystemPromptWithSchema(albumReq)
	albumUserPrompt := buildAlbumInsightUserPrompt(albumReq)

	t.Log("==================== MOCK ALBUM SYSTEM PROMPT ====================")
	t.Log(albumSystemPrompt)
	t.Log("==================== MOCK ALBUM USER PROMPT ====================")
	t.Log(albumUserPrompt)
}

func TestPrintMultiStepTrackPrompts(t *testing.T) {
	trackReq := TrackAnalysisRequest{
		Title:           "Hotel California",
		Artist:          "Eagles",
		Album:           "Hotel California",
		Lyrics:          "On a dark desert highway, cool wind in my hair...",
		LangSource:      "English",
		LangTarget:      "Chinese",
		FeedbackContext: "之前翻译的修辞不够优雅，请注意信雅达的表现。",
	}

	// Step 1
	sys1 := buildTrackInsightStep1SystemPrompt(trackReq)
	t.Log("==================== MOCK STEP 1 (Translation) SYSTEM PROMPT ====================")
	t.Log(sys1)

	// Step 2
	mockTranslation := "<original>On a dark desert highway</original>\n<translation>在黑暗的沙漠公路上</translation>"
	sys2 := buildTrackInsightStep2SystemPrompt(trackReq, mockTranslation)
	t.Log("==================== MOCK STEP 2 (Appreciate) SYSTEM PROMPT ====================")
	t.Log(sys2)

	// Step 3
	mockAppreciate := "<original>On a dark desert highway</original>\n<translation>在黑暗的沙漠公路上</translation>\n<explain>以公路隐喻人生旅途，奠定神秘冷冽的基调</explain>"
	sys3 := buildTrackInsightStep3SystemPrompt(trackReq, mockTranslation, mockAppreciate)
	t.Log("==================== MOCK STEP 3 (Summary/Deep) SYSTEM PROMPT ====================")
	t.Log(sys3)
}

func TestIsInstrumentalOrEmptyLyrics(t *testing.T) {
	tests := []struct {
		name     string
		lyrics   string
		expected bool
	}{
		{"空歌词", "", true},
		{"全空格歌词", "   \n\t  ", true},
		{"纯音乐标记1", "[Instrumental]", true},
		{"纯音乐标记2", "纯音乐，请欣赏", true},
		{"无歌词标记", "音乐暂无歌词", true},
		{"正常非中文歌词", "On a dark desert highway", false},
		{"正常中文歌词", "草莓 就在一瞬间 划过天际", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := TrackAnalysisRequest{Lyrics: tt.lyrics}
			got := IsInstrumentalOrEmptyLyrics(req)
			if got != tt.expected {
				t.Errorf("IsInstrumentalOrEmptyLyrics() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestIsChineseLyricsBridge(t *testing.T) {
	t.Parallel()

	chineseLyrics := "[00:12.34]故事的小黄花\n[00:15.67]从出生那年就飘着"
	require.True(t, IsChineseLyrics(chineseLyrics))

	fmtResult := FormatChineseLyricsTranslation(chineseLyrics)
	require.Contains(t, fmtResult, "<original>故事的小黄花</original>")
	require.Contains(t, fmtResult, "<translation></translation>")

	englishLyrics := "On a dark desert highway"
	require.False(t, IsChineseLyrics(englishLyrics))
}
