import SwiftUI

struct TrackInfoPosterView: View {
    let payload: TrackInfoSharePayload
    var renderedImage: PlatformShareImage? = nil
    var continuationLabel: String? = nil
    var pageText: String? = nil
    var showsFooter: Bool = true

    var body: some View {
        SharePosterShell(
            header: payload.header,
            footer: payload.footer,
            renderedImage: renderedImage,
            continuationLabel: continuationLabel,
            pageText: pageText,
            showsFooter: showsFooter
        ) {
            InfoPosterContent(fields: payload.fields)
        }
    }
}

struct AlbumInfoPosterView: View {
    let payload: AlbumInfoSharePayload
    var renderedImage: PlatformShareImage? = nil
    var continuationLabel: String? = nil
    var pageText: String? = nil
    var showsFooter: Bool = true

    var body: some View {
        SharePosterShell(
            header: payload.header,
            footer: payload.footer,
            renderedImage: renderedImage,
            continuationLabel: continuationLabel,
            pageText: pageText,
            showsFooter: showsFooter
        ) {
            InfoPosterContent(fields: payload.fields)
        }
    }
}

struct LyricsLongPosterView: View {
    let payload: TrackLyricsSharePayload
    var renderedImage: PlatformShareImage? = nil
    var continuationLabel: String? = nil
    var pageText: String? = nil
    var showsFooter: Bool = true

    var body: some View {
        SharePosterShell(
            header: payload.header,
            footer: payload.footer,
            renderedImage: renderedImage,
            continuationLabel: continuationLabel,
            pageText: pageText,
            showsFooter: showsFooter
        ) {
            LyricsLongPosterContent(blocks: payload.blocks)
        }
    }
}

struct InsightLongPosterView: View {
    let payload: TrackInsightSharePayload
    var renderedImage: PlatformShareImage? = nil
    var continuationLabel: String? = nil
    var pageText: String? = nil
    var showsFooter: Bool = true

    var body: some View {
        SharePosterShell(
            header: payload.header,
            footer: payload.footer,
            renderedImage: renderedImage,
            continuationLabel: continuationLabel,
            pageText: pageText,
            showsFooter: showsFooter
        ) {
            InsightLongPosterContent(document: payload.document)
        }
    }
}

struct AlbumInsightPosterView: View {
    let payload: AlbumInsightSharePayload
    var renderedImage: PlatformShareImage? = nil
    var continuationLabel: String? = nil
    var pageText: String? = nil
    var showsFooter: Bool = true

    var body: some View {
        SharePosterShell(
            header: payload.header,
            footer: payload.footer,
            renderedImage: renderedImage,
            continuationLabel: continuationLabel,
            pageText: pageText,
            showsFooter: showsFooter
        ) {
            InsightLongPosterContent(document: payload.document)
        }
    }
}

private struct InfoPosterContent: View {
    let fields: [ShareInfoField]

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            VStack(alignment: .leading, spacing: 5) {
                Text("内容提要")
                    .font(.system(size: 20, weight: .bold, design: .rounded))
                    .foregroundStyle(.white)

                Text("想看更完整的解读与延伸内容，可回到 App 继续浏览。")
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(Color.white.opacity(0.56))
                    .fixedSize(horizontal: false, vertical: true)
                    .lineSpacing(3)
            }

            ForEach(fields) { field in
                VStack(alignment: .leading, spacing: 6) {
                    Text(field.title)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(Color.white.opacity(0.58))
                    Text(displayValue(for: field))
                        .font(.system(size: 15, weight: .medium))
                        .lineSpacing(6)
                        .foregroundStyle(.white.opacity(0.94))
                        .fixedSize(horizontal: false, vertical: true)

                    if let note = field.note {
                        Text("\n\(note)")
                            .font(.system(size: 8, weight: .medium))
                            .foregroundStyle(Color.white.opacity(0.5))
                            .fixedSize(horizontal: false, vertical: true)
                            .lineSpacing(3)
                    }
                }
                .padding(16)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
            }
        }
    }

    private func displayValue(for field: ShareInfoField) -> String {
        if let maxCharacterCount = field.maxCharacterCount {
            return field.value.truncatedCharacterLimit(maxCharacterCount)
        }
        return field.value
    }
}

private extension String {
    func truncatedCharacterLimit(_ limit: Int) -> String {
        guard limit > 0 else { return "" }
        if count <= limit {
            return self
        }

        let prefixText = prefix(limit)
        return String(prefixText) + "..."
    }
}

private struct LyricsLongPosterContent: View {
    let blocks: [ShareTextBlock]

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("歌词全文")
                .font(.system(size: 20, weight: .bold, design: .rounded))
                .foregroundStyle(.white)

