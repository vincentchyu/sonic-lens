import SwiftUI
import Foundation

private enum InsightWorkbenchSelection: Hashable {
    case primary
    case history(Int64)
}

private struct InsightWorkbenchVersionCard: Identifiable {
    let id: Int64
    let selection: InsightWorkbenchSelection
    let title: String
    let provider: String?
    let createdAt: String?
    let score: Int?
    let feedbackStatusText: String
    let analysisSummary: String?
    let badges: [String]
}

private func compareInsightWorkbenchVersionCards(
    _ lhs: InsightWorkbenchVersionCard,
    _ rhs: InsightWorkbenchVersionCard
) -> Bool {
    let lhsScore = lhs.score ?? 0
    let rhsScore = rhs.score ?? 0
    if lhsScore != rhsScore {
        return lhsScore > rhsScore
    }

    let lhsCreatedAt = lhs.createdAt ?? ""
    let rhsCreatedAt = rhs.createdAt ?? ""
    if lhsCreatedAt != rhsCreatedAt {
        return lhsCreatedAt > rhsCreatedAt
    }

    return lhs.id > rhs.id
}

private enum InsightWorkbenchDeviceClass {
    case phone
    case pad
    case desktop

    static var current: InsightWorkbenchDeviceClass {
        #if os(iOS)
        switch UIDevice.current.userInterfaceIdiom {
        case .phone:
            return .phone
        case .pad:
            return .pad
        case .mac:
            return .desktop
        default:
            return .desktop
        }
        #else
        return .desktop
        #endif
    }
}

private struct InsightWorkbenchLayoutMetrics {
    let outerPadding: CGFloat
    let pageSectionSpacing: CGFloat
    let headerContentSpacing: CGFloat
    let headerBadgeSpacing: CGFloat
    let contentSpacing: CGFloat
    let columnSpacing: CGFloat
    let railWidth: CGFloat

    static func make(for deviceClass: InsightWorkbenchDeviceClass) -> InsightWorkbenchLayoutMetrics {
        switch deviceClass {
        case .phone:
            return InsightWorkbenchLayoutMetrics(
                outerPadding: 16,
                pageSectionSpacing: 20,
                headerContentSpacing: 14,
                headerBadgeSpacing: 10,
                contentSpacing: 14,
                columnSpacing: 14,
                railWidth: .infinity
            )
        case .pad:
            return InsightWorkbenchLayoutMetrics(
                outerPadding: 24,
                pageSectionSpacing: 24,
                headerContentSpacing: 18,
                headerBadgeSpacing: 12,
                contentSpacing: 18,
                columnSpacing: 18,
                railWidth: 320
            )
        case .desktop:
            return InsightWorkbenchLayoutMetrics(
                outerPadding: 32,
                pageSectionSpacing: 28,
                headerContentSpacing: 22,
                headerBadgeSpacing: 12,
                contentSpacing: 24,
                columnSpacing: 24,
                railWidth: 336
            )
        }
    }
}

struct InsightDetailView: View {
    @EnvironmentObject private var store: AppStore

    let insight: Insight
    let allInsights: [Insight]
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode
    let showsContextHeader: Bool

    @StateObject private var feedbackViewModel = InsightFeedbackViewModel()
    @State private var historySummaries: [InsightSummary] = []
    @State private var recommendedInsightID: Int64?
    @State private var selectedWorkbenchVersion: InsightWorkbenchSelection = .primary
    @State private var primaryInsightOverride: Insight?
    @State private var historyPreviewInsight: Insight?
    @State private var historyPreviewCache: [Int64: Insight] = [:]
    @State private var historyLoading: Bool = false
    @State private var historyPreviewLoading: Bool = false
    @State private var historyErrorMessage: String?
    @State private var historyPreviewErrorMessage: String?
    @State private var historyLoadedForInsightID: Int64?
    @State private var historyPreviewRequestToken = UUID()
    @State private var historyPreviewTask: Task<Void, Never>?

    init(
        insight: Insight,
        allInsights: [Insight] = [],
        selectedInsightIndex: Binding<Int> = .constant(0),
        insightViewMode: Binding<InsightViewMode> = .constant(.current),
        showsContextHeader: Bool = true
    ) {
        self.insight = insight
        self.allInsights = allInsights
        self._selectedInsightIndex = selectedInsightIndex
        self._insightViewMode = insightViewMode
        self.showsContextHeader = showsContextHeader
    }

    private var isPhoneLayout: Bool {
        deviceClass == .phone
    }

    private var deviceClass: InsightWorkbenchDeviceClass {
        InsightWorkbenchDeviceClass.current
    }

    private var layoutMetrics: InsightWorkbenchLayoutMetrics {
        .make(for: deviceClass)
    }

    private var versionCards: [InsightWorkbenchVersionCard] {
        let primaryInsight = primaryWorkbenchInsight
        let currentScore = summaryForInsightID(primaryInsight.id)?.totalScore ?? primaryInsight.totalScore ?? 0
        let currentCard = InsightWorkbenchVersionCard(
            id: primaryInsight.id,
            selection: .primary,
            title: "当前推荐",
            provider: primaryInsight.llmProvider,
            createdAt: primaryInsight.createdAt,
            score: currentScore,
            feedbackStatusText: summaryForInsightID(primaryInsight.id)?.feedbackStatusText ?? "未评价",
            analysisSummary: primaryInsight.analysisSummary,
            badges: primaryVersionBadges
        )
        let historyCards = availableHistorySummaries.map { summary in
            InsightWorkbenchVersionCard(
                id: summary.id,
                selection: .history(summary.id),
                title: "历史版本",
                provider: summary.llmProvider,
                createdAt: summary.createdAt,
                score: summary.totalScore,
                feedbackStatusText: summary.feedbackStatusText,
                analysisSummary: summary.analysisSummary,
                badges: badges(for: summary)
            )
        }
        return ([currentCard] + historyCards).sorted(by: compareInsightWorkbenchVersionCards)
    }

    private var availableHistorySummaries: [InsightSummary] {
        historySummaries.filter { $0.id != primaryWorkbenchInsight.id }
    }

    private var primaryWorkbenchInsight: Insight {
        primaryInsightOverride ?? insight
    }

    private var selectedHistoryInsightID: Int64? {
        guard case let .history(id) = selectedWorkbenchVersion else { return nil }
        return id
    }

