import SwiftUI

struct GenreAlbumsSheet: View {
    @EnvironmentObject var store: AppStore
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: GenreAlbumsViewModel

    let onSelectAlbum: (Int64) -> Void

    init(
        item: HomeHotGenrePresentationItem,
        onSelectAlbum: @escaping (Int64) -> Void
    ) {
        _viewModel = StateObject(wrappedValue: GenreAlbumsViewModel(item: item))
        self.onSelectAlbum = onSelectAlbum
    }

    init(
        rawGenreName: String,
        genreTitle: String,
        rank: Int? = nil,
        playCount: Int? = nil,
        accentKey: HomeHotAccentKey = .tide,
        onSelectAlbum: @escaping (Int64) -> Void
    ) {
        _viewModel = StateObject(wrappedValue: GenreAlbumsViewModel(
            rawGenreName: rawGenreName,
            genreTitle: genreTitle,
            rank: rank,
            playCount: playCount,
            accentKey: accentKey
        ))
        self.onSelectAlbum = onSelectAlbum
    }

    var body: some View {
        ZStack {
            // 背景沉浸玻璃氛围
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [SonicTheme.background, SonicTheme.background],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: [
                    AmbientOrb(
                        color: viewModel.accentKey.solidColor.opacity(0.32),
                        size: 380,
                        blur: 90,
                        opacity: 0.65,
                        offsetFrom: CGSize(width: -120, height: -160),
                        offsetTo: CGSize(width: -40, height: -80),
                        duration: 16
                    ),
                    AmbientOrb(
                        color: viewModel.accentKey.solidColor.opacity(0.18),
                        size: 300,
                        blur: 110,
                        opacity: 0.50,
                        offsetFrom: CGSize(width: 160, height: 120),
                        offsetTo: CGSize(width: 80, height: 180),
                        duration: 20
                    )
                ],
                renderingStyle: .staticHome
            )

            VStack(spacing: 0) {
                // 1. 顶部 Header 导航栏
                topHeaderControlBar

                // 2. 紧凑型面板核心视图
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        // 整合型 Hero 统计卡片
                        heroSummaryHeaderCard

                        // 专辑列表与状态
                        if viewModel.isLoading && viewModel.albums.isEmpty {
                            loadingStateView
                        } else if let error = viewModel.errorMessage, viewModel.albums.isEmpty {
                            ErrorBanner(message: error)
                        } else if viewModel.albums.isEmpty {
                            emptyStateView
                        } else {
                            albumsGridView
                        }
                    }
                    .padding(.horizontal, 18)
                    .padding(.bottom, 24)
                }
            }
        }
        .frame(
            minWidth: 480, idealWidth: 520, maxWidth: 560,
            minHeight: 460, idealHeight: 530, maxHeight: 620
        )
        .onAppear {
            viewModel.loadAlbums(server: store.currentServer)
        }
    }

    // MARK: - 1. 顶部 Header 导航栏
    private var topHeaderControlBar: some View {
        HStack(alignment: .center) {
            Button {
                dismiss()
            } label: {
                Image(systemName: "xmark.circle.fill")
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundStyle(SonicTheme.textSecondary.opacity(0.85))
            }
            .buttonStyle(.plain)

            Spacer()

            HStack(spacing: 6) {
                Image(systemName: "sparkles")
                    .font(.system(size: 12, weight: .semibold))
                Text("流派关联探查")
                    .font(.system(size: 12, weight: .semibold))
            }
            .foregroundStyle(viewModel.accentKey.solidColor)
            .padding(.horizontal, 12)
            .padding(.vertical, 5)
            .background(
                Capsule()
                    .fill(viewModel.accentKey.solidColor.opacity(0.12))
            )
        }
        .padding(.horizontal, 18)
        .padding(.top, 14)
        .padding(.bottom, 10)
    }

    // MARK: - 2. 整合型 Hero 统计卡片
    private var heroSummaryHeaderCard: some View {
        GlassPanel(cornerRadius: 18, padding: 16) {
            VStack(alignment: .leading, spacing: 12) {
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    if let rank = viewModel.rank {
                        RankBadge(rank: rank, accentKey: viewModel.accentKey, style: .prominent)
                    }

                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Text(viewModel.genreTitle)
                            .font(.system(size: 24, weight: .bold, design: .rounded))
                            .foregroundStyle(SonicTheme.textPrimary)

                        if !viewModel.rawGenreName.isEmpty && viewModel.rawGenreName.lowercased() != viewModel.genreTitle.lowercased() {
                            Text(viewModel.rawGenreName.uppercased())
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
                                .foregroundStyle(viewModel.accentKey.solidColor.opacity(0.90))
                        }
                    }

                    Spacer()

                    Image(systemName: "guitars.fill")
                        .font(.system(size: 26))
                        .foregroundStyle(
                            LinearGradient(
                                colors: [
                                    viewModel.accentKey.solidColor,
                                    viewModel.accentKey.solidColor.opacity(0.5)
                                ],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            )
                        )
                }

                Divider()
                    .background(Color.white.opacity(0.12))

                HStack(spacing: 16) {
                    if let playCount = viewModel.playCount {
                        HStack(spacing: 5) {
                            Image(systemName: "play.circle.fill")
                                .font(.system(size: 12))
                                .foregroundStyle(viewModel.accentKey.solidColor)
                            Text("累计播放 \(playCount) 次")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(SonicTheme.textPrimary)
                        }
                    }

                    HStack(spacing: 5) {
                        Image(systemName: "square.stack.3d.up.fill")
                            .font(.system(size: 12))
                            .foregroundStyle(viewModel.accentKey.solidColor)
                        Text("关联收录 \(viewModel.totalCount > 0 ? viewModel.totalCount : viewModel.albums.count) 张专辑")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(SonicTheme.textPrimary)
                    }

                    Spacer()
                }
            }
        }
    }

    // MARK: - 3. 专辑 2 列自适应网格
    private var albumsGridView: some View {
        let columns = [
            GridItem(.adaptive(minimum: 180, maximum: 230), spacing: 14)
        ]

        return LazyVGrid(columns: columns, spacing: 14) {
            ForEach(viewModel.albums) { album in
                Button {
                    dismiss()
                    onSelectAlbum(album.id)
                } label: {
                    AlbumGridCard(
                        album: album,
                        artworkBaseURL: store.currentServer?.artworkBaseURL
                    )
                }
                .buttonStyle(.plain)
            }
        }
    }

    private var loadingStateView: some View {
        VStack(spacing: 14) {
            ProgressView()
                .scaleEffect(1.1)
            Text("正在加载关联流派专辑...")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(SonicTheme.textSecondary)
        }
        .frame(maxWidth: .infinity, minHeight: 180)
    }

    private var emptyStateView: some View {
        GlassPanel(cornerRadius: 18, padding: 24) {
            VStack(spacing: 12) {
                Image(systemName: "music.note.house")
                    .font(.system(size: 36))
                    .foregroundStyle(viewModel.accentKey.solidColor.opacity(0.6))
                Text("暂无关联专辑")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(SonicTheme.textPrimary)
                Text("资料库中暂未检测到标记为 “\(viewModel.genreTitle)” 的物理专辑。")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(SonicTheme.textSecondary)
                    .multilineTextAlignment(.center)
            }
            .frame(maxWidth: .infinity, minHeight: 160)
        }
    }
}
