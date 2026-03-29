import SwiftUI

#if os(macOS)
import AppKit

enum SidebarDestination: String, CaseIterable, Hashable {
    case home
    case futureFeatures
    case albums
    case tracks
    case sonicLens
    case unreported

    var title: String {
        switch self {
        case .home:
            return "主页"
        case .futureFeatures:
            return "未来功能"
        case .albums:
            return "专辑"
        case .tracks:
            return "曲目"
        case .sonicLens:
            return "音眸"
        case .unreported:
            return "未上报"
        }
    }

    var systemImage: String {
        switch self {
        case .home:
            return "house"
        case .futureFeatures:
            return "sparkles"
        case .albums:
            return "square.grid.2x2"
        case .tracks:
            return "music.note.list"
        case .sonicLens:
            return "waveform.and.magnifyingglass"
        case .unreported:
            return "exclamationmark.triangle"
        }
    }

    var subtitle: String? {
        switch self {
        case .home:
            return "仪表盘与播放中入口"
        case .futureFeatures:
            return "规划中的能力"
        case .albums:
            return "本地分页浏览"
        case .tracks:
            return "搜索、筛选与收藏"
        case .sonicLens:
            return "洞察与解析"
        case .unreported:
            return "待上报记录"
        }
    }
}

struct AppLayoutView: View {
    @EnvironmentObject private var store: AppStore
    @State private var selection: SidebarDestination = .home
    @State private var showNowPlaying = false
    @State private var albumSort: LibrarySort = .recent
    @State private var trackSort: LibrarySort = .recent
    @State private var trackFilter: TrackFilter = .all
    @State private var albumQuery: String = ""
    @State private var trackQuery: String = ""
    @StateObject private var libraryViewModel = LibraryViewModel()
    @AppStorage("soniclens.performanceMode") private var performanceModeEnabled: Bool = false
    @State private var windowChromeBackup: WindowChromeBackup?

    var body: some View {
        ZStack(alignment: .bottom) {
            NavigationSplitView {
                SidebarView(selection: $selection)
            } detail: {
                NavigationStack {
                    contentView
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background(AppWindowBackground(useMaterial: false))
                        .toolbar {
                            ToolbarItemGroup(placement: .principal) {
                                ToolbarTitleSubtitleView(
                                    title: selection.title,
                                    subtitle: toolbarSubtitle
                                )
                                .equatable()
                            }
                            ToolbarItemGroup(placement: .automatic) {
                                Toggle("性能模式", isOn: $performanceModeEnabled)
                                    .toggleStyle(.switch)
                                    .help("降低动效/材质/阴影开销，提升长时间运行稳定性")
                                toolbarContent
                            }
                        }
                }
            }
            .navigationSplitViewStyle(.balanced)
            // .padding(.bottom, PlaybackBarView.regularHeight)

            PlaybackBarView(isExpanded: $showNowPlaying)
        }
        .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
        .background(
            WindowAccessor { window in
                configureWindowChrome(window)
            }
        )
        .overlay {
            if showNowPlaying, let nowPlaying = store.nowPlaying {
                NowPlayingView(nowPlaying: nowPlaying) {
                    showNowPlaying = false
                }
                .ignoresSafeArea()
                .transition(.opacity.combined(with: .move(edge: .bottom)))
            }
        }
        .animation(.easeInOut(duration: 0.35), value: showNowPlaying)
        .task {
            guard let server = store.currentServer else { return }
            await libraryViewModel.load(using: server)
        }
        .onChange(of: store.currentServer) { _, server in
            guard let server else { return }
            Task {
                await libraryViewModel.load(using: server)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: NSApplication.didBecomeActiveNotification)) { _ in
            guard let server = store.currentServer else { return }
            Task {
                await libraryViewModel.refresh(using: server)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .librarySyncDidUpdate)) { _ in
            guard let server = store.currentServer else { return }
            Task {
                await libraryViewModel.refresh(using: server)
            }
        }
        .frame(minWidth: 1100, minHeight: 720)
    }

