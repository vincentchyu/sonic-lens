import SwiftUI
import Combine

@ViewBuilder
func albumDetailDestination(albumID: Int64, selectedTab: AlbumDetailTab = .info) -> some View {
    AlbumDetailView(albumID: albumID, selectedTab: selectedTab)
}

struct AlbumDetailView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(FavoriteStore.self) private var favoriteStore
    @EnvironmentObject private var insightCoordinator: InsightAnalysisCoordinator
    @StateObject private var viewModel: AlbumDetailViewModel

    let albumID: Int64
    @State private var isCurationExpanded: Bool = true
    @State private var selectedTab: AlbumDetailTab = .info
    @State private var sharePreviewRequest: SharePreviewRequest?

    init(albumID: Int64, selectedTab: AlbumDetailTab = .info) {
        self.albumID = albumID
        _selectedTab = State(initialValue: selectedTab)
        _viewModel = StateObject(wrappedValue: AlbumDetailViewModel())
    }

    private var isPhoneLayout: Bool {
        #if os(iOS)
        UIDevice.current.userInterfaceIdiom == .phone
        #else
        false
        #endif
    }

    var body: some View {
        ZStack {
            if let message = viewModel.errorMessage {
                ErrorBanner(message: message)
                    .padding(16)
            }

            if let detail = viewModel.detail {
                AlbumDetailPlatformContainer(
                    albumID: albumID,
                    detail: detail,
                    resolvedArtworkURL: viewModel.resolvedArtworkURL,
                    candidates: viewModel.candidates,
                    favoriteTrackIDs: viewModel.favoriteTrackIDs,
                    trackPresentation: viewModel.trackPresentation,
                    albumInsights: viewModel.albumInsights,
                    albumInsightGenerationState: viewModel.albumInsightGenerationState,
                    generationStatusMessage: viewModel.generationStatusMessage,
                    selectedTab: $selectedTab,
                    isCurationExpanded: $isCurationExpanded,
                    isSearchingCandidates: viewModel.isSearchingCandidates,
                    onSearch: searchCandidates,
                    onConfirm: confirmCandidate,
                    onGenerateInsight: startAlbumInsightGeneration,
                    selectedInsightIndex: $viewModel.selectedInsightIndex,
                    insightViewMode: $viewModel.insightViewMode
                )
            }

            if viewModel.isLoading && viewModel.detail == nil {
                LoadingOverlay()
            }

            if !viewModel.isLoading, viewModel.detail == nil, viewModel.errorMessage == nil {
                EmptyStateView(
                    title: "暂无专辑详情",
                    subtitle: "请稍后重试或返回重进。"
                )
                .padding(32)
            }
        }
        .navigationTitle("专辑详情")
        .toolbar {
            if isPhoneLayout {
                shareMenu
            } else {
                exportMenu
            }
        }
        .task(id: albumID) {
            if let server = store.currentServer {
                await viewModel.load(using: server, albumID: albumID, favoriteKeys: favoriteStore.favoriteKeys)
                await viewModel.syncInsightJob(insightCoordinator.activeJob, using: server, albumID: albumID, forceRefresh: true)
                await insightCoordinator.reconcileIfNeeded(using: server)
            }
        }
        .task(id: insightJobTaskToken) {
            guard let server = store.currentServer else { return }
            await viewModel.syncInsightJob(matchingInsightJob, using: server, albumID: albumID)
        }
        .onChange(of: viewModel.selectedAIPlatform) { _, newValue in
            guard viewModel.isModelPickerPresented, let server = store.currentServer, !newValue.isEmpty else { return }
            Task {
                try? await viewModel.selectAIPlatform(newValue, using: server)
            }
        }
        #if os(macOS)
        .popover(isPresented: modelPickerPresentedBinding, arrowEdge: .top) {
            InsightModelPickerContent(
                subjectLabel: "专辑",
                selectedAIPlatform: $viewModel.selectedAIPlatform,
                selectedAIModel: $viewModel.selectedAIModel,
                availableAIPlatforms: viewModel.availableAIPlatforms,
                availableAIModels: viewModel.availableAIModels,
                isConfirmDisabled: viewModel.selectedAIPlatform.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    || viewModel.availableAIModels.isEmpty
                    || viewModel.albumInsightGenerationState == .loadingModels
                    || viewModel.albumInsightGenerationState == .generating
                    || viewModel.selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                onCancel: { viewModel.dismissModelPicker() },
                onConfirm: { confirmAlbumInsightGeneration() }
            )
                .padding(18)
                .frame(width: 360)
        }
        #else
        .sheet(isPresented: modelPickerPresentedBinding) {
            NavigationStack {
                InsightModelPickerContent(
                    subjectLabel: "专辑",
                    selectedAIPlatform: $viewModel.selectedAIPlatform,
                    selectedAIModel: $viewModel.selectedAIModel,
                    availableAIPlatforms: viewModel.availableAIPlatforms,
                    availableAIModels: viewModel.availableAIModels,
                    isConfirmDisabled: viewModel.selectedAIPlatform.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || viewModel.availableAIModels.isEmpty
                        || viewModel.albumInsightGenerationState == .loadingModels
                        || viewModel.albumInsightGenerationState == .generating
                        || viewModel.selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                    onCancel: { viewModel.dismissModelPicker() },
                    onConfirm: { confirmAlbumInsightGeneration() }
                )
                .padding(20)
                .navigationTitle("选择音眸模型")
                .navigationBarTitleDisplayMode(.inline)
            }
            .presentationDetents(isPhoneLayout ? [.medium, .large] : [.fraction(0.45), .large])
            .presentationDragIndicator(.visible)
        }
        .fullScreenCover(item: $sharePreviewRequest) { request in
            SharePreviewView(payload: request.payload)
        }
        #endif
    }

    private func searchCandidates() {
        guard let server = store.currentServer else { return }
        Task { await viewModel.searchCandidates(using: server, albumID: albumID) }
    }

    private func confirmCandidate(_ candidate: ReleaseCandidate) {
        guard let server = store.currentServer else { return }
        Task { await viewModel.confirmSelection(using: server, albumID: albumID, candidate: candidate) }
    }

    @ToolbarContentBuilder
    private var exportMenu: some ToolbarContent {
        ToolbarItem(placement: .primaryAction) {
            Menu {
                Button("导出：专辑详情") {
                    if let detail = viewModel.detail {
                        exportSnapshotPNG(
                            AlbumSnapshotView(
                                detail: detail,
                                resolvedArtworkURL: viewModel.resolvedArtworkURL,
                                candidates: viewModel.candidates
                            )
                            .padding(32),
                            suggestedFilename: "\(detail.artist)-\(detail.displayName)-专辑"
                        )
                    }
                }
                Button("导出：音眸专辑") {
                    if let detail = viewModel.detail {
                        exportSnapshotPNG(
                            AlbumInsightSnapshotView(
                                detail: detail,
                                trackPresentation: viewModel.trackPresentation,
                                insight: currentAlbumInsight,
                                resolvedArtworkURL: viewModel.resolvedArtworkURL
                            )
                            .padding(32),
                            suggestedFilename: "\(detail.artist)-\(detail.displayName)-专辑音眸"
                        )
                    }
                }
            } label: {
                Label("导出快照", systemImage: "square.and.arrow.up")
            }
            .disabled(viewModel.detail == nil)
        }
    }

    @ToolbarContentBuilder
    private var shareMenu: some ToolbarContent {
        ToolbarItem(placement: .primaryAction) {
            Menu {
                Button("分享：基础信息") {
                    openSharePreview(scene: .albumInfo)
                }
                Button("分享：音眸") {
                    openSharePreview(scene: .albumInsight)
                }
            } label: {
                Label("分享", systemImage: "square.and.arrow.up")
            }
            .disabled(viewModel.detail == nil)
        }
    }

    private var modelPickerPresentedBinding: Binding<Bool> {
        Binding(
            get: { viewModel.isModelPickerPresented },
            set: { presented in
                if presented {
                    viewModel.isModelPickerPresented = true
                } else {
                    viewModel.dismissModelPicker()
                }
            }
        )
    }

    private func startAlbumInsightGeneration() {
        guard let server = store.currentServer else { return }
        selectedTab = .insights
        Task {
            await viewModel.beginAlbumInsightGeneration(using: server)
        }
    }

    private func confirmAlbumInsightGeneration() {
        guard let server = store.currentServer else { return }
        selectedTab = .insights
        Task {
            await viewModel.confirmAlbumInsightGeneration(
                using: server,
                coordinator: insightCoordinator,
                albumID: albumID
            )
        }
    }

    private var matchingInsightJob: InsightAnalysisJob? {
        guard let activeJob = insightCoordinator.activeJob, activeJob.matches(albumID: albumID) else {
            return nil
        }
        return activeJob
    }

    private var insightJobTaskToken: String {
        guard let job = matchingInsightJob else { return "none" }
        return "\(job.id)::\(job.phase.rawValue)::\(job.updatedAt ?? "")"
    }

    private func openSharePreview(scene: ShareScene) {
        guard let detail = viewModel.detail else { return }
        let payload = SharePayloadBuilder.buildAlbum(
            scene: scene,
            albumDetail: detail,
            albumInsight: currentAlbumInsight,
            resolvedArtworkURL: viewModel.resolvedArtworkURL
        )
        sharePreviewRequest = SharePreviewRequest(payload: payload)
    }

    private var currentAlbumInsight: AlbumInsight? {
        guard viewModel.albumInsights.indices.contains(viewModel.selectedInsightIndex) else {
            return viewModel.albumInsights.primaryInsight
        }
        return viewModel.albumInsights[viewModel.selectedInsightIndex]
    }
}

