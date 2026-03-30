import SwiftUI

struct SonicLensInsightsView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel
    @State private var selectedTargetType: InsightTargetType = .track
    @State private var hasAppearedOnce = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Picker("分析对象", selection: $selectedTargetType) {
                    Text("曲目").tag(InsightTargetType.track)
                    Text("专辑").tag(InsightTargetType.album)
                }
                .pickerStyle(.segmented)
                .tint(SonicTheme.primary)
                .frame(maxWidth: 360)

                insightList
            }
            .padding(.horizontal, 32)
            .padding(.vertical, 20)
        }
        .task(id: insightTaskID) {
            guard let server = store.currentServer else { return }
            await viewModel.ensureInsightsLoaded(using: server, targetType: selectedTargetType)
        }
        .onAppear {
            guard let server = store.currentServer else { return }
            if hasAppearedOnce {
                Task {
                    await viewModel.reloadInsights(using: server, targetType: selectedTargetType)
                }
            } else {
                hasAppearedOnce = true
            }
        }
        .onChange(of: selectedTargetType) { _, newValue in
            guard let server = store.currentServer else { return }
            Task {
                await viewModel.reloadInsights(using: server, targetType: newValue)
            }
        }
    }

    private var insightList: some View {
        Group {
            let items = currentInsights
            if items.isEmpty {
                EmptyStateView(
                    title: selectedTargetType == .album ? "暂无专辑音眸" : "暂无曲目音眸",
                    subtitle: selectedTargetType == .album ? "生成专辑分析后会显示在这里。" : "生成赏析后会显示在这里。"
                )
            } else {
                LazyVStack(spacing: 12) {
                    ForEach(Array(items.enumerated()), id: \.element.id) { index, insight in
                        NavigationLink(destination: destination(for: insight)) {
                            InsightRowCard(insight: insight)
                        }
                        .buttonStyle(.plain)
                        .onAppear {
                            guard let server = store.currentServer else { return }
                            if viewModel.shouldLoadMoreInsights(at: index, targetType: selectedTargetType) {
                                Task { await viewModel.loadMoreInsights(using: server, targetType: selectedTargetType) }
                            }
                        }
                    }
                }
            }
        }
    }

    private var currentInsights: [InsightSummary] {
        selectedTargetType == .album ? viewModel.albumInsights : viewModel.insights
    }

    private var insightTaskID: String {
        "\(store.currentServer?.id.uuidString ?? "no-server")-\(selectedTargetType.rawValue)"
    }

    @ViewBuilder
    private func destination(for insight: InsightSummary) -> some View {
        InsightDetailLoaderView(viewModel: viewModel, summary: insight)
    }
}

struct InsightRowCard: View {
    let insight: InsightSummary
    @State private var isHovered = false
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(insight.displayTitle)
                        .font(.body)
                        .fontWeight(.semibold)
                        .lineLimit(1)
                    Text(insight.displaySubtitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                Spacer()
                Text(insight.badgeText)
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.primary.opacity(0.06), in: Capsule())
            }

            if let summary = insight.analysisSummary, !summary.isEmpty {
                Text(summary)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
            }
        }
        .padding(14)
        .glassCard(cornerRadius: 12, isSimplified: performanceModeEnabled)
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

struct InsightDetailLoaderView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel
    let summary: InsightSummary

    @State private var trackDetail: Insight?
    @State private var albumDetail: AlbumInsight?
    @State private var isLoading = true
    @State private var errorMessage: String?

    var body: some View {
        Group {
            if let trackDetail {
                InsightDetailView(insight: trackDetail)
            } else if let albumDetail {
                AlbumInsightDetailView(insight: albumDetail)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        InsightDetailHeaderPlaceholder(summary: summary)
                        if isLoading {
                            ProgressView("音眸详情加载中...")
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(20)
                                .glassCard(cornerRadius: 14)
                        } else if let errorMessage {
                            Text(errorMessage)
                                .font(.body)
                                .foregroundStyle(.secondary)
                                .padding(20)
                                .glassCard(cornerRadius: 14)
                        } else {
                            if summary.isAlbum {
                                AlbumInsightPrimaryContentView(
                                    insight: nil,
                                    compact: false,
                                    emptyTitle: "暂无专辑音眸",
                                    emptySubtitle: "当前专辑还没有可展示的音眸内容。"
                                )
                            } else {
                                InsightPrimaryContentView(
                                    insight: nil,
                                    style: .detail,
                                    emptyTitle: "暂无音眸",
                                    emptySubtitle: "当前曲目还没有可展示的音眸内容。"
                                )
                            }
                        }
                    }
                    .padding(32)
                }
            }
        }
        .task(id: summary.id) {
            await loadDetail()
        }
    }

    private func loadDetail() async {
        guard let server = store.currentServer else {
            isLoading = false
            errorMessage = "当前未连接服务器"
            return
        }
        isLoading = true
        errorMessage = nil
        trackDetail = nil
        albumDetail = nil
        do {
            if summary.isAlbum {
                albumDetail = try await viewModel.fetchAlbumInsightDetail(using: server, id: summary.id)
            } else {
                trackDetail = try await viewModel.fetchTrackInsightDetail(using: server, id: summary.id)
            }
        } catch {
            errorMessage = "音眸详情加载失败"
        }
        isLoading = false
    }
}

private struct InsightDetailHeaderPlaceholder: View {
    let summary: InsightSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(summary.displayTitle)
                .font(.title2)
                .fontWeight(.semibold)
            Text(summary.displaySubtitle)
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassCard(cornerRadius: 16)
    }
}
