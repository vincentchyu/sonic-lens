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
            return "本地索引专辑库"
        case .tracks:
            return "长列表与筛选"
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
                    .background(AppBackground())
                    .toolbar {
                        ToolbarItem(placement: .principal) {
                            VStack(spacing: 2) {
                                Text(selection.title)
                                    .font(.headline)
                                Text(selection.subtitle)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }

                        ToolbarItemGroup(placement: .topBarTrailing) {
                            performanceMenu
                            toolbarContent
                        }
                    }
            }
        }
        .navigationSplitViewStyle(.balanced)
        .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
        .safeAreaInset(edge: .bottom, spacing: 0) {
            PlaybackBarView(isExpanded: $showNowPlaying)
                .environmentObject(store)
        }
        .fullScreenCover(isPresented: $showNowPlaying) {
            Group {
                if let nowPlaying = store.nowPlaying {
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
            .onChange(of: store.nowPlaying != nil) { oldValue, newValue in
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
            guard phase == .active, let server = store.currentServer else { return }
            Task { await libraryViewModel.refresh(using: server) }
        }
        .onReceive(NotificationCenter.default.publisher(for: .librarySyncDidUpdate)) { _ in
            guard let server = store.currentServer else { return }
            Task { await libraryViewModel.refresh(using: server) }
        }
    }

    @ViewBuilder
    private var detailContent: some View {
        switch selection {
        case .home:
            PadHomeView(onOpenNowPlaying: {
                if store.nowPlaying != nil {
                    showNowPlaying = true
                }
            })
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

    private var performanceMenu: some View {
        Menu {
            Toggle("性能模式", isOn: $performanceModeEnabled)
            Text("降低动效、阴影和复杂材质负担")
                .font(.caption)
        } label: {
            ToolbarPillLabel(title: "性能", systemImage: "gauge.medium")
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
            }

            Section("深度内容") {
                sidebarLink(for: .sonicLens)
                sidebarLink(for: .unreported)
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
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.vertical, 6)
                .padding(.horizontal, 8)
                .background(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .fill(selection == destination ? Color.accentColor.opacity(0.18) : Color.clear)
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(selection == destination ? Color.accentColor.opacity(0.28) : Color.clear, lineWidth: 1)
                )
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