    private func configureWindowChrome(_ window: NSWindow?) {
        guard let window else { return }

        if showNowPlaying {
            if windowChromeBackup == nil {
                windowChromeBackup = WindowChromeBackup(
                    titlebarAppearsTransparent: window.titlebarAppearsTransparent,
                    titleVisibility: window.titleVisibility,
                    toolbarVisible: window.toolbar?.isVisible ?? true,
                    hasFullSizeContentView: window.styleMask.contains(.fullSizeContentView)
                )
            }

            window.titleVisibility = .hidden
            window.titlebarAppearsTransparent = true
            window.toolbar?.isVisible = false
            window.styleMask.insert(.fullSizeContentView)
            return
        }

        guard let backup = windowChromeBackup else { return }
        window.titlebarAppearsTransparent = backup.titlebarAppearsTransparent
        window.titleVisibility = backup.titleVisibility
        window.toolbar?.isVisible = backup.toolbarVisible
        if backup.hasFullSizeContentView {
            window.styleMask.insert(.fullSizeContentView)
        } else {
            window.styleMask.remove(.fullSizeContentView)
        }
        windowChromeBackup = nil
    }

    @ViewBuilder
    private var contentView: some View {
        switch selection {
        case .home:
            HomeView()
        case .futureFeatures:
            FutureFeaturesView()
        case .albums:
            AlbumGridView(
                viewModel: libraryViewModel,
                sort: albumSort,
                query: albumQuery,
                artworkBaseURL: store.currentServer?.artworkBaseURL
            )
        case .tracks:
            TrackListView(
                viewModel: libraryViewModel,
                sort: trackSort,
                filter: trackFilter,
                query: trackQuery,
                showsInlineControls: false
            )
        case .sonicLens:
            SonicLensInsightsView(viewModel: libraryViewModel)
        case .unreported:
            UnreportedListView(viewModel: libraryViewModel)
        }
    }

    private var toolbarSubtitle: String? {
        switch selection {
        case .albums:
            return LibraryStatusSummary.album(sort: albumSort)
        case .tracks:
            return LibraryStatusSummary.track(sort: trackSort, filter: trackFilter)
        default:
            return selection.subtitle
        }
    }

    @ViewBuilder
    private var toolbarContent: some View {
        switch selection {
        case .albums:
            LibrarySortMenu(title: "排序", selection: $albumSort, options: LibrarySort.albumOptions)
            ToolbarSearchField(text: $albumQuery)
        case .tracks:
            LibrarySortMenu(title: "排序", selection: $trackSort, options: LibrarySort.trackOptions)
            TrackFilterMenu(selection: $trackFilter)
            ToolbarSearchField(text: $trackQuery)
        default:
            EmptyView()
        }
    }
}

private struct WindowChromeBackup {
    let titlebarAppearsTransparent: Bool
    let titleVisibility: NSWindow.TitleVisibility
    let toolbarVisible: Bool
    let hasFullSizeContentView: Bool
}

private struct WindowAccessor: NSViewRepresentable {
    let onWindowChange: (NSWindow?) -> Void

    func makeNSView(context: Context) -> NSView {
        let view = NSView()
        DispatchQueue.main.async {
            onWindowChange(view.window)
        }
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        DispatchQueue.main.async {
            onWindowChange(nsView.window)
        }
    }
}

struct SidebarView: View {
    @Binding var selection: SidebarDestination

    private let browseItems: [SidebarDestination] = [.home, .albums, .tracks, .unreported]
    private let deepContentItems: [SidebarDestination] = [.sonicLens]
    private let planningItems: [SidebarDestination] = [.futureFeatures]

    var body: some View {
        List(selection: $selection) {
            Section("浏览") {
                ForEach(browseItems, id: \.self) { item in
                    SidebarItemView(item: item)
                        .tag(item)
                        .listRowInsets(EdgeInsets(top: 4, leading: 6, bottom: 4, trailing: 6))
                }
            }

            Section("深度内容") {
                ForEach(deepContentItems, id: \.self) { item in
                    SidebarItemView(item: item)
                        .tag(item)
                        .listRowInsets(EdgeInsets(top: 4, leading: 6, bottom: 4, trailing: 6))
                }
            }

            Section("规划") {
                ForEach(planningItems, id: \.self) { item in
                    SidebarItemView(item: item)
                        .tag(item)
                        .listRowInsets(EdgeInsets(top: 4, leading: 6, bottom: 4, trailing: 6))
                }
            }
        }
        .listStyle(.sidebar)
        .navigationTitle("音眸")
        .frame(minWidth: 220, idealWidth: 220, maxWidth: 260)
    }
}

struct SidebarItemView: View {
    let item: SidebarDestination

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: item.systemImage)
                .font(.system(size: 14, weight: .semibold))
                .frame(width: 18)
            Text(item.title)
                .font(.body)
            Spacer(minLength: 0)
        }
        .padding(.vertical, 6)
        .padding(.horizontal, 8)
        .contentShape(Rectangle())
    }
}