    private var displayedInsightID: Int64 {
        if let preview = historyPreviewInsight,
           preview.id == selectedHistoryInsightID {
            return preview.id
        }
        if let selectedHistoryInsightID {
            return selectedHistoryInsightID
        }
        return primaryWorkbenchInsight.id
    }

    private var displayedInsight: Insight {
        if let preview = historyPreviewInsight,
           preview.id == selectedHistoryInsightID {
            return preview
        }
        return primaryWorkbenchInsight
    }

    private var versionCountText: String {
        "共 \(1 + availableHistorySummaries.count) 个版本"
    }

    private var bestVersionID: Int64? {
        let currentInsight = primaryWorkbenchInsight
        let currentScore = summaryForInsightID(currentInsight.id)?.totalScore ?? currentInsight.totalScore ?? 0
        let historyBest = availableHistorySummaries.max { lhs, rhs in
            lhs.totalScore < rhs.totalScore
        }
        if let historyBest, historyBest.totalScore > currentScore {
            return historyBest.id
        }
        return currentInsight.id
    }

    private var feedbackTaskToken: String {
        guard let server = store.currentServer else { return "no-server" }
        return "\(server.id.uuidString)-\(displayedInsightID)"
    }

    private var historyLoadToken: String {
        "\(insight.id)"
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: layoutMetrics.pageSectionSpacing) {
                if showsContextHeader {
                    InsightDetailHeader(insight: insight)
                }

                VStack(alignment: .leading, spacing: layoutMetrics.headerContentSpacing) {
                    workbenchHeader
                    workbenchContent
                }
            }
            .padding(layoutMetrics.outerPadding)
        }
        .navigationTitle("曲目详情")
        .task(id: feedbackTaskToken) {
            await loadFeedbackIfNeeded()
        }
        .task(id: historyLoadToken) {
            await loadHistoryIfNeeded()
        }
        .onChange(of: insight.id) { _, newValue in
            if selectedInsightIndex != 0 {
                selectedInsightIndex = 0
            }
            cancelHistoryPreview()
            recommendedInsightID = nil
            selectedWorkbenchVersion = .primary
            primaryInsightOverride = nil
            historyPreviewInsight = nil
            historyPreviewCache = [:]
            historySummaries = []
            historyLoadedForInsightID = nil
            historyErrorMessage = nil
            historyPreviewErrorMessage = nil
            if newValue > 0 {
                feedbackViewModel.reset()
            }
        }
    }

    private var workbenchHeader: some View {
        Group {
            if isPhoneLayout {
                VStack(alignment: .leading, spacing: layoutMetrics.headerBadgeSpacing) {
                    workbenchHeaderText(
                        subtitle: "版本按总分优先排序，当前推荐会单独标记，选择任意版本后右侧只切换预览和反馈对象。"
                    )
                    versionCountBadge
                }
            } else {
                HStack(alignment: .center, spacing: 12) {
                    workbenchHeaderText(
                        subtitle: "版本按总分优先排序，当前推荐会单独标记，选择任意版本后右侧只切换预览和反馈对象。"
                    )
                    Spacer()
                    versionCountBadge
                }
            }
        }
    }

    @ViewBuilder
    private var workbenchContent: some View {
        if isPhoneLayout {
            VStack(alignment: .leading, spacing: layoutMetrics.contentSpacing) {
                versionRailCard
                versionPreviewCard
            }
        } else {
            HStack(alignment: .top, spacing: layoutMetrics.columnSpacing) {
                versionRailCard
                    .frame(maxWidth: layoutMetrics.railWidth)
                versionPreviewCard
                    .frame(maxWidth: .infinity, alignment: .topLeading)
            }
        }
    }

    private func workbenchHeaderText(subtitle: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("版本工作台")
                .font(.headline.weight(.semibold))
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private var versionCountBadge: some View {
        Text(versionCountText)
            .font(.caption.weight(.semibold))
            .foregroundStyle(.secondary)
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Color.primary.opacity(0.06), in: Capsule())
    }

    private var versionRailCard: some View {
        DetailSectionCard(title: "版本列表", compact: false) {
            VStack(alignment: .leading, spacing: 12) {
                Text("版本按总分降序排列，分数相同再按创建时间新旧排序。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if historyLoading && availableHistorySummaries.isEmpty {
                    ProgressView("正在加载历史版本...")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 12)
                } else {
                    VStack(spacing: 10) {
                        if let historyErrorMessage, availableHistorySummaries.isEmpty {
                            Text(historyErrorMessage)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8)
                        }

                        ForEach(versionCards) { card in
                            InsightWorkbenchVersionRow(
                                title: card.title,
                                provider: card.provider,
                                createdAt: card.createdAt,
                                score: card.score,
                                feedbackStatusText: card.feedbackStatusText,
                                analysisSummary: card.analysisSummary,
                                isSelected: selectedWorkbenchVersion == card.selection,
                                badges: card.badges
                            ) {
                                switch card.selection {
                                case .primary:
                                    selectPrimaryVersion()
                                case let .history(id):
                                    guard let summary = availableHistorySummaries.first(where: { $0.id == id }) else { return }
                                    selectHistorySummary(summary)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    private var versionPreviewCard: some View {
        DetailSectionCard(title: previewCardTitle, compact: false) {
            VStack(alignment: .leading, spacing: 14) {
                if historyPreviewLoading {
                    ProgressView("正在加载所选版本...")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 18)
                } else if let historyPreviewErrorMessage {
                    versionPreviewErrorState(message: historyPreviewErrorMessage)
                } else {
                    previewIdentityLine

                    InsightPrimaryContentView(
                        insight: displayedInsight,
                        style: .detail,
                        emptyTitle: "暂无音眸",
                        emptySubtitle: "当前选中版本还没有可展示的音眸内容。"
                    )

                    InsightFeedbackSection(
                        summary: feedbackViewModel.summary,
                        history: feedbackViewModel.history,
                        isLoading: feedbackViewModel.isLoading,
                        isSubmitting: feedbackViewModel.isSubmitting,
                        errorMessage: feedbackViewModel.errorMessage,
                        isCompact: false,
                        onHelpful: submitHelpfulFeedback,
                        onSubmitIssue: submitIssueFeedback
                    )
                }
            }
        }
    }

    private func loadHistoryIfNeeded() async {
        guard let server = store.currentServer else {
            historyLoading = false
            historyErrorMessage = "当前未连接服务器"
            return
        }
        guard historyLoadedForInsightID != insight.id else { return }

        historyLoading = true
        historyErrorMessage = nil
        historyPreviewErrorMessage = nil
        historyPreviewInsight = nil
        do {
            let response = try await InsightAPIClient.fetchTrackInsightHistory(using: server, id: insight.id)
            historySummaries = response.insights
            recommendedInsightID = response.recommendedInsightID
            historyLoadedForInsightID = insight.id
            await loadPrimaryInsightIfNeeded(recommendedID: response.recommendedInsightID, server: server)
            if case let .history(id) = selectedWorkbenchVersion,
               !availableHistorySummaries.contains(where: { $0.id == id }) {
                selectedWorkbenchVersion = .primary
                historyPreviewInsight = nil
            }
        } catch {
            historyErrorMessage = "历史版本加载失败"
        }
        historyLoading = false
    }

    private func selectPrimaryVersion() {
        cancelHistoryPreview()
        selectedWorkbenchVersion = .primary
        historyPreviewRequestToken = UUID()
        historyPreviewInsight = nil
        historyPreviewErrorMessage = nil
        historyPreviewLoading = false
    }

    private func selectHistorySummary(_ summary: InsightSummary) {
        guard selectedWorkbenchVersion != .history(summary.id) || historyPreviewInsight?.id != summary.id else { return }
        cancelHistoryPreview()
        selectedWorkbenchVersion = .history(summary.id)
        historyPreviewErrorMessage = nil
        if let cached = historyPreviewCache[summary.id] {
            historyPreviewInsight = cached
            historyPreviewLoading = false
            return
        }
        historyPreviewInsight = nil
        historyPreviewLoading = true
        let requestToken = UUID()
        historyPreviewRequestToken = requestToken
        historyPreviewTask = Task {
            await loadHistoryPreview(id: summary.id, requestToken: requestToken)
        }
    }

    private func loadHistoryPreview(id: Int64, requestToken: UUID) async {
        guard let server = store.currentServer else {
            historyPreviewLoading = false
            historyPreviewErrorMessage = "当前未连接服务器"
            return
        }

        historyPreviewLoading = true
        historyPreviewErrorMessage = nil
        do {
            let detail = try await InsightAPIClient.fetchTrackInsightDetail(using: server, id: id)
            try Task.checkCancellation()
            guard requestToken == historyPreviewRequestToken else { return }
            historyPreviewInsight = detail
            historyPreviewCache[id] = detail
        } catch is CancellationError {
            return
        } catch {
            guard requestToken == historyPreviewRequestToken else { return }
            historyPreviewErrorMessage = "历史版本详情加载失败"
        }
        if requestToken == historyPreviewRequestToken {
            historyPreviewTask = nil
            historyPreviewLoading = false
        }
    }

    private func loadFeedbackIfNeeded() async {
        guard let server = store.currentServer else {
            feedbackViewModel.reset()
            return
        }

        await feedbackViewModel.load(using: server, insightID: displayedInsightID, targetType: .track)
    }

    private func submitHelpfulFeedback() {
        guard let server = store.currentServer else { return }
        Task {
            await feedbackViewModel.submitHelpful(using: server, insightID: displayedInsightID, targetType: .track)
        }
    }

    private func submitIssueFeedback(_ draft: InsightIssueDraft) {
        guard let server = store.currentServer else { return }
        Task {
            await feedbackViewModel.submitIssue(using: server, insightID: displayedInsightID, targetType: .track, draft: draft)
        }
    }

    private func cancelHistoryPreview() {
        historyPreviewTask?.cancel()
        historyPreviewTask = nil
    }

    private var previewCardTitle: String {
        selectedHistoryInsightID == nil ? "当前推荐版本" : "版本预览"
    }

    private var primaryVersionBadges: [String] {
        var items = ["当前推荐"]
        if latestVersionID == primaryWorkbenchInsight.id {
            items.append("最新")
        }
        if bestVersionID == primaryWorkbenchInsight.id {
            items.append("最佳")
        }
        return items
    }

    private func badges(for summary: InsightSummary) -> [String] {
        var items: [String] = []
        if summary.id == bestVersionID {
            items.append("最佳")
        }
        return items
    }

    private func summaryForInsightID(_ id: Int64) -> InsightSummary? {
        historySummaries.first(where: { $0.id == id })
    }

    private var latestVersionID: Int64? {
        historySummaries.max { lhs, rhs in
            let leftCreatedAt = lhs.createdAt ?? ""
            let rightCreatedAt = rhs.createdAt ?? ""
            if leftCreatedAt != rightCreatedAt {
                return leftCreatedAt < rightCreatedAt
            }
            return lhs.id < rhs.id
        }?.id ?? primaryWorkbenchInsight.id
    }

    private func loadPrimaryInsightIfNeeded(recommendedID: Int64?, server: ServerConfig) async {
        guard let recommendedID, recommendedID > 0 else {
            primaryInsightOverride = nil
            return
        }
        guard recommendedID != insight.id else {
            primaryInsightOverride = nil
            return
        }
        if let cached = historyPreviewCache[recommendedID] {
            primaryInsightOverride = cached
            return
        }
        do {
            let detail = try await InsightAPIClient.fetchTrackInsightDetail(using: server, id: recommendedID)
            historyPreviewCache[recommendedID] = detail
            primaryInsightOverride = detail
        } catch {
            primaryInsightOverride = nil
        }
    }

    @ViewBuilder
    private var previewIdentityLine: some View {
        HStack(alignment: .center, spacing: 8) {
            Text(selectedHistoryInsightID == nil ? "当前推荐版本" : "已选历史版本")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            if let providerLine = displayedInsight.providerLine {
                Text(providerLine)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
        }
    }

    @ViewBuilder
    private func versionPreviewErrorState(message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(message)
                .font(.body)
                .foregroundStyle(.secondary)
            Button("返回当前推荐版本") {
                selectPrimaryVersion()
            }
            .buttonStyle(.bordered)
        }
        .padding(.vertical, 8)
    }
}

struct AlbumInsightDetailView: View {
    @EnvironmentObject private var store: AppStore

    let insight: AlbumInsight
    let allInsights: [AlbumInsight]
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode
    let showsContextHeader: Bool

    @StateObject private var feedbackViewModel = InsightFeedbackViewModel()
    @State private var historySummaries: [InsightSummary] = []
    @State private var recommendedInsightID: Int64?
    @State private var selectedWorkbenchVersion: InsightWorkbenchSelection = .primary
    @State private var primaryInsightOverride: AlbumInsight?
    @State private var historyPreviewInsight: AlbumInsight?
    @State private var historyPreviewCache: [Int64: AlbumInsight] = [:]
    @State private var historyLoading: Bool = false
    @State private var historyPreviewLoading: Bool = false
    @State private var historyErrorMessage: String?
    @State private var historyPreviewErrorMessage: String?
    @State private var historyLoadedForInsightID: Int64?
    @State private var historyPreviewRequestToken = UUID()
    @State private var historyPreviewTask: Task<Void, Never>?

    init(
        insight: AlbumInsight,
        allInsights: [AlbumInsight] = [],
        selectedInsightIndex: Binding<Int> = .constant(0),
        insightViewMode: Binding<InsightViewMode> = .constant(.current),
        showsContextHeader: Bool = true
    ) {
        self.insight = insight
        self.allInsights = allInsights
        self._selectedInsightIndex = selectedInsightIndex
        self._insightViewMode = insightViewMode
        self.showsContextHeader = showsContextHeader
    }

    private var isPhoneLayout: Bool {
        deviceClass == .phone
    }

    private var deviceClass: InsightWorkbenchDeviceClass {
        InsightWorkbenchDeviceClass.current
    }

    private var layoutMetrics: InsightWorkbenchLayoutMetrics {
        .make(for: deviceClass)
    }

    private var versionCards: [InsightWorkbenchVersionCard] {
        let primaryInsight = primaryWorkbenchInsight
        let currentScore = summaryForInsightID(primaryInsight.id)?.totalScore ?? 0
        let currentCard = InsightWorkbenchVersionCard(
            id: primaryInsight.id,
            selection: .primary,
            title: "当前推荐",
            provider: primaryInsight.llmProvider,
            createdAt: primaryInsight.createdAt,
            score: currentScore,
            feedbackStatusText: summaryForInsightID(primaryInsight.id)?.feedbackStatusText ?? "未评价",
            analysisSummary: primaryInsight.analysisSummary,
            badges: primaryVersionBadges
        )
        let historyCards = availableHistorySummaries.map { summary in
            InsightWorkbenchVersionCard(
                id: summary.id,
                selection: .history(summary.id),
                title: "历史版本",
                provider: summary.llmProvider,
                createdAt: summary.createdAt,
                score: summary.totalScore,
                feedbackStatusText: summary.feedbackStatusText,
                analysisSummary: summary.analysisSummary,
                badges: badges(for: summary)
            )
        }
        return ([currentCard] + historyCards).sorted(by: compareInsightWorkbenchVersionCards)
    }

    private var availableHistorySummaries: [InsightSummary] {
        historySummaries.filter { $0.id != primaryWorkbenchInsight.id }
    }

    private var primaryWorkbenchInsight: AlbumInsight {
        primaryInsightOverride ?? insight
    }

    private var selectedHistoryInsightID: Int64? {
        guard case let .history(id) = selectedWorkbenchVersion else { return nil }
        return id
    }

    private var displayedInsightID: Int64 {
        if let preview = historyPreviewInsight,
           preview.id == selectedHistoryInsightID {
            return preview.id
        }
        if let selectedHistoryInsightID {
            return selectedHistoryInsightID
        }
        return primaryWorkbenchInsight.id
    }

    private var displayedInsight: AlbumInsight {
        if let preview = historyPreviewInsight,
           preview.id == selectedHistoryInsightID {
            return preview
        }
        return primaryWorkbenchInsight
    }

    private var versionCountText: String {
        "共 \(1 + availableHistorySummaries.count) 个版本"
    }

    private var bestVersionID: Int64? {
        let currentInsight = primaryWorkbenchInsight
        let currentScore = summaryForInsightID(currentInsight.id)?.totalScore ?? 0
        let historyBest = availableHistorySummaries.max { lhs, rhs in
            lhs.totalScore < rhs.totalScore
        }
        if let historyBest, historyBest.totalScore > currentScore {
            return historyBest.id
        }
        return currentInsight.id
    }

    private var feedbackTaskToken: String {
        guard let server = store.currentServer else { return "no-server" }
        return "\(server.id.uuidString)-album-\(displayedInsightID)"
    }

    private var historyLoadToken: String {
        "\(insight.id)"
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: layoutMetrics.pageSectionSpacing) {
                if showsContextHeader {
                    AlbumInsightDetailHeader(insight: insight)
                }

                VStack(alignment: .leading, spacing: layoutMetrics.headerContentSpacing) {
                    workbenchHeader
                    workbenchContent
                }
            }
            .padding(layoutMetrics.outerPadding)
        }
        .navigationTitle("专辑详情")
        .task(id: feedbackTaskToken) {
            await loadFeedbackIfNeeded()
        }
        .task(id: historyLoadToken) {
            await loadHistoryIfNeeded()
        }
        .onChange(of: insight.id) { _, newValue in
            if selectedInsightIndex != 0 {
                selectedInsightIndex = 0
            }
            cancelHistoryPreview()
            recommendedInsightID = nil
            selectedWorkbenchVersion = .primary
            primaryInsightOverride = nil
            historyPreviewInsight = nil
            historyPreviewCache = [:]
            historySummaries = []
            historyLoadedForInsightID = nil
            historyErrorMessage = nil
            historyPreviewErrorMessage = nil
            if newValue > 0 {
                feedbackViewModel.reset()
            }
        }
    }

    private var workbenchHeader: some View {
        Group {
            if isPhoneLayout {
                VStack(alignment: .leading, spacing: layoutMetrics.headerBadgeSpacing) {
                    workbenchHeaderText(
                        subtitle: "版本按总分优先排序，当前推荐会单独标记，选择任意版本后右侧只切换预览和反馈对象。"
                    )
                    versionCountBadge
                }
            } else {
                HStack(alignment: .center, spacing: 12) {
                    workbenchHeaderText(
                        subtitle: "版本按总分优先排序，当前推荐会单独标记，选择任意版本后右侧只切换预览和反馈对象。"
                    )
                    Spacer()
                    versionCountBadge
                }
            }
        }
    }

    @ViewBuilder
    private var workbenchContent: some View {
        if isPhoneLayout {
            VStack(alignment: .leading, spacing: layoutMetrics.contentSpacing) {
                versionRailCard
                versionPreviewCard
            }
        } else {
            HStack(alignment: .top, spacing: layoutMetrics.columnSpacing) {
                versionRailCard
                    .frame(maxWidth: layoutMetrics.railWidth)
                versionPreviewCard
                    .frame(maxWidth: .infinity, alignment: .topLeading)
            }
        }
    }

    private func workbenchHeaderText(subtitle: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("版本工作台")
                .font(.headline.weight(.semibold))
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private var versionCountBadge: some View {
        Text(versionCountText)
            .font(.caption.weight(.semibold))
            .foregroundStyle(.secondary)
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Color.primary.opacity(0.06), in: Capsule())
    }

    private var versionRailCard: some View {
        DetailSectionCard(title: "版本列表", compact: false) {
            VStack(alignment: .leading, spacing: 12) {
                Text("版本按总分降序排列，分数相同再按创建时间新旧排序。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if historyLoading && availableHistorySummaries.isEmpty {
                    ProgressView("正在加载历史版本...")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 12)
                } else {
                    VStack(spacing: 10) {
                        if let historyErrorMessage, availableHistorySummaries.isEmpty {
                            Text(historyErrorMessage)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8)
                        }

                        ForEach(versionCards) { card in
                            InsightWorkbenchVersionRow(
                                title: card.title,
                                provider: card.provider,
                                createdAt: card.createdAt,
                                score: card.score,
                                feedbackStatusText: card.feedbackStatusText,
                                analysisSummary: card.analysisSummary,
                                isSelected: selectedWorkbenchVersion == card.selection,
                                badges: card.badges
                            ) {
                                switch card.selection {
                                case .primary:
                                    selectPrimaryVersion()
                                case let .history(id):
                                    guard let summary = availableHistorySummaries.first(where: { $0.id == id }) else { return }
                                    selectHistorySummary(summary)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    private var versionPreviewCard: some View {
        DetailSectionCard(title: previewCardTitle, compact: false) {
            VStack(alignment: .leading, spacing: 14) {
                if historyPreviewLoading {
                    ProgressView("正在加载所选版本...")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 18)
                } else if let historyPreviewErrorMessage {
                    versionPreviewErrorState(message: historyPreviewErrorMessage)
                } else {
                    previewIdentityLine

                    AlbumInsightPrimaryContentView(
                        insight: displayedInsight,
                        compact: false,
                        emptyTitle: "暂无专辑音眸",
                        emptySubtitle: "当前选中版本还没有可展示的音眸内容。"
                    )

                    InsightFeedbackSection(
                        summary: feedbackViewModel.summary,
                        history: feedbackViewModel.history,
                        isLoading: feedbackViewModel.isLoading,
                        isSubmitting: feedbackViewModel.isSubmitting,
                        errorMessage: feedbackViewModel.errorMessage,
                        isCompact: false,
                        onHelpful: submitHelpfulFeedback,
                        onSubmitIssue: submitIssueFeedback
                    )
                }
            }
        }
    }

    private func loadHistoryIfNeeded() async {
        guard let server = store.currentServer else {
            historyLoading = false
            historyErrorMessage = "当前未连接服务器"
            return
        }
        guard historyLoadedForInsightID != insight.id else { return }

        historyLoading = true
        historyErrorMessage = nil
        historyPreviewErrorMessage = nil
        historyPreviewInsight = nil
        do {
            let response = try await InsightAPIClient.fetchAlbumInsightHistory(using: server, id: insight.id)
            historySummaries = response.insights
            recommendedInsightID = response.recommendedInsightID
            historyLoadedForInsightID = insight.id
            await loadPrimaryInsightIfNeeded(recommendedID: response.recommendedInsightID, server: server)
            if case let .history(id) = selectedWorkbenchVersion,
               !availableHistorySummaries.contains(where: { $0.id == id }) {
                selectedWorkbenchVersion = .primary
                historyPreviewInsight = nil
            }
        } catch {
            historyErrorMessage = "历史版本加载失败"
        }
        historyLoading = false
    }

    private func selectPrimaryVersion() {
        cancelHistoryPreview()
        selectedWorkbenchVersion = .primary
        historyPreviewRequestToken = UUID()
        historyPreviewInsight = nil
        historyPreviewErrorMessage = nil
        historyPreviewLoading = false
    }

    private func selectHistorySummary(_ summary: InsightSummary) {
        guard selectedWorkbenchVersion != .history(summary.id) || historyPreviewInsight?.id != summary.id else { return }
        cancelHistoryPreview()
        selectedWorkbenchVersion = .history(summary.id)
        historyPreviewErrorMessage = nil
        if let cached = historyPreviewCache[summary.id] {
            historyPreviewInsight = cached
            historyPreviewLoading = false
            return
        }
        historyPreviewInsight = nil
        historyPreviewLoading = true
        let requestToken = UUID()
        historyPreviewRequestToken = requestToken
        historyPreviewTask = Task {
            await loadHistoryPreview(id: summary.id, requestToken: requestToken)
        }
    }

    private func loadHistoryPreview(id: Int64, requestToken: UUID) async {
        guard let server = store.currentServer else {
            historyPreviewLoading = false
            historyPreviewErrorMessage = "当前未连接服务器"
            return
        }

        historyPreviewLoading = true
        historyPreviewErrorMessage = nil
        do {
            let detail = try await InsightAPIClient.fetchAlbumInsightDetail(using: server, id: id)
            try Task.checkCancellation()
            guard requestToken == historyPreviewRequestToken else { return }
            historyPreviewInsight = detail
            historyPreviewCache[id] = detail
        } catch is CancellationError {
            return
        } catch {
            guard requestToken == historyPreviewRequestToken else { return }
            historyPreviewErrorMessage = "历史版本详情加载失败"
        }
        if requestToken == historyPreviewRequestToken {
            historyPreviewTask = nil
            historyPreviewLoading = false
        }
    }

    private func loadFeedbackIfNeeded() async {
        guard let server = store.currentServer else {
            feedbackViewModel.reset()
            return
        }

        await feedbackViewModel.load(using: server, insightID: displayedInsightID, targetType: .album)
    }

    private func submitHelpfulFeedback() {
        guard let server = store.currentServer else { return }
        Task {
            await feedbackViewModel.submitHelpful(using: server, insightID: displayedInsightID, targetType: .album)
        }
    }

    private func submitIssueFeedback(_ draft: InsightIssueDraft) {
        guard let server = store.currentServer else { return }
        Task {
            await feedbackViewModel.submitIssue(using: server, insightID: displayedInsightID, targetType: .album, draft: draft)
        }
    }

    private func cancelHistoryPreview() {
        historyPreviewTask?.cancel()
        historyPreviewTask = nil
    }

    private var previewCardTitle: String {
        selectedHistoryInsightID == nil ? "当前推荐版本" : "版本预览"
    }

    private var primaryVersionBadges: [String] {
        var items = ["当前推荐"]
        if latestVersionID == primaryWorkbenchInsight.id {
            items.append("最新")
        }
        if bestVersionID == primaryWorkbenchInsight.id {
            items.append("最佳")
        }
        return items
    }

    private func badges(for summary: InsightSummary) -> [String] {
        var items: [String] = []
        if summary.id == bestVersionID {
            items.append("最佳")
        }
        return items
    }

    private func summaryForInsightID(_ id: Int64) -> InsightSummary? {
        historySummaries.first(where: { $0.id == id })
    }

    private var latestVersionID: Int64? {
        historySummaries.max { lhs, rhs in
            let leftCreatedAt = lhs.createdAt ?? ""
            let rightCreatedAt = rhs.createdAt ?? ""
            if leftCreatedAt != rightCreatedAt {
                return leftCreatedAt < rightCreatedAt
            }
            return lhs.id < rhs.id
        }?.id ?? primaryWorkbenchInsight.id
    }

    private func loadPrimaryInsightIfNeeded(recommendedID: Int64?, server: ServerConfig) async {
        guard let recommendedID, recommendedID > 0 else {
            primaryInsightOverride = nil
            return
        }
        guard recommendedID != insight.id else {
            primaryInsightOverride = nil
            return
        }
        if let cached = historyPreviewCache[recommendedID] {
            primaryInsightOverride = cached
            return
        }
        do {
            let detail = try await InsightAPIClient.fetchAlbumInsightDetail(using: server, id: recommendedID)
            historyPreviewCache[recommendedID] = detail
            primaryInsightOverride = detail
        } catch {
            primaryInsightOverride = nil
        }
    }

    @ViewBuilder
    private var previewIdentityLine: some View {
        HStack(alignment: .center, spacing: 8) {
            Text(selectedHistoryInsightID == nil ? "当前推荐版本" : "已选历史版本")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            if let providerLine = displayedInsight.providerLine {
                Text(providerLine)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
        }
    }

    @ViewBuilder
    private func versionPreviewErrorState(message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(message)
                .font(.body)
                .foregroundStyle(.secondary)
            Button("返回当前推荐版本") {
                selectPrimaryVersion()
            }
            .buttonStyle(.bordered)
        }
        .padding(.vertical, 8)
    }
}

struct InsightDetailHeader: View {
    let insight: Insight

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(insight.track)
                .font(.title2)
                .fontWeight(.semibold)
            Text("\(insight.artist) · \(insight.album)")
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassCard(cornerRadius: 16)
    }
}

struct AlbumInsightDetailHeader: View {
    let insight: AlbumInsight

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(insight.album)
                .font(.title2)
                .fontWeight(.semibold)
            Text(insight.artist)
                .font(.body)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassCard(cornerRadius: 16)
    }
}

struct InsightWorkbenchVersionRow: View {
    let title: String
    let provider: String?
    let createdAt: String?
    let score: Int?
    let feedbackStatusText: String
    let analysisSummary: String?
    let isSelected: Bool
    let badges: [String]
    let action: () -> Void

    private var scoreText: String {
        let score = score ?? 0
        if score == 0 {
            return "0"
        }
        return score > 0 ? "+\(score)" : "\(score)"
    }

    var body: some View {
        Button(action: action) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(title)
                        .font(.headline)
                        .lineLimit(1)

                    ForEach(badges, id: \.self) { badge in
                        Text(badge)
                            .font(.caption2.weight(.bold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(badgeBackground(for: badge), in: Capsule())
                            .foregroundStyle(badgeForeground(for: badge))
                    }

                    Spacer(minLength: 8)

                    Text(feedbackStatusText)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(feedbackStatusText == "待修正" ? Color.orange : .secondary)
                }

                HStack(spacing: 8) {
                    Text(provider ?? "未知模型")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let createdAt {
                        Text(createdAt)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                }

                HStack(spacing: 8) {
                    Text("评分 \(scoreText)")
                        .font(.caption)
                        .foregroundStyle(scoreColor)
                    if let trimmed = analysisSummary?.trimmingCharacters(in: .whitespacesAndNewlines),
                       !trimmed.isEmpty {
                        Text("·")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Text(trimmed)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                }
            }
            .padding(14)
            .frame(maxWidth: .infinity, alignment: .leading)
            .glassCard(cornerRadius: 12, isSimplified: true)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? Color.blue.opacity(0.45) : Color.white.opacity(0.08), lineWidth: isSelected ? 2 : 1)
            )
        }
        .buttonStyle(.plain)
    }

    private var scoreColor: Color {
        let value = score ?? 0
        if value < 0 {
            return .red
        }
        if value > 0 {
            return .green
        }
        return .secondary
    }

    private func badgeBackground(for badge: String) -> Color {
        switch badge {
        case "当前推荐":
            return SonicTheme.primary.opacity(0.18)
        case "最新":
            return Color.orange.opacity(0.2)
        case "最佳":
            return Color.green.opacity(0.18)
        default:
            return Color.primary.opacity(0.08)
        }
    }

    private func badgeForeground(for badge: String) -> Color {
        switch badge {
        case "当前推荐":
            return SonicTheme.primary
        case "最新":
            return .orange
        case "最佳":
            return .green
        default:
            return .secondary
        }
    }
}

struct InsightVersionPicker: View {
    @Binding var viewMode: InsightViewMode
    let historyCount: Int?

    var body: some View {
        HStack {
            Picker("", selection: $viewMode) {
                Text(InsightViewMode.current.rawValue).tag(InsightViewMode.current)
                if let historyCount {
                    Text("\(InsightViewMode.history.rawValue) (\(historyCount))").tag(InsightViewMode.history)
                } else {
                    Text(InsightViewMode.history.rawValue).tag(InsightViewMode.history)
                }
            }
            .pickerStyle(.segmented)
            .frame(maxWidth: 320)
        }
        .frame(maxWidth: .infinity, alignment: .center)
        .padding(.vertical, 8)
    }
}

struct InsightVersionHistoryRow: View {
    let summary: InsightSummary
    let index: Int
    let isSelected: Bool
    let action: () -> Void

    private var scoreText: String {
        let score = summary.totalScore
        if score == 0 {
            return "0"
        }
        return score > 0 ? "+\(score)" : "\(score)"
    }

    var body: some View {
        Button(action: action) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(summary.llmProvider ?? "未知模型")
                        .font(.headline)
                        .lineLimit(1)
                    if index == 0 {
                        Text("最新版本")
                            .font(.caption2.weight(.bold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.orange.opacity(0.2), in: Capsule())
                            .foregroundStyle(.orange)
                    }
                    Spacer()
                    Text(summary.feedbackStatusText)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(summary.feedbackStatusText == "待修正" ? Color.orange : .secondary)
                }

                HStack(spacing: 8) {
                    Text(summary.createdAt ?? "未知时间")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("评分 \(scoreText)")
                        .font(.caption)
                        .foregroundStyle(summary.totalScore < 0 ? Color.red : (summary.totalScore > 0 ? Color.green : .secondary))
                }

                if let analysisSummary = summary.analysisSummary?.trimmingCharacters(in: .whitespacesAndNewlines), !analysisSummary.isEmpty {
                    Text(analysisSummary)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }
            }
            .padding(14)
            .frame(maxWidth: .infinity, alignment: .leading)
            .glassCard(cornerRadius: 12, isSimplified: true)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? Color.blue.opacity(0.5) : Color.white.opacity(0.08), lineWidth: isSelected ? 2 : 1)
            )
        }
        .buttonStyle(.plain)
    }
}

