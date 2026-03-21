import Foundation

enum InsightShareParser {
    static func parse(_ insight: Insight?) -> InsightShareDocument {
        guard let insight else {
            return InsightShareDocument(cards: [])
        }

        var cards: [InsightShareCard] = []

        if let lyricsTranslation = trimmed(insight.lyricsTranslation) {
            let result = parseTaggedText(lyricsTranslation)
            if !result.groups.isEmpty {
                cards.append(
                    .tagged(
                        id: "lyrics_translation",
                        title: "歌词对照",
                        groups: result.groups,
                        text: result.plainText
                    )
                )
            } else {
                cards.append(.text(id: "lyrics_translation", title: "歌词对照", text: lyricsTranslation))
            }
        }

        if let summary = trimmed(insight.analysisSummary) {
            cards.append(.text(id: "analysis_summary", title: "曲目解读", text: summary))
        }

        let sections = makeSectionCards(from: insight.analysisBySection.orderedBlocks)
        if !sections.isEmpty {
            cards.append(.section(id: "analysis_by_section", title: "分段解析", sections: sections))
        }

        if let background = trimmed(insight.backgroundInfo) {
            cards.append(.text(id: "background_info", title: "背景信息", text: background))
        }

        if let era = trimmed(insight.eraContext) {
            cards.append(.text(id: "era_context", title: "时代语境", text: era))
        }

        return InsightShareDocument(cards: cards)
    }

    static func parseAlbum(_ insight: AlbumInsight?) -> InsightShareDocument {
        guard let insight else {
            return InsightShareDocument(cards: [])
        }

        var cards: [InsightShareCard] = []

        if let summary = trimmed(insight.analysisSummary) {
            cards.append(.text(id: "analysis_summary", title: "专辑总评", text: summary))
        }

        let sections = makeSectionCards(from: insight.orderedSections)
        if !sections.isEmpty {
            cards.append(.section(id: "analysis_by_section", title: "专辑分析", sections: sections))
        }

        if let background = trimmed(insight.backgroundInfo) {
            cards.append(.text(id: "background_info", title: "背景信息", text: background))
        }

        if let era = trimmed(insight.eraContext) {
            cards.append(.text(id: "era_context", title: "时代语境", text: era))
        }

        return InsightShareDocument(cards: cards)
    }

    private static func makeSectionCards(from blocks: [InsightSectionBlock]) -> [InsightShareSection] {
        blocks.compactMap { block in
            let result = parseTaggedText(block.content)
            let plainText = trimmed(result.plainText)
            guard plainText != nil || !result.groups.isEmpty else { return nil }
            return InsightShareSection(
                id: block.id,
                title: block.title,
                text: plainText,
                groups: result.groups
            )
        }
    }

    private static func parseTaggedText(_ text: String) -> (groups: [InsightShareGroup], plainText: String?) {
        guard let taggedSegments = InsightTaggedContentParser.parse(text) else {
            return ([], trimmed(text))
        }

        var groups: [InsightShareGroup] = []
        var pendingRows: [InsightShareRow] = []
        var plainBlocks: [String] = []

        func flushRows(explain: String? = nil) {
            guard !pendingRows.isEmpty else { return }
            groups.append(InsightShareGroup(rows: pendingRows, explain: explain))
            pendingRows.removeAll(keepingCapacity: true)
        }

        for segment in taggedSegments {
            switch segment {
            case let .rows(rows):
                pendingRows.append(contentsOf: rows.compactMap { row in
                    let original = trimmed(row.original)
                    let translation = trimmed(row.translation)
                    guard original != nil || translation != nil else { return nil }
                    return InsightShareRow(original: original ?? "", translation: translation)
                })
            case let .explain(text):
                if let explain = trimmed(text) {
                    if pendingRows.isEmpty {
                        plainBlocks.append(explain)
                    } else {
                        flushRows(explain: explain)
                    }
                }
            case let .text(text):
                flushRows()
                if let text = trimmed(text) {
                    plainBlocks.append(text)
                }
            }
        }

        flushRows()
        return (groups, plainBlocks.joined(separator: "\n\n"))
    }

    private static func trimmed(_ text: String?) -> String? {
        ShareMetadataFormatter.trimmed(text)
    }
}
