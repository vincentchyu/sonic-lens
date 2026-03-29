import SwiftUI
import NukeUI

struct PlaybackBarView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled
    @Binding var isExpanded: Bool
    var style: PlaybackBarStyle = .regular
    @StateObject private var progressModel = PlaybackBarProgressModel()

    static let regularHeight: CGFloat = 82
    static let compactHeight: CGFloat = 72

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
            VStack(spacing: style == .compact ? 7 : 8) {
                HStack(spacing: style == .compact ? 10 : 12) {
                    PlaybackArtworkView(
                        artworkURL: nowPlaying?.artwork,
                        fallbackTitle: nowPlaying?.album ?? nowPlaying?.track,
                        style: style
                    )

                    VStack(alignment: .leading, spacing: 2) {
                        Text(nowPlaying?.track ?? "未播放")
                            .font(style == .compact ? .subheadline.weight(.semibold) : .body.weight(.semibold))
                            .foregroundStyle(.primary)
                            .lineLimit(1)

                        Text([nowPlaying?.artist, nowPlaying?.album].compactMap { $0 }.joined(separator: " · "))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                HStack(spacing: 10) {
                    Text(currentTimeText)
                        .font(.caption2.monospacedDigit())
                        .foregroundStyle(.secondary)
                        .frame(width: 36, alignment: .leading)

                    ProgressBarView(progress: progressValue)
                        .frame(maxWidth: .infinity)

                    Text(durationText)
                        .font(.caption2.monospacedDigit())
                        .foregroundStyle(.secondary)
                        .frame(width: 36, alignment: .trailing)
                }
            }
            .padding(.horizontal, style == .compact ? 14 : 18)
            .padding(.vertical, style == .compact ? 10 : 11)
            .frame(height: height)
            .frame(maxWidth: .infinity)
            .background(
                RoundedRectangle(cornerRadius: style == .compact ? 16 : 18, style: .continuous)
                    .fill(SonicTheme.card.opacity(performanceModeEnabled ? 0.96 : 0.90))
                //.fill(.regularMaterial)
            )
            .overlay(
                RoundedRectangle(cornerRadius: style == .compact ? 16 : 18, style: .continuous)
                    .stroke(SonicTheme.glassBorder.opacity(0.75), lineWidth: 1)
                    //.fill(Color.black.opacity(performanceModeEnabled ? 0.08 : 0.14))
            )

            .clipShape(RoundedRectangle(cornerRadius: style == .compact ? 16 : 18, style: .continuous))
            .shadow(
                color: Color.black.opacity(performanceModeEnabled ? 0.03 : 0.06),
                radius: performanceModeEnabled ? 6 : 10,
                x: 0,
                y: performanceModeEnabled ? 2 : 4
            )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .padding(.horizontal, style == .compact ? 12 : 16)
        .padding(.bottom, style == .compact ? 8 : 10)
        .onAppear {
            syncPlaybackProgress()
        }
        .onChange(of: playbackIdentity) { _, _ in
            syncPlaybackProgress(forceRestart: true)
        }
        .onChange(of: store.nowPlaying?.position) { _, _ in
            syncPlaybackProgress()
        }
        .onChange(of: store.nowPlaying?.positionMs) { _, _ in
            syncPlaybackProgress()
        }
        .onDisappear {
            progressModel.stop()
        }
    }

    private var progressValue: Double {
        guard let duration = store.nowPlaying?.duration,
              duration > 0 else {
            return 0
        }
        return min(max(progressModel.currentTime / Double(duration), 0), 1)
    }

    private var currentTimeText: String {
        formatTime(progressModel.currentTime)
    }

    private var durationText: String {
        formatTime(Double(store.nowPlaying?.duration ?? 0))
    }

    private func formatTime(_ seconds: Double) -> String {
        let totalSeconds = max(Int(seconds.rounded(.down)), 0)
        let minutes = totalSeconds / 60
        let remainingSeconds = totalSeconds % 60
        return String(format: "%d:%02d", minutes, remainingSeconds)
    }

    private var playbackIdentity: String {
        let current = store.nowPlaying
        return [
            current?.artist ?? "",
            current?.album ?? "",
            current?.track ?? "",
            String(current?.duration ?? 0)
        ].joined(separator: "•")
    }

    private func syncPlaybackProgress(forceRestart: Bool = false) {
        guard let nowPlaying = store.nowPlaying else {
            progressModel.stop()
            return
        }

        if forceRestart {
            progressModel.start(position: nowPlaying.position, positionMs: nowPlaying.positionMs)
        } else {
            progressModel.sync(position: nowPlaying.position, positionMs: nowPlaying.positionMs)
        }
    }
}