private struct AlbumDetailPlatformContainer: View {
    let albumID: Int64
    let detail: AlbumDetail
    let resolvedArtworkURL: String?
    let candidates: [ReleaseCandidate]
    let favoriteTrackIDs: Set<Int64>
    let trackPresentation: AlbumTrackPresentation
    let albumInsights: [AlbumInsight]
    let albumInsightGenerationState: InsightGenerationState
    let generationStatusMessage: String?
    @Binding var selectedTab: AlbumDetailTab
    @Binding var isCurationExpanded: Bool
    let isSearchingCandidates: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void
    let onGenerateInsight: () -> Void
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode

    var body: some View {
        #if os(iOS)
        if UIDevice.current.userInterfaceIdiom == .phone {
            PhoneAlbumDetailView(
                albumID: albumID,
                detail: detail,
                resolvedArtworkURL: resolvedArtworkURL,
                candidates: candidates,
                favoriteTrackIDs: favoriteTrackIDs,
                trackPresentation: trackPresentation,
                albumInsights: albumInsights,
                albumInsightGenerationState: albumInsightGenerationState,
                generationStatusMessage: generationStatusMessage,
                selectedTab: $selectedTab,
                isCurationExpanded: $isCurationExpanded,
                isSearchingCandidates: isSearchingCandidates,
                onSearch: onSearch,
                onConfirm: onConfirm,
                onGenerateInsight: onGenerateInsight,
                selectedInsightIndex: $selectedInsightIndex,
                insightViewMode: $insightViewMode
            )
        } else {
            RegularAlbumDetailView(
                albumID: albumID,
                detail: detail,
                resolvedArtworkURL: resolvedArtworkURL,
                candidates: candidates,
                favoriteTrackIDs: favoriteTrackIDs,
                trackPresentation: trackPresentation,
                albumInsights: albumInsights,
                albumInsightGenerationState: albumInsightGenerationState,
                generationStatusMessage: generationStatusMessage,
                selectedTab: $selectedTab,
                isCurationExpanded: $isCurationExpanded,
                isSearchingCandidates: isSearchingCandidates,
                onSearch: onSearch,
                onConfirm: onConfirm,
                onGenerateInsight: onGenerateInsight,
                selectedInsightIndex: $selectedInsightIndex,
                insightViewMode: $insightViewMode
            )
        }
        #else
        RegularAlbumDetailView(
            albumID: albumID,
            detail: detail,
            resolvedArtworkURL: resolvedArtworkURL,
            candidates: candidates,
            favoriteTrackIDs: favoriteTrackIDs,
            trackPresentation: trackPresentation,
            albumInsights: albumInsights,
            albumInsightGenerationState: albumInsightGenerationState,
            generationStatusMessage: generationStatusMessage,
            selectedTab: $selectedTab,
            isCurationExpanded: $isCurationExpanded,
            isSearchingCandidates: isSearchingCandidates,
            onSearch: onSearch,
            onConfirm: onConfirm,
            onGenerateInsight: onGenerateInsight,
            selectedInsightIndex: $selectedInsightIndex,
            insightViewMode: $insightViewMode
        )
        #endif
    }
}

