import SwiftUI

private struct SonicPerformanceModeEnabledKey: EnvironmentKey {
    static let defaultValue: Bool = false
}

extension EnvironmentValues {
    var sonicPerformanceModeEnabled: Bool {
        get { self[SonicPerformanceModeEnabledKey.self] }
        set { self[SonicPerformanceModeEnabledKey.self] = newValue }
    }
}

struct GlassCardStyle: ViewModifier {
    let cornerRadius: CGFloat
    var isSimplified: Bool = false // 新增：性能模式，用于长列表
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    func body(content: Content) -> some View {
        let simplified = isSimplified || performanceModeEnabled
        content
            .background {
                if simplified {
                    SonicTheme.card
                        .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
                } else {
                    RoundedRectangle(cornerRadius: cornerRadius)
                        .fill(.ultraThinMaterial)
                }
            }
            .overlay(
                RoundedRectangle(cornerRadius: cornerRadius)
                    .stroke(SonicTheme.glassBorder, lineWidth: 1)
            )
            .shadow(color: Color.black.opacity(simplified ? 0.04 : 0.08), radius: simplified ? 8 : 16, x: 0, y: simplified ? 4 : 10)
    }
}

extension View {
    func glassCard(cornerRadius: CGFloat = 12, isSimplified: Bool = false) -> some View {
        modifier(GlassCardStyle(cornerRadius: cornerRadius, isSimplified: isSimplified))
    }
}