enum PlaybackBarStyle {
    case regular
    case compact
}

struct PlaybackArtworkView: View {
    let artworkURL: String?
    var fallbackTitle: String? = nil
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
        ArtworkSquareView(
            artworkURL: artworkURL,
            fallbackTitle: fallbackTitle,
            size: size,
            cornerRadius: style == .compact ? 9 : 10,
            style: .subtle
        )
    }
}

struct ArtworkSquareView: View {
    enum SurfaceStyle {
        case subtle
        case vivid
    }

    let artworkURL: String?
    let fallbackTitle: String?
    let size: CGFloat
    let cornerRadius: CGFloat
    var style: SurfaceStyle = .vivid

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
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
    }

    private var placeholder: some View {
        RoundedRectangle(cornerRadius: cornerRadius)
            .fill(placeholderGradient)
            .overlay {
                if let initial = fallbackTitle?.sonicArtworkInitial {
                    Text(initial)
                        .font(.system(size: max(size * 0.32, 14), weight: .bold, design: .rounded))
                        .foregroundStyle(.white.opacity(0.82))
                } else {
                    Image(systemName: "music.note")
                        .font(.system(size: max(size * 0.34, 14), weight: .semibold))
                        .foregroundStyle(.white.opacity(0.68))
                }
            }
    }

    private var placeholderGradient: LinearGradient {
        switch style {
        case .subtle:
            return LinearGradient(
                colors: [Color.primary.opacity(0.15), Color.primary.opacity(0.06)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        case .vivid:
            return LinearGradient(
                colors: [Color.accentColor.opacity(0.55), Color.accentColor.opacity(0.18)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        }
    }
}

private extension String {
    var sonicArtworkInitial: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).first.map { String($0).uppercased() }
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
                    .fill(SonicTheme.progressTrack)
                Capsule()
                    .fill(SonicTheme.progressFill)
                    .frame(width: geo.size.width * max(0.02, progress))
            }
        }
        .frame(height: 6)
    }
}

@MainActor
private final class PlaybackBarProgressModel: ObservableObject {
    @Published var currentTime: TimeInterval = 0

    private var timer: Timer?
    private var anchorDate: Date?
    private var anchorTime: TimeInterval = 0

    func start(position: Int?, positionMs: Int?) {
        stop()
        let startTime = resolvedTime(position: position, positionMs: positionMs)
        anchorTime = startTime
        anchorDate = Date()
        currentTime = startTime

        timer = Timer.scheduledTimer(withTimeInterval: 0.35, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.tick()
            }
        }
        timer?.tolerance = 0.12
    }

    func stop() {
        timer?.invalidate()
        timer = nil
        anchorDate = nil
        currentTime = 0
    }

    func sync(position: Int?, positionMs: Int?) {
        if timer == nil {
            start(position: position, positionMs: positionMs)
            return
        }

        let incoming = resolvedTime(position: position, positionMs: positionMs)
        if abs(incoming - currentTime) > 0.35 {
            anchorTime = incoming
            anchorDate = Date()
            currentTime = incoming
        }
    }

    private func tick() {
        guard let anchorDate else { return }
        currentTime = anchorTime + Date().timeIntervalSince(anchorDate)
    }

    private func resolvedTime(position: Int?, positionMs: Int?) -> TimeInterval {
        if let positionMs {
            return TimeInterval(positionMs) / 1000
        }
        return TimeInterval(position ?? 0)
    }
}
