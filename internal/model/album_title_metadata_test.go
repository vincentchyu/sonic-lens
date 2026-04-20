package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
)

func TestAlbumTitleMetadataJSONValueAndScan(t *testing.T) {
	original := common.AlbumTitleMetadata{
		SourceDisplayTitle: "Kid A (Deluxe Edition)",
		OfficialTitle:      "Kid A",
		TitleVersions: []common.AlbumTitleVersion{
			{
				Text:          "Deluxe Edition",
				Type:          common.AlbumTitleVersionTypeEdition,
				Parenthesized: true,
			},
		},
		NormalizedDisplayTitle: "Kid A (Deluxe Edition)",
	}

	wrapped := AlbumTitleMetadataJSON(original)
	raw, err := wrapped.Value()
	require.NoError(t, err)
	require.Equal(
		t,
		`{"source_display_title":"Kid A (Deluxe Edition)","official_title":"Kid A","title_versions":[{"text":"Deluxe Edition","type":"edition","bracketed":false,"parenthesized":true}],"normalized_display_title":"Kid A (Deluxe Edition)"}`,
		raw,
	)

	var decoded AlbumTitleMetadataJSON
	require.NoError(t, decoded.Scan(raw))

	roundTrip := decoded.ToCommon()
	require.NotNil(t, roundTrip)
	require.Equal(t, original.SourceDisplayTitle, roundTrip.SourceDisplayTitle)
	require.Equal(t, original.OfficialTitle, roundTrip.OfficialTitle)
	require.Equal(t, original.NormalizedDisplayTitle, roundTrip.NormalizedDisplayTitle)
	require.Len(t, roundTrip.TitleVersions, 1)
	require.Equal(t, original.TitleVersions[0].Text, roundTrip.TitleVersions[0].Text)
	require.Equal(t, original.TitleVersions[0].Type, roundTrip.TitleVersions[0].Type)
}
