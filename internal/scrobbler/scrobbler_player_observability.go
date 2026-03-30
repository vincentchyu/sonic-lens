package scrobbler

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

func startPlayerControllerSpan(
	ctx context.Context,
	source common.PlayerType,
	operation string,
) (context.Context, trace.Span) {
	spanCtx, span := telemetry.StartSpanForTracerName(
		ctx,
		_TracerName,
		"player."+operation,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("player.source", string(source)),
		attribute.String("player.operation", operation),
	)
	return spanCtx, span
}

func markPlayerSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func setTrackPlaybackSpanAttributes(
	span trace.Span,
	source common.PlayerType,
	snapshot playingTrackSnapshot,
) {
	span.SetAttributes(
		attribute.String("player.source", string(source)),
		attribute.String("track.title", snapshot.playerInfo.GetTitle()),
		attribute.String("track.artist", snapshot.playerInfo.GetArtist()),
		attribute.String("track.album", snapshot.playerInfo.GetAlbum()),
		attribute.String("track.album_artist", snapshot.playerInfo.GetAlbumArtist()),
		attribute.Int64("track.track_number", snapshot.playerInfo.GetTrackNumber()),
		attribute.Int("track.disc_number", int(snapshot.playerInfo.GetDiscNumber())),
		attribute.Int64("track.duration_sec", snapshot.duration),
		attribute.String("track.metadata_confidence", trackMetadataConfidenceLabel(snapshot.metadata.Confidence)),
	)
}

func startPlayerTrackStageSpan(
	ctx context.Context,
	source common.PlayerType,
	stage string,
) (context.Context, trace.Span) {
	spanCtx, span := telemetry.StartSpanForTracerName(
		ctx,
		_TracerName,
		"player."+stage,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("player.source", string(source)),
		attribute.String("player.stage", stage),
	)
	return spanCtx, span
}

func addTrackPlaybackEvent(span trace.Span, eventName string, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}

func setFavoriteStateSpanAttributes(
	span trace.Span,
	favoriteState common.TrackFavoriteState,
	appleMusic bool,
	lastFM bool,
) {
	if span == nil {
		return
	}
	span.SetAttributes(
		attribute.String("track.favorite_state", string(favoriteState)),
		attribute.Bool("track.favorite.apple_music", appleMusic),
		attribute.Bool("track.favorite.lastfm", lastFM),
	)
}

func trackMetadataConfidenceLabel(confidence common.TrackMetadataConfidence) string {
	switch confidence {
	case common.TrackMetadataConfidenceLow:
		return "low"
	case common.TrackMetadataConfidenceMedium:
		return "medium"
	case common.TrackMetadataConfidenceHigh:
		return "high"
	case common.TrackMetadataConfidenceAuthoritative:
		return "authoritative"
	default:
		return "unknown"
	}
}
