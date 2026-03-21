import SwiftUI
import Combine

@ViewBuilder
func albumDetailDestination(albumID: Int64) -> some View {
    AlbumDetailView(albumID: albumID)
}

struct AlbumDetailView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var viewModel: AlbumDetailViewModel

    let albumID: Int64
    @State private var isCurationExpanded: Bool = true

    init(albumID: Int64) {
        self.albumID = albumID
        _viewModel = StateObject(wrappedValue: AlbumDetailViewModel())
    }

    var body: some View {
        ZStack {
            if let message = viewModel.errorMessage {
                ErrorBanner(message: message)
                    .padding(16)
            }

            if let detail = viewModel.detail {
                AlbumDetailPlatformContainer(
                    albumID: albumID,
                    detail: detail,
                    candidates: viewModel.candidates,
                    isCurationExpanded: $isCurationExpanded,
                    onSearch: searchCandidates,
                    onConfirm: confirmCandidate
                )
            }

            if viewModel.isLoading && viewModel.detail == nil {
                LoadingOverlay()
            }

            if !viewModel.isLoading, viewModel.detail == nil, viewModel.errorMessage == nil {
                EmptyStateView(
                    title: "暂无专辑详情",
                    subtitle: "请稍后重试或返回重进。"
                )
                .padding(32)
            }
        }
        .navigationTitle("专辑详情")
        .toolbar {
            Button {
                if let detail = viewModel.detail {
                    exportSnapshotPNG(
                        AlbumSnapshotView(detail: detail, candidates: viewModel.candidates).padding(32),
                        suggestedFilename: "\(detail.artist)-\(detail.name)-专辑"
                    )
                }
            } label: {
                Label("导出快照", systemImage: "square.and.arrow.up")
            }
            .disabled(viewModel.detail == nil)
        }
        .task(id: albumID) {
            if let server = store.currentServer {
                await viewModel.load(using: server, albumID: albumID)
            }
        }
    }

    private func searchCandidates() {
        guard let server = store.currentServer else { return }
        Task { await viewModel.searchCandidates(using: server, albumID: albumID) }
    }

    private func confirmCandidate(_ candidate: ReleaseCandidate) {
        guard let server = store.currentServer else { return }
        Task { await viewModel.confirmSelection(using: server, albumID: albumID, candidate: candidate) }
    }
}

private struct AlbumDetailPlatformContainer: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    @Binding var isCurationExpanded: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        #if os(iOS)
        if UIDevice.current.userInterfaceIdiom == .phone {
            PhoneAlbumDetailView(
                albumID: albumID,
                detail: detail,
                candidates: candidates,
                isCurationExpanded: $isCurationExpanded,
                onSearch: onSearch,
                onConfirm: onConfirm
            )
        } else {
            RegularAlbumDetailView(
                albumID: albumID,
                detail: detail,
                candidates: candidates,
                isCurationExpanded: $isCurationExpanded,
                onSearch: onSearch,
                onConfirm: onConfirm
            )
        }
        #else
        RegularAlbumDetailView(
            albumID: albumID,
            detail: detail,
            candidates: candidates,
            isCurationExpanded: $isCurationExpanded,
            onSearch: onSearch,
            onConfirm: onConfirm
        )
        #endif
    }
}

private struct RegularAlbumDetailView: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    @Binding var isCurationExpanded: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        ScrollView {
            AlbumDetailContentView(
                albumID: albumID,
                detail: detail,
                candidates: candidates,
                isCurationExpanded: $isCurationExpanded,
                heroLayout: .regular,
                onSearch: onSearch,
                onConfirm: onConfirm
            )
            .padding(32)
        }
    }
}

private struct PhoneAlbumDetailView: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    @Binding var isCurationExpanded: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        ScrollView {
            AlbumDetailContentView(
                albumID: albumID,
                detail: detail,
                candidates: candidates,
                isCurationExpanded: $isCurationExpanded,
                heroLayout: .phone,
                onSearch: onSearch,
                onConfirm: onConfirm
            )
            .padding(.horizontal, 16)
            .padding(.vertical, 20)
        }
    }
}

