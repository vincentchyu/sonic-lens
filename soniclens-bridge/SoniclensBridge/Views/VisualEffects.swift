import SwiftUI

struct HoverCardEffect: ViewModifier {
    let isHovered: Bool

    func body(content: Content) -> some View {
        content
            .scaleEffect(isHovered ? 1.02 : 1.0)
            .shadow(color: Color.black.opacity(isHovered ? 0.16 : 0.08), radius: isHovered ? 20 : 14, x: 0, y: isHovered ? 14 : 10)
    }
}

extension View {
    func hoverCard(_ isHovered: Bool) -> some View {
        modifier(HoverCardEffect(isHovered: isHovered))
    }
}

struct PressableButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.96 : 1.0)
            .opacity(configuration.isPressed ? 0.85 : 1.0)
            .animation(.easeInOut(duration: 0.12), value: configuration.isPressed)
    }
}
