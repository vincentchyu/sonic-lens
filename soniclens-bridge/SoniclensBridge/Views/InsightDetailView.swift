import SwiftUI

struct InsightDetailView: View {
    let insight: Insight

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                InsightDetailHeader(insight: insight)
                InsightPrimaryContentView(
                    insight: insight,
                    style: .detail,
                    emptyTitle: "暂无音眸",
                    emptySubtitle: "当前曲目还没有可展示的音眸内容。"
                )
            }
            .padding(32)
        }
        .navigationTitle("音眸")
    }
}

struct AlbumInsightDetailView: View {
    let insight: AlbumInsight

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                AlbumInsightDetailHeader(insight: insight)
                AlbumInsightPrimaryContentView(
                    insight: insight,
                    compact: false,
                    emptyTitle: "暂无专辑音眸",
                    emptySubtitle: "当前专辑还没有可展示的音眸内容。"
                )
            }
            .padding(32)
        }
        .navigationTitle("音眸")
    }
}

struct InsightDetailHeader: View {
    let insight: Insight

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(insight.track)
                .font(.title2)
                .fontWeight(.semibold)
            Text("\(insight.artist) · \(insight.album)")
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassCard(cornerRadius: 16)
    }
}

struct AlbumInsightDetailHeader: View {
    let insight: AlbumInsight

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(insight.album)
                .font(.title2)
                .fontWeight(.semibold)
            Text(insight.artist)
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassCard(cornerRadius: 16)
    }
}

struct InsightRenderStyle {
    let sectionSpacing: CGFloat
    let blockSpacing: CGFloat
    let blockPadding: CGFloat
    let innerSpacing: CGFloat
    let rowSpacing: CGFloat
    let stacksTaggedRows: Bool
    let wrapBlocksInCards: Bool
    let showProviderLine: Bool
    let simplifiedCards: Bool
    let translationUsesItalic: Bool
    let providerFont: Font
    let blockTitleFont: Font
    let sectionTitleFont: Font
    let bodyFont: Font
    let originalFont: Font
    let translationFont: Font
    let explainFont: Font
    let textColor: Color
    let secondaryTextColor: Color
    let originalColor: Color
    let translationColor: Color
    let explainColor: Color
    let explainBackground: Color
    let plainBlockBackground: Color
    let plainBlockBorder: Color

    static let detail = InsightRenderStyle(
        sectionSpacing: 16,
        blockSpacing: 14,
        blockPadding: 16,
        innerSpacing: 12,
        rowSpacing: 10,
        stacksTaggedRows: false,
        wrapBlocksInCards: true,
        showProviderLine: true,
        simplifiedCards: true,
        translationUsesItalic: false,
        providerFont: .caption,
        blockTitleFont: .headline,
        sectionTitleFont: .subheadline.weight(.semibold),
        bodyFont: .body,
        originalFont: .body.weight(.medium),
        translationFont: .callout,
        explainFont: .callout.weight(.medium),
        textColor: SonicTheme.textPrimary,
        secondaryTextColor: SonicTheme.textSecondary,
        originalColor: SonicTheme.textPrimary,
        translationColor: SonicTheme.textSecondary,
        explainColor: SonicTheme.textSecondary,
        explainBackground: SonicTheme.primary.opacity(0.08),
        plainBlockBackground: SonicTheme.card,
        plainBlockBorder: SonicTheme.glassBorder
    )

    static let immersive = InsightRenderStyle(
        sectionSpacing: 18,
        blockSpacing: 18,
        blockPadding: 18,
        innerSpacing: 14,
        rowSpacing: 12,
        stacksTaggedRows: false,
        wrapBlocksInCards: false,
        showProviderLine: true,
        simplifiedCards: true,
        translationUsesItalic: false,
        providerFont: .system(size: 13, weight: .medium),
        blockTitleFont: .system(size: 20, weight: .bold, design: .rounded),
        sectionTitleFont: .system(size: 16, weight: .semibold),
        bodyFont: .system(size: 19, weight: .medium),
        originalFont: .system(size: 17, weight: .medium),
        translationFont: .system(size: 17, weight: .medium),
        explainFont: .system(size: 21, weight: .bold),
        textColor: .white.opacity(0.95),
        secondaryTextColor: .white.opacity(0.8),
        originalColor: .white.opacity(0.8),
        translationColor: .white.opacity(0.62),
        explainColor: .white.opacity(0.95),
        explainBackground: Color.white.opacity(0.045),
        plainBlockBackground: .clear,
        plainBlockBorder: .clear
    )

    static let phoneCompact = InsightRenderStyle(
        sectionSpacing: 14,
        blockSpacing: 12,
        blockPadding: 14,
        innerSpacing: 10,
        rowSpacing: 8,
        stacksTaggedRows: true,
        wrapBlocksInCards: false,
        showProviderLine: true,
        simplifiedCards: true,
        translationUsesItalic: true,
        providerFont: .caption,
        blockTitleFont: .system(size: 18, weight: .bold, design: .rounded),
        sectionTitleFont: .system(size: 13, weight: .semibold),
        bodyFont: .system(size: 16, weight: .medium),
        originalFont: .system(size: 15, weight: .medium),
        translationFont: .system(size: 14, weight: .medium),
        explainFont: .system(size: 17, weight: .semibold),
        textColor: .white,
        secondaryTextColor: Color.white.opacity(0.78),
        originalColor: Color.white.opacity(0.8),
        translationColor: Color.white.opacity(0.58),
        explainColor: Color.white.opacity(0.95),
        explainBackground: Color.white.opacity(0.05),
        plainBlockBackground: .clear,
        plainBlockBorder: .clear
    )
}