struct AppWindowBackground: View {
    var useMaterial: Bool = true
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        let simplified = performanceModeEnabled || !useMaterial
        ZStack {
            LinearGradient(
                colors: [
                    Color(nsColor: NSColor.windowBackgroundColor),
                    Color(nsColor: NSColor.controlBackgroundColor),
                    Color(nsColor: NSColor.controlBackgroundColor).opacity(0.7)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .opacity(0.95)

            if !simplified {
                Rectangle()
                    .fill(.ultraThinMaterial)
                    .opacity(0.28)
            }
        }
        .ignoresSafeArea()
    }
}

struct AppBackground: View {
    var useMaterial: Bool = true
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        AppWindowBackground(useMaterial: useMaterial && !performanceModeEnabled)
    }
}
#else
struct AppBackground: View {
    var useMaterial: Bool = true
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        let simplified = performanceModeEnabled || !useMaterial
        ZStack {
            LinearGradient(
                colors: [
                    SonicTheme.background,
                    SonicTheme.background.opacity(0.92),
                    Color.accentColor.opacity(0.12)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .ignoresSafeArea()

            if !simplified {
                Rectangle()
                    .fill(.ultraThinMaterial)
                    .opacity(0.2)
                    .ignoresSafeArea()
            }
        }
    }
}
#endif

struct FutureFeaturesView: View {
    private let ideas = [
        ("跨播放器统一控制", "Apple Music / Roon / Audirvana 一处掌控"),
        ("AI 赏析深度报告", "曲目故事、歌词翻译、时代背景"),
        ("智能混音", "根据喜好自动生成歌单"),
        ("空间聆听", "为不同场景优化听感")
    ]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                SectionHeader(title: "未来功能")

                ForEach(ideas, id: \.0) { idea in
                    FeatureCard(title: idea.0, subtitle: idea.1)
                }
            }
            .padding(32)
        }
    }
}

struct FeatureCard: View {
    let title: String
    let subtitle: String

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 6) {
                Text(title)
                    .font(.headline)
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Image(systemName: "sparkle")
                .font(.system(size: 20, weight: .semibold))
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .glassCard(cornerRadius: 12)
    }
}

struct SonicLensPlaceholderView: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                SectionHeader(title: "音眸")
                Text("智能赏析、歌词与背景解读将呈现在这里。")
                    .font(.body)
                    .foregroundStyle(.secondary)
                FeatureCard(title: "赏析卡片", subtitle: "生成曲目背景与深度解读")
                FeatureCard(title: "歌词智能", subtitle: "实时歌词、翻译与注解")
            }
            .padding(32)
        }
    }
}

struct UnreportedListView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                SectionHeader(title: "未上报")

                if viewModel.unscrobbled.isEmpty {
                    EmptyStateView(
                        title: "已全部完成",
                        subtitle: "暂无待上报曲目。"
                    )
                } else {
                    VStack(spacing: 12) {
                        ForEach(viewModel.unscrobbled) { record in
                            UnreportedRow(record: record)
                        }
                    }
                }
            }
            .padding(32)
        }
        .task(id: store.currentServer?.baseURL) {
            guard let server = store.currentServer else { return }
            guard viewModel.unscrobbled.isEmpty, viewModel.unscrobbledCount != 0 else { return }
            await viewModel.reloadUnscrobbled(using: server)
        }
    }
}

struct UnreportedRow: View {
    let record: UnscrobbledRecord
    @State private var isHovered = false
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        let hoverEnabled = !performanceModeEnabled
        let hoverFill = isHovered && hoverEnabled ? Color.primary.opacity(0.05) : Color.clear

        HStack(spacing: 16) {
            Image(systemName: "waveform")
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(.secondary)
            VStack(alignment: .leading, spacing: 4) {
                Text(record.track)
                    .font(.body)
                Text("\(record.artist) · \(record.album)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Text(record.playTime)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(14)
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(hoverFill)
        )
        .background(
            Group {
                if performanceModeEnabled {
                    RoundedRectangle(cornerRadius: 12)
                        .fill(SonicTheme.card)
                } else {
                    RoundedRectangle(cornerRadius: 12)
                        .fill(.ultraThinMaterial)
                }
            }
        )
        .onHover { hovering in
            guard hoverEnabled else { return }
            withAnimation(.easeInOut(duration: 0.15)) {
                isHovered = hovering
            }
        }
    }
}

struct EmptyStateView: View {
    let title: String
    let subtitle: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassCard(cornerRadius: 12)
    }
}

