import SwiftUI

#if os(iOS)
import UIKit
typealias PlatformShareImage = UIImage
#else
import AppKit
typealias PlatformShareImage = NSImage
#endif

struct SharePosterBackground<Content: View>: View {
    let content: Content

    init(@ViewBuilder content: () -> Content) {
        self.content = content()
    }

    var body: some View {
        ZStack {
            LinearGradient(
                colors: [
                    Color(red: 0.08, green: 0.11, blue: 0.24),
                    Color(red: 0.14, green: 0.10, blue: 0.33),
                    Color(red: 0.04, green: 0.06, blue: 0.16)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            Circle()
                .fill(Color(red: 0.37, green: 0.49, blue: 1).opacity(0.32))
                .frame(width: 260, height: 260)
                .blur(radius: 70)
                .offset(x: -120, y: -240)

            Circle()
                .fill(Color(red: 0.56, green: 0.34, blue: 1).opacity(0.32))
                .frame(width: 300, height: 300)
                .blur(radius: 80)
                .offset(x: 130, y: -150)

            Rectangle()
                .fill(
                    LinearGradient(
                        colors: [Color.white.opacity(0.05), .clear],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .ignoresSafeArea()

            content
        }
        .background(Color.black)
    }
}

struct SharePosterCard<Content: View>: View {
    let cornerRadius: CGFloat
    let content: Content

    init(cornerRadius: CGFloat = 28, @ViewBuilder content: () -> Content) {
        self.cornerRadius = cornerRadius
        self.content = content()
    }

    var body: some View {
        content
            .padding(20)
            .background(Color.white.opacity(0.08), in: RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                    .stroke(Color.white.opacity(0.12), lineWidth: 1)
            )
    }
}

struct SharePosterArtworkView: View {
    let artworkURL: String?
    let fallbackTitle: String
    let size: CGFloat
    let cornerRadius: CGFloat
    var renderedImage: PlatformShareImage? = nil

    var body: some View {
        Group {
            if let renderedImage {
                #if os(iOS)
                Image(uiImage: renderedImage)
                    .resizable()
                    .scaledToFill()
                #else
                Image(nsImage: renderedImage)
                    .resizable()
                    .scaledToFill()
                #endif
            } else {
                ArtworkSquareView(
                    artworkURL: artworkURL,
                    fallbackTitle: fallbackTitle,
                    size: size,
                    cornerRadius: cornerRadius,
                    style: .vivid
                )
            }
        }
        .frame(width: size, height: size)
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                .stroke(Color.white.opacity(0.18), lineWidth: 1)
        )
    }
}

struct SharePosterHeader: View {
    let header: ShareHeaderPayload
    let continuationLabel: String?
    var continuationPreview: String? = nil
    var renderedImage: PlatformShareImage? = nil

    private let heroMinHeight: CGFloat = 274

    var body: some View {
        if let continuationLabel {
            SharePosterCard(cornerRadius: 28) {
                HStack(alignment: .center, spacing: 14) {
                    Circle()
                        .fill(Color.white.opacity(0.14))
                        .frame(width: 40, height: 40)
                        .overlay {
                            Image(systemName: "sparkles")
                                .font(.system(size: 15, weight: .semibold))
                                .foregroundStyle(.white.opacity(0.88))
                        }

                    VStack(alignment: .leading, spacing: 5) {
                        Text(continuationLabel)
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(Color.white.opacity(0.68))
                        Text(header.trackName)
                            .font(.system(size: 18, weight: .bold, design: .rounded))
                            .foregroundStyle(.white)
                            .fixedSize(horizontal: false, vertical: true)
                            .minimumScaleFactor(0.72)
                        if let compactSubtitle {
                            Text(compactSubtitle)
                                .font(.system(size: 13, weight: .medium))
                                .foregroundStyle(Color.white.opacity(0.76))
                                .fixedSize(horizontal: false, vertical: true)
                                .minimumScaleFactor(0.75)
                        }
                        if let previewText = formattedPreviewText {
                            HStack(spacing: 4) {
                                Image(systemName: "arrow.turn.down.right")
                                    .font(.system(size: 9, weight: .semibold))
                                Text("接上: \(previewText)")
                                    .font(.system(size: 10, weight: .medium))
                                    .lineLimit(1)
                            }
                            .foregroundStyle(Color.white.opacity(0.55))
                            .padding(.top, 2)
                        }
                    }

                    Spacer(minLength: 0)
                }
            }
        } else {
            SharePosterCard(cornerRadius: 32) {
                VStack(alignment: .leading, spacing: 16) {
                    HStack(alignment: .top, spacing: 18) {
                        SharePosterArtworkView(
                            artworkURL: header.artworkURL,
                            fallbackTitle: header.artworkFallbackTitle ?? header.albumName,
                            size: 142,
                            cornerRadius: 28,
                            renderedImage: renderedImage
                        )
                        .shadow(color: Color.black.opacity(0.24), radius: 24, x: 0, y: 16)

                        VStack(alignment: .leading, spacing: 10) {
                            Text(header.trackName)
                                .font(.system(size: 28, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                                .fixedSize(horizontal: false, vertical: true)
                                .minimumScaleFactor(0.5)
                                .lineLimit(nil)

                            if let compactSubtitle {
                                Text(compactSubtitle)
                                    .font(.system(size: 15, weight: .semibold))
                                    .foregroundStyle(Color.white.opacity(0.82))
                                    .fixedSize(horizontal: false, vertical: true)
                                    .minimumScaleFactor(0.58)
                                    .lineLimit(nil)
                            }

                            Spacer(minLength: 0)
                        }
                        .frame(maxWidth: .infinity, minHeight: 142, alignment: .topLeading)
                    }

                    HStack(alignment: .bottom, spacing: 12) {
                        VStack(alignment: .leading, spacing: 10) {
                            HStack(spacing: 8) {
                                ForEach(positionTags, id: \.self) { text in
                                    SharePosterPositionBadge(text: text)
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

                        Spacer(minLength: 20)

                        if header.showsFavoriteBadge {
                            SharePosterFavoriteBadge(isFavorite: header.isFavorite)
                        }
                    }
                }
                .frame(maxWidth: .infinity, minHeight: heroMinHeight, alignment: .topLeading)
            }
        }
    }

    private var positionTags: [String] {
        guard let positionTag = header.positionTag?.trimmingCharacters(in: .whitespacesAndNewlines),
              !positionTag.isEmpty else {
            return [header.sceneTitle]
        }

        let parts = positionTag
            .components(separatedBy: "·")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        return parts.isEmpty ? [positionTag] : parts
    }

    private var compactSubtitle: String? {
        if let subtitleText = header.subtitleText?.trimmingCharacters(in: .whitespacesAndNewlines), !subtitleText.isEmpty {
            return subtitleText
        }
        let artist = header.artistName.trimmingCharacters(in: .whitespacesAndNewlines)
        let album = header.albumName.trimmingCharacters(in: .whitespacesAndNewlines)
        switch (artist.isEmpty, album.isEmpty) {
        case (false, false):
            return "\(artist) · \(album)"
        case (false, true):
            return artist
        case (true, false):
            return album
        case (true, true):
            return nil
        }
    }

    private var formattedPreviewText: String? {
        guard let text = continuationPreview?.trimmingCharacters(in: .whitespacesAndNewlines),
              !text.isEmpty else {
            return nil
        }
        let singleLine = text.replacingOccurrences(of: "\n", with: " ")
        if singleLine.count > 25 {
            return String(singleLine.prefix(25)) + "..."
        }
        return singleLine
    }
}

private struct SharePosterFavoriteBadge: View {
    let isFavorite: Bool

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: isFavorite ? "heart.fill" : "heart")
                .font(.system(size: 11, weight: .semibold))
            Text(isFavorite ? "已收藏" : "未收藏")
                .font(.system(size: 11, weight: .semibold))
        }
        .foregroundStyle(isFavorite ? Color(red: 1, green: 0.50, blue: 0.62) : Color.white.opacity(0.72))
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background(Color.white.opacity(0.08), in: Capsule())
        .overlay(
            Capsule()
                .stroke(Color.white.opacity(0.12), lineWidth: 1)
        )
    }
}

private struct SharePosterPositionBadge: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 11, weight: .bold))
            .foregroundStyle(.white)
            .lineLimit(1)
            .minimumScaleFactor(0.7)
            .frame(width: 72, height: 30)
            .background(Color.white.opacity(0.1), in: Capsule())
            .overlay(
                Capsule()
                    .stroke(Color.white.opacity(0.12), lineWidth: 1)
            )
    }
}

private struct SharePosterMetricTag: View {
    let item: ShareMetaItem

