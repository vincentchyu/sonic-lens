import SwiftUI

struct SonicLensInsightsView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel
    @State private var selectedInsight: Insight?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                SectionHeader(title: "音眸")

                if viewModel.insights.isEmpty {
                    EmptyStateView(
                        title: "暂无解析",
                        subtitle: "生成赏析后会显示在这里。"
                    )
                } else {
                    VStack(spacing: 12) {
                        ForEach(viewModel.insights) { insight in
                            InsightRowCard(insight: insight)
                                .onTapGesture {
                                    selectedInsight = insight
                                }
                                .onAppear {
                                    if insight.id == viewModel.insights.last?.id, let server = store.currentServer {
                                        Task { await viewModel.loadMoreInsights(using: server) }
                                    }
                                }
                        }
                    }
                }
            }
            .padding(32)
        }
        .navigationDestination(item: $selectedInsight) { insight in
            InsightDetailView(insight: insight)
        }
        .task(id: store.currentServer?.id) {
            guard let server = store.currentServer else { return }
            await viewModel.ensureInsightsLoaded(using: server)
        }
    }
}

struct InsightRowCard: View {
    let insight: Insight
    @State private var isHovered = false

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(insight.track)
                        .font(.body)
                        .fontWeight(.semibold)
                    Text("\(insight.artist) · \(insight.album)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Image(systemName: "sparkle.magnifyingglass")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(.secondary)
            }

            if let summary = insight.analysisSummary, !summary.isEmpty {
                Text(summary)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
            }
        }
        .padding(14)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Color.white.opacity(isHovered ? 0.2 : 0.08), lineWidth: 1)
        )
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }
}
