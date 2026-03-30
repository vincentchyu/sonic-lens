import SwiftUI

struct ErrorBanner: View {
    let message: String

    var body: some View {
        HStack {
            Image(systemName: "exclamationmark.triangle")
            Text(message)
                .font(.subheadline)
            Spacer()
        }
        .padding(10)
        .background(Color.red.opacity(0.1))
        .foregroundColor(.red)
        .cornerRadius(10)
    }
}

struct LoadingOverlay: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        ZStack {
            backdropColor
                .ignoresSafeArea()

            VStack(spacing: 12) {
                Capsule()
                    .fill(SonicTheme.accentGradient)
                    .frame(width: 42, height: 4)

                ProgressView()
                    .progressViewStyle(.circular)
                    .tint(SonicTheme.primary)

                Text("加载中...")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(SonicTheme.textPrimary)

                Text("正在获取内容")
                    .font(.caption)
                    .foregroundStyle(SonicTheme.textSecondary)
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 18)
            .frame(minWidth: 180)
            .glassCard(cornerRadius: 18, isSimplified: true)
        }
    }

    private var backdropColor: Color {
        switch colorScheme {
        case .dark:
            return Color.black.opacity(0.26)
        default:
            return Color(red: 0.93, green: 0.95, blue: 0.98).opacity(0.78)
        }
    }
}

struct ContentHeader: View {
    let title: String

    var body: some View {
        HStack {
            Text(title)
                .font(.title3)
                .fontWeight(.semibold)
            Spacer()
        }
    }
}

struct SectionHeader: View {
    let title: String

    var body: some View {
        Text(title)
            .font(.title3)
            .fontWeight(.semibold)
    }
}

/// 播放静默时显示的状态条，用来提示当前是暂停更新还是完全无活动播放。
struct PlaybackStatusBanner: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.caption2.weight(.semibold))
            .foregroundStyle(SonicTheme.textSecondary)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(
                Capsule(style: .continuous)
                    .fill(SonicTheme.card.opacity(0.86))
            )
            .overlay(
                Capsule(style: .continuous)
                    .stroke(SonicTheme.glassBorder.opacity(0.72), lineWidth: 1)
            )
    }
}
