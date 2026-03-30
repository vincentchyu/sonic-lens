import SwiftUI

struct NowPlayingBarView: View {
    @Environment(PlaybackStore.self) private var playbackStore
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled
    @Binding var isExpanded: Bool

    // 封面尺寸
    private let artworkSize: CGFloat = 60

    // 整个迷你播放器最小高度
    static let height: CGFloat = 92

    var body: some View {
        let nowPlaying = playbackStore.nowPlaying

        Button(action: {
            guard nowPlaying != nil else { return }
            isExpanded = true
        }) {
            HStack(alignment: .center, spacing: 16) {
                // 左边：封面
                PlaybackArtworkView(artworkURL: nowPlaying?.artwork)
                    .frame(width: artworkSize, height: artworkSize)
                    .cornerRadius(8)

                // 右边：标题 / 副标题 / 进度条
                VStack(alignment: .leading, spacing: 6) {
                    Text(nowPlaying?.track ?? "未播放-这么看这个是废弃了、在使用PlaybackBarView.swift")
                        .font(.system(size: 14, weight: .semibold))
                        .lineLimit(1)
                        .truncationMode(.tail)

                    Text(
                        [nowPlaying?.artist, nowPlaying?.album]
                            .compactMap { $0 }
                            .joined(separator: " · ")
                    )
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
                    .truncationMode(.tail)

                    Group {
                        if let duration = nowPlaying?.duration, duration > 0 {
                            ProgressView(value: progressValue)
                                .progressViewStyle(.linear)
                        } else {
                            ProgressView(value: 0.0)
                                .progressViewStyle(.linear)
                                .opacity(0.4)
                        }
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 12)
            .frame(maxWidth: .infinity, minHeight: Self.height)
            .background {
                if performanceModeEnabled {
                    Rectangle()
                        .fill(SonicTheme.card)
                } else {
                    Rectangle()
                        .fill(.ultraThinMaterial)
                }
            }
            .overlay(
                Rectangle()
                    .fill(Color.black.opacity(0.08))
                    .frame(height: 1),
                alignment: .top
            )
            .shadow(
                color: Color.black.opacity(performanceModeEnabled ? 0.10 : 0.18),
                radius: performanceModeEnabled ? 10 : 16,
                x: 0,
                y: performanceModeEnabled ? -4 : -6
            )
        }
        .buttonStyle(.plain)
    }

    private var progressValue: Double {
        guard let nowPlaying = playbackStore.nowPlaying,
              let duration = nowPlaying.duration,
              duration > 0
        else { return 0 }

        let position = Double(nowPlaying.position ?? 0)
        return min(position / Double(duration), 1.0)
    }
}