    var body: some View {
        HStack(spacing: 7) {
            Image(systemName: item.systemImage)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(Color.white.opacity(0.72))

            Text(displayValue)
                .font(.system(size: 11, weight: .bold))
                .foregroundStyle(.white)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
        }
        .frame(width: 72, height: 30)
        .background(Color.white.opacity(0.06), in: Capsule())
        .overlay(
            Capsule()
                .stroke(Color.white.opacity(0.10), lineWidth: 1)
        )
    }

    private var displayValue: String {
        if item.id == "play_count" {
            return "\(item.value)次"
        }
        return item.value
    }
}

struct SharePosterMetaGrid: View {
    let meta: ShareMetaPayload

    private let columns = [
        GridItem(.flexible(), spacing: 12),
        GridItem(.flexible(), spacing: 12)
    ]

    var body: some View {
        SharePosterCard(cornerRadius: 24) {
            VStack(alignment: .leading, spacing: 12) {
                Text("基础信息")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(Color.white.opacity(0.64))

                LazyVGrid(columns: columns, spacing: 10) {
                    ForEach(meta.items) { item in
                        HStack(alignment: .top, spacing: 10) {
                            Image(systemName: item.systemImage)
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundStyle(Color.white.opacity(0.76))
                                .frame(width: 16, height: 16)

                            VStack(alignment: .leading, spacing: 3) {
                                Text(item.title)
                                    .font(.system(size: 10, weight: .semibold))
                                    .foregroundStyle(Color.white.opacity(0.52))

                                Text(item.value)
                                    .font(.system(size: 13, weight: .semibold))
                                    .foregroundStyle(.white)
                                    .fixedSize(horizontal: false, vertical: true)
                                    .minimumScaleFactor(0.75)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 10)
                        .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                    }
                }
            }
        }
    }
}

struct SharePosterFooter: View {
    let footer: ShareFooterPayload
    let pageText: String?

    var body: some View {
        VStack(alignment: .center, spacing: 5) {
            Text(footer.brandText)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(Color.white.opacity(0.82))

            Text(footer.sloganText)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(Color.white.opacity(0.62))

            Text(footer.authorText)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(Color.white.opacity(0.68))

            if let timestampText = footer.timestampText {
                Text(timestampText)
                    .font(.system(size: 10, weight: .medium))
                    .foregroundStyle(Color.white.opacity(0.52))
            }

            if let pageText {
                Text(pageText)
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(Color.white.opacity(0.58))
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(Color.white.opacity(0.08), in: Capsule())
                    .padding(.top, 3)
            }
        }
        .frame(maxWidth: .infinity, alignment: .center)
        .multilineTextAlignment(.center)
    }
}
