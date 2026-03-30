#if os(iOS)
import ActivityKit
import OSLog
import SwiftUI
import UIKit
import WidgetKit

@available(iOS 16.2, *)
struct InsightJobLiveActivityWidget: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: InsightLiveActivityAttributes.self) { context in
            InsightActivityLockScreenView(context: context)
                .widgetURL(activityURL(jobID: context.attributes.jobID))
        } dynamicIsland: { context in
            DynamicIsland {
                DynamicIslandExpandedRegion(.leading) { //封面展示区
                    InsightActivityArtworkView(
                        artworkLocalIdentifier: context.state.artworkLocalIdentifier,
                        size: 54,
                        cornerRadius: 14,
                        debugContext: "展开态-左侧封面"
                    )
                    .padding(.leading, 6)
                    .padding(.top, 18)
                }
                DynamicIslandExpandedRegion(.center) {//音乐元数据展示区
                    InsightActivityExpandedTextColumn(context: context)
                        .padding(.leading, 4)
                        .padding(.top, 0)
                }
                DynamicIslandExpandedRegion(.trailing) { //脉冲区
                    InsightActivityPulsePanel(
                        phase: context.state.phase,
                        pulseSize: 50,
                        showStatusText: true
                    )
                    .padding(.top, 16)
                    .padding(.trailing, 4)
                }
                DynamicIslandExpandedRegion(.bottom) {//底部展示区包含模型数据
                    InsightActivityMetadataRow(context: context)
                        .padding(.top, 6)
                        .padding(.leading, 6)
                }
            } compactLeading: {//紧凑封面展示区
                HStack(spacing: 0) {
                    //Spacer(minLength: 1)
                    InsightActivityArtworkView(
                        artworkLocalIdentifier: context.state.artworkLocalIdentifier,
                        size: 24,
                        cornerRadius: 7,
                        debugContext: "紧凑态-封面"
                    )
                    //.padding(.top, 0)
                    .padding(.leading, 2).padding(.trailing, 2)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            } compactTrailing: {//紧凑脉冲区
                InsightPulseView(phase: context.state.phase, size: 24, ringCount: 3)
                    .frame(width: 24, height: 24)
                    //.padding(.top, 0).padding(.trailing, 0)
            } minimal: {
                InsightPulseView(phase: context.state.phase, size: 18, ringCount: 2)
                    .frame(width: 18, height: 18)
            }
            .widgetURL(activityURL(jobID: context.attributes.jobID))
        }
    }

    private func activityURL(jobID: String) -> URL? {
        URL(string: "soniclens://insight-job/\(jobID)")
    }
}

@available(iOS 16.2, *)
private struct InsightActivityLockScreenView: View {
    let context: ActivityViewContext<InsightLiveActivityAttributes>

    var body: some View {
        HStack(alignment: .center, spacing: 18) {
            VStack(alignment: .leading, spacing: 14) {
                Text(headerText)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.white.opacity(0.75))
                    .padding(.leading, 2)

                HStack(alignment: .top, spacing: 12) {
                    InsightActivityArtworkView(
                        artworkLocalIdentifier: context.state.artworkLocalIdentifier,
                        size: 64,
                        cornerRadius: 18,
                        debugContext: "锁屏态-封面"
                    )
                    .padding(.top, 1)

                    VStack(alignment: .leading, spacing: 6) {
                        Text(context.state.title)
                            .font(.headline.weight(.semibold))
                            .foregroundStyle(.white)
                            .lineLimit(1)

                        Text(subtitleText)
                            .font(.subheadline)
                            .foregroundStyle(.white.opacity(0.72))
                            .lineLimit(2)
                    }
                }

                InsightActivityMetadataRow(context: context)
            }

            Spacer(minLength: 12)

            VStack {
                // Spacer(minLength: 12)
                InsightActivityPulsePanel(
                    phase: context.state.phase,
                    pulseSize: 74,
                    showStatusText: true
                )
            }
            .padding(.top, 12)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 16)
        .activityBackgroundTint(Color.black.opacity(0.92))
        .activitySystemActionForegroundColor(.white)
    }

    private var headerText: String {
        context.attributes.targetType == "album" ? "专辑分析" : "曲目分析"
    }

    private var subtitleText: String {
        switch context.attributes.targetType {
        case "album":
            return context.state.artist
        default:
            return [context.state.artist, context.state.album]
                .filter { !$0.isEmpty }
                .joined(separator: " · ")
        }
    }
}

