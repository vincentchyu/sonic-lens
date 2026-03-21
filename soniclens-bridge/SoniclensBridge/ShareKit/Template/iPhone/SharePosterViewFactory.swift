import SwiftUI

enum SharePosterViewFactory {
    @MainActor
    static func makeView(
        payload: SharePayload,
        renderedImage: PlatformShareImage? = nil,
        continuationLabel: String? = nil,
        pageText: String? = nil,
        showsFooter: Bool = true
    ) -> AnyView {
        switch payload {
        case let .insight(insightPayload):
            return AnyView(
                InsightLongPosterView(
                    payload: insightPayload,
                    renderedImage: renderedImage,
                    continuationLabel: continuationLabel,
                    pageText: pageText,
                    showsFooter: showsFooter
                )
            )
        case let .lyrics(lyricsPayload):
            return AnyView(
                LyricsLongPosterView(
                    payload: lyricsPayload,
                    renderedImage: renderedImage,
                    continuationLabel: continuationLabel,
                    pageText: pageText,
                    showsFooter: showsFooter
                )
            )
        case let .info(infoPayload):
            return AnyView(
                TrackInfoPosterView(
                    payload: infoPayload,
                    renderedImage: renderedImage,
                    continuationLabel: continuationLabel,
                    pageText: pageText,
                    showsFooter: showsFooter
                )
            )
        case let .albumInfo(infoPayload):
            return AnyView(
                AlbumInfoPosterView(
                    payload: infoPayload,
                    renderedImage: renderedImage,
                    continuationLabel: continuationLabel,
                    pageText: pageText,
                    showsFooter: showsFooter
                )
            )
        case let .albumInsight(insightPayload):
            return AnyView(
                AlbumInsightPosterView(
                    payload: insightPayload,
                    renderedImage: renderedImage,
                    continuationLabel: continuationLabel,
                    pageText: pageText,
                    showsFooter: showsFooter
                )
            )
        }
    }
}
