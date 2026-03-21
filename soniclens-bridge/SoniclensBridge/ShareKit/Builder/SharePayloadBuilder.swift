import Foundation

enum SharePayloadBuilder {
    static func build(
        scene: ShareScene,
        track: Track,
        resolvedArtworkURL: String?,
        lyrics: TrackLyricsResponse?,
        insight: Insight?,
        isFavorite: Bool
    ) -> SharePayload {
        let metaItems = ShareMetadataFormatter.makeMetaItems(track: track, isFavorite: isFavorite)

        switch scene {
        case .trackInsight:
            return .insight(
                TrackInsightSharePayload(
                    header: makeTrackHeader(track: track, artworkURL: resolvedArtworkURL, scene: scene, isFavorite: isFavorite, metaItems: metaItems),
                    meta: ShareMetaPayload(items: metaItems),
                    document: InsightShareParser.parse(insight),
                    footer: ShareMetadataFormatter.makeFooter()
                )
            )
        case .trackLyrics:
            return .lyrics(
                TrackLyricsSharePayload(
                    header: makeTrackHeader(track: track, artworkURL: resolvedArtworkURL, scene: scene, isFavorite: isFavorite, metaItems: metaItems),
                    meta: ShareMetaPayload(items: metaItems),
                    blocks: ShareMetadataFormatter.makeLyricsBlocks(from: lyrics),
                    footer: ShareMetadataFormatter.makeFooter()
                )
            )
        case .trackInfo:
            return .info(
                TrackInfoSharePayload(
                    header: makeTrackHeader(track: track, artworkURL: resolvedArtworkURL, scene: scene, isFavorite: isFavorite, metaItems: metaItems),
                    meta: ShareMetaPayload(items: metaItems),
                    fields: makeInfoFields(insight: insight),
                    footer: ShareMetadataFormatter.makeFooter()
                )
            )
        case .albumInfo, .albumInsight:
            fatalError("Use buildAlbum(scene:albumDetail:albumInsight:resolvedArtworkURL:) for album share scenes.")
        }
    }

    static func buildAlbum(
        scene: ShareScene,
        albumDetail: AlbumDetail,
        albumInsight: AlbumInsight?,
        resolvedArtworkURL: String?
    ) -> SharePayload {
        let metaItems = ShareMetadataFormatter.makeAlbumMetaItems(
            albumDetail: albumDetail,
            totalDuration: albumDetail.tracks.compactMap(\.duration).reduce(0, +)
        )
        let header = makeAlbumHeader(
            albumDetail: albumDetail,
            artworkURL: resolvedArtworkURL,
            scene: scene,
            metaItems: metaItems
        )

        switch scene {
        case .albumInfo:
            return .albumInfo(
                AlbumInfoSharePayload(
                    header: header,
                    meta: ShareMetaPayload(items: metaItems),
                    fields: makeAlbumInfoFields(insight: albumInsight),
                    footer: ShareMetadataFormatter.makeFooter()
                )
            )
        case .albumInsight:
            return .albumInsight(
                AlbumInsightSharePayload(
                    header: header,
                    meta: ShareMetaPayload(items: metaItems),
                    document: InsightShareParser.parseAlbum(albumInsight),
                    footer: ShareMetadataFormatter.makeFooter()
                )
            )
        case .trackInsight, .trackInfo, .trackLyrics:
            fatalError("Use build(scene:track:resolvedArtworkURL:lyrics:insight:isFavorite:) for track share scenes.")
        }
    }

    private static func makeTrackHeader(
        track: Track,
        artworkURL: String?,
        scene: ShareScene,
        isFavorite: Bool,
        metaItems: [ShareMetaItem]
    ) -> ShareHeaderPayload {
        ShareHeaderPayload(
            artworkURL: artworkURL,
            trackName: track.track,
            artistName: track.artist,
            albumName: track.album,
            sceneTitle: scene.title,
            positionTag: ShareMetadataFormatter.formatDiscTrack(trackNumber: track.trackNumber, discNumber: track.discNumber),
            isFavorite: isFavorite,
            showsFavoriteBadge: true,
            subtitleText: nil,
            artworkFallbackTitle: nil,
            metricTags: metaItems
        )
    }

    private static func makeAlbumHeader(
        albumDetail: AlbumDetail,
        artworkURL: String?,
        scene: ShareScene,
        metaItems: [ShareMetaItem]
    ) -> ShareHeaderPayload {
        ShareHeaderPayload(
            artworkURL: artworkURL,
            trackName: albumDetail.name,
            artistName: albumDetail.artist,
            albumName: "",
            sceneTitle: scene.title,
            positionTag: nil,
            isFavorite: false,
            showsFavoriteBadge: false,
            subtitleText: albumDetail.artist,
            artworkFallbackTitle: albumDetail.name,
            metricTags: metaItems
        )
    }

    private static func makeInfoFields(insight: Insight?) -> [ShareInfoField] {
        let summary = trimmed(insight?.analysisSummary) ?? "音眸还没有生成这首歌的整体解读，可以先分享基础信息，稍后再回来看看。"
        let literaryAnalysis = trimmed(insight?.analysisBySection.values["literary_analysis"])
            ?? "暂时还没有提炼出文学解读，等音眸补全后，这里会展示更偏文本与意象层面的分析。"

        return [
            ShareInfoField(id: "analysis_summary", title: "曲目解读", value: summary, note: nil, maxCharacterCount: nil),
            ShareInfoField(
                id: "literary_analysis",
                title: "文学解读",
                value: literaryAnalysis,
                note: "更多意象、隐喻与深度解读，可回到 App 查看。",
                maxCharacterCount: 200
            )
        ]
    }

    private static func makeAlbumInfoFields(insight: AlbumInsight?) -> [ShareInfoField] {
        let summary = trimmed(insight?.analysisSummary) ?? "音眸还没有生成这张专辑的整体总评，可以先分享专辑基础信息，稍后再回来查看完整分析。"
        let sections = insight?.analysisBySection.values ?? [:]

        return [
            ShareInfoField(id: "analysis_summary", title: "专辑总评", value: summary, note: nil, maxCharacterCount: nil),
            ShareInfoField(
                id: "key_tracks",
                title: "关键曲目",
                value: trimmed(sections["key_tracks"]) ?? "音眸尚未提炼关键曲目与它们在专辑中的作用。",
                note: "完整关键曲目解析，请回到 App 查看。",
                maxCharacterCount: 120
            ),
            ShareInfoField(
                id: "philosophical_reflection",
                title: "哲学反思",
                value: trimmed(sections["philosophical_reflection"]) ?? "音眸尚未提炼这张专辑折射出的价值观与存在主题。",
                note: "完整哲学反思，请回到 App 查看。",
                maxCharacterCount: 120
            ),
            ShareInfoField(
                id: "listening_guide",
                title: "聆听指南",
                value: trimmed(sections["listening_guide"]) ?? "音眸尚未给出这张专辑的完整收听切入角度与建议。",
                note: "更多聆听建议，请回到 App 查看。",
                maxCharacterCount: 120
            )
        ]
    }

    private static func trimmed(_ text: String?) -> String? {
        guard let text else { return nil }
        let value = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
    }
}