@available(iOS 16.2, *)
private struct InsightActivityExpandedTextColumn: View {
    let context: ActivityViewContext<InsightLiveActivityAttributes>

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(headerText)
                .font(.caption2.weight(.semibold))
               .foregroundStyle(.white.opacity(0.72))
               .lineLimit(1)

            Text(context.state.title)
                .font(.headline.weight(.semibold))
                .foregroundStyle(.white)
                .lineLimit(1)

            Text(subtitleText)
                .font(.caption)
                .foregroundStyle(.white.opacity(0.7))
                .lineLimit(2)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var headerText: String {
        context.attributes.targetType == "album" ? "专辑分析" : "曲目分析"
    }

    private var subtitleText: String {
        switch context.attributes.targetType {
        case "album":
            return context.state.artist
        default:
            return [context.state.artist, context.state.album]
                .filter { !$0.isEmpty }
                .joined(separator: " · ")
        }
    }
}

@available(iOS 16.2, *)
private struct InsightActivityMetadataRow: View {
    let context: ActivityViewContext<InsightLiveActivityAttributes>

    var body: some View {
        HStack(spacing: 3) {
            Text(context.attributes.targetType == "album" ? "@音眸轨迹" : "@音眸轨迹")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.white.opacity(0.88))
            //竖线
            Rectangle()
                .fill(.white.opacity(0.16))
                .frame(width: 1, height: 12)

            Text(modelText)
                .font(.caption2)
                .foregroundStyle(.white.opacity(0.74))
                .lineLimit(1)

            Spacer(minLength: 0)
        }
    }

    private var modelText: String {
        "\(context.state.providerDisplayName) * \(context.state.modelDisplayName)"
    }
}

private struct InsightActivityPulsePanel: View {
    let phase: InsightJobPhase
    let pulseSize: CGFloat
    let showStatusText: Bool

    var body: some View {
        VStack(spacing: 5) {
            InsightPulseView(phase: phase, size: pulseSize, ringCount: pulseRingCount)
                .frame(width: pulseSize, height: pulseSize)

            if showStatusText {
                Text(phase.statusText)
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(InsightPulsePalette.palette(for: phase).accent)
                    .lineLimit(1)
            }
        }
        .frame(minWidth: pulseSize + 6)
    }

    private var pulseRingCount: Int {
        pulseSize >= 60 ? 4 : 3
    }
}

