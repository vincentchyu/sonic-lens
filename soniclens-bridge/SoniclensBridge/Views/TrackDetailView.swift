import SwiftUI
import Combine

@ViewBuilder
func trackDetailDestination(track: Track, selectedTab: TrackDetailTab = .info) -> some View {
    TrackDetailView(track: track, selectedTab: selectedTab)
}

struct TrackDetailView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(FavoriteStore.self) private var favoriteStore
    @EnvironmentObject private var insightCoordinator: InsightAnalysisCoordinator
    @StateObject private var viewModel = TrackDetailViewModel()

    let track: Track
    @State private var selectedTab: TrackDetailTab = .info
    @State private var previewTime: TimeInterval = 0
    @State private var sharePreviewRequest: SharePreviewRequest?

    init(track: Track, selectedTab: TrackDetailTab = .info) {
        self.track = track
        _selectedTab = State(initialValue: selectedTab)
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

            ScrollView {
                VStack(alignment: .leading, spacing: isPhoneLayout ? 16 : 20) {
                    TrackDetailHeader(
                        track: track,
                        playCount: track.playCount,
                        artworkURL: viewModel.resolvedArtworkURL,
                        isFavorite: isCurrentTrackFavorite,
                        layout: isPhoneLayout ? .phone : .regular
                    )

                    Picker("内容", selection: $selectedTab) {
                        Text("信息").tag(TrackDetailTab.info)
                        Text("歌词").tag(TrackDetailTab.lyrics)
                        Text("音眸").tag(TrackDetailTab.insights)
                    }
                    .pickerStyle(.segmented)
                    .tint(SonicTheme.primary)
                    .frame(maxWidth: isPhoneLayout ? .infinity : 320)

                    if selectedTab == .info {
                        infoSection
                    } else if selectedTab == .lyrics {
                        lyricsSection
                    } else {
                        insightsSection
                    }
                }
                .padding(isPhoneLayout ? 16 : 32)
            }

            if viewModel.isLoading {
                LoadingOverlay()
            }
        }
        .navigationTitle("曲目详情")
        .toolbar {
            if isPhoneLayout {
                shareMenu
            } else {
                exportMenu
            }
        }
        .task(id: store.currentServer?.id) {
            guard let server = store.currentServer else { return }
            await viewModel.load(
                using: server,
                artist: track.artist,
                album: track.album,
                track: track.track,
                trackNumber: track.trackNumber,
                discNumber: track.discNumber
            )
            await viewModel.syncInsightJob(insightCoordinator.activeJob, using: server, track: track, forceRefresh: true)
            await insightCoordinator.reconcileIfNeeded(using: server)
        }
        .task(id: insightJobTaskToken) {
            guard let server = store.currentServer else { return }
            await viewModel.syncInsightJob(matchingInsightJob, using: server, track: track)
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
                subjectLabel: "曲目",
                selectedAIPlatform: $viewModel.selectedAIPlatform,
                selectedAIModel: $viewModel.selectedAIModel,
                availableAIPlatforms: viewModel.availableAIPlatforms,
                availableAIModels: viewModel.availableAIModels,
                isConfirmDisabled: viewModel.selectedAIPlatform.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    || viewModel.availableAIModels.isEmpty
                    || viewModel.insightGenerationState == .loadingModels
                    || viewModel.insightGenerationState == .generating
                    || viewModel.selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                onCancel: { viewModel.dismissModelPicker() },
                onConfirm: { confirmInsightGeneration() }
            )
                .padding(18)
                .frame(width: 360)
        }
        #else
        .sheet(isPresented: modelPickerPresentedBinding) {
            NavigationStack {
                InsightModelPickerContent(
                    subjectLabel: "曲目",
                    selectedAIPlatform: $viewModel.selectedAIPlatform,
                    selectedAIModel: $viewModel.selectedAIModel,
                    availableAIPlatforms: viewModel.availableAIPlatforms,
                    availableAIModels: viewModel.availableAIModels,
                    isConfirmDisabled: viewModel.selectedAIPlatform.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || viewModel.availableAIModels.isEmpty
                        || viewModel.insightGenerationState == .loadingModels
                        || viewModel.insightGenerationState == .generating
                        || viewModel.selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                    onCancel: { viewModel.dismissModelPicker() },
                    onConfirm: { confirmInsightGeneration() }
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

    @ToolbarContentBuilder
    private var exportMenu: some ToolbarContent {
        ToolbarItem(placement: .primaryAction) {
            Menu {
                Button("导出：基础信息") {
                    exportSnapshotPNG(infoSection.padding(32), suggestedFilename: "\(track.artist)-\(track.track)-信息")
                }
                Button("导出：歌词") {
                    exportSnapshotPNG(lyricsSection.padding(32), suggestedFilename: "\(track.artist)-\(track.track)-歌词")
                }
                Button("导出：音眸") {
                    exportSnapshotPNG(insightsSection.padding(32), suggestedFilename: "\(track.artist)-\(track.track)-音眸")
                }
            } label: {
                Label("导出快照", systemImage: "square.and.arrow.up")
            }
        }
    }

    @ToolbarContentBuilder
    private var shareMenu: some ToolbarContent {
        ToolbarItem(placement: .primaryAction) {
            Menu {
                Button("分享：基础信息") {
                    openSharePreview(scene: .trackInfo)
                }
                Button("分享：歌词") {
                    openSharePreview(scene: .trackLyrics)
                }
                Button("分享：音眸") {
                    openSharePreview(scene: .trackInsight)
                }
            } label: {
                Label("分享", systemImage: "square.and.arrow.up")
            }
        }
    }

    private var lyricsSection: some View {
        DetailSectionCard(title: "歌词", compact: isPhoneLayout) {
            HStack {
                Spacer()
                if viewModel.lyricLines.contains(where: { $0.time != nil }) {
                    Text("LRC")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
                }
            }

            if viewModel.lyricLines.isEmpty {
                Text("暂无歌词")
                    .foregroundColor(.secondary)
            } else {
                if let duration = track.duration, duration > 0, viewModel.lyricLines.contains(where: { $0.time != nil }) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("时间预览（仅用于歌词高亮，不会控制播放器）")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Slider(value: $previewTime, in: 0...TimeInterval(duration), step: 1)
                    }
                }

                LyricsPane(lines: viewModel.lyricLines, currentLineID: viewModel.currentLineID(forPreviewTime: previewTime))
                    .frame(minHeight: 260)
            }
        }
    }

    private var infoSection: some View {
        DetailSectionCard(title: "基础信息", compact: isPhoneLayout) {
            VStack(alignment: .leading, spacing: isPhoneLayout ? 8 : 10) {
                InfoRow(title: "曲目", value: track.track, compact: isPhoneLayout)
                InfoRow(title: "艺术家", value: track.artist, compact: isPhoneLayout)
                InfoRow(title: "专辑", value: track.album, compact: isPhoneLayout)
                InfoRow(title: "播放次数", value: "\(track.playCount)", compact: isPhoneLayout)
                if let duration = track.duration {
                    InfoRow(title: "时长", value: formatDuration(duration), compact: isPhoneLayout)
                }
                if let disc = track.discNumber {
                    InfoRow(title: "碟号", value: "\(disc)", compact: isPhoneLayout)
                }
                if let no = track.trackNumber {
                    InfoRow(title: "曲序", value: "\(no)", compact: isPhoneLayout)
                }
            }
        }
    }

    private var insightsSection: some View {
        DetailSectionCard(title: "操作", compact: isPhoneLayout) {
            VStack(alignment: .leading, spacing: 12) {
                insightActionRow

                if let message = viewModel.generationStatusMessage {
                    Text(message)
                        .font(.caption)
                        .foregroundStyle(viewModel.insightGenerationState == .error ? Color.red : Color.secondary)
                }

                if let inFlightHint = insightInFlightHint {
                    Text(inFlightHint)
                        .font(.caption)
                        .foregroundStyle(Color.orange)
                }

                if let currentInsight {
                    InsightDetailView(
                        insight: currentInsight,
                        allInsights: viewModel.insights,
                        selectedInsightIndex: $viewModel.selectedInsightIndex,
                        insightViewMode: $viewModel.insightViewMode,
                        showsContextHeader: false
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
    }

    private var insightActionRow: some View {
        HStack(spacing: 10) {
            if viewModel.insights.isEmpty {
                Button(action: startInsightGeneration) {
                    Label("生成音眸解析", systemImage: "sparkles")
                }
                .buttonStyle(.borderedProminent)
                .disabled(isInsightActionDisabled)
            } else {
                Button(action: startInsightGeneration) {
                    Label("重新生成", systemImage: "sparkles")
                }
                .buttonStyle(.bordered)
                .disabled(isInsightActionDisabled)
            }

            if viewModel.insightGenerationState == .loadingModels || viewModel.insightGenerationState == .generating {
                ProgressView()
                    .controlSize(.small)
            }
        }
    }

    private var isInsightActionDisabled: Bool {
        store.currentServer == nil
            || viewModel.insightGenerationState == .loadingModels
            || viewModel.insightGenerationState == .generating
    }

    private var isCurrentTrackFavorite: Bool {
        track.isFavorited
            || favoriteStore.favoriteKeys.contains(
                [track.artist, track.album, track.track, String(track.trackNumber ?? 0), String(track.discNumber ?? 0)].joined(separator: "•")
            )
    }

    private var currentInsight: Insight? {
        guard viewModel.insights.indices.contains(viewModel.selectedInsightIndex) else {
            return viewModel.insights.primaryInsight
        }
        return viewModel.insights[viewModel.selectedInsightIndex]
    }

    private func openSharePreview(scene: ShareScene) {
        let payload = SharePayloadBuilder.build(
            scene: scene,
            track: track,
            resolvedArtworkURL: viewModel.resolvedArtworkURL,
            lyrics: viewModel.lyrics,
            insight: currentInsight,
            isFavorite: isCurrentTrackFavorite
        )
        sharePreviewRequest = SharePreviewRequest(payload: payload)
    }

    private var insightInFlightHint: String? {
        switch viewModel.insightGenerationState {
        case .loadingModels:
            return "正在加载可用模型，请稍候。"
        case .generating:
            return "音眸解析可能持续数分钟，切到桌面后可通过灵动岛继续关注状态。"
        default:
            return nil
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

    private func startInsightGeneration() {
        guard let server = store.currentServer else { return }
        selectedTab = .insights
        Task {
            await viewModel.beginInsightGeneration(
                using: server,
                artist: track.artist,
                album: track.album,
                track: track.track,
                trackNumber: track.trackNumber,
                discNumber: track.discNumber
            )
        }
    }

    private func confirmInsightGeneration() {
        guard let server = store.currentServer else { return }
        selectedTab = .insights
        Task {
            await viewModel.confirmInsightGeneration(
                using: server,
                coordinator: insightCoordinator,
                track: track
            )
        }
    }

    private var matchingInsightJob: InsightAnalysisJob? {
        guard let activeJob = insightCoordinator.activeJob, activeJob.matches(track: track) else {
            return nil
        }
        return activeJob
    }

    private var insightJobTaskToken: String {
        guard let job = matchingInsightJob else { return "none" }
        return "\(job.id)::\(job.phase.rawValue)::\(job.updatedAt ?? "")"
    }


    private func formatDuration(_ duration: Int64) -> String {
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}

enum TrackDetailTab: String, Hashable {
    case info
    case lyrics
    case insights
}

struct LyricsPane: View {
    let lines: [LyricLine]
    let currentLineID: UUID?

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 12) {
                    if lines.isEmpty {
                        Text("暂无歌词")
                            .font(.body)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(lines) { line in
                            let isCurrent = line.id == currentLineID
                            Group {
                                if line.isSectionLabel {
                                    Text(line.text)
                                        .font(.system(size: 11, weight: .semibold))
                                        .tracking(2.0)
                                        .textCase(.uppercase)
                                        .foregroundStyle(.secondary)
                                        .frame(maxWidth: .infinity, alignment: .center)
                                        .padding(.vertical, 6)
                                } else {
                                    Text(line.text)
                                        .font(isCurrent ? .title3 : .body)
                                        .fontWeight(isCurrent ? .semibold : .regular)
                                        .foregroundStyle(isCurrent ? Color.primary : Color.secondary)
                                        .padding(.vertical, 4)
                                        .padding(.horizontal, 6)
                                        .background(isCurrent ? Color.accentColor.opacity(0.12) : Color.clear)
                                        .clipShape(RoundedRectangle(cornerRadius: 6))
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                }
                            }
                            .id(line.id)
                        }
                    }
                }
                .padding(16)
            }
            .mask(
                LinearGradient(
                    colors: [.clear, .black, .black, .clear],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )
            .onChange(of: currentLineID) { _, lineID in
                guard let lineID else { return }
                withAnimation(.easeInOut(duration: 0.35)) {
                    proxy.scrollTo(lineID, anchor: .center)
                }
            }
        }
    }
}

struct InfoRow: View {
    let title: String
    let value: String
    var compact: Bool = false

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(title)
                .font(compact ? .caption2 : .caption)
                .foregroundStyle(.secondary)
                .frame(width: compact ? 56 : 64, alignment: .leading)
            Text(value)
                .font(compact ? .subheadline : .body)
                .foregroundStyle(.primary)
            Spacer()
        }
    }
}

struct InsightDetailCard: View {
    let insight: Insight

    var body: some View {
        InsightPrimaryContentView(
            insight: insight,
            style: .detail,
            emptyTitle: "暂无音眸",
            emptySubtitle: "当前曲目还没有可展示的音眸内容。"
        )
    }
}

enum TrackDetailHeaderLayout {
    case regular
    case phone

    var artworkSize: CGFloat {
        switch self {
        case .regular:
            return 160
        case .phone:
            return 124
        }
    }
}

struct TrackDetailHeader: View {
    let track: Track
    let playCount: Int
    let artworkURL: String?
    let isFavorite: Bool
    let layout: TrackDetailHeaderLayout
    @EnvironmentObject private var store: AppStore

    var body: some View {
        switch layout {
        case .regular:
            HStack(spacing: 24) {
                ArtworkSquareView(
                    artworkURL: artworkURL,
                    fallbackTitle: track.album,
                    size: layout.artworkSize,
                    cornerRadius: 16,
                    style: .vivid
                )
                    .shadow(color: Color.black.opacity(0.16), radius: 24, x: 0, y: 16)

                VStack(alignment: .leading, spacing: 10) {
                    Text(track.track)
                        .font(.title2)
                        .fontWeight(.semibold)
                    Text("\(track.artist) · \(track.album)")
                        .font(.body)
                        .foregroundStyle(.secondary)

                    HStack(spacing: 16) {
                        DetailMetaChip(title: "播放次数", value: "\(playCount)")
                        if let duration = track.duration {
                            DetailMetaChip(title: "时长", value: formatDuration(duration))
                        }
                    }
                }
                Spacer()
                favoriteButton
            }
            .padding(18)
            .glassCard(cornerRadius: 16)
        case .phone:
            VStack(alignment: .leading, spacing: 14) {
                HStack(alignment: .top, spacing: 14) {
                    ArtworkSquareView(
                        artworkURL: artworkURL,
                        fallbackTitle: track.album,
                        size: layout.artworkSize,
                        cornerRadius: 14,
                        style: .vivid
                    )
                        .shadow(color: Color.black.opacity(0.14), radius: 18, x: 0, y: 12)

                    Spacer(minLength: 12)

                    favoriteButton
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text(track.track)
                        .font(.title3)
                        .fontWeight(.semibold)
                    Text("\(track.artist) · \(track.album)")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)

                    HStack(spacing: 12) {
                        DetailMetaChip(title: "播放次数", value: "\(playCount)")
                        if let duration = track.duration {
                            DetailMetaChip(title: "时长", value: formatDuration(duration))
                        }
                    }
                }
            }
            .padding(16)
            .glassCard(cornerRadius: 16)
        }
    }

    private var favoriteButton: some View {
        FavoriteButton(
            isFavorite: isFavorite,
            action: {
                Task {
                    await store.toggleFavorite(
                        artist: track.artist,
                        album: track.album,
                        track: track.track,
                        trackNumber: track.trackNumber,
                        discNumber: track.discNumber
                    )
                }
            }
        )
    }

    private func formatDuration(_ duration: Int64) -> String {
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}

struct DetailMetaChip: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct FavoriteButton: View {
    let isFavorite: Bool
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            Image(systemName: isFavorite ? "heart.fill" : "heart")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(isFavorite ? Color.red : SonicTheme.textPrimary)
                .frame(width: 32, height: 32)
                .background(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .fill(isHovered ? Color.primary.opacity(0.12) : Color.primary.opacity(0.06))
                )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
        .help(isFavorite ? "取消收藏" : "收藏")
    }
}