private struct RegularAlbumDetailView: View {
    let albumID: Int64
    let detail: AlbumDetail
    let resolvedArtworkURL: String?
    let candidates: [ReleaseCandidate]
    let favoriteTrackIDs: Set<Int64>
    let trackPresentation: AlbumTrackPresentation
    let albumInsights: [AlbumInsight]
    let albumInsightGenerationState: InsightGenerationState
    let generationStatusMessage: String?
    @Binding var selectedTab: AlbumDetailTab
    @Binding var isCurationExpanded: Bool
    let isSearchingCandidates: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void
    let onGenerateInsight: () -> Void
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                AlbumHeroSection(
                    detail: detail,
                    resolvedArtworkURL: resolvedArtworkURL,
                    layout: .regular,
                    favoriteTrackCount: favoriteTrackIDs.count,
                    trackCountOverride: trackPresentation.trackCount,
                    totalDurationOverride: trackPresentation.totalDuration
                )

                AlbumDetailTabContainer(
                    albumID: albumID,
                    detail: detail,
                    candidates: candidates,
                    favoriteTrackIDs: favoriteTrackIDs,
                    trackPresentation: trackPresentation,
                    albumInsights: albumInsights,
                    albumInsightGenerationState: albumInsightGenerationState,
                    generationStatusMessage: generationStatusMessage,
                    selectedTab: $selectedTab,
                    isCurationExpanded: $isCurationExpanded,
                    isSearchingCandidates: isSearchingCandidates,
                    heroLayout: .regular,
                    onSearch: onSearch,
                    onConfirm: onConfirm,
                    onGenerateInsight: onGenerateInsight,
                    selectedInsightIndex: $selectedInsightIndex,
                    insightViewMode: $insightViewMode
                )
            }
            .padding(32)
        }
    }
}

private struct PhoneAlbumDetailView: View {
    let albumID: Int64
    let detail: AlbumDetail
    let resolvedArtworkURL: String?
    let candidates: [ReleaseCandidate]
    let favoriteTrackIDs: Set<Int64>
    let trackPresentation: AlbumTrackPresentation
    let albumInsights: [AlbumInsight]
    let albumInsightGenerationState: InsightGenerationState
    let generationStatusMessage: String?
    @Binding var selectedTab: AlbumDetailTab
    @Binding var isCurationExpanded: Bool
    let isSearchingCandidates: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void
    let onGenerateInsight: () -> Void
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                AlbumHeroSection(
                    detail: detail,
                    resolvedArtworkURL: resolvedArtworkURL,
                    layout: .phone,
                    favoriteTrackCount: favoriteTrackIDs.count,
                    trackCountOverride: trackPresentation.trackCount,
                    totalDurationOverride: trackPresentation.totalDuration
                )

                AlbumDetailTabContainer(
                    albumID: albumID,
                    detail: detail,
                    candidates: candidates,
                    favoriteTrackIDs: favoriteTrackIDs,
                    trackPresentation: trackPresentation,
                    albumInsights: albumInsights,
                    albumInsightGenerationState: albumInsightGenerationState,
                    generationStatusMessage: generationStatusMessage,
                    selectedTab: $selectedTab,
                    isCurationExpanded: $isCurationExpanded,
                    isSearchingCandidates: isSearchingCandidates,
                    heroLayout: .phone,
                    onSearch: onSearch,
                    onConfirm: onConfirm,
                    onGenerateInsight: onGenerateInsight,
                    selectedInsightIndex: $selectedInsightIndex,
                    insightViewMode: $insightViewMode
                )
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 20)
        }
    }
}