struct InsightHistoryList<T: Identifiable>: View {
    let insights: [T]
    @Binding var selectedIndex: Int
    var isAlbum: Bool = false

    var body: some View {
        VStack(spacing: 12) {
            ForEach(Array(insights.enumerated()), id: \.offset) { index, item in
                InsightHistoryItemView(
                    index: index,
                    isSelected: selectedIndex == index,
                    llmProvider: (item as? Insight)?.llmProvider ?? (item as? AlbumInsight)?.llmProvider,
                    createdAt: (item as? Insight)?.createdAt ?? (item as? AlbumInsight)?.createdAt,
                    totalScore: (item as? Insight)?.totalScore ?? 0,
                    isAlbum: isAlbum
                ) {
                    selectedIndex = index
                }
            }
        }
    }
}

struct InsightHistoryItemView: View {
    let index: Int
    let isSelected: Bool
    let llmProvider: String?
    let createdAt: String?
    let totalScore: Int
    let isAlbum: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 16) {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 8) {
                        Text(llmProvider ?? "未知模型")
                            .font(.headline)
                        if index == 0 {
                            Text("最佳版本")
                                .font(.caption2)
                                .fontWeight(.bold)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.orange.opacity(0.2), in: Capsule())
                                .foregroundStyle(.orange)
                        }
                    }
                    Text(createdAt ?? "未知时间")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                if totalScore != 0 {
                    HStack(spacing: 4) {
                        Image(systemName: totalScore > 0 ? "hand.thumbsup.fill" : "hand.thumbsdown.fill")
                        Text("\(abs(totalScore))")
                    }
                    .font(.caption)
                    .foregroundStyle(totalScore > 0 ? .green : .red)
                }
                Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                    .foregroundStyle(isSelected ? Color.blue : Color.secondary)
            }
            .padding(16)
            .glassCard(cornerRadius: 12, isSimplified: true)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? Color.blue.opacity(0.5) : Color.clear, lineWidth: 2)
            )
        }
        .buttonStyle(.plain)
    }
}

