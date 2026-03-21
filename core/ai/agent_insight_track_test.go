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
