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
}

struct AppLayoutView: View {
    @EnvironmentObject private var store: AppStore
    @State private var selection: SidebarDestination = .home
    @State private var searchText: String = ""
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
                SidebarView(selection: $selection, searchText: $searchText)
            } detail: {
                NavigationStack {
                    contentView
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background(AppWindowBackground())
                        .toolbar {
                            ToolbarItemGroup(placement: .principal) {
                                Text(selection.title)
                                    .font(.title3)
                                    .fontWeight(.semibold)
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
            .padding(.bottom, PlaybackBarView.regularHeight)

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
            AlbumGridView(viewModel: libraryViewModel, sort: albumSort, query: albumQuery)
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
    @Binding var searchText: String

    private let focusItems: [SidebarDestination] = [.home, .futureFeatures]
    private let libraryItems: [SidebarDestination] = [.albums, .tracks, .sonicLens, .unreported]

    var body: some View {
        List(selection: $selection) {
            Section("聚焦") {
                ForEach(focusItems, id: \.self) { item in
                    SidebarItemView(item: item, isSelected: selection == item)
                        .tag(item)
                        .listRowInsets(EdgeInsets(top: 4, leading: 6, bottom: 4, trailing: 6))
                        .listRowBackground(Color.clear)
                }
            }

            Section("我的资料库") {
                ForEach(libraryItems, id: \.self) { item in
                    SidebarItemView(item: item, isSelected: selection == item)
                        .tag(item)
                        .listRowInsets(EdgeInsets(top: 4, leading: 6, bottom: 4, trailing: 6))
                        .listRowBackground(Color.clear)
                }
            }
        }
        .listStyle(.sidebar)
        .searchable(text: $searchText, placement: .sidebar)
        .navigationTitle("音眸")
        .frame(minWidth: 220, idealWidth: 220, maxWidth: 260)
        .background(
            ZStack {
                LinearGradient(
                    colors: [
                        Color(nsColor: NSColor.windowBackgroundColor).opacity(0.9),
                        Color(nsColor: NSColor.controlBackgroundColor).opacity(0.8)
                    ],
                    startPoint: .top,
                    endPoint: .bottom
                )
                Rectangle()
                    .fill(.ultraThinMaterial)
            }
        )
        .overlay(
            Rectangle()
                .fill(Color.white.opacity(0.08))
                .frame(width: 1),
            alignment: .trailing
        )
    }
}

struct SidebarItemView: View {
    let item: SidebarDestination
    let isSelected: Bool
    @State private var isHovered = false

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
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(isSelected ? Color.accentColor.opacity(0.2) : (isHovered ? Color.primary.opacity(0.06) : Color.clear))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.white.opacity(isSelected ? 0.35 : 0.1), lineWidth: 1)
        )
        .contentShape(Rectangle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.15)) {
                isHovered = hovering
            }
        }
    }
}

struct AppWindowBackground: View {
    var body: some View {
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

            Rectangle()
                .fill(.ultraThinMaterial)
                .opacity(0.28)
        }
        .ignoresSafeArea()
    }
}

struct AppBackground: View {
    var body: some View {
        AppWindowBackground()
    }
}
#else
struct AppBackground: View {
    var body: some View {
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

            Rectangle()
                .fill(.ultraThinMaterial)
                .opacity(0.2)
                .ignoresSafeArea()
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
    }
}

struct UnreportedRow: View {
    let record: UnscrobbledRecord
    @State private var isHovered = false

    var body: some View {
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
                .fill(isHovered ? Color.primary.opacity(0.05) : .clear)
        )
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 12))
        .onHover { hovering in
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
                Button(sort.rawValue) {
                    selection = sort
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
                Button(filter.rawValue) {
                    selection = filter
                }
            }
        } label: {
            ToolbarPillLabel(title: "筛选", systemImage: "line.3.horizontal.decrease.circle")
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
