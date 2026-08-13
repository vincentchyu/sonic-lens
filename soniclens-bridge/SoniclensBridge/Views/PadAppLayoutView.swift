import SwiftUI

#if os(iOS)
enum PadSidebarDestination: String, CaseIterable, Hashable {
    case home
    case albums
    case tracks
    case sonicLens
    case unreported

    var title: String {
        switch self {
        case .home:
            return "主页"
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

    var subtitle: String {
        switch self {
        case .home:
            return "仪表盘与播放中入口"
        case .albums:
            return "快乐中国的喇叭花"
        case .tracks:
            return "列表与筛选"
        case .sonicLens:
            return "洞察与解析"
        case .unreported:
            return "待上报记录"
        }
    }

    var systemImage: String {
        switch self {
        case .home:
            return "square.grid.2x2"
        case .albums:
            return "square.stack.3d.up"
        case .tracks:
            return "music.note.list"
        case .sonicLens:
            return "sparkles.rectangle.stack"
        case .unreported:
            return "clock.badge.exclamationmark"
        }
    }
}
#endif

struct PadAppLayoutView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(PlaybackStore.self) private var playbackStore
    @Environment(\.scenePhase) private var scenePhase
    @State private var selection: PadSidebarDestination = .home
    @State private var showNowPlaying = false
    @State private var albumSort: LibrarySort = .recent
    @State private var trackSort: LibrarySort = .recent
    @State private var trackFilter: TrackFilter = .all
    @State private var albumQuery = ""
    @State private var trackQuery = ""
    @StateObject private var libraryViewModel = LibraryViewModel()
    @AppStorage("soniclens.performanceMode") private var performanceModeEnabled = false

    var body: some View {
        NavigationSplitView {
            PadSidebarView(selection: $selection)
        } detail: {
            NavigationStack {
                detailContent
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(AppBackground(useMaterial: false))
                    .toolbar {
                        ToolbarItem(placement: .principal) {
                            ToolbarTitleSubtitleView(
                                title: selection.title,
                                subtitle: toolbarSubtitle
                            )
                            .equatable()
                        }

                        UnifiedLibraryToolbarGroup(performanceModeEnabled: $performanceModeEnabled) {
                            toolbarContent
                        }
                    }
            }
            .safeAreaInset(edge: .bottom, spacing: 0) {
                PlaybackBarView(isExpanded: $showNowPlaying)
                    .environmentObject(store)
            }
        }
        .navigationSplitViewStyle(.balanced)
        .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
        .fullScreenCover(isPresented: $showNowPlaying) {
            Group {
                if let nowPlaying = playbackStore.nowPlaying, playbackStore.hasActiveNowPlaying {
                    PadNowPlayingView(nowPlaying: nowPlaying) {
                        showNowPlaying = false
                    }
                    .environmentObject(store)
                } else {
                    Color.clear
                        .onAppear {
                            showNowPlaying = false
                        }
                }
            }
            .onChange(of: playbackStore.hasActiveNowPlaying) { _, newValue in
                if !newValue {
                    showNowPlaying = false
                }
            }
        }
        .task {
            guard let server = store.currentServer else { return }
            await libraryViewModel.load(using: server)
        }
        .onChange(of: store.currentServer) { _, server in
            guard let server else { return }
            Task { await libraryViewModel.load(using: server) }
        }
        .onChange(of: scenePhase) { _, phase in
            guard phase == .active else { return }
            Task {
                await store.performForegroundConnectionHealthCheckIfNeeded()
                guard let server = store.currentServer, store.isConnectionHealthy else { return }
                await libraryViewModel.refresh(using: server)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .librarySyncDidUpdate)) { _ in
            guard let server = store.currentServer else { return }
            Task { await libraryViewModel.refresh(using: server) }
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
    private var detailContent: some View {
        switch selection {
        case .home:
            PadHomeView(onOpenNowPlaying: {
                if playbackStore.hasActiveNowPlaying {
                    showNowPlaying = true
                }
            })
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

    @ViewBuilder
    private var disconnectToolbarButton: some View {
        if store.currentServer != nil {
            Button {
                store.disconnect()
            } label: {
                ToolbarIconButton(systemImage: "power", helpText: "断开当前服务端")
            }
            .buttonStyle(.plain)
            .help("断开当前服务端")
        }
    }
}

private struct PadSidebarView: View {
    @Binding var selection: PadSidebarDestination

    var body: some View {
        List {
            Section("浏览") {
                sidebarLink(for: .home)
                sidebarLink(for: .albums)
                sidebarLink(for: .tracks)
                sidebarLink(for: .unreported)
            }

            Section("深度内容") {
                sidebarLink(for: .sonicLens)
            }
        }
        .listStyle(.sidebar)
        .navigationTitle("音眸桥接")
    }

    private func sidebarLink(for destination: PadSidebarDestination) -> some View {
        Button {
            selection = destination
        } label: {
            Label(destination.title, systemImage: destination.systemImage)
                .foregroundStyle(selection == destination ? Color.accentColor : Color.primary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.vertical, 6)
                .padding(.horizontal, 8)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