struct AlbumSnapshotView: View {
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]

    var body: some View {
        AlbumDetailContentView(
            albumID: detail.id,
            detail: detail,
            candidates: candidates,
            isCurationExpanded: .constant(true),
            heroLayout: .regular,
            onSearch: {},
            onConfirm: { _ in }
        )
    }
}

private struct AlbumDetailContentView: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    @Binding var isCurationExpanded: Bool
    let heroLayout: AlbumHeroLayout
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: heroLayout.sectionSpacing) {
            AlbumHeroSection(detail: detail, layout: heroLayout)

            AlbumTrackListSection(
                tracks: detail.tracks,
                isCompact: heroLayout == .phone
            )

            AlbumCurationSection(
                albumID: albumID,
                detail: detail,
                candidates: candidates,
                isExpanded: $isCurationExpanded,
                isCompact: heroLayout == .phone,
                onSearch: onSearch,
                onConfirm: onConfirm
            )
        }
    }
}

private enum AlbumHeroLayout {
    case regular
    case phone

    var artworkSize: CGFloat {
        switch self {
        case .regular:
            return 180
        case .phone:
            return 188
        }
    }

    var sectionSpacing: CGFloat {
        switch self {
        case .regular:
            return 20
        case .phone:
            return 16
        }
    }
}

struct AlbumTrackListSection: View {
    let tracks: [Track]
    let isCompact: Bool