struct InsightPrimaryContentView: View {
    let insight: Insight?
    var style: InsightRenderStyle = .detail
    var emptyTitle: String = "暂无音眸"
    var emptySubtitle: String = "当前曲目还没有生成洞察内容。"

    var body: some View {
        Group {
            if let insight, insight.hasDisplayContent {
                InsightRichContentView(insight: insight, style: style)
            } else {
                InsightEmptyStateCard(title: emptyTitle, subtitle: emptySubtitle, style: style)
            }
        }
    }
}

struct InsightRichContentView: View {
    let insight: Insight
    let style: InsightRenderStyle

    var body: some View {
        VStack(alignment: .leading, spacing: style.blockSpacing) {
            if style.showProviderLine, let providerLine = insight.providerLine {
                Text(providerLine)
                    .font(style.providerFont)
                    .foregroundStyle(style.secondaryTextColor)
            }

            ForEach(insight.displayBlocks) { block in
                InsightDisplayBlockView(block: block, style: style)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct InsightDisplayBlockView: View {
    let block: InsightDisplayBlock
    let style: InsightRenderStyle

    var body: some View {
        let content = VStack(alignment: .leading, spacing: style.innerSpacing) {
            Text(blockTitle)
                .font(style.blockTitleFont)
                .foregroundStyle(style.textColor)

            switch block {
            case let .text(_, _, text):
                InsightTaggedContentView(text: text, style: style)
            case let .sections(_, _, sections):
                VStack(alignment: .leading, spacing: style.sectionSpacing) {
                    ForEach(sections) { section in
                        VStack(alignment: .leading, spacing: 10) {
                            Text("【\(section.title)】")
                                .font(style.sectionTitleFont)
                                .foregroundStyle(style.secondaryTextColor)
                            InsightTaggedContentView(text: section.content, style: style)
                        }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)

        if style.wrapBlocksInCards {
            content
                .padding(style.blockPadding)
                .glassCard(cornerRadius: 14, isSimplified: style.simplifiedCards)
        } else {
            content
                .padding(style.blockPadding)
                .background(style.plainBlockBackground, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .stroke(style.plainBlockBorder, lineWidth: 1)
                )
        }
    }

    private var blockTitle: String {
        switch block {
        case let .text(_, title, _):
            return title
        case let .sections(_, title, _):
            return title
        }
    }
}

private struct InsightTaggedContentView: View {
    let text: String
    let style: InsightRenderStyle

    var body: some View {
        if let segments = InsightTaggedContentParser.parse(text) {
            VStack(alignment: .leading, spacing: style.innerSpacing) {
                ForEach(Array(segments.enumerated()), id: \.offset) { item in
                    switch item.element {
                    case let .text(text):
                        InsightBodyText(text: text, color: style.textColor, font: style.bodyFont)
                    case let .rows(rows):
                        InsightTaggedRowsView(rows: rows, style: style)
                    case let .explain(text):
                        InsightExplainText(text: text, style: style)
                    }
                }
            }
        } else {
            InsightBodyText(text: text, color: style.textColor, font: style.bodyFont)
        }
    }
}

private struct InsightTaggedRowsView: View {
    let rows: [InsightTaggedRow]
    let style: InsightRenderStyle

    var body: some View {
        VStack(alignment: .leading, spacing: style.rowSpacing) {
            ForEach(Array(rows.enumerated()), id: \.offset) { item in
                let row = item.element
                if style.stacksTaggedRows {
                    VStack(alignment: .leading, spacing: 6) {
                        if !row.original.isEmpty {
                            Text(row.original)
                                .font(style.originalFont)
                                .foregroundStyle(style.originalColor)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        if !row.translation.isEmpty {
                            InsightTranslationText(text: row.translation, style: style)
                        }
                    }
                    .padding(.vertical, 2)
                } else {
                    HStack(alignment: .top, spacing: 16) {
                        Text(row.original.isEmpty ? " " : row.original)
                            .font(style.originalFont)
                            .foregroundStyle(style.originalColor)
                            .frame(maxWidth: .infinity, alignment: .leading)

                        if !row.translation.isEmpty {
                            InsightTranslationText(text: row.translation, style: style)
                        }
                    }
                    .padding(.vertical, 2)
                }
            }
        }
    }
}

private struct InsightTranslationText: View {
    let text: String
    let style: InsightRenderStyle

    var body: some View {
        let translation = Text(text).font(style.translationFont)
        Group {
            if style.translationUsesItalic {
                translation.italic()
            } else {
                translation
            }
        }
        .foregroundStyle(style.translationColor)
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct InsightBodyText: View {
    let text: String
    let color: Color
    let font: Font

    var body: some View {
        Text(text.replacingOccurrences(of: "\\n", with: "\n"))
            .font(font)
            .foregroundStyle(color)
            .fixedSize(horizontal: false, vertical: true)
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct InsightExplainText: View {
    let text: String
    let style: InsightRenderStyle

    var body: some View {
        InsightBodyText(text: text, color: style.explainColor, font: style.explainFont)
            .padding(12)
            .background(style.explainBackground, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}

private struct InsightEmptyStateCard: View {
    let title: String
    let subtitle: String
    let style: InsightRenderStyle

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
                .foregroundStyle(style.textColor)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(style.secondaryTextColor)
        }
        .padding(style.blockPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(style.plainBlockBackground, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(style.plainBlockBorder, lineWidth: 1)
        )
    }
}