struct AlbumSnapshotView: View {
    let detail: AlbumDetail
    let resolvedArtworkURL: String?
    let candidates: [ReleaseCandidate]

    var body: some View {
        let favoriteTrackIDs = Set(detail.tracks.filter(\.isFavorited).map(\.id))
        VStack(alignment: .leading, spacing: 20) {
            AlbumHeroSection(
                detail: detail,
                resolvedArtworkURL: resolvedArtworkURL ?? ArtworkURLResolver.resolveArtworkPath(detail.coverArtURL, artworkBaseURL: nil),
                layout: .regular,
                favoriteTrackCount: favoriteTrackIDs.count,
                trackCountOverride: detail.tracks.count,
                totalDurationOverride: detail.tracks.reduce(0) { $0 + ($1.duration ?? 0) }
            )

            AlbumDetailContentView(
                albumID: detail.id,
                detail: detail,
                candidates: candidates,
                favoriteTrackIDs: favoriteTrackIDs,
                trackPresentation: AlbumTrackPresentation.build(from: detail.tracks),
                isCurationExpanded: .constant(true),
                isSearchingCandidates: false,
                heroLayout: .regular,
                onSearch: {},
                onConfirm: { _ in }
            )
        }
    }
}

struct AlbumInsightSnapshotView: View {
    let detail: AlbumDetail
    let trackPresentation: AlbumTrackPresentation
    let insight: AlbumInsight?
    let resolvedArtworkURL: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            AlbumHeroSection(
                detail: detail,
                resolvedArtworkURL: resolvedArtworkURL,
                layout: .regular,
                favoriteTrackCount: 0,
                trackCountOverride: trackPresentation.trackCount,
                totalDurationOverride: trackPresentation.totalDuration
            )

            AlbumInsightContentView(
                insight: insight,
                generationState: .idle,
                generationStatusMessage: nil,
                isCompact: false,
                onGenerateInsight: nil
            )
        }
    }
}

private struct AlbumDetailTabContainer: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    let favoriteTrackIDs: Set<Int64>
    let trackPresentation: AlbumTrackPresentation
    let albumInsights: [AlbumInsight]
    let albumInsightGenerationState: InsightGenerationState
    let generationStatusMessage: String?
    @Binding var selectedTab: AlbumDetailTab
    @Binding var isCurationExpanded: Bool
    let isSearchingCandidates: Bool
    let heroLayout: AlbumHeroLayout
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void
    let onGenerateInsight: () -> Void
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode

    var body: some View {
        VStack(alignment: .leading, spacing: heroLayout.sectionSpacing) {
            Picker("内容", selection: $selectedTab) {
                Text("信息").tag(AlbumDetailTab.info)
                Text("音眸").tag(AlbumDetailTab.insights)
            }
            .pickerStyle(.segmented)
            .tint(SonicTheme.primary)
            .frame(maxWidth: heroLayout == .phone ? .infinity : 320)

            if selectedTab == .info {
                AlbumDetailContentView(
                    albumID: albumID,
                    detail: detail,
                    candidates: candidates,
                    favoriteTrackIDs: favoriteTrackIDs,
                    trackPresentation: trackPresentation,
                    isCurationExpanded: $isCurationExpanded,
                    isSearchingCandidates: isSearchingCandidates,
                    heroLayout: heroLayout,
                    onSearch: onSearch,
                    onConfirm: onConfirm
                )
            } else {
                let currentInsight = albumInsights.indices.contains(selectedInsightIndex)
                    ? albumInsights[selectedInsightIndex]
                    : albumInsights.primaryInsight
                DetailSectionCard(title: "操作", ) {
                VStack(alignment: .leading, spacing: heroLayout.sectionSpacing) {
                    AlbumInsightActionCard(
                        hasInsight: currentInsight != nil,
                        generationState: albumInsightGenerationState,
                        generationStatusMessage: generationStatusMessage,
                        isCompact: heroLayout == .phone,
                        onGenerateInsight: onGenerateInsight
                    )
                    
                    if let insight = currentInsight {
                        AlbumInsightDetailView(
                            insight: insight,
                            allInsights: albumInsights,
                            selectedInsightIndex: $selectedInsightIndex,
                            insightViewMode: $insightViewMode,
                            showsContextHeader: false
                        )
                    } else {
                        AlbumInsightPrimaryContentView(
                            insight: nil,
                            compact: heroLayout == .phone,
                            emptyTitle: "暂无专辑音眸",
                            emptySubtitle: "当前专辑还没有可展示的音眸内容，可在此直接触发生成。"
                        )
                    }
                }}
            }
        }
    }
}

enum AlbumDetailTab: String, Hashable {
    case info
    case insights
}

private struct AlbumDetailContentView: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    let favoriteTrackIDs: Set<Int64>
    let trackPresentation: AlbumTrackPresentation
    @Binding var isCurationExpanded: Bool
    let isSearchingCandidates: Bool
    let heroLayout: AlbumHeroLayout
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: heroLayout.sectionSpacing) {
            AlbumTrackListSection(
                presentation: trackPresentation,
                isCompact: heroLayout == .phone,
                favoriteTrackIDs: favoriteTrackIDs
            )

            AlbumCurationSection(
                albumID: albumID,
                detail: detail,
                candidates: candidates,
                isExpanded: $isCurationExpanded,
                isCompact: heroLayout == .phone,
                isSearchingCandidates: isSearchingCandidates,
                onSearch: onSearch,
                onConfirm: onConfirm
            )
        }
    }
}