enum LibrarySort: String, CaseIterable {
    case recent = "最近添加"
    case updated = "最近更新"
    case releaseDate = "发行时间"
    case alpha = "按字母"
    case plays = "播放最多"

    static let albumOptions: [LibrarySort] = [.recent, .updated, .releaseDate, .alpha, .plays]
    static let trackOptions: [LibrarySort] = [.recent, .updated, .alpha, .plays]
}

enum TrackFilter: String, CaseIterable {
    case all = "全部曲目"
    case favorites = "已收藏"
    case unreported = "未上报"
}

struct LibrarySortMenu: View {
    let title: String
    @Binding var selection: LibrarySort
    var options: [LibrarySort] = LibrarySort.albumOptions

    var body: some View {
        Menu {
            ForEach(options, id: \.self) { sort in
                Button {
                    selection = sort
                } label: {
                    MenuSelectionRow(title: sort.rawValue, isSelected: selection == sort)
                }
            }
        } label: {
            ToolbarPillLabel(title: title, systemImage: "arrow.up.arrow.down")
        }
        .menuStyle(.borderlessButton)
    }
}

struct TrackFilterMenu: View {
    @Binding var selection: TrackFilter

    var body: some View {
        Menu {
            ForEach(TrackFilter.allCases, id: \.self) { filter in
                Button {
                    selection = filter
                } label: {
                    MenuSelectionRow(title: filter.rawValue, isSelected: selection == filter)
                }
            }
        } label: {
            ToolbarPillLabel(
                title: selection.isDefault ? "筛选" : selection.rawValue,
                systemImage: "line.3.horizontal.decrease.circle"
            )
        }
        .menuStyle(.borderlessButton)
    }
}

struct ToolbarSearchField: View {
    @Binding var text: String

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: "magnifyingglass")
                .font(.caption)
                .foregroundStyle(.secondary)
            TextField("搜索", text: $text)
                .textFieldStyle(.plain)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .frame(width: 200)
        .background(
            Capsule()
                .fill(Color.white.opacity(0.14))
        )
        .overlay(
            Capsule()
                .stroke(Color.white.opacity(0.25), lineWidth: 1)
        )
        .accessibilityLabel("搜索")
    }
}

struct ToolbarPillLabel: View {
    let title: String
    let systemImage: String

    var body: some View {
        Label(title, systemImage: systemImage)
            .font(.caption)
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(
                Capsule()
                    .fill(Color.white.opacity(0.14))
            )
            .overlay(
                Capsule()
                    .stroke(Color.white.opacity(0.25), lineWidth: 1)
            )
    }
}

struct ToolbarTitleSubtitleView: View, Equatable {
    let title: String
    let subtitle: String?

    var body: some View {
        VStack(spacing: 2) {
            Text(title)
                .font(.title3)
                .fontWeight(.semibold)
                .lineLimit(1)

            if let subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
        }
    }
}

struct MenuSelectionRow: View {
    let title: String
    let isSelected: Bool

    var body: some View {
        HStack(spacing: 10) {
            Text(title)
            Spacer(minLength: 12)
            if isSelected {
                Image(systemName: "checkmark")
                    .font(.caption.weight(.semibold))
            }
        }
    }
}

enum LibraryStatusSummary {
    static func album(sort: LibrarySort) -> String? {
        guard !sort.isDefault else { return nil }
        return sort.rawValue
    }

    static func track(sort: LibrarySort, filter: TrackFilter) -> String? {
        var parts: [String] = []
        if !sort.isDefault {
            parts.append(sort.rawValue)
        }
        if !filter.isDefault {
            parts.append(filter.rawValue)
        }
        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }
}

struct LibraryStatusSummaryChip: View {
    let text: String

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "arrow.up.arrow.down")
                .font(.caption2.weight(.semibold))
                .foregroundStyle(.secondary)
            Text(text)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(1)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(
            Capsule()
                .fill(Color.white.opacity(0.12))
        )
        .overlay(
            Capsule()
                .stroke(Color.white.opacity(0.18), lineWidth: 1)
        )
    }
}

extension LibrarySort {
    var isDefault: Bool {
        self == .recent
    }
}

extension TrackFilter {
    var isDefault: Bool {
        self == .all
    }
}
