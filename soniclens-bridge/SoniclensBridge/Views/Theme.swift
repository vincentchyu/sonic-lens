import SwiftUI

enum SonicTheme {
    // MARK: - Core Palette
    static let background = dynamicColor(
        light: .sonicRGBA(0.96, 0.97, 0.99, 1),
        dark: .sonicRGBA(0.05, 0.06, 0.08, 1)
    )

    static let card = dynamicColor(
        // light: .sonicWhite(1.0, alpha: 0.45),
        light: .sonicWhite(1.0, alpha: 0.8),
        // dark: .sonicWhite(0.1, alpha: 0.3)
        dark: .sonicWhite(0.1, alpha: 0.7)

    )

    static let glassBorder = dynamicColor(
        light: .sonicWhite(1.0, alpha: 0.6),
        dark: .sonicWhite(1.0, alpha: 0.12)
    )

    static let textPrimary = dynamicColor(
        light: .sonicRGBA(0.1, 0.12, 0.18, 1),
        dark: .sonicRGBA(0.94, 0.95, 0.98, 1)
    )

    static let textSecondary = dynamicColor(
        light: .sonicRGBA(0.4, 0.45, 0.55, 1),
        dark: .sonicRGBA(0.55, 0.6, 0.7, 1)
    )

    // 渐变色定义，用于 Ambient 背景和图标
    static let accentGradient = LinearGradient(
        colors: [Color.accentColor, Color.accentColor.opacity(0.7)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let primary = Color.accentColor
    static let accent = Color.accentColor // 别名兼容
    static let lyricsAccent = Color.accentColor // 别名兼容
    static let secondaryAccent = Color.blue

    static let progressTrack = dynamicColor(
        light: .sonicWhite(0, alpha: 0.1),
        dark: .sonicWhite(1, alpha: 0.1)
    )

    static let progressFill = accentGradient

    // MARK: - Helpers
    static func dynamicColor(light: PlatformColor, dark: PlatformColor) -> Color {
        Color(platformColor: .sonicDynamic(light: light, dark: dark))
    }
}

struct GlassPanel<Content: View>: View {
    let cornerRadius: CGFloat
    let padding: CGFloat
    let clipsContent: Bool
    let content: Content
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    init(
        cornerRadius: CGFloat = 16,
        padding: CGFloat = 16,
        clipsContent: Bool = false,
        @ViewBuilder content: () -> Content
    ) {
        self.cornerRadius = cornerRadius
        self.padding = padding
        self.clipsContent = clipsContent
        self.content = content()
    }

    var body: some View {
        let simplified = performanceModeEnabled
        let shape = RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)

        Group {
            if clipsContent {
                content
                    .padding(padding)
                    .clipShape(shape)
            } else {
                content
                    .padding(padding)
            }
        }
        .background {
            // 阴影挂在纯圆角背景形状上，走硬件解析阴影，避免复杂内容离屏栅格化
            shape
                .fill(SonicTheme.card)
                .shadow(
                    color: Color.black.opacity(simplified ? 0.04 : 0.06),
                    radius: simplified ? 6 : 10,
                    x: 0,
                    y: simplified ? 3 : 5
                )
        }
        .overlay {
            shape
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
        }
    }
}

struct AmbientBackgroundView: View {
    let gradient: LinearGradient
    let orbs: [AmbientOrb]
    var renderingStyle: AmbientBackgroundRenderingStyle = .standard

    @State private var animate = false
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        let simplified = performanceModeEnabled || renderingStyle != .standard
        let animated = !performanceModeEnabled && renderingStyle == .standard
        let visibleOrbs = renderingStyle == .standard ? Array(orbs.prefix(2)) : Array(orbs.prefix(1))

        ZStack {
            gradient
            if !visibleOrbs.isEmpty {
                // 使用稳定序号作为身份，避免每次重求值因 UUID 变化销毁重建光斑
                ForEach(Array(visibleOrbs.enumerated()), id: \.offset) { _, orb in
                    if simplified {
                        let diameter = orb.size * 0.96
                        Circle()
                            .fill(
                                RadialGradient(
                                    colors: [
                                        orb.color.opacity(orb.opacity * 0.62),
                                        orb.color.opacity(orb.opacity * 0.22),
                                        .clear
                                    ],
                                    center: .center,
                                    startRadius: 0,
                                    endRadius: diameter * 0.5
                                )
                            )
                            .frame(width: diameter, height: diameter)
                            .offset(orb.offsetFrom)
                    } else {
                        Circle()
                            .fill(orb.color)
                            .frame(width: orb.size * 0.92, height: orb.size * 0.92)
                            .blur(radius: orb.blur * 0.72)
                            .opacity(orb.opacity * 0.82)
                            .offset(animated && animate ? orb.offsetTo : orb.offsetFrom)
                            .animation(
                                animated
                                ? .easeInOut(duration: orb.duration).repeatForever(autoreverses: true)
                                : nil,
                                value: animate
                            )
                    }
                }
            }
        }
        .ignoresSafeArea()
        .onAppear { animate = animated }
    }
}

enum AmbientBackgroundRenderingStyle {
    case standard
    case compact
    case staticHome
}

struct AmbientOrb: Identifiable {
    let id = UUID()
    let color: Color
    let size: CGFloat
    let blur: CGFloat
    let opacity: Double
    let offsetFrom: CGSize
    let offsetTo: CGSize
    let duration: Double
}

@MainActor
private final class WidthDebounceController {
    var task: Task<Void, Never>?
    var lastCommittedWidth: CGFloat = 0

    func cancel() {
        task?.cancel()
        task = nil
    }
}

private struct DebouncedWidthModifier: ViewModifier {
    let delayMs: UInt64
    let threshold: CGFloat
    let onChange: @MainActor (CGFloat, Bool) -> Void

    @State private var controller = WidthDebounceController()

    func body(content: Content) -> some View {
        content
            .background {
                GeometryReader { proxy in
                    Color.clear
                        .onAppear {
                            let width = proxy.size.width
                            guard width > 0 else { return }
                            if controller.lastCommittedWidth == 0 || abs(width - controller.lastCommittedWidth) >= threshold {
                                controller.lastCommittedWidth = width
                                onChange(width, true)
                            }
                        }
                        .onChange(of: proxy.size.width) { _, newWidth in
                            guard newWidth > 0 else { return }
                            if controller.lastCommittedWidth == 0 {
                                controller.cancel()
                                controller.lastCommittedWidth = newWidth
                                onChange(newWidth, true)
                                return
                            }
                            controller.cancel()
                            controller.task = Task { @MainActor in
                                do {
                                    try await Task.sleep(nanoseconds: delayMs * 1_000_000)
                                } catch {
                                    return
                                }
                                guard !Task.isCancelled else { return }
                                guard abs(newWidth - controller.lastCommittedWidth) >= threshold else { return }
                                controller.lastCommittedWidth = newWidth
                                onChange(newWidth, false)
                            }
                        }
                        .onDisappear {
                            controller.cancel()
                        }
                }
            }
    }
}

extension View {
    /// 防抖监听容器宽度变化，避免侧边栏或窗口缩放动画期间每帧触发外层重排
    func measureDebouncedWidth(
        delayMs: UInt64 = 120,
        threshold: CGFloat = 4,
        onChange: @escaping @MainActor (CGFloat, Bool) -> Void
    ) -> some View {
        modifier(DebouncedWidthModifier(delayMs: delayMs, threshold: threshold, onChange: onChange))
    }
}
