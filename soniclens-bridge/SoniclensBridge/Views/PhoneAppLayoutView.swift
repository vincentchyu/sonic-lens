import SwiftUI

#if os(iOS)
enum PhoneTabDestination: String, CaseIterable, Hashable {
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

    var systemImage: String {
        switch self {
        case .home:
            return "house"
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

struct PhoneAppLayoutView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(\.scenePhase) private var scenePhase
    @State private var selection: PhoneTabDestination = .home
    @State private var showNowPlaying = false
    @State private var albumSort: LibrarySort = .recent
    @State private var trackSort: LibrarySort = .recent
    @State private var trackFilter: TrackFilter = .all
    @State private var albumQuery = ""
    @State private var trackQuery = ""
    @StateObject private var libraryViewModel = LibraryViewModel()
    @AppStorage("soniclens.performanceMode") private var performanceModeEnabled = false
    private let tabBarHeight: CGFloat = 49

    var body: some View {
        GeometryReader { geo in
            ZStack(alignment: .bottom) {
                TabView(selection: $selection) {
                    phoneNavigationTab(for: .home) {
                        PhoneHomeView(onOpenNowPlaying: openNowPlaying)
                    }

                    phoneNavigationTab(for: .albums) {
                        AlbumGridView(
                            viewModel: libraryViewModel,
                            sort: albumSort,
                            query: albumQuery,
                            prefersCompactLayout: true
                        )
                        .toolbar {
                            ToolbarItem(placement: .topBarTrailing) {
                                LibrarySortMenu(title: "排序", selection: $albumSort, options: LibrarySort.albumOptions)
                            }
                        }
                        .searchable(text: $albumQuery, placement: .navigationBarDrawer(displayMode: .always), prompt: "搜索专辑")
                    }

                    phoneNavigationTab(for: .tracks) {
                        TrackListView(
                            viewModel: libraryViewModel,
                            sort: trackSort,
                            filter: trackFilter,
                            query: trackQuery,
                            showsInlineControls: true
                        )
                    }

                    phoneNavigationTab(for: .sonicLens) {
                        SonicLensInsightsView(viewModel: libraryViewModel)
                    }

                    phoneNavigationTab(for: .unreported) {
                        UnreportedListView(viewModel: libraryViewModel)
                    }
                }
                .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)

                // 临时隐藏 iPhone 底部播放条，保留 fullScreenCover 播放页能力。
//                PlaybackBarView(isExpanded: $showNowPlaying, style: .compact)
//                    .environmentObject(store)
//                    .padding(.bottom, tabBarHeight + max(geo.safeAreaInsets.bottom, 0))
//                    .zIndex(1)
            }
//            .safeAreaInset(edge: .bottom, spacing: 0) {
//                Color.clear
//                    .frame(height: PlaybackBarView.compactHeight + tabBarHeight + max(geo.safeAreaInsets.bottom, 0))
//                    .allowsHitTesting(false)
//            }
        }
        .fullScreenCover(isPresented: $showNowPlaying) {
            Group {
                if let nowPlaying = store.nowPlaying {
                    PhoneNowPlayingView(nowPlaying: nowPlaying) {
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
            .onChange(of: store.nowPlaying != nil) { _, newValue in
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
    private func phoneNavigationTab<Content: View>(
        for destination: PhoneTabDestination,
        @ViewBuilder content: () -> Content
    ) -> some View {
        NavigationStack {
            content()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(AppBackground())
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Text(destination.title)
                            .font(.headline.weight(.semibold))
                    }

                    ToolbarItem(placement: .topBarTrailing) {
                        performanceMenu
                    }
                }
        }
        .tabItem {
            Label(destination.title, systemImage: destination.systemImage)
        }
        .tag(destination)
    }

    private var performanceMenu: some View {
        Menu {
            Toggle("性能模式", isOn: $performanceModeEnabled)
            Text("降低动效、阴影和复杂材质负担")
                .font(.caption)
        } label: {
            Image(systemName: "gauge.medium")
        }
    }

    private func openNowPlaying() {
        guard store.nowPlaying != nil else { return }
        showNowPlaying = true
    }
}
#endif