struct InsightRenderStyle {
    let sectionSpacing: CGFloat
    let blockSpacing: CGFloat
    let blockPadding: CGFloat
    let innerSpacing: CGFloat
    let rowSpacing: CGFloat
    let stacksTaggedRows: Bool
    let wrapBlocksInCards: Bool
    let showProviderLine: Bool
    let simplifiedCards: Bool
    let translationUsesItalic: Bool
    let providerFont: Font
    let blockTitleFont: Font
    let sectionTitleFont: Font
    let bodyFont: Font
    let originalFont: Font
    let translationFont: Font
    let explainFont: Font
    let textColor: Color
    let secondaryTextColor: Color
    let originalColor: Color
    let translationColor: Color
    let explainColor: Color
    let explainBackground: Color
    let plainBlockBackground: Color
    let plainBlockBorder: Color

    static let detail = InsightRenderStyle(
        sectionSpacing: 16,
        blockSpacing: 14,
        blockPadding: 16,
        innerSpacing: 12,
        rowSpacing: 10,
        stacksTaggedRows: false,
        wrapBlocksInCards: true,
        showProviderLine: true,
        simplifiedCards: true,
        translationUsesItalic: false,
        providerFont: .caption,
        blockTitleFont: .headline,
        sectionTitleFont: .subheadline.weight(.semibold),
        bodyFont: .body,
        originalFont: .body.weight(.medium),
        translationFont: .callout,
        explainFont: .callout.weight(.medium),
        textColor: SonicTheme.textPrimary,
        secondaryTextColor: SonicTheme.textSecondary,
        originalColor: SonicTheme.textPrimary,
        translationColor: SonicTheme.textSecondary,
        explainColor: SonicTheme.textSecondary,
        explainBackground: SonicTheme.primary.opacity(0.08),
        plainBlockBackground: SonicTheme.card,
        plainBlockBorder: SonicTheme.glassBorder
    )