private struct InsightActivityArtworkView: View {
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "InsightActivityArtwork")
    let artworkLocalIdentifier: String?
    let size: CGFloat
    let cornerRadius: CGFloat
    let debugContext: String

    var body: some View {
        Group {
            if let image = loadArtworkImage() {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
                    .onAppear {
                        logger.debug("封面加载成功 场景=\(debugContext, privacy: .public) 标识=\(describeArtworkIdentifier(artworkLocalIdentifier), privacy: .public)")
                    }
            } else {
                placeholder
                    .onAppear {
                        logger.debug("封面缺失，使用占位图 场景=\(debugContext, privacy: .public) 标识=\(describeArtworkIdentifier(artworkLocalIdentifier), privacy: .public)")
                    }
            }
        }
        .frame(width: size, height: size)
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                .stroke(.white.opacity(0.10), lineWidth: 0.8)
        )
    }

    private var placeholder: some View {
        ZStack {
            RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                .fill(
                    LinearGradient(
                        colors: [
                            Color(red: 0.12, green: 0.16, blue: 0.28),
                            Color(red: 0.06, green: 0.08, blue: 0.16)
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )

            InsightPulseView(phase: .queued, size: size * 0.68, ringCount: 3)
                .opacity(0.9)
        }
    }

    private func loadArtworkImage() -> UIImage? {
        guard let artworkLocalIdentifier,
              let fileURL = LiveActivityArtworkSupport.fileURL(for: artworkLocalIdentifier),
              FileManager.default.fileExists(atPath: fileURL.path) else {
            return nil
        }

        guard let image = UIImage(contentsOfFile: fileURL.path) else {
            logger.error("读取本地封面失败 场景=\(debugContext, privacy: .public) 标识=\(describeArtworkIdentifier(artworkLocalIdentifier), privacy: .public)")
            return nil
        }
        return image
    }

    private func describeArtworkIdentifier(_ artworkIdentifier: String?) -> String {
        guard let artworkIdentifier else { return "空" }
        let trimmed = artworkIdentifier.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "空字符串" }
        return String(trimmed.prefix(12))
    }
}

private struct InsightPulseView: View {
    let phase: InsightJobPhase
    let size: CGFloat
    let ringCount: Int

    private var palette: InsightPulsePalette { .palette(for: phase) }

    var body: some View {
        ZStack {
            ForEach(Array((0..<ringCount).enumerated()), id: \.offset) { index, _ in
                let factor = ringScaleFactor(index: index)
                Circle()
                    .fill(ringFill(for: index))
                    .frame(width: size * factor, height: size * factor)
            }

            Circle()
                .fill(centerGradient)
                .frame(width: size * 0.28, height: size * 0.28)
                .shadow(color: palette.accent.opacity(0.36), radius: size * 0.08, x: 0, y: 0)
        }
        .compositingGroup()
        .scaleEffect(scaleFactor)
        .opacity(opacity)
        .animation(.easeInOut(duration: 0.35), value: phase)
    }

    private func ringScaleFactor(index: Int) -> CGFloat {
        let maxScale: CGFloat = 1.0
        let minScale: CGFloat = 0.42
        guard ringCount > 1 else { return maxScale }
        let progress = CGFloat(index) / CGFloat(ringCount - 1)
        return maxScale - ((maxScale - minScale) * progress)
    }

    private func ringFill(for index: Int) -> some ShapeStyle {
        let opacityStep = max(0.10, 0.28 - (CGFloat(index) * 0.045))
        return RadialGradient(
            colors: [
                palette.soft.opacity(opacityStep),
                palette.fill.opacity(opacityStep * 0.92)
            ],
            center: .center,
            startRadius: 0,
            endRadius: size * 0.55
        )
    }

    private var centerGradient: some ShapeStyle {
        RadialGradient(
            colors: [palette.center, palette.fill],
            center: .center,
            startRadius: 0,
            endRadius: size * 0.20
        )
    }

    private var scaleFactor: CGFloat {
        switch phase {
        case .queued:
            return 0.94
        case .running:
            return 1.0
        case .completed:
            return 0.98
        case .failed, .canceled:
            return 1.02
        }
    }

    private var opacity: Double {
        switch phase {
        case .queued:
            return 0.90
        case .running:
            return 1.0
        case .completed:
            return 0.98
        case .failed, .canceled:
            return 0.96
        }
    }
}

private struct InsightPulsePalette {
    let accent: Color
    let center: Color
    let fill: Color
    let soft: Color

