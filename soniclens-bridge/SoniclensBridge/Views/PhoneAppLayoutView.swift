import SwiftUI
import OSLog

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
    @Environment(PlaybackStore.self) private var playbackStore
    @EnvironmentObject private var insightCoordinator: InsightAnalysisCoordinator
    @Environment(\.scenePhase) private var scenePhase
    @State private var selection: PhoneTabDestination = .home
    @State private var showNowPlaying = false
    @State private var routedTrack: TrackDetailRoute?
    @State private var routedAlbum: AlbumDetailRoute?
    @StateObject private var libraryViewModel = LibraryViewModel()
    @AppStorage("soniclens.performanceMode") private var performanceModeEnabled = false
    private let tabBarHeight: CGFloat = 49
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "PhoneAppLayout")

    var body: some View {
        GeometryReader { geo in
            ZStack(alignment: .bottom) {
                TabView(selection: $selection) {
                    phoneNavigationTab(for: .home) {
                        PhoneHomeView(onOpenNowPlaying: openNowPlaying)
                            .toolbar {
                                UnifiedLibraryToolbarGroup(performanceModeEnabled: $performanceModeEnabled)
                            }
                    }

                    phoneNavigationTab(for: .albums) {
                        PhoneAlbumLibraryTab(performanceModeEnabled: $performanceModeEnabled, viewModel: libraryViewModel, routedAlbum: $routedAlbum)
                    }

                    phoneNavigationTab(for: .tracks) {
                        PhoneTrackLibraryTab(performanceModeEnabled: $performanceModeEnabled, viewModel: libraryViewModel, routedTrack: $routedTrack)
                    }

                    phoneNavigationTab(for: .sonicLens) {
                        SonicLensInsightsView(viewModel: libraryViewModel)
                            .toolbar {
                                UnifiedLibraryToolbarGroup(performanceModeEnabled: $performanceModeEnabled)
                            }
                    }

                    phoneNavigationTab(for: .unreported) {
                        UnreportedListView(viewModel: libraryViewModel)
                            .toolbar {
                                UnifiedLibraryToolbarGroup(performanceModeEnabled: $performanceModeEnabled)
                            }
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
                if let nowPlaying = playbackStore.nowPlaying, playbackStore.hasActiveNowPlaying {
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
                await insightCoordinator.reconcileIfNeeded(using: server)
            }
        }
        .onOpenURL { url in
            Task {
                await insightCoordinator.handleDeepLink(url, using: store.currentServer)
                await applyPendingInsightRoute()
            }
        }
        .task(id: pendingInsightRouteToken) {
            await applyPendingInsightRoute()
        }
        .onReceive(NotificationCenter.default.publisher(for: .librarySyncDidUpdate)) { _ in
            guard let server = store.currentServer else { return }
            Task { await libraryViewModel.refresh(using: server) }
        }
        .onChange(of: selection) { _, newValue in
            logger.info("切换手机底部标签 \(newValue.title, privacy: .public)")
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

                    ToolbarItem(placement: .topBarLeading) {
                        Text(destination.title)
                            .font(.headline.weight(.semibold))
                    }
                }
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            if playbackStore.hasActiveNowPlaying {
                Color.clear
                    .frame(height: PlaybackBarView.compactHeight + 8)
                    .allowsHitTesting(false)
            }
        }
        .tabItem {
            Label(destination.title, systemImage: destination.systemImage)
        }
        .tag(destination)
    }

    private func openNowPlaying() {
        guard playbackStore.hasActiveNowPlaying else { return }
        showNowPlaying = true
    }

    private var pendingInsightRouteToken: String {
        guard let route = insightCoordinator.pendingRoute else { return "none" }
        return "\(route.jobID)::\(route.targetType.rawValue)"
    }

    @MainActor
    private func applyPendingInsightRoute() async {
        guard let route = insightCoordinator.pendingRoute else { return }
        switch route.targetType {
        case .track:
            guard let track = route.track else { return }
            selection = .tracks
            try? await Task.sleep(nanoseconds: 120_000_000)
            routedTrack = TrackDetailRoute(track: track, selectedTab: .insights)
        case .album:
            guard let albumID = route.albumID else { return }
            selection = .albums
            try? await Task.sleep(nanoseconds: 120_000_000)
            routedAlbum = AlbumDetailRoute(albumID: albumID, selectedTab: .insights)
        }
        insightCoordinator.consumePendingRoute()
    }
}


private struct PhoneAlbumLibraryTab: View {
    @EnvironmentObject private var store: AppStore
    @Binding var performanceModeEnabled: Bool
    @State private var sort: LibrarySort = .recent
    @State private var searchText = ""
    @State private var committedQuery = ""
    @State private var searchCommitTask: Task<Void, Never>?

    @ObservedObject var viewModel: LibraryViewModel
    @Binding var routedAlbum: AlbumDetailRoute?

    var body: some View {
        VStack(spacing: 0) {
            PhoneInlineSearchBar(text: $searchText, prompt: "搜索专辑")
                .padding(.horizontal, 16)
                .padding(.top, 12)
                .padding(.bottom, 6)

            AlbumGridView(
                viewModel: viewModel,
                sort: sort,
                query: committedQuery,
                artworkBaseURL: store.currentServer?.artworkBaseURL,
                statusSummary: LibraryStatusSummary.album(sort: sort),
                prefersCompactLayout: true
            )
            .scrollDismissesKeyboard(.immediately)
        }
        .toolbar {
            UnifiedLibraryToolbarGroup(performanceModeEnabled: $performanceModeEnabled) {
                LibrarySortMenu(title: "排序", selection: $sort, options: LibrarySort.albumOptions)
            }
        }
        .onChange(of: searchText) { _, value in
            scheduleSearchCommit(value)
        }
        .onDisappear {
            searchCommitTask?.cancel()
        }
        .navigationDestination(item: $routedAlbum) { route in
            albumDetailDestination(albumID: route.albumID, selectedTab: route.selectedTab)
        }
    }

    private func scheduleSearchCommit(_ text: String) {
        searchCommitTask?.cancel()
        let pendingText = text
        searchCommitTask = Task { @MainActor in
            do {
                try await Task.sleep(nanoseconds: 250_000_000)
            } catch {
                return
            }

            guard !Task.isCancelled else { return }
            guard committedQuery != pendingText else { return }
            committedQuery = pendingText
        }
    }
}

private struct PhoneTrackLibraryTab: View {
    @Binding var performanceModeEnabled: Bool
    @State private var sort: LibrarySort = .recent
    @State private var filter: TrackFilter = .all
    @State private var searchText = ""
    @State private var committedQuery = ""
    @State private var searchCommitTask: Task<Void, Never>?

    @ObservedObject var viewModel: LibraryViewModel
    @Binding var routedTrack: TrackDetailRoute?

    var body: some View {
        VStack(spacing: 0) {
            PhoneInlineSearchBar(text: $searchText, prompt: "搜索曲目")
                .padding(.horizontal, 16)
                .padding(.top, 12)
                .padding(.bottom, 6)

            TrackListView(
                viewModel: viewModel,
                sort: sort,
                filter: filter,
                query: committedQuery,
                showsInlineControls: false
            )
            .scrollDismissesKeyboard(.immediately)
        }
        .toolbar {
            UnifiedLibraryToolbarGroup(performanceModeEnabled: $performanceModeEnabled) {
                LibrarySortMenu(title: "排序", selection: $sort, options: LibrarySort.trackOptions)
                TrackFilterMenu(selection: $filter)
            }
        }
        .navigationDestination(item: $routedTrack) { route in
            trackDetailDestination(track: route.track, selectedTab: route.selectedTab)
        }
        .onChange(of: searchText) { _, value in
            scheduleSearchCommit(value)
        }
        .onDisappear {
            searchCommitTask?.cancel()
        }
    }

    private func scheduleSearchCommit(_ text: String) {
        searchCommitTask?.cancel()
        let pendingText = text
        searchCommitTask = Task { @MainActor in
            do {
                try await Task.sleep(nanoseconds: 250_000_000)
            } catch {
                return
            }

            guard !Task.isCancelled else { return }
            guard committedQuery != pendingText else { return }
            committedQuery = pendingText
        }
    }
}

private struct PhoneInlineSearchBar: View {
    @Binding var text: String
    let prompt: String

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: "magnifyingglass")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(SonicTheme.textSecondary)

            TextField(prompt, text: $text)
                .textFieldStyle(.plain)
                .autocorrectionDisabled(true)

            if !text.isEmpty {
                Button {
                    text = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.subheadline)
                        .foregroundStyle(SonicTheme.textSecondary)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .fill(SonicTheme.card.opacity(0.92))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
        )
    }
}
#endif
