import SwiftUI

struct MacSharePosterShell<Content: View>: View {
    let header: ShareHeaderPayload
    let footer: ShareFooterPayload
    let renderedImage: PlatformShareImage?
    let continuationLabel: String?
    let pageText: String?
    let showsFooter: Bool
    let showsTrafficLights: Bool
    let topCaptionText: String?
    let content: Content

    init(
        header: ShareHeaderPayload,
        footer: ShareFooterPayload,
        renderedImage: PlatformShareImage? = nil,
        continuationLabel: String? = nil,
        pageText: String? = nil,
        showsFooter: Bool = true,
        showsTrafficLights: Bool = true,
        topCaptionText: String? = "SonicLens Bridge · macOS Ultra HD",
        @ViewBuilder content: () -> Content
    ) {
        self.header = header
        self.footer = footer
        self.renderedImage = renderedImage
        self.continuationLabel = continuationLabel
        self.pageText = pageText
        self.showsFooter = showsFooter
        self.showsTrafficLights = showsTrafficLights
        self.topCaptionText = topCaptionText
        self.content = content()
    }

    var body: some View {
        SharePosterBackground {
            VStack(alignment: .leading, spacing: 14) {
                // 顶部工具条：macOS 窗口三色点 + 大屏 Caption 标题
                HStack(spacing: 8) {
                    if showsTrafficLights {
                        HStack(spacing: 6) {
                            Circle().fill(Color(red: 1.0, green: 0.38, blue: 0.35)).frame(width: 10, height: 10)
                            Circle().fill(Color(red: 1.0, green: 0.76, blue: 0.22)).frame(width: 10, height: 10)
                            Circle().fill(Color(red: 0.16, green: 0.80, blue: 0.25)).frame(width: 10, height: 10)
                        }
                    }

                    Spacer()

                    if let topCaptionText {
                        Text(topCaptionText)
                            .font(.system(size: 10, weight: .semibold, design: .monospaced))
                            .foregroundStyle(Color.white.opacity(0.60))
                    }

                    Spacer()
                }
                .padding(.horizontal, 4)
                .padding(.top, 2)

                // 主体横向双栏区域 (HStack Dual-Column)
                HStack(alignment: .top, spacing: 20) {
                    // 左侧：Hero 视角与指标侧栏 (Fixed Width Left Hero Sidebar)
                    MacSharePosterHeroSidebar(
                        header: header,
                        footer: footer,
                        continuationLabel: continuationLabel,
                        renderedImage: renderedImage,
                        showsFooter: showsFooter,
                        pageText: pageText
                    )
                    .frame(width: 380)

                    // 右侧：内容主画布 (Flex Right Content Canvas)
                    VStack(alignment: .leading, spacing: 16) {
                        SharePosterCard(cornerRadius: 28) {
                            content
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                }
            }
            .padding(24)
        }
    }
}

/// macOS 专属左侧 Hero 侧边栏（包含大图封面、沉浸音轨元数据与底端水印）
struct MacSharePosterHeroSidebar: View {
    let header: ShareHeaderPayload
    let footer: ShareFooterPayload
    let continuationLabel: String?
    let renderedImage: PlatformShareImage?
    let showsFooter: Bool
    let pageText: String?

    var body: some View {
        SharePosterCard(cornerRadius: 32) {
            VStack(alignment: .leading, spacing: 18) {
                // 封面
                SharePosterArtworkView(
                    artworkURL: header.artworkURL,
                    fallbackTitle: header.artworkFallbackTitle ?? header.albumName,
                    size: 340,
                    cornerRadius: 24,
                    renderedImage: renderedImage
                )
                .shadow(color: Color.black.opacity(0.35), radius: 30, x: 0, y: 16)

                // 歌曲与专辑名称
                VStack(alignment: .leading, spacing: 6) {
                    Text(header.trackName)
                        .font(.system(size: 26, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                        .fixedSize(horizontal: false, vertical: true)
                        .lineLimit(2)

                    if let subtitle = compactSubtitle {
                        Text(subtitle)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(Color.white.opacity(0.80))
                            .fixedSize(horizontal: false, vertical: true)
                            .lineLimit(2)
                    }
                }

                // 标签与收藏状态
                VStack(alignment: .leading, spacing: 10) {
                    HStack(spacing: 8) {
                        ForEach(positionTags, id: \.self) { text in
                            SharePosterPositionBadge(text: text)
                        }

                        if header.showsFavoriteBadge {
                            SharePosterFavoriteBadge(isFavorite: header.isFavorite)
                        }
                    }

                    if !header.metricTags.isEmpty {
                        HStack(spacing: 8) {
                            ForEach(header.metricTags) { item in
                                SharePosterMetricTag(item: item)
                            }
                        }
                    }
                }

                Spacer(minLength: 12)

                // 底端品牌水印
                if showsFooter {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(footer.brandText)
                            .font(.system(size: 11, weight: .bold))
                            .foregroundStyle(Color.white.opacity(0.85))

                        Text(footer.sloganText)
                            .font(.system(size: 9, weight: .medium))
                            .foregroundStyle(Color.white.opacity(0.60))

                        if let pageText {
                            Text(pageText)
                                .font(.system(size: 9, weight: .semibold))
                                .foregroundStyle(Color.white.opacity(0.70))
                                .padding(.horizontal, 8)
                                .padding(.vertical, 3)
                                .background(Color.white.opacity(0.1), in: Capsule())
                                .padding(.top, 2)
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    private var positionTags: [String] {
        guard let positionTag = header.positionTag?.trimmingCharacters(in: .whitespacesAndNewlines),
              !positionTag.isEmpty else {
            return [header.sceneTitle]
        }
        let parts = positionTag.components(separatedBy: "·").map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }
        return parts.isEmpty ? [positionTag] : parts
    }

    private var compactSubtitle: String? {
        if let subtitleText = header.subtitleText?.trimmingCharacters(in: .whitespacesAndNewlines), !subtitleText.isEmpty {
            return subtitleText
        }
        let artist = header.artistName.trimmingCharacters(in: .whitespacesAndNewlines)
        let album = header.albumName.trimmingCharacters(in: .whitespacesAndNewlines)
        switch (!artist.isEmpty, !album.isEmpty) {
        case (true, true): return "\(artist) · \(album)"
        case (true, false): return artist
        case (false, true): return album
        default: return nil
        }
    }
}

private struct SharePosterFavoriteBadge: View {
    let isFavorite: Bool

    var body: some View {
        HStack(spacing: 5) {
            Image(systemName: isFavorite ? "heart.fill" : "heart")
                .font(.system(size: 10, weight: .semibold))
            Text(isFavorite ? "已收藏" : "未收藏")
                .font(.system(size: 10, weight: .semibold))
        }
        .foregroundStyle(isFavorite ? Color(red: 1, green: 0.50, blue: 0.62) : Color.white.opacity(0.72))
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(Color.white.opacity(0.08), in: Capsule())
        .overlay(Capsule().stroke(Color.white.opacity(0.12), lineWidth: 1))
    }
}

private struct SharePosterPositionBadge: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 10, weight: .bold))
            .foregroundStyle(.white)
            .lineLimit(1)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(Color.white.opacity(0.1), in: Capsule())
            .overlay(Capsule().stroke(Color.white.opacity(0.12), lineWidth: 1))
    }
}

private struct SharePosterMetricTag: View {
    let item: ShareMetaItem

    var body: some View {
        HStack(spacing: 5) {
            Image(systemName: item.systemImage)
                .font(.system(size: 9, weight: .semibold))
                .foregroundStyle(Color.white.opacity(0.72))

            Text(displayValue)
                .font(.system(size: 10, weight: .bold))
                .foregroundStyle(.white)
                .lineLimit(1)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(Color.white.opacity(0.06), in: Capsule())
        .overlay(Capsule().stroke(Color.white.opacity(0.10), lineWidth: 1))
    }

    private var displayValue: String {
        item.id == "play_count" ? "\(item.value)次" : item.value
    }
}
