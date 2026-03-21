import SwiftUI

enum SonicTheme {
    // MARK: - Core Palette
    static let background = dynamicColor(
        light: .sonicRGBA(0.96, 0.97, 0.99, 1),
        dark: .sonicRGBA(0.05, 0.06, 0.08, 1)
    )

    static let card = dynamicColor(
        light: .sonicWhite(1.0, alpha: 0.45),
        dark: .sonicWhite(0.1, alpha: 0.3)
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
    let content: Content

    init(cornerRadius: CGFloat = 16, padding: CGFloat = 16, @ViewBuilder content: () -> Content) {
        self.cornerRadius = cornerRadius
        self.padding = padding
        self.content = content()
    }

    var body: some View {
        content
            .padding(padding)
            .background(SonicTheme.card)
            .overlay(
                RoundedRectangle(cornerRadius: cornerRadius)
                    .stroke(SonicTheme.glassBorder, lineWidth: 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
            .shadow(color: Color.black.opacity(0.08), radius: 18, x: 0, y: 12)
    }
}

struct AmbientBackgroundView: View {
    let gradient: LinearGradient
    let orbs: [AmbientOrb]

    @State private var animate = false
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        ZStack {
            gradient
            ForEach(orbs) { orb in
                Circle()
                    .fill(orb.color)
                    .frame(width: orb.size, height: orb.size)
                    .blur(radius: orb.blur)
                    .opacity(orb.opacity)
                    .offset(performanceModeEnabled ? orb.offsetFrom : (animate ? orb.offsetTo : orb.offsetFrom))
                    .animation(
                        performanceModeEnabled ? .none : .easeInOut(duration: orb.duration).repeatForever(autoreverses: true),
                        value: animate
                    )
            }
        }
        .ignoresSafeArea()
        .onAppear { animate = !performanceModeEnabled }
    }
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
