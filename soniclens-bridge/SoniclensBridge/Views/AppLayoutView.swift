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

    var keyboardShortcutKey: KeyEquivalent {
        switch self {
        case .home:
            return "1"
        case .albums:
            return "2"
        case .tracks:
            return "3"
        case .unreported:
            return "4"
        case .sonicLens:
            return "5"
        case .futureFeatures:
            return "6"
        }
    }
}

struct AppLayoutView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(PlaybackStore.self) private var playbackStore
    @State private var selection: SidebarDestination = .home
    @State private var showNowPlaying = false
    @State private var albumSort: LibrarySort = .recent
    @State private var trackSort: LibrarySort = .recent
    @State private var trackFilter: TrackFilter = .all
    @State private var albumQuery: String = ""
    @State private var trackQuery: String = ""
    @StateObject private var libraryViewModel = LibraryViewModel()
    @AppStorage("soniclens.performanceMode") private var performanceModeEnabled: Bool = false
    @State private var playbackOverlayController = PlaybackBarWindowOverlayController()
    @State private var nowPlayingOverlayController = NowPlayingWindowOverlayController()

    var body: some View {
        NavigationSplitView {
            SidebarView(selection: $selection)
        } detail: {
            NavigationStack {
                contentView
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(AppWindowBackground(useMaterial: false))
                    .toolbar {
                        ToolbarItemGroup(placement: .principal) {
                            ToolbarTitleMenu(
                                selection: $selection,
                                subtitle: toolbarSubtitle
                            )
                        }
                        ToolbarItemGroup(placement: .automatic) {
                            toolbarPageOperations

                            ToolbarDivider()

                            PerformanceModeToolbarButton(isEnabled: $performanceModeEnabled)

                            if store.currentServer != nil {
                                Button {
                                    store.disconnect()
                                } label: {
                                    ToolbarIconButton(systemImage: "power", helpText: "断开当前服务端")
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
            }
            .safeAreaInset(edge: .bottom, spacing: 0) {
                if !showNowPlaying {
                    Color.clear
                        .frame(height: PlaybackBarView.regularHeight + 14)
                        .allowsHitTesting(false)
                }
            }
        }
        .navigationSplitViewStyle(.balanced)
        .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
        .background(
            PlaybackBarWindowOverlayBridge(
                controller: playbackOverlayController,
                nowPlaying: playbackStore.nowPlaying,
                performanceModeEnabled: performanceModeEnabled,
                isVisible: !showNowPlaying,
                style: .regular,
                onActivate: {
                    guard playbackStore.hasActiveNowPlaying else { return }
                    showNowPlaying = true
                }
            )
        )
        .background(
            NowPlayingWindowOverlayBridge(
                controller: nowPlayingOverlayController,
                nowPlaying: playbackStore.nowPlaying,
                appStore: store,
                playbackStore: playbackStore,
                isVisible: showNowPlaying,
                onClose: {
                    showNowPlaying = false
                }
            )
        )
        .onChange(of: playbackStore.hasActiveNowPlaying) { _, hasActiveNowPlaying in
            guard !hasActiveNowPlaying else { return }
            showNowPlaying = false
        }
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
            Task {
                await store.performForegroundConnectionHealthCheckIfNeeded()
                guard let server = store.currentServer, store.isConnectionHealthy else { return }
                await libraryViewModel.refresh(using: server)
            }
        }
        .sheet(item: $store.sharePreviewRequest) { request in
            SharePreviewView(payload: request.payload)
        }
        .onReceive(NSWorkspace.shared.notificationCenter.publisher(for: NSWorkspace.didWakeNotification)) { _ in
            Task {
                await store.performForegroundConnectionHealthCheckIfNeeded()
                guard let server = store.currentServer, store.isConnectionHealthy else { return }
                await libraryViewModel.refresh(using: server)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .librarySyncDidUpdate)) { _ in
            guard let server = store.currentServer else { return }
            Task {
                await libraryViewModel.refresh(using: server)
            }
        }
        .onDisappear {
            playbackOverlayController.detach()
            nowPlayingOverlayController.detach()
        }
        .frame(minWidth: 1100, minHeight: 720)
        .background(
            Group {
                Button("") { selection = .home }.keyboardShortcut("1", modifiers: .control)
                Button("") { selection = .albums }.keyboardShortcut("2", modifiers: .control)
                Button("") { selection = .tracks }.keyboardShortcut("3", modifiers: .control)
                Button("") { selection = .unreported }.keyboardShortcut("4", modifiers: .control)
                Button("") { selection = .sonicLens }.keyboardShortcut("5", modifiers: .control)
                Button("") { selection = .futureFeatures }.keyboardShortcut("6", modifiers: .control)
            }
            .opacity(0)
            .allowsHitTesting(false)
        )
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
    private var toolbarPageOperations: some View {
        switch selection {
        case .albums:
            ToolbarSearchField(text: $albumQuery)
            LibrarySortMenu(title: "排序", selection: $albumSort, options: LibrarySort.albumOptions)
        case .tracks:
            ToolbarSearchField(text: $trackQuery)
            LibrarySortMenu(title: "排序", selection: $trackSort, options: LibrarySort.trackOptions)
            TrackFilterMenu(selection: $trackFilter)
        default:
            EmptyView()
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
                Text("\(record.artist) · \(record.displayAlbum)")
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

    var systemImage: String {
        switch self {
        case .recent:
            return "clock"
        case .updated:
            return "arrow.clockwise"
        case .releaseDate:
            return "calendar"
        case .alpha:
            return "textformat.abc"
        case .plays:
            return "flame.fill"
        }
    }
}

enum TrackFilter: String, CaseIterable {
    case all = "全部曲目"
    case favorites = "已收藏"
    case unreported = "未上报"

    var systemImage: String {
        switch self {
        case .all:
            return "music.note.list"
        case .favorites:
            return "heart.fill"
        case .unreported:
            return "exclamationmark.triangle"
        }
    }
}

struct LibrarySortMenu: View {
    let title: String
    @Binding var selection: LibrarySort
    var options: [LibrarySort] = LibrarySort.albumOptions

    var body: some View {
        Menu {
            Picker(title, selection: $selection) {
                ForEach(options, id: \.self) { sort in
                    Label(sort.rawValue, systemImage: sort.systemImage)
                        .tag(sort)
                }
            }
            .pickerStyle(.inline)
        } label: {
            ToolbarIconButton(
                systemImage: selection.systemImage,
                isActive: !selection.isDefault,
                helpText: "排序：\(selection.rawValue)"
            )
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
    }
}

struct TrackFilterMenu: View {
    @Binding var selection: TrackFilter

    var body: some View {
        Menu {
            Picker("筛选", selection: $selection) {
                ForEach(TrackFilter.allCases, id: \.self) { filter in
                    Label(filter.rawValue, systemImage: filter.systemImage)
                        .tag(filter)
                }
            }
            .pickerStyle(.inline)
        } label: {
            ToolbarIconButton(
                systemImage: selection.systemImage,
                isActive: !selection.isDefault,
                activeTint: .red,
                helpText: "筛选：\(selection.rawValue)"
            )
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
    }
}

struct PerformanceModeToolbarButton: View {
    @Binding var isEnabled: Bool

    var body: some View {
        Button {
            withAnimation(.easeInOut(duration: 0.2)) {
                isEnabled.toggle()
            }
        } label: {
            ToolbarIconButton(
                systemImage: isEnabled ? "gauge.with.dots.needle.50percent" : "speedometer",
                isActive: isEnabled,
                activeTint: .orange,
                helpText: isEnabled ? "性能模式已开启（极简渲染）" : "开启性能模式",
                showIndicatorDot: isEnabled
            )
        }
        .buttonStyle(.plain)
    }
}

struct ToolbarDivider: View {
    var body: some View {
        Rectangle()
            .fill(Color.primary.opacity(0.15))
            .frame(width: 1, height: 14)
            .padding(.horizontal, 4)
    }
}

struct ToolbarSearchField: View {
    @Binding var text: String
    var isFocused: FocusState<Bool>.Binding? = nil

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: "magnifyingglass")
                .font(.caption)
                .foregroundStyle(.secondary)
            if let isFocused {
                TextField("搜索", text: $text)
                    .textFieldStyle(.plain)
                    .focused(isFocused)
            } else {
                TextField("搜索", text: $text)
                    .textFieldStyle(.plain)
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .frame(width: 160, height: 28)
        .background(
            RoundedRectangle(cornerRadius: 7, style: .continuous)
                .fill(Color.primary.opacity(0.06))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 7, style: .continuous)
                .stroke((isFocused?.wrappedValue ?? false) ? Color.accentColor.opacity(0.6) : Color.primary.opacity(0.12), lineWidth: 1)
        )
        .accessibilityLabel("搜索")
    }
}

struct ToolbarIconButton: View {
    let systemImage: String
    var title: String? = nil
    var isActive: Bool = false
    var activeTint: Color = .accentColor
    var helpText: String? = nil
    var showIndicatorDot: Bool = false
    var action: (() -> Void)? = nil

    @State private var isHovered = false

    var body: some View {
        Group {
            if let action {
                Button(action: action) {
                    content
                }
                .buttonStyle(.plain)
            } else {
                content
            }
        }
        .ifLet(helpText) { view, help in
            view.help(help)
        }
    }

    private var content: some View {
        HStack(spacing: 4) {
            Image(systemName: systemImage)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(isActive ? activeTint : (isHovered ? Color.primary : Color.secondary))

            if let title {
                Text(title)
                    .font(.caption)
                    .foregroundStyle(isActive ? Color.primary : Color.secondary)
            }

            if showIndicatorDot {
                Circle()
                    .fill(activeTint)
                    .frame(width: 4, height: 4)
            }
        }
        .padding(.horizontal, title == nil ? 0 : 8)
        .frame(width: title == nil ? 28 : nil, height: 28)
        .frame(minWidth: 28)
        .background(
            RoundedRectangle(cornerRadius: 7, style: .continuous)
                .fill(isActive ? activeTint.opacity(0.15) : (isHovered ? Color.primary.opacity(0.08) : Color.primary.opacity(0.04)))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 7, style: .continuous)
                .stroke(isActive ? activeTint.opacity(0.35) : (isHovered ? Color.primary.opacity(0.18) : Color.primary.opacity(0.08)), lineWidth: 1)
        )
        .contentShape(Rectangle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }
}

private extension View {
    @ViewBuilder
    func ifLet<Value, Content: View>(_ value: Value?, transform: (Self, Value) -> Content) -> some View {
        if let value {
            transform(self, value)
        } else {
            self
        }
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

struct ToolbarTitleMenu: View {
    @Binding var selection: SidebarDestination
    let subtitle: String?

    @State private var isHovered = false

    private let browseItems: [SidebarDestination] = [.home, .albums, .tracks, .unreported]
    private let deepContentItems: [SidebarDestination] = [.sonicLens]
    private let planningItems: [SidebarDestination] = [.futureFeatures]

    var body: some View {
        Menu {
            Section("浏览") {
                ForEach(browseItems, id: \.self) { item in
                    destinationButton(for: item)
                }
            }

            Section("深度内容") {
                ForEach(deepContentItems, id: \.self) { item in
                    destinationButton(for: item)
                }
            }

            Section("规划") {
                ForEach(planningItems, id: \.self) { item in
                    destinationButton(for: item)
                }
            }
        } label: {
            HStack(spacing: 5) {
                ToolbarTitleSubtitleView(
                    title: selection.title,
                    subtitle: subtitle
                )
                .equatable()

                Image(systemName: "chevron.down")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(.secondary)
                    .padding(.top, 1)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 4)
            .background(
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(isHovered ? Color.primary.opacity(0.06) : Color.clear)
            )
            .contentShape(Rectangle())
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.15)) {
                isHovered = hovering
            }
        }
        .help("点击快速切换模块 (⌃1~⌃6)")
    }

    @ViewBuilder
    private func destinationButton(for item: SidebarDestination) -> some View {
        Toggle(isOn: Binding(
            get: { selection == item },
            set: { if $0 { selection = item } }
        )) {
            Label(item.title, systemImage: item.systemImage)
        }
        .keyboardShortcut(item.keyboardShortcutKey, modifiers: .control)
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