    var body: some View {
        DetailSectionCard(title: "曲目", compact: isCompact) {
            VStack(alignment: .leading, spacing: 12) {
                AlbumTracksSummary(tracks: tracks, compact: isCompact)

                if discGroups.count > 1 {
                    LazyVStack(alignment: .leading, spacing: isCompact ? 14 : 16) {
                        ForEach(discGroups) { discGroup in
                            VStack(alignment: .leading, spacing: isCompact ? 8 : 10) {
                                AlbumDiscHeader(title: discGroup.title, compact: isCompact)

                                LazyVStack(spacing: isCompact ? 8 : 10) {
                                    ForEach(discGroup.tracks) { track in
                                        NavigationLink(destination: TrackDetailView(track: track)) {
                                            AlbumTrackRow(track: track, compact: isCompact)
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                        }
                    }
                } else {
                    LazyVStack(spacing: isCompact ? 8 : 10) {
                        ForEach(tracks) { track in
                            NavigationLink(destination: TrackDetailView(track: track)) {
                                AlbumTrackRow(track: track, compact: isCompact)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
        }
    }

    private var discGroups: [AlbumDiscGroup] {
        let grouped = Dictionary(grouping: tracks) { track in
            track.discNumber
        }

        return grouped
            .map { discNumber, tracks in
                AlbumDiscGroup(discNumber: discNumber, tracks: tracks.sorted(by: trackOrder))
            }
            .sorted { lhs, rhs in
                switch (lhs.discNumber, rhs.discNumber) {
                case let (l?, r?):
                    return l < r
                case (_?, nil):
                    return true
                case (nil, _?):
                    return false
                case (nil, nil):
                    return false
                }
            }
    }

    private func trackOrder(_ lhs: Track, _ rhs: Track) -> Bool {
        let lhsTrack = lhs.trackNumber ?? Int.max
        let rhsTrack = rhs.trackNumber ?? Int.max
        if lhsTrack != rhsTrack {
            return lhsTrack < rhsTrack
        }
        return lhs.id < rhs.id
    }
}

private struct AlbumDiscGroup: Identifiable {
    let discNumber: Int?
    let tracks: [Track]

    var id: String {
        discNumber.map { "disc-\($0)" } ?? "disc-unknown"
    }

    var title: String {
        if let discNumber {
            return "光盘 \(discNumber)"
        }
        return "未标记光盘"
    }
}

private struct AlbumDiscHeader: View {
    let title: String
    let compact: Bool

    var body: some View {
        Text(title)
            .font(compact ? .subheadline.weight(.semibold) : .headline.weight(.semibold))
            .foregroundStyle(.secondary)
            .padding(.top, compact ? 2 : 4)
    }
}

struct DetailSectionCard<Content: View>: View {
    let title: String
    var compact: Bool = false
    let content: Content

    init(title: String, compact: Bool = false, @ViewBuilder content: () -> Content) {
        self.title = title
        self.compact = compact
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: compact ? 10 : 12) {
            Text(title)
                .font(compact ? .headline : .title3)
                .fontWeight(.semibold)

            content
        }
        .padding(compact ? 14 : 16)
        .glassCard(cornerRadius: compact ? 16 : 14, isSimplified: true)
    }
}

struct AlbumTracksSummary: View {
    let tracks: [Track]
    var compact: Bool = false

    var body: some View {
        HStack(spacing: compact ? 8 : 10) {
            SummaryChip(title: "曲目数", value: "\(tracks.count)", compact: compact)
            SummaryChip(title: "总时长", value: formatDuration(totalDuration), compact: compact)
            Spacer()
        }
    }

    private var totalDuration: Int64 {
        tracks.compactMap(\.duration).reduce(0, +)
    }

    private func formatDuration(_ seconds: Int64) -> String {
        guard seconds > 0 else { return "—" }
        let total = Int(seconds)
        let hours = total / 3600
        let minutes = (total % 3600) / 60
        let secs = total % 60
        if hours > 0 {
            return String(format: "%d:%02d:%02d", hours, minutes, secs)
        }
        return String(format: "%02d:%02d", minutes, secs)
    }
}

struct SummaryChip: View {
    let title: String
    let value: String
    var compact: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(compact ? .caption : .caption)
        }
        .padding(.horizontal, compact ? 8 : 10)
        .padding(.vertical, compact ? 5 : 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 10))
    }
}

struct AlbumCurationSection: View {
    let albumID: Int64
    let detail: AlbumDetail
    let candidates: [ReleaseCandidate]
    @Binding var isExpanded: Bool
    let isCompact: Bool
    let onSearch: () -> Void
    let onConfirm: (ReleaseCandidate) -> Void

    var body: some View {
        DetailSectionCard(title: "候选版本", compact: isCompact) {
            VStack(alignment: .leading, spacing: isCompact ? 10 : 12) {
                HStack(alignment: .top, spacing: 12) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("核对 MusicBrainz 候选并标记精选")
                            .font(isCompact ? .subheadline.weight(.semibold) : .body.weight(.semibold))
                        Text("不会控制播放器，仅用于专辑版本整理。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button(isExpanded ? "收起" : "展开") {
                        withAnimation(.easeInOut(duration: 0.15)) {
                            isExpanded.toggle()
                        }
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                }

                if let link = detail.releaseMB {
                    HStack(spacing: 12) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("当前精选")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            Text(link.mbid)
                                .font(.caption)
                                .fontDesign(.monospaced)
                                .lineLimit(1)
                        }
                        Spacer()
                        if link.confirmed {
                            Text("已确认")
                                .font(.caption)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.green.opacity(0.14), in: RoundedRectangle(cornerRadius: 8))
                        }
                    }
                    .padding(12)
                    .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 12))
                } else {
                    Text("尚未选择精选版本")
                        .foregroundStyle(.secondary)
                }

                if isExpanded {
                    HStack(spacing: 10) {
                        Button("搜索候选") { onSearch() }
                            .buttonStyle(.plain)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 10))
                        Text("共 \(candidates.count) 条")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    if candidates.isEmpty {
                        Text("暂无候选。可点击“搜索候选”从 MusicBrainz 补全。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.top, 4)
                    } else {
                        VStack(spacing: 10) {
                            ForEach(candidates.prefix(12)) { candidate in
                                CandidateRow(
                                    candidate: candidate,
                                    isSelected: candidate.id == detail.releaseMB?.releaseMBID,
                                    onConfirm: { onConfirm(candidate) }
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

struct CandidateRow: View {
    let candidate: ReleaseCandidate
    let isSelected: Bool
    let onConfirm: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(candidate.name ?? "未命名版本")
                    .font(.body)
                    .lineLimit(1)
                Text(candidate.mbid)
                    .font(.caption2)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer()
            if isSelected {
                Text("已选")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                Button("确认精选", action: onConfirm)
                    .buttonStyle(.plain)
                    .font(.caption)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(Color.accentColor.opacity(0.14), in: RoundedRectangle(cornerRadius: 10))
            }
        }
        .padding(12)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Color.white.opacity(isSelected ? 0.18 : 0.08), lineWidth: 1)
        )
    }
}

private struct AlbumHeroSection: View {
    let detail: AlbumDetail
    let layout: AlbumHeroLayout

    var body: some View {
        Group {
            if layout == .phone {
                VStack(alignment: .leading, spacing: 16) {
                    artwork
                        .frame(maxWidth: .infinity)

                    VStack(alignment: .leading, spacing: 10) {
                        titleBlock
                        metaFlow
                    }
                }
            } else {
                HStack(spacing: 24) {
                    artwork
                    VStack(alignment: .leading, spacing: 10) {
                        titleBlock
                        metaFlow
                    }
                    Spacer()
                }
            }
        }
        .padding(layout == .phone ? 16 : 18)
        .glassCard(cornerRadius: layout == .phone ? 18 : 16, isSimplified: true)
    }

    private var artwork: some View {
        RoundedRectangle(cornerRadius: 18)
            .fill(
                LinearGradient(
                    colors: [
                        Color.accentColor.opacity(0.55),
                        Color.accentColor.opacity(0.18)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .frame(width: layout.artworkSize, height: layout.artworkSize)
            .overlay(
                RoundedRectangle(cornerRadius: 18)
                    .stroke(Color.white.opacity(0.22), lineWidth: 1)
            )
            .overlay(
                Image(systemName: "music.note")
                    .font(.system(size: layout == .phone ? 34 : 36, weight: .semibold))
                    .foregroundStyle(.secondary)
            )
    }

    private var titleBlock: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(detail.name)
                .font(layout == .phone ? .title3.weight(.semibold) : .title2.weight(.semibold))
            Text(detail.artist)
                .font(layout == .phone ? .headline : .body)
                .foregroundStyle(.secondary)
        }
    }

    private var metaFlow: some View {
        FlexibleChipWrap(spacing: 10, lineSpacing: 10) {
            if let release = detail.releaseDate, !release.isEmpty {
                AlbumMetaChip(title: "发行日期", value: release)
            }
            if let genre = detail.genre, !genre.isEmpty {
                AlbumMetaChip(title: "流派", value: genre)
            }
            if let discs = detail.totalDiscs {
                AlbumMetaChip(title: "碟数", value: "\(discs)")
            }
        }
    }
}

private struct FlexibleChipWrap<Content: View>: View {
    let spacing: CGFloat
    let lineSpacing: CGFloat
    let content: Content

    init(spacing: CGFloat, lineSpacing: CGFloat, @ViewBuilder content: () -> Content) {
        self.spacing = spacing
        self.lineSpacing = lineSpacing
        self.content = content()
    }

    var body: some View {
        ViewThatFits(in: .vertical) {
            HStack(spacing: spacing) {
                content
                Spacer(minLength: 0)
            }
            VStack(alignment: .leading, spacing: lineSpacing) {
                content
            }
        }
    }
}

struct AlbumMetaChip: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption)
                .lineLimit(1)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct AlbumTrackRow: View {
    let track: Track
    var compact: Bool = false

    var body: some View {
        HStack(spacing: compact ? 10 : 12) {
            Text(trackNumber)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: compact ? 24 : 28, alignment: .leading)

            VStack(alignment: .leading, spacing: 2) {
                Text(track.track)
                    .font(compact ? .subheadline : .body)
                    .lineLimit(1)
                Text(track.artist)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            Spacer()

            Text(formatDuration(track.duration))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(compact ? 10 : 12)
        .background(
            RoundedRectangle(cornerRadius: 10)
                .fill(SonicTheme.card)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(SonicTheme.glassBorder.opacity(0.7), lineWidth: 1)
        )
    }

    private var trackNumber: String {
        if let trackNumber = track.trackNumber {
            return "\(trackNumber)"
        }
        return "—"
    }

    private func formatDuration(_ duration: Int64?) -> String {
        guard let duration else { return "--:--" }
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}
