import SwiftUI
import Combine

public enum ToastStyle: Equatable {
    case info
    case success
    case warning
    case error

    var iconName: String {
        switch self {
        case .info: return "info.circle.fill"
        case .success: return "checkmark.circle.fill"
        case .warning: return "exclamationmark.triangle.fill"
        case .error: return "xmark.octagon.fill"
        }
    }

    var color: Color {
        switch self {
        case .info: return .blue
        case .success: return .green
        case .warning: return .orange
        case .error: return .red
        }
    }
}

public struct ToastMessage: Identifiable, Equatable {
    public let id: UUID
    public let message: String
    public let style: ToastStyle
    public let duration: TimeInterval
    public let actionTitle: String?
    public let action: (() -> Void)?

    public init(
        id: UUID = UUID(),
        message: String,
        style: ToastStyle = .info,
        duration: TimeInterval = 3.0,
        actionTitle: String? = nil,
        action: (() -> Void)? = nil
    ) {
        self.id = id
        self.message = message
        self.style = style
        self.duration = duration
        self.actionTitle = actionTitle
        self.action = action
    }

    public static func == (lhs: ToastMessage, rhs: ToastMessage) -> Bool {
        lhs.id == rhs.id && lhs.message == rhs.message && lhs.style == rhs.style
    }
}

@MainActor
public final class ToastManager: ObservableObject {
    public static let shared = ToastManager()

    @Published public var currentToast: ToastMessage?
    private var dismissTask: Task<Void, Never>?

    public func show(
        _ message: String,
        style: ToastStyle = .info,
        duration: TimeInterval = 3.0,
        actionTitle: String? = nil,
        action: (() -> Void)? = nil
    ) {
        dismissTask?.cancel()
        withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
            currentToast = ToastMessage(
                message: message,
                style: style,
                duration: duration,
                actionTitle: actionTitle,
                action: action
            )
        }

        if duration > 0 {
            dismissTask = Task {
                try? await Task.sleep(nanoseconds: UInt64(duration * 1_000_000_000))
                guard !Task.isCancelled else { return }
                withAnimation(.easeInOut(duration: 0.25)) {
                    self.currentToast = nil
                }
            }
        }
    }

    public func dismiss() {
        dismissTask?.cancel()
        withAnimation(.easeInOut(duration: 0.25)) {
            currentToast = nil
        }
    }
}

struct ToastOverlayView: View {
    @ObservedObject var toastManager: ToastManager = .shared

    var body: some View {
        VStack {
            if let toast = toastManager.currentToast {
                HStack(spacing: 10) {
                    Image(systemName: toast.style.iconName)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(toast.style.color)

                    Text(toast.message)
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .foregroundStyle(.primary)

                    if let actionTitle = toast.actionTitle, let action = toast.action {
                        Button(action: {
                            toastManager.dismiss()
                            action()
                        }) {
                            Text(actionTitle)
                                .font(.caption)
                                .fontWeight(.bold)
                                .foregroundStyle(toast.style.color)
                                .padding(.horizontal, 10)
                                .padding(.vertical, 4)
                                .background(toast.style.color.opacity(0.12), in: Capsule())
                        }
                        .buttonStyle(.plain)
                    }

                    Spacer(minLength: 8)

                    Button(action: { toastManager.dismiss() }) {
                        Image(systemName: "xmark")
                            .font(.system(size: 11, weight: .bold))
                            .foregroundStyle(.secondary)
                            .frame(width: 20, height: 20)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
                .glassCard(cornerRadius: 18)
                .shadow(color: Color.black.opacity(0.14), radius: 14, x: 0, y: 8)
                .padding(.horizontal, 24)
                .padding(.top, 12)
                .transition(.move(edge: .top).combined(with: .opacity))
            }
            Spacer()
        }
        .animation(.spring(response: 0.35, dampingFraction: 0.8), value: toastManager.currentToast)
        .allowsHitTesting(toastManager.currentToast != nil)
    }
}

public extension View {
    func withGlobalToast() -> some View {
        ZStack(alignment: .top) {
            self
            ToastOverlayView()
        }
    }
}
