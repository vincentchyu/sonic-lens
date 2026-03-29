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