struct AlbumInsightPrimaryContentView: View {
    let insight: AlbumInsight?
    let compact: Bool
    var emptyTitle: String = "暂无专辑音眸"
    var emptySubtitle: String = "当前专辑还没有可展示的音眸内容。"

    var body: some View {
        Group {
            if let insight, insight.hasDisplayContent {
                AlbumInsightRichContentView(insight: insight, compact: compact)
            } else {
                DetailSectionCard(title: "内容状态", compact: compact) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(emptyTitle)
                            .font(compact ? .subheadline.weight(.semibold) : .headline.weight(.semibold))
                        Text(emptySubtitle)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        }
    }
}

private struct AlbumInsightRichContentView: View {
    let insight: AlbumInsight
    let compact: Bool

    private var extraSectionBlocks: [InsightSectionBlock] {
        insight.analysisBySection.values.keys
            .filter { !AlbumInsightSectionCatalog.orderedKeys.contains($0) }
            .sorted()
            .compactMap { key in
                guard let value = insight.analysisBySection.values[key]?.trimmingCharacters(in: .whitespacesAndNewlines),
                      !value.isEmpty else {
                    return nil
                }
                return InsightSectionBlock(id: key, title: AlbumInsightSectionCatalog.titleMap[key] ?? key, content: value)
            }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: compact ? 14 : 18) {
            if let providerLine = insight.providerLine {
                Text(providerLine)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if let summary = trimmed(insight.analysisSummary) {
                AlbumInsightTextSectionCard(
                    title: "专辑总评",
                    text: summary,
                    compact: compact
                )
            }

            DetailSectionCard(title: "音眸分析", compact: compact) {
                VStack(alignment: .leading, spacing: compact ? 10 : 12) {
                    ForEach(AlbumInsightSectionCatalog.orderedKeys, id: \.self) { key in
                        AlbumInsightSectionCard(
                            title: AlbumInsightSectionCatalog.titleMap[key] ?? key,
                            text: insight.analysisBySection.values[key],
                            compact: compact
                        )
                    }
                }
            }

            if !extraSectionBlocks.isEmpty {
                DetailSectionCard(title: "扩展分区", compact: compact) {
                    VStack(alignment: .leading, spacing: compact ? 10 : 12) {
                        ForEach(extraSectionBlocks) { block in
                            AlbumInsightSectionCard(
                                title: block.title,
                                text: block.content,
                                compact: compact
                            )
                        }
                    }
                }
            }

            if let backgroundInfo = trimmed(insight.backgroundInfo) {
                AlbumInsightTextSectionCard(
                    title: "背景信息",
                    text: backgroundInfo,
                    compact: compact
                )
            }

            if let eraContext = trimmed(insight.eraContext) {
                AlbumInsightTextSectionCard(
                    title: "时代语境",
                    text: eraContext,
                    compact: compact
                )
            }
        }
    }