    static let immersive = InsightRenderStyle(
        sectionSpacing: 18,
        blockSpacing: 18,
        blockPadding: 18,
        innerSpacing: 14,
        rowSpacing: 12,
        stacksTaggedRows: false,
        wrapBlocksInCards: false,
        showProviderLine: true,
        simplifiedCards: true,
        translationUsesItalic: false,
        providerFont: .system(size: 13, weight: .medium),
        blockTitleFont: .system(size: 20, weight: .bold, design: .rounded),
        sectionTitleFont: .system(size: 16, weight: .semibold),
        bodyFont: .system(size: 19, weight: .medium),
        originalFont: .system(size: 17, weight: .medium),
        translationFont: .system(size: 17, weight: .medium),
        explainFont: .system(size: 21, weight: .bold),
        textColor: .white.opacity(0.95),
        secondaryTextColor: .white.opacity(0.8),
        originalColor: .white.opacity(0.8),
        translationColor: .white.opacity(0.62),
        explainColor: .white.opacity(0.95),
        explainBackground: Color.white.opacity(0.045),
        plainBlockBackground: .clear,
        plainBlockBorder: .clear
    )

    static let phoneCompact = InsightRenderStyle(
        sectionSpacing: 14,
        blockSpacing: 12,
        blockPadding: 14,
        innerSpacing: 10,
        rowSpacing: 8,
        stacksTaggedRows: true,
        wrapBlocksInCards: false,
        showProviderLine: true,
        simplifiedCards: true,
        translationUsesItalic: true,
        providerFont: .caption,
        blockTitleFont: .system(size: 18, weight: .bold, design: .rounded),
        sectionTitleFont: .system(size: 13, weight: .semibold),
        bodyFont: .system(size: 16, weight: .medium),
        originalFont: .system(size: 15, weight: .medium),
        translationFont: .system(size: 14, weight: .medium),
        explainFont: .system(size: 17, weight: .semibold),
        textColor: SonicTheme.textPrimary,
        secondaryTextColor: SonicTheme.textSecondary,
        originalColor: SonicTheme.textPrimary,
        translationColor: SonicTheme.textSecondary,
        explainColor: SonicTheme.textPrimary,
        explainBackground: Color.black.opacity(0.04),
        plainBlockBackground: Color.black.opacity(0.03),
        plainBlockBorder: Color.black.opacity(0.07)
    )
}

