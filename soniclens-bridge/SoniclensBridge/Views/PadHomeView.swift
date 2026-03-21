import SwiftUI

#if os(iOS)
struct PadHomeView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var viewModel = HomeViewModel()
    @State private var selectedRecentTrack: Track?
    @State private var selectedAlbumID: Int64?

    let onOpenNowPlaying: () -> Void

    var body: some View {
        ZStack {
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [SonicTheme.background, SonicTheme.background.opacity(0.98), Color.accentColor.opacity(0.10)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: [
                    AmbientOrb(color: SonicTheme.primary.opacity(0.26), size: 520, blur: 140, opacity: 0.7, offsetFrom: CGSize(width: -220, height: -280), offsetTo: CGSize(width: -140, height: -160), duration: 20),
                    AmbientOrb(color: SonicTheme.secondaryAccent.opacity(0.18), size: 420, blur: 150, opacity: 0.7, offsetFrom: CGSize(width: 280, height: 120), offsetTo: CGSize(width: 180, height: 220), duration: 26)
                ]
            )

            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    PadHomeHero(nowPlaying: store.nowPlaying, onOpenNowPlaying: onOpenNowPlaying)

                    DashboardStatsSection(stats: viewModel.stats)

                    DashboardTrendSection(
                        points: viewModel.trendPoints,
                        hourlyData: viewModel.hourlyData
                    )
                    .frame(maxWidth: .infinity)

                    ViewThatFits(in: .horizontal) {
                        HStack(alignment: .top, spacing: 16) {
                            TopGenresCard(topGenres: viewModel.topGenres)
                                .frame(maxWidth: .infinity)
                            TopArtistsCard(topArtists: viewModel.topArtistsByPlays)
                                .frame(maxWidth: .infinity)
                        }

                        VStack(spacing: 16) {
                            TopGenresCard(topGenres: viewModel.topGenres)
                            TopArtistsCard(topArtists: viewModel.topArtistsByPlays)
                        }
                    }

                    TopAlbumsCard(topAlbums: viewModel.topAlbums, onAlbumTap: { selectedAlbumID = $0 })

                    ViewThatFits(in: .horizontal) {
                        HStack(alignment: .top, spacing: 16) {
                            RecentPlaysSection(
                                items: viewModel.recentPlays,
                                onTrackTap: { selectedRecentTrack = $0 }
                            )
                            .frame(maxWidth: .infinity)

                            RankingsCard(
                                topArtistsByPlays: viewModel.topArtistsByPlays,
                                topTracks: viewModel.topTracks,
                                onTrackTap: { selectedRecentTrack = $0 }
                            )
                            .frame(maxWidth: .infinity)
                        }

                        VStack(spacing: 16) {
                            RecentPlaysSection(
                                items: viewModel.recentPlays,
                                onTrackTap: { selectedRecentTrack = $0 }
                            )
                            RankingsCard(
                                topArtistsByPlays: viewModel.topArtistsByPlays,
                                topTracks: viewModel.topTracks,
                                onTrackTap: { selectedRecentTrack = $0 }
                            )
                        }
                    }
                }
                .padding(.horizontal, 24)
                .padding(.top, 20)
                .padding(.bottom, 36)
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.stats == nil {
                LoadingOverlay()
            }
        }
        .task {
            if let server = store.currentServer {
                await viewModel.load(using: server)
            }
        }
        .onChange(of: store.currentServer) { _, server in
            guard let server else { return }
            Task { await viewModel.load(using: server) }
        }
        .navigationDestination(item: $selectedRecentTrack) { track in
            TrackDetailView(track: track)
        }
        .navigationDestination(item: $selectedAlbumID) { albumID in
            albumDetailDestination(albumID: albumID)
        }
    }
}
#endif

private struct PadHomeHero: View {
    let nowPlaying: NowPlaying?
    let onOpenNowPlaying: () -> Void

    var body: some View {
        GlassPanel(cornerRadius: 28, padding: 0) {
            VStack(alignment: .leading, spacing: 18) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("SonicLens Bridge for iPad")
                        .font(.system(size: 14, weight: .semibold, design: .rounded))
                        .foregroundStyle(Color.white.opacity(0.88))
                    Text("声之透镜 · 深度解析 · 聆听之印记")
                        .font(.system(size: 28, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                    Text("音乐不仅是流动的空气，更是我们生命中不曾停歇的数字资产")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(Color.white.opacity(0.82))
                        .fixedSize(horizontal: false, vertical: true)
                }

                if let nowPlaying {
                    Button(action: onOpenNowPlaying) {
                        HStack(spacing: 16) {
                            PlaybackArtworkView(artworkURL: nowPlaying.artwork)
                                .frame(width: 60, height: 60)

                            VStack(alignment: .leading, spacing: 4) {
                                Text("正在播放")
                                    .font(.caption.weight(.semibold))
                                    .foregroundStyle(Color.white.opacity(0.72))
                                Text(nowPlaying.track)
                                    .font(.title3.weight(.semibold))
                                    .foregroundStyle(.white)
                                    .lineLimit(1)
                                Text([nowPlaying.artist, nowPlaying.album].compactMap { $0 }.joined(separator: " · "))
                                    .font(.subheadline)
                                    .foregroundStyle(Color.white.opacity(0.76))
                                    .lineLimit(1)
                            }

                            Spacer()

                            Label("进入沉浸播放中", systemImage: "arrow.up.left.and.arrow.down.right")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 10)
                                .background(Color.white.opacity(0.14), in: Capsule())
                        }
                        .padding(18)
                        .background(Color.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 22))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(24)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                LinearGradient(
                    colors: [
                        SonicTheme.primary.opacity(0.86),
                        SonicTheme.secondaryAccent.opacity(0.52),
                        Color.black.opacity(0.18)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
        }
    }
}