            if blocks.isEmpty {
                SharePosterEmptyContentView(
                    title: "暂无歌词",
                    subtitle: "当前曲目还没有可分享的歌词内容。"
                )
            } else {
                ForEach(blocks) { block in
                    VStack(alignment: .leading, spacing: 8) {
                        if let title = block.title {
                            Text(title)
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(Color.white.opacity(0.56))
                        }
                        Text(block.text)
                            .font(.system(size: 18, weight: .medium))
                            .lineSpacing(8)
                            .foregroundStyle(.white.opacity(0.94))
                            .fixedSize(horizontal: false, vertical: true)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    .padding(16)
                    .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
                }
            }
        }
    }
}

private struct InsightLongPosterContent: View {
    let document: InsightShareDocument

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("音眸全文")
                .font(.system(size: 20, weight: .bold, design: .rounded))
                .foregroundStyle(.white)

            if document.isEmpty {
                SharePosterEmptyContentView(
                    title: "暂无音眸内容",
                    subtitle: "当前曲目还没有可分享的音眸解析结果。"
                )
            } else {
                ForEach(document.cards) { card in
                    SharePosterInsightCard(card: card)
                }
            }
        }
    }
}

private struct SharePosterInsightCard: View {
    let card: InsightShareCard

    var body: some View {
        switch card {
        case let .text(_, title, text):
            SharePosterInsightTextCard(title: title, text: text)
        case let .tagged(_, title, groups, text):
            SharePosterInsightTaggedCard(title: title, groups: groups, text: text)
        case let .section(_, title, sections):
            SharePosterInsightSectionCard(title: title, sections: sections)
        }
    }
}

private struct SharePosterInsightTextCard: View {
    let title: String
    let text: String

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(Color.white.opacity(0.64))

            Text(text)
                .font(.system(size: 16, weight: .medium))
                .lineSpacing(7)
                .foregroundStyle(.white.opacity(0.94))
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 24, style: .continuous))
    }
}

private struct SharePosterInsightTaggedCard: View {
    let title: String
    let groups: [InsightShareGroup]
    let text: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(Color.white.opacity(0.64))

            if let text {
                Text(text)
                    .font(.system(size: 15, weight: .medium))
                    .lineSpacing(6)
                    .foregroundStyle(.white.opacity(0.9))
                    .fixedSize(horizontal: false, vertical: true)
            }

            VStack(alignment: .leading, spacing: 14) {
                ForEach(groups) { group in
                    SharePosterInsightGroupCard(group: group, indexLabel: nil, showExplainPanel: false)
                }
            }
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 24, style: .continuous))
    }
}

private struct SharePosterInsightSectionCard: View {
    let title: String
    let sections: [InsightShareSection]

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(title)
                .font(.system(size: 16, weight: .bold, design: .rounded))
                .foregroundStyle(.white)

            ForEach(sections) { section in
                VStack(alignment: .leading, spacing: 12) {
                    Text(section.title)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(Color.white.opacity(0.64))

                    if let text = section.text {
                        if section.id != "appreciate_analysis" {
                            Text(text)
                                .font(.system(size: 15, weight: .medium))
                                .lineSpacing(6)
                                .foregroundStyle(.white.opacity(0.9))
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }

                    if !section.groups.isEmpty {
                        VStack(alignment: .leading, spacing: 12) {
                            ForEach(section.groups) { group in
                                SharePosterInsightGroupCard(
                                    group: group,
                                    indexLabel: nil,
                                    showExplainPanel: true
                                )
                            }
                        }
                    }
                }
                .padding(16)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
            }
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 24, style: .continuous))
    }
}

private struct SharePosterInsightGroupCard: View {
    let group: InsightShareGroup
    let indexLabel: String?
    let showExplainPanel: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            if let indexLabel {
                Text(indexLabel)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(Color.white.opacity(0.54))
            }

            ForEach(group.rows) { row in
                VStack(alignment: .leading, spacing: 6) {
                    if !row.original.isEmpty {
                        Text(row.original)
                            .font(.system(size: 14, weight: .medium))
                            .foregroundStyle(Color.white.opacity(0.58))
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    if let translation = row.translation, !translation.isEmpty {
                        Text(translation)
                            .font(.system(size: 16, weight: .medium))
                            .italic()
                            .foregroundStyle(Color.white.opacity(0.72))
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }

            if let explain = group.explain {
                Text(explain)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(.white.opacity(0.94))
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(14)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.white.opacity(showExplainPanel ? 0.08 : 0.04), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(showExplainPanel ? 16 : 0)
        .background(
            (showExplainPanel ? Color.white.opacity(0.04) : Color.clear),
            in: RoundedRectangle(cornerRadius: 20, style: .continuous)
        )
    }
}

private struct SharePosterEmptyContentView: View {
    let title: String
    let subtitle: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(.white)
            Text(subtitle)
                .font(.system(size: 14, weight: .medium))
                .foregroundStyle(Color.white.opacity(0.68))
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
    }
}