    private func trimmed(_ text: String?) -> String? {
        guard let text else { return nil }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

private struct AlbumInsightContentView: View {
    let insight: AlbumInsight?
    let generationState: InsightGenerationState
    let generationStatusMessage: String?
    let isCompact: Bool
    let onGenerateInsight: (() -> Void)?

    private var isActionDisabled: Bool {
        generationState == .loadingModels || generationState == .generating
    }

    private var inFlightHint: String? {
        switch generationState {
        case .loadingModels:
            return "正在加载可用模型，请稍候。"
        case .generating:
            return "专辑音眸通常需要数分钟，切到桌面后可通过灵动岛继续关注状态。"
        default:
            return nil
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: isCompact ? 14 : 18) {
            DetailSectionCard(title: "音眸专辑", compact: isCompact) {
                VStack(alignment: .leading, spacing: 12) {
                    if let onGenerateInsight {
                        HStack(spacing: 10) {
                            if insight == nil {
                                Button(action: onGenerateInsight) {
                                    Label("生成专辑音眸", systemImage: "sparkles")
                                }
                                .buttonStyle(.borderedProminent)
                                .disabled(isActionDisabled)
                            } else {
                                Button(action: onGenerateInsight) {
                                    Label("重新生成", systemImage: "sparkles")
                                }
                                .buttonStyle(.bordered)
                                .disabled(isActionDisabled)
                            }

                            if generationState == .loadingModels || generationState == .generating {
                                ProgressView()
                                    .controlSize(.small)
                            }
                        }
                    }

                    if let generationStatusMessage {
                        Text(generationStatusMessage)
                            .font(.caption)
                            .foregroundStyle(generationState == .error ? Color.red : Color.secondary)
                    }

                    if let inFlightHint {
                        Text(inFlightHint)
                            .font(.caption)
                            .foregroundStyle(Color.orange)
                    }
                }
            }

            AlbumInsightPrimaryContentView(
                insight: insight,
                compact: isCompact,
                emptyTitle: "暂无专辑音眸",
                emptySubtitle: "当前专辑还没有可展示的音眸内容，可在此直接触发生成。"
            )
        }
    }
}

private struct AlbumInsightActionCard: View {
    let hasInsight: Bool
    let generationState: InsightGenerationState
    let generationStatusMessage: String?
    let isCompact: Bool
    let onGenerateInsight: () -> Void

    private var isActionDisabled: Bool {
        generationState == .loadingModels || generationState == .generating
    }

    private var inFlightHint: String? {
        switch generationState {
        case .loadingModels:
            return "正在加载可用模型，请稍候。"
        case .generating:
            return "专辑音眸通常需要数分钟，切到桌面后可通过灵动岛继续关注状态。"
        default:
            return nil
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 10) {
                if hasInsight {
                    Button(action: onGenerateInsight) {
                        Label("重新生成", systemImage: "sparkles")
                    }
                    .buttonStyle(.bordered)
                    .disabled(isActionDisabled)
                } else {
                    Button(action: onGenerateInsight) {
                        Label("生成专辑音眸", systemImage: "sparkles")
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(isActionDisabled)
                }

                if generationState == .loadingModels || generationState == .generating {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let generationStatusMessage {
                Text(generationStatusMessage)
                    .font(.caption)
                    .foregroundStyle(generationState == .error ? Color.red : Color.secondary)
            }

            if let inFlightHint {
                Text(inFlightHint)
                    .font(.caption)
                    .foregroundStyle(Color.orange)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct AlbumInsightTextSectionCard: View {
    let title: String
    let text: String
    let compact: Bool

    var body: some View {
        DetailSectionCard(title: title, compact: compact) {
            Text(text)
                .font(compact ? .subheadline : .body)
                .foregroundStyle(.primary)
                .lineSpacing(compact ? 5 : 6)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct AlbumInsightSectionCard: View {
    let title: String
    let text: String?
    let compact: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(compact ? .subheadline.weight(.semibold) : .headline.weight(.semibold))

            if let text = trimmed(text) {
                Text(text)
                    .font(compact ? .subheadline : .body)
                    .foregroundStyle(.primary)
                    .lineSpacing(compact ? 5 : 6)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else {
                Text("本次结果暂未生成这一分区内容。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(compact ? 12 : 14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(Color.white.opacity(0.08), lineWidth: 1)
        )
    }

    private func trimmed(_ text: String?) -> String? {
        guard let text else { return nil }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

private enum AlbumHeroLayout {
    case regular
    case phone

    var artworkSize: CGFloat {
        switch self {
        case .regular:
            return 180
        case .phone:
            return 188
        }
    }

    var sectionSpacing: CGFloat {
        switch self {
        case .regular:
            return 20
        case .phone:
            return 16
        }
    }
}

struct AlbumTrackListSection: View {
    let presentation: AlbumTrackPresentation
    let isCompact: Bool
    let favoriteTrackIDs: Set<Int64>

    var body: some View {
        DetailSectionCard(title: "曲目", compact: isCompact) {
            VStack(alignment: .leading, spacing: 12) {
                AlbumTracksSummary(
                    trackCount: presentation.trackCount,
                    totalDuration: presentation.totalDuration,
                    compact: isCompact
                )

                if presentation.discGroups.count > 1 {
                    LazyVStack(alignment: .leading, spacing: isCompact ? 14 : 16) {
                        ForEach(presentation.discGroups) { discGroup in
                            VStack(alignment: .leading, spacing: isCompact ? 8 : 10) {
                                AlbumDiscHeader(title: discGroup.title, compact: isCompact)

                                LazyVStack(spacing: isCompact ? 8 : 10) {
                                    ForEach(discGroup.tracks) { track in
                                        let isFavorite = favoriteTrackIDs.contains(track.id)
                                        NavigationLink(destination: TrackDetailView(track: track)) {
                                            AlbumTrackRow(track: track, compact: isCompact, isFavorite: isFavorite)
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                        }
                    }
                } else {
                    LazyVStack(spacing: isCompact ? 8 : 10) {
                        ForEach(presentation.discGroups.first?.tracks ?? []) { track in
                            let isFavorite = favoriteTrackIDs.contains(track.id)
                            NavigationLink(destination: TrackDetailView(track: track)) {
                                AlbumTrackRow(track: track, compact: isCompact, isFavorite: isFavorite)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
        }
    }

}

private struct AlbumDiscHeader: View {
    let title: String
    let compact: Bool

    var body: some View {
        Text(title)
            .font(compact ? .subheadline.weight(.semibold) : .headline.weight(.semibold))
            .foregroundStyle(.secondary)
            .padding(.top, compact ? 2 : 4)
    }
}

struct DetailSectionCard<Content: View>: View {
    let title: String
    var compact: Bool = false
    let content: Content

    init(title: String, compact: Bool = false, @ViewBuilder content: () -> Content) {
        self.title = title
        self.compact = compact
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: compact ? 10 : 12) {
            Text(title)
                .font(compact ? .headline : .title3)
                .fontWeight(.semibold)

            content
        }
        .padding(compact ? 14 : 16)
        .glassCard(cornerRadius: compact ? 16 : 14, isSimplified: true)
    }
}

struct AlbumTracksSummary: View {
    let trackCount: Int
    let totalDuration: Int64
    var compact: Bool = false

    var body: some View {
        HStack(spacing: compact ? 8 : 10) {
            SummaryChip(title: "曲目数", value: "\(trackCount)", compact: compact)
            SummaryChip(title: "总时长", value: formatDuration(totalDuration), compact: compact)
            Spacer()
        }
    }

    private func formatDuration(_ seconds: Int64) -> String {
        guard seconds > 0 else { return "—" }
        let total = Int(seconds)
        let hours = total / 3600
        let minutes = (total % 3600) / 60
        let secs = total % 60
        if hours > 0 {
            return String(format: "%d:%02d:%02d", hours, minutes, secs)
        }
        return String(format: "%02d:%02d", minutes, secs)
    }
}

struct SummaryChip: View {
    let title: String
    let value: String
    var compact: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(compact ? .caption : .caption)
        }
        .padding(.horizontal, compact ? 8 : 10)
        .padding(.vertical, compact ? 5 : 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 10))
    }
}

struct AlbumCurationSection: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    @Binding var isExpanded: Bool
    let isCompact: Bool
    let isSearchingCandidates: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        DetailSectionCard(title: "候选版本", compact: isCompact) {
            VStack(alignment: .leading, spacing: isCompact ? 10 : 12) {
                HStack(alignment: .top, spacing: 12) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("核对 MusicBrainz 候选并标记精选")
                            .font(isCompact ? .subheadline.weight(.semibold) : .body.weight(.semibold))
                        Text("不会控制播放器，仅用于专辑版本整理。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button(isExpanded ? "收起" : "展开") {
                        withAnimation(.easeInOut(duration: 0.15)) {
                            isExpanded.toggle()
                        }
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                }

                if let link = detail.releaseMB {
                    HStack(spacing: 12) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("当前精选")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            Text(link.mbid)
                                .font(.caption)
                                .fontDesign(.monospaced)
                                .lineLimit(1)
                        }
                        Spacer()
                        if link.confirmed {
                            Text("已确认")
                                .font(.caption)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.green.opacity(0.14), in: RoundedRectangle(cornerRadius: 8))
                        }
                    }
                    .padding(12)
                    .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 12))
                } else {
                    Text("尚未选择精选版本")
                        .foregroundStyle(.secondary)
                }

                if isExpanded {
                    HStack(spacing: 10) {
                        Button(isSearchingCandidates ? "搜索中..." : "搜索候选") { onSearch() }
                            .buttonStyle(.plain)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 10))
                            .disabled(isSearchingCandidates)
                        if isSearchingCandidates {
                            ProgressView()
                                .controlSize(.small)
                        }
                        Text("共 \(candidates.count) 条")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    if candidates.isEmpty {
                        Text("暂无候选。可点击“搜索候选”从 MusicBrainz 补全。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.top, 4)
                    } else {
                        VStack(spacing: 10) {
                            ForEach(candidates.prefix(12)) { candidate in
                                CandidateRow(
                                    candidate: candidate,
                                    isSelected: candidate.id == detail.releaseMB?.releaseMBID,
                                    onConfirm: { onConfirm(candidate) }
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

struct CandidateRow: View {
    let candidate: ReleaseCandidate
    let isSelected: Bool
    let onConfirm: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(candidate.name ?? "未命名版本")
                    .font(.body)
                    .lineLimit(1)
                Text(candidate.mbid)
                    .font(.caption2)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer()
            if isSelected {
                Text("已选")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                Button("确认精选", action: onConfirm)
                    .buttonStyle(.plain)
                    .font(.caption)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(Color.accentColor.opacity(0.14), in: RoundedRectangle(cornerRadius: 10))
            }
        }
        .padding(12)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Color.white.opacity(isSelected ? 0.18 : 0.08), lineWidth: 1)
        )
    }
}

private struct AlbumHeroSection: View {
    let detail: AlbumDetail
    let resolvedArtworkURL: String?
    let layout: AlbumHeroLayout
    let favoriteTrackCount: Int
    var trackCountOverride: Int? = nil
    var totalDurationOverride: Int64? = nil

    var body: some View {
        Group {
            if layout == .phone {
                VStack(alignment: .leading, spacing: 16) {
                    artwork
                        .frame(maxWidth: .infinity)

                    VStack(alignment: .leading, spacing: 10) {
                        titleBlock
                        metaFlow
                    }
                }
            } else {
                HStack(spacing: 24) {
                    artwork
                    VStack(alignment: .leading, spacing: 10) {
                        titleBlock
                        metaFlow
                    }
                    Spacer()
                }
            }
        }
        .padding(layout == .phone ? 16 : 18)
        .glassCard(cornerRadius: layout == .phone ? 18 : 16, isSimplified: true)
    }

    private var artwork: some View {
        ArtworkSquareView(
            artworkURL: resolvedArtworkURL,
            fallbackTitle: detail.displayName,
            size: layout.artworkSize,
            cornerRadius: 18,
            style: .vivid
        )
        .overlay(
            RoundedRectangle(cornerRadius: 18)
                .stroke(Color.white.opacity(0.22), lineWidth: 1)
        )
    }

    private var titleBlock: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(detail.displayName)
                .font(layout == .phone ? .title3.weight(.semibold) : .title2.weight(.semibold))
            Text(detail.artist)
                .font(layout == .phone ? .headline : .body)
                .foregroundStyle(.secondary)
        }
    }

    private var metaFlow: some View {
        FlexibleChipWrap(spacing: 10, lineSpacing: 10) {
            if let trackCountOverride {
                AlbumMetaChip(title: "曲目数", value: "\(trackCountOverride)")
            }
            if let totalDurationOverride, totalDurationOverride > 0 {
                AlbumMetaChip(title: "总时长", value: formatDuration(totalDurationOverride))
            }
            if let release = detail.releaseDate, !release.isEmpty {
                AlbumReleaseDateChip(
                    releaseDate: release,
                    originalReleaseDate: detail.originalReleaseDate
                )
            }
            if let genre = detail.genre, !genre.isEmpty {
                AlbumMetaChip(title: "流派", value: genre)
            }
            if let releaseType = detail.releaseType, !releaseType.isEmpty {
                AlbumMetaChip(title: "类型", value: releaseType.uppercased())
            }
            if let discs = detail.totalDiscs {
                AlbumMetaChip(title: "碟数", value: "\(discs)")
            }
            if favoriteTrackCount > 0 {
                AlbumMetaChip(title: "收藏", value: "\(favoriteTrackCount) 首")
            }
        }
    }

    private func formatDuration(_ seconds: Int64) -> String {
        guard seconds > 0 else { return "—" }
        let total = Int(seconds)
        let hours = total / 3600
        let minutes = (total % 3600) / 60
        let secs = total % 60
        if hours > 0 {
            return String(format: "%d:%02d:%02d", hours, minutes, secs)
        }
        return String(format: "%02d:%02d", minutes, secs)
    }
}

private struct FlexibleChipWrap<Content: View>: View {
    let spacing: CGFloat
    let lineSpacing: CGFloat
    let content: Content

    init(spacing: CGFloat, lineSpacing: CGFloat, @ViewBuilder content: () -> Content) {
        self.spacing = spacing
        self.lineSpacing = lineSpacing
        self.content = content()
    }

    var body: some View {
        ViewThatFits(in: .vertical) {
            HStack(spacing: spacing) {
                content
                Spacer(minLength: 0)
            }
            VStack(alignment: .leading, spacing: lineSpacing) {
                content
            }
        }
    }
}

struct AlbumMetaChip: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption)
                .lineLimit(1)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
    }
}

private struct AlbumReleaseDateChip: View {
    let releaseDate: String
    let originalReleaseDate: String?
    @State private var isOriginalReleasePopoverPresented = false

    private var trimmedOriginalReleaseDate: String? {
        originalReleaseDate?.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var shouldShowOriginalReleaseHint: Bool {
        guard let original = trimmedOriginalReleaseDate, !original.isEmpty else {
            return false
        }
        return original != releaseDate.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var originalReleaseHintText: String {
        guard let original = trimmedOriginalReleaseDate, !original.isEmpty else {
            return ""
        }
        return "原始发行时间：\(original)"
    }

    var body: some View {
        HStack(alignment: .center, spacing: 6) {
            VStack(alignment: .leading, spacing: 2) {
                Text("发行日期")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                Text(releaseDate)
                    .font(.caption)
                    .lineLimit(1)
            }

            #if os(macOS)
            if shouldShowOriginalReleaseHint {
                OriginalReleaseHintButton(
                    isPresented: $isOriginalReleasePopoverPresented,
                    helpText: originalReleaseHintText
                )
                .popover(isPresented: $isOriginalReleasePopoverPresented, arrowEdge: .top) {
                    OriginalReleasePopoverContent(
                        releaseDate: releaseDate,
                        originalReleaseDate: trimmedOriginalReleaseDate ?? ""
                    )
                    .padding(14)
                    .frame(width: 220)
                }
            }
            #endif
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
    }
}

#if os(macOS)
private struct OriginalReleaseHintButton: View {
    @Binding var isPresented: Bool
    let helpText: String

    var body: some View {
        Button {
            isPresented.toggle()
        } label: {
            Circle()
                .fill(Color.red)
                .frame(width: 7, height: 7)
                .frame(width: 14, height: 14, alignment: .center)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .help("点击查看原始发行时间")
        .accessibilityLabel(helpText)
    }
}

private struct OriginalReleasePopoverContent: View {
    let releaseDate: String
    let originalReleaseDate: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("原始发行时间")
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(originalReleaseDate)
                .font(.subheadline.weight(.semibold))
            Divider()
            Text("当前发行时间")
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(releaseDate)
                .font(.subheadline)
        }
    }
}
#endif

struct AlbumTrackRow: View {
    let track: Track
    var compact: Bool = false
    let isFavorite: Bool

    var body: some View {
        HStack(spacing: compact ? 10 : 12) {
            Text(trackNumber)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: compact ? 24 : 28, alignment: .leading)

            VStack(alignment: .leading, spacing: 2) {
                Text(track.track)
                    .font(compact ? .subheadline : .body)
                    .lineLimit(1)
                Text(track.artist)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            Spacer()

            HStack(spacing: 8) {
                Text(formatDuration(track.duration))
                    .font(.caption)
                    .foregroundStyle(.secondary)

                TrackFavoriteBadge(isFavorite: isFavorite)
            }
        }
        .padding(compact ? 10 : 12)
        .background(
            RoundedRectangle(cornerRadius: 10)
                .fill(SonicTheme.card)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(SonicTheme.glassBorder.opacity(0.7), lineWidth: 1)
        )
    }

    private var trackNumber: String {
        if let trackNumber = track.trackNumber {
            return "\(trackNumber)"
        }
        return "—"
    }

    private func formatDuration(_ duration: Int64?) -> String {
        guard let duration else { return "--:--" }
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}