    static func palette(for phase: InsightJobPhase) -> InsightPulsePalette {
        switch phase {
        case .queued:
            return InsightPulsePalette(
                accent: Color(red: 0.46, green: 0.83, blue: 0.96),
                center: Color(red: 0.20, green: 0.49, blue: 0.72),
                fill: Color(red: 0.10, green: 0.18, blue: 0.32),
                soft: Color(red: 0.60, green: 0.88, blue: 0.96)
            )
        case .running:
            return InsightPulsePalette(
                accent: Color(red: 0.22, green: 0.79, blue: 1.0),
                center: Color(red: 0.10, green: 0.52, blue: 0.98),
                fill: Color(red: 0.05, green: 0.12, blue: 0.36),
                soft: Color(red: 0.56, green: 0.88, blue: 1.0)
            )
        case .completed:
            return InsightPulsePalette(
                accent: Color(red: 0.36, green: 0.92, blue: 0.55),
                center: Color(red: 0.13, green: 0.70, blue: 0.33),
                fill: Color(red: 0.05, green: 0.23, blue: 0.12),
                soft: Color(red: 0.70, green: 0.97, blue: 0.77)
            )
        case .failed, .canceled:
            return InsightPulsePalette(
                accent: Color(red: 1.0, green: 0.45, blue: 0.45),
                center: Color(red: 0.86, green: 0.18, blue: 0.18),
                fill: Color(red: 0.28, green: 0.05, blue: 0.08),
                soft: Color(red: 1.0, green: 0.76, blue: 0.76)
            )
        }
    }
}

@available(iOS 17.0, *)
#Preview("Track Running - Lock Screen", as: .content, using: trackAttributes) {
    InsightJobLiveActivityWidget()
} contentStates: {
    trackRunningState
}

@available(iOS 17.0, *)
#Preview("Track Running - Compact", as: .dynamicIsland(.compact), using: trackAttributes) {
    InsightJobLiveActivityWidget()
} contentStates: {
    trackRunningState
}

@available(iOS 17.0, *)
#Preview("Track Completed - Minimal", as: .dynamicIsland(.minimal), using: trackAttributes) {
    InsightJobLiveActivityWidget()
} contentStates: {
    trackCompletedState
}

@available(iOS 17.0, *)
#Preview("Album Failed - Expanded", as: .dynamicIsland(.expanded), using: albumAttributes) {
    InsightJobLiveActivityWidget()
} contentStates: {
    albumFailedState
}

@available(iOS 17.0, *)
#Preview("Album Queued - Lock Screen", as: .content, using: albumAttributes) {
    InsightJobLiveActivityWidget()
} contentStates: {
    albumQueuedState
}

@available(iOS 17.0, *)
private let trackAttributes = InsightLiveActivityAttributes(jobID: "track-preview", targetType: "track")

@available(iOS 17.0, *)
private let albumAttributes = InsightLiveActivityAttributes(jobID: "album-preview", targetType: "album")

@available(iOS 17.0, *)
private let trackRunningState = InsightLiveActivityAttributes.ContentState(
    title: "光明大道",
    artist: "张楚",
    album: "孤独的人是可耻的",
    artworkLocalIdentifier: nil,
    providerDisplayName: "Doubao",
    modelDisplayName: "SonicLens-dev",
    phase: .running
)

@available(iOS 17.0, *)
private let trackCompletedState = InsightLiveActivityAttributes.ContentState(
    title: "光明大道",
    artist: "张楚",
    album: "孤独的人是可耻的",
    artworkLocalIdentifier: nil,
    providerDisplayName: "Doubao",
    modelDisplayName: "SonicLens-dev",
    phase: .completed
)

@available(iOS 17.0, *)
private let albumQueuedState = InsightLiveActivityAttributes.ContentState(
    title: "孤独的人是可耻的",
    artist: "张楚",
    album: "孤独的人是可耻的",
    artworkLocalIdentifier: nil,
    providerDisplayName: "Doubao",
    modelDisplayName: "SonicLens-dev",
    phase: .queued
)

@available(iOS 17.0, *)
private let albumFailedState = InsightLiveActivityAttributes.ContentState(
    title: "孤独的人是可耻的",
    artist: "张楚",
    album: "孤独的人是可耻的",
    artworkLocalIdentifier: nil,
    providerDisplayName: "Doubao",
    modelDisplayName: "SonicLens-dev",
    phase: .failed
)
#endif
