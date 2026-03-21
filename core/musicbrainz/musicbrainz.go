package musicbrainz

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uploadedlobster.com/mbtypes"
	"go.uploadedlobster.com/musicbrainzws2"

	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

var musicbrainzClient *musicbrainzws2.Client
var one sync.Once

const tracerName = "sonic-lens/core/musicbrainz"

func InitClient() {
	one.Do(
		func() {
			musicbrainzClient = musicbrainzws2.NewClient(
				musicbrainzws2.AppInfo{
					Name:    "sonic-lens",
					Version: "1.0",
					URL:     "https://blog-vincent.chyu.org/web/sonic-lens/",
				},
			)
		},
	)
}

func GetClient() *musicbrainzws2.Client {
	return musicbrainzClient
}

// SearchReleases 包装 MusicBrainz Release 搜索，补齐标准客户端 span。
func SearchReleases(
	ctx context.Context, filter musicbrainzws2.SearchFilter, paginator musicbrainzws2.Paginator,
) (musicbrainzws2.SearchReleasesResult, error) {
	client := GetClient()
	if client == nil {
		return musicbrainzws2.SearchReleasesResult{}, fmt.Errorf("musicbrainz client is not initialized")
	}

	spanCtx, span := telemetry.StartSpanForTracerName(
		ctx,
		tracerName,
		"musicbrainz.search_releases",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("external.system", "musicbrainz"),
		attribute.String("musicbrainz.operation", "search_releases"),
		attribute.Int("page.limit", paginator.Limit),
		attribute.Int("page.offset", paginator.Offset),
	)
	defer span.End()

	result, err := client.SearchReleases(spanCtx, filter, paginator)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return musicbrainzws2.SearchReleasesResult{}, err
	}

	span.SetAttributes(attribute.Int("musicbrainz.result_count", len(result.Releases)))
	return result, nil
}

// LookupRelease 包装 MusicBrainz Release 详情查询，补齐标准客户端 span。
func LookupRelease(
	ctx context.Context, id mbtypes.MBID, filter musicbrainzws2.IncludesFilter,
) (musicbrainzws2.Release, error) {
	client := GetClient()
	if client == nil {
		return musicbrainzws2.Release{}, fmt.Errorf("musicbrainz client is not initialized")
	}

	spanCtx, span := telemetry.StartSpanForTracerName(
		ctx,
		tracerName,
		"musicbrainz.lookup_release",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("external.system", "musicbrainz"),
		attribute.String("musicbrainz.operation", "lookup_release"),
		attribute.String("musicbrainz.release_mbid", string(id)),
		attribute.Int("musicbrainz.includes_count", len(filter.Includes)),
	)
	defer span.End()

	result, err := client.LookupRelease(spanCtx, id, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return musicbrainzws2.Release{}, err
	}

	return result, nil
}

// Close
func Close(ctx context.Context) {
	if musicbrainzClient != nil {
		err := musicbrainzClient.Close()
		if err != nil {
			fmt.Println("error closing musicbrainz client", err)
		}
	}
}

// TrackTitleWithFeat 检查 Track 的 ArtistCredit 是否包含 feat 合作者，
// 如果包含，将其格式化为 Title(feat.ArtistName) 的形式，否则返回原 Title。
// Hikky Burr (feat. Bill Cosby)
func TrackTitleWithFeat(track musicbrainzws2.Track) string {
	title := track.Title
	if len(track.ArtistCredit) <= 1 {
		return title
	}

	for i := 0; i < len(track.ArtistCredit)-1; i++ {
		joinPhrase := strings.ToLower(strings.TrimSpace(track.ArtistCredit[i].JoinPhrase))
		if strings.Contains(joinPhrase, "feat") || strings.Contains(joinPhrase, "ft") {
			// 找到 feat，提取下一个艺术家作为合作者
			featArtist := track.ArtistCredit[i+1].Name
			title = fmt.Sprintf("%s(feat.%s)", title, featArtist)
		}
	}

	return title
}