struct InsightPrimaryContentView: View {
    let insight: Insight?
    var style: InsightRenderStyle = .detail
    var emptyTitle: String = "暂无音眸"
    var emptySubtitle: String = "当前曲目还没有生成洞察内容。"

    var body: some View {
        Group {
            if let insight, insight.hasDisplayContent {
                InsightRichContentView(insight: insight, style: style)
            } else {
                InsightEmptyStateCard(title: emptyTitle, subtitle: emptySubtitle, style: style)
            }
        }
    }
}

struct InsightRichContentView: View {
    let insight: Insight
    let style: InsightRenderStyle

    var body: some View {
        VStack(alignment: .leading, spacing: style.blockSpacing) {
            if style.showProviderLine, let providerLine = insight.providerLine {
                Text(providerLine)
                    .font(style.providerFont)
                    .foregroundStyle(style.secondaryTextColor)
            }

            ForEach(insight.displayBlocks) { block in
                InsightDisplayBlockView(block: block, style: style)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct InsightDisplayBlockView: View {
    let block: InsightDisplayBlock
    let style: InsightRenderStyle

    var body: some View {
        let content = VStack(alignment: .leading, spacing: style.innerSpacing) {
            Text(blockTitle)
                .font(style.blockTitleFont)
                .foregroundStyle(style.textColor)

            switch block {
            case let .text(_, _, text):
                InsightTaggedContentView(text: text, style: style)
            case let .sections(_, _, sections):
                VStack(alignment: .leading, spacing: style.sectionSpacing) {
                    ForEach(sections) { section in
                        VStack(alignment: .leading, spacing: 10) {
                            Text("【\(section.title)】")
                                .font(style.sectionTitleFont)
                                .foregroundStyle(style.secondaryTextColor)
                            InsightTaggedContentView(text: section.content, style: style)
                        }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)

        if style.wrapBlocksInCards {
            content
                .padding(style.blockPadding)
                .glassCard(cornerRadius: 14, isSimplified: style.simplifiedCards)
        } else {
            content
                .padding(style.blockPadding)
                .background(style.plainBlockBackground, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .stroke(style.plainBlockBorder, lineWidth: 1)
                )
        }
    }

    private var blockTitle: String {
        switch block {
        case let .text(_, title, _):
            return title
        case let .sections(_, title, _):
            return title
        }
    }
}

private struct InsightTaggedContentView: View {
    let text: String
    let style: InsightRenderStyle

    var body: some View {
        if let segments = InsightTaggedContentParser.parse(text) {
            VStack(alignment: .leading, spacing: style.innerSpacing) {
                ForEach(Array(segments.enumerated()), id: \.offset) { item in
                    switch item.element {
                    case let .text(text):
                        InsightBodyText(text: text, color: style.textColor, font: style.bodyFont)
                    case let .rows(rows):
                        InsightTaggedRowsView(rows: rows, style: style)
                    case let .explain(text):
                        InsightExplainText(text: text, style: style)
                    }
                }
            }
        } else {
            InsightBodyText(text: text, color: style.textColor, font: style.bodyFont)
        }
    }
}

private struct InsightTaggedRowsView: View {
    let rows: [InsightTaggedRow]
    let style: InsightRenderStyle

    var body: some View {
        VStack(alignment: .leading, spacing: style.rowSpacing) {
            ForEach(Array(rows.enumerated()), id: \.offset) { item in
                let row = item.element
                if style.stacksTaggedRows {
                    VStack(alignment: .leading, spacing: 6) {
                        if !row.original.isEmpty {
                            Text(row.original)
                                .font(style.originalFont)
                                .foregroundStyle(style.originalColor)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        if !row.translation.isEmpty {
                            InsightTranslationText(text: row.translation, style: style)
                        }
                    }
                    .padding(.vertical, 2)
                } else {
                    HStack(alignment: .top, spacing: 16) {
                        Text(row.original.isEmpty ? " " : row.original)
                            .font(style.originalFont)
                            .foregroundStyle(style.originalColor)
                            .frame(maxWidth: .infinity, alignment: .leading)

                        if !row.translation.isEmpty {
                            InsightTranslationText(text: row.translation, style: style)
                        }
                    }
                    .padding(.vertical, 2)
                }
            }
        }
    }
}

private struct InsightTranslationText: View {
    let text: String
    let style: InsightRenderStyle

    var body: some View {
        let translation = Text(text).font(style.translationFont)
        Group {
            if style.translationUsesItalic {
                translation.italic()
            } else {
                translation
            }
        }
        .foregroundStyle(style.translationColor)
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct InsightBodyText: View {
    let text: String
    let color: Color
    let font: Font

    var body: some View {
        Text(text.replacingOccurrences(of: "\\n", with: "\n"))
            .font(font)
            .foregroundStyle(color)
            .fixedSize(horizontal: false, vertical: true)
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct InsightExplainText: View {
    let text: String
    let style: InsightRenderStyle

    var body: some View {
        InsightBodyText(text: text, color: style.explainColor, font: style.explainFont)
            .padding(12)
            .background(style.explainBackground, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}

private struct InsightEmptyStateCard: View {
    let title: String
    let subtitle: String
    let style: InsightRenderStyle

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
                .foregroundStyle(style.textColor)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(style.secondaryTextColor)
        }
        .padding(style.blockPadding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(style.plainBlockBackground, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(style.plainBlockBorder, lineWidth: 1)
        )
    }
}
