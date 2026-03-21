import SwiftUI
import NukeUI

struct PlaybackBarView: View {
    @EnvironmentObject private var store: AppStore
    @Binding var isExpanded: Bool
    var style: PlaybackBarStyle = .regular

    static let regularHeight: CGFloat = 90
    static let compactHeight: CGFloat = 64

    private var height: CGFloat {
        switch style {
        case .regular:
            return Self.regularHeight
        case .compact:
            return Self.compactHeight
        }
    }

    var body: some View {
        let nowPlaying = store.nowPlaying

        Button(action: {
            guard nowPlaying != nil else { return }
            isExpanded = true
        }) {
            HStack(spacing: style == .compact ? 12 : 16) {
                PlaybackArtworkView(artworkURL: nowPlaying?.artwork, style: style)

                VStack(alignment: .leading, spacing: style == .compact ? 4 : 8) {
                    Text(nowPlaying?.track ?? "未播放")
                        .font(style == .compact ? .subheadline.weight(.semibold) : .body.weight(.semibold))
                        .lineLimit(1)
                    Text([nowPlaying?.artist, nowPlaying?.album].compactMap { $0 }.joined(separator: " · "))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                Spacer(minLength: 0)
            }
            .padding(.horizontal, style == .compact ? 16 : 20)
            .padding(.vertical, style == .compact ? 10 : 0)
            .frame(height: height)
            .frame(maxWidth: .infinity)
            .background(.ultraThinMaterial)
            .overlay(
                Rectangle()
                    .fill(Color.black.opacity(0.08))
                    .frame(height: 1),
                alignment: .top
            )
            .shadow(
                color: Color.black.opacity(style == .compact ? 0.0 : 0.14),
                radius: style == .compact ? 0 : 18,
                x: 0,
                y: style == .compact ? 0 : -8
            )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
    }
}

enum PlaybackBarStyle {
    case regular
    case compact
}

struct PlaybackArtworkView: View {
    let artworkURL: String?
    var style: PlaybackBarStyle = .regular

    private var size: CGFloat {
        switch style {
        case .regular:
            return 48
        case .compact:
            return 42
        }
    }

    var body: some View {
        Group {
            if let artworkURL, let url = URL(string: artworkURL) {
                LazyImage(url: url) { state in
                    if let image = state.image {
                        image
                            .resizable()
                            .scaledToFill()
                    } else {
                        placeholder
                    }
                }
            } else {
                placeholder
            }
        }
        .frame(width: size, height: size)
        .clipShape(RoundedRectangle(cornerRadius: style == .compact ? 9 : 10))
    }

    private var placeholder: some View {
        RoundedRectangle(cornerRadius: style == .compact ? 9 : 10)
            .fill(Color.primary.opacity(0.08))
            .overlay(
                Image(systemName: "music.note")
                    .foregroundStyle(.secondary)
            )
    }

}

struct PlayerControlsView: View {
    let isPlaying: Bool

    var body: some View {
        EmptyView()
    }
}

struct ControlButton: View {
    let symbol: String
    var size: CGFloat = 14
    var isPrimary: Bool = false
    @State private var isHovered = false

    var body: some View {
        Button(action: {}) {
            Image(systemName: symbol)
                .font(.system(size: size, weight: .semibold))
                .frame(width: 32, height: 32)
                .background(backgroundShape)
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }

    private var backgroundShape: some View {
        RoundedRectangle(cornerRadius: 10)
            .fill(
                isPrimary
                ? AnyShapeStyle(
                    LinearGradient(
                        colors: [
                            Color.accentColor.opacity(isHovered ? 0.45 : 0.35),
                            Color.accentColor.opacity(isHovered ? 0.25 : 0.18)
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                : AnyShapeStyle(Color.primary.opacity(isHovered ? 0.08 : 0.04))
            )
    }
}

struct ProgressBarView: View {
    let progress: Double

    var body: some View {
        GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule()
                    .fill(Color.primary.opacity(0.1))
                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [Color.accentColor, Color.accentColor.opacity(0.6)],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(width: geo.size.width * max(0.02, progress))
            }
        }
        .frame(height: 6)
    }
}
