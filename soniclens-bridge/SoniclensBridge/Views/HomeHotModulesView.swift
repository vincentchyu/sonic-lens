import Foundation
import SwiftUI

struct HotModuleSectionHeader: View {
    let title: String
    let subtitle: String
    var metricTag: String? = nil
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil
    var accentKey: HomeHotAccentKey? = nil

    var body: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .center, spacing: 12) {
                titleBlock
                Spacer(minLength: 12)
                trailingContent
            }

            VStack(alignment: .leading, spacing: 10) {
                titleBlock
                trailingContent
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var titleBlock: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                if let accentKey {
                    Capsule()
                        .fill(accentKey.gradient)
                        .frame(width: 16, height: 6)
                }
                Text(title)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(SonicTheme.textPrimary)
            }
            Text(subtitle)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(SonicTheme.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    @ViewBuilder
    private var trailingContent: some View {
        HStack(spacing: 8) {
            if let metricTag {
                Text(metricTag)
                    .font(.system(size: 11, weight: .semibold, design: .rounded))
                    .foregroundStyle(SonicTheme.primary)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .lineLimit(1)
                    .frame(minHeight: 28)
                    .fixedSize(horizontal: true, vertical: false)
                    .background(
                        Capsule()
                            .fill(SonicTheme.dynamicColor(light: .sonicWhite(1, alpha: 0.58), dark: .sonicWhite(0.12, alpha: 0.72)))
                    )
                    .overlay(
                        Capsule()
                            .stroke(SonicTheme.dynamicColor(light: .sonicWhite(0, alpha: 0.05), dark: .sonicWhite(1, alpha: 0.08)), lineWidth: 1)
                    )
            }

            if let actionTitle, let onAction {
                Button(action: onAction) {
                    HStack(spacing: 6) {
                        Image(systemName: "arrow.right")
                            .font(.system(size: 11, weight: .semibold))
                        Text(actionTitle)
                            .lineLimit(1)
                    }
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(SonicTheme.primary)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 6)
                        .frame(minHeight: 28)
                        .fixedSize(horizontal: true, vertical: false)
                        .background(
                            Capsule()
                                .fill(SonicTheme.primary.opacity(0.10))
                        )
                }
                .buttonStyle(.plain)
            }
        }
        .frame(maxWidth: .infinity, alignment: .trailing)
        .fixedSize(horizontal: true, vertical: false)
    }
}

struct GenreCapsuleMap: View {
    enum Style {
        case cloud
        case rail
    }

    let items: [HomeHotGenrePresentationItem]
    let selectedGenreID: String?
    let style: Style
    let onSelect: (HomeHotGenrePresentationItem) -> Void

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "热门流派",
                    subtitle: "把最近偏好的口味轮廓压成更直观的热度胶囊。"
                )

                content
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    @ViewBuilder
    private var content: some View {
        if items.isEmpty {
            HotModuleEmptyState(
                title: "还没有流派热度",
                message: "等更多播放记录同步进来，这里会逐渐长出口味轮廓。"
            )
        } else {
            switch style {
            case .cloud:
                cloudContent
            case .rail:
                railContent
            }
        }
    }

    private var cloudContent: some View {
        VStack(alignment: .leading, spacing: 10) {
            if let hero = items.first {
                genreCapsule(hero, isHero: true)
            }

            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                ForEach(Array(items.dropFirst())) { item in
                    genreCapsule(item, isHero: false)
                }
            }

            Spacer(minLength: 0)
        }
    }

    private var railContent: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 10) {
                ForEach(items) { item in
                    genreCapsule(item, isHero: item.rank == 1)
                        .frame(width: item.rank == 1 ? 188 : 150)
                }
            }
        }
    }

    private func genreCapsule(_ item: HomeHotGenrePresentationItem, isHero: Bool) -> some View {
        Button {
            onSelect(item)
        } label: {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .center) {
                    RankBadge(rank: item.rank, accentKey: item.accentKey, style: .light)
                    Spacer(minLength: 8)
                    Text(compactCount(item.count))
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(.white.opacity(0.86))
                }

                Text(item.title)
                    .font(.system(size: isHero ? 18 : 15, weight: .bold, design: .rounded))
                    .foregroundStyle(.white)
                    .lineLimit(2)

                VStack(alignment: .leading, spacing: 6) {
                    Capsule()
                        .fill(Color.white.opacity(0.20))
                        .frame(height: 6)
                        .overlay(alignment: .leading) {
                            Capsule()
                                .fill(Color.white.opacity(0.88))
                                .frame(maxWidth: CGFloat(max(24.0, 120.0 * item.relativeWeight)), maxHeight: 6)
                        }

                    Text("总热度 \(compactCount(item.secondaryCount))")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(.white.opacity(0.72))
                }
            }
            .frame(maxWidth: .infinity, minHeight: isHero ? 116 : 96, alignment: .leading)
            .padding(16)
            .background(item.accentKey.gradient.opacity(selectedGenreID == item.id ? 0.98 : 0.88))
            .overlay(
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .stroke(Color.white.opacity(selectedGenreID == item.id ? 0.34 : 0.12), lineWidth: 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
            .scaleEffect(selectedGenreID == item.id ? 1.01 : 1)
        }
        .buttonStyle(.plain)
    }
}

struct ListeningProfileCard: View {
    enum LayoutStyle {
        case split
        case stacked
        case sidebarSummary
    }

    let summaryText: String
    let footnoteText: String
    let genres: [HomeHotGenrePresentationItem]
    let sources: [HomeHotSourcePresentationItem]
    let accentKey: HomeHotAccentKey?
    let layoutStyle: LayoutStyle
    var onSelectGenre: ((HomeHotGenrePresentationItem) -> Void)? = nil
    var onOpenDetailMode: ((ListeningProfileDetailMode) -> Void)? = nil

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            if layoutStyle == .sidebarSummary {
                ListeningProfileSummarySidebar(
                    summaryText: summaryText,
                    footnoteText: footnoteText,
                    genres: genres,
                    sources: sources,
                    accentKey: accentKey,
                    onSelectGenre: onSelectGenre,
                    onOpenDetailMode: onOpenDetailMode
                )
            } else {
                VStack(alignment: .leading, spacing: 14) {
                    HotModuleSectionHeader(
                        title: "聆听画像",
                        subtitle: summaryText,
                        accentKey: accentKey
                    )

                    profileContent

                    Text(footnoteText)
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
        }
    }

    @ViewBuilder
    private var profileContent: some View {
        switch layoutStyle {
        case .split:
            HStack(alignment: .top, spacing: 12) {
                ListeningProfileGenrePanel(items: Array(genres.prefix(3)), accentKey: accentKey, onSelectGenre: onSelectGenre, onOpenDetail: { onOpenDetailMode?(.genre) })
                ListeningProfileSourcePanel(items: Array(sources.prefix(3)), accentKey: accentKey, onOpenDetail: { onOpenDetailMode?(.source) })
            }
        case .stacked:
            VStack(spacing: 12) {
                ListeningProfileGenrePanel(items: Array(genres.prefix(3)), accentKey: accentKey, onSelectGenre: onSelectGenre, onOpenDetail: { onOpenDetailMode?(.genre) })
                ListeningProfileSourcePanel(items: Array(sources.prefix(3)), accentKey: accentKey, onOpenDetail: { onOpenDetailMode?(.source) })
            }
        case .sidebarSummary:
            EmptyView()
        }
    }
}

struct ListeningProfileRingPairCard: View {
    let summaryText: String
    let genres: [HomeHotGenrePresentationItem]
    let sources: [HomeHotSourcePresentationItem]
    let accentKey: HomeHotAccentKey?
    let onSelect: (ListeningProfileDetailMode) -> Void

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 16) {
            VStack(alignment: .leading, spacing: 12) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Capsule()
                        .fill(
                            LinearGradient(
                                colors: [
                                    (accentKey ?? .tide).solidColor.opacity(0.92),
                                    (accentKey ?? .tide).solidColor.opacity(0.68)
                                ],
                                startPoint: .leading,
                                endPoint: .trailing
                            )
                        )
                        .frame(width: 16, height: 6)

                    Text("聆听画像")
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)

                    Spacer(minLength: 8)
                }

                HStack(spacing: 12) {
                    ringButton(
                        title: "口味偏好",
                        detail: summaryText,
                        slices: genreSlices,
                        centerTitle: genres.first?.title ?? "口味",
                        centerSubtitle: genres.first.map { "\(percentLabel(share(for: $0, in: genres))) 偏好占比" } ?? "等待更多数据",
                        accentKey: genres.first?.accentKey ?? accentKey,
                        mode: .genre
                    )

                    ringButton(
                        title: "播放渠道",
                        detail: summaryText,
                        slices: sourceSlices,
                        centerTitle: sources.first?.title ?? "渠道",
                        centerSubtitle: sources.first.map { "\(percentLabel($0.share)) 当前主力" } ?? "等待更多数据",
                        accentKey: sources.first?.accentKey ?? accentKey,
                        mode: .source
                    )
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func ringButton(
        title: String,
        detail: String,
        slices: [ProfileDonutSlice],
        centerTitle: String,
        centerSubtitle: String,
        accentKey: HomeHotAccentKey?,
        mode: ListeningProfileDetailMode
    ) -> some View {
        Button {
            onSelect(mode)
        } label: {
            ProfileDonutChart(
                slices: slices,
                centerBadge: slices.first.map { (rank: $0.rank, accentKey: $0.accentKey) },
                centerTitle: centerTitle,
                centerSubtitle: centerSubtitle,
                chartSize: 104,
                lineWidth: 12,
                showsCenterContent: true
            )
            .frame(maxWidth: .infinity)
            .frame(height: 118)
            .background(
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .fill((accentKey ?? .tide).tintedSurface(opacity: 0.05))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .stroke((accentKey ?? .tide).tintedSurface(opacity: 0.10), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(title)
        .accessibilityHint(detail)
    }

    private var genreSlices: [ProfileDonutSlice] {
        let total = max(genres.reduce(0) { $0 + $1.count }, 1)
        return genres.map { item in
            ProfileDonutSlice(
                id: item.id,
                title: item.title,
                count: item.count,
                share: Double(item.count) / Double(total),
                rank: item.rank,
                accentKey: item.accentKey,
                symbolName: "sparkles"
            )
        }
    }

    private var sourceSlices: [ProfileDonutSlice] {
        sources.map { item in
            ProfileDonutSlice(
                id: item.id,
                title: item.title,
                count: item.count,
                share: item.share,
                rank: item.rank,
                accentKey: item.accentKey,
                symbolName: item.symbolName
            )
        }
    }

    private func share(for item: HomeHotGenrePresentationItem, in items: [HomeHotGenrePresentationItem]) -> Double {
        let total = max(items.reduce(0) { $0 + $1.count }, 1)
        return Double(item.count) / Double(total)
    }
}

enum ListeningProfileDetailMode: String, Identifiable {
    case genre
    case source

    var id: String { rawValue }

    var title: String {
        switch self {
        case .genre: return "口味偏好"
        case .source: return "播放渠道"
        }
    }

    var subtitle: String {
        switch self {
        case .genre: return "查看最近积累的口味偏好完整数据。"
        case .source: return "查看最近积累的播放渠道完整数据。"
        }
    }
}

struct ListeningProfileDetailSheet: View {
    @Environment(\.dismiss) private var dismiss

    let mode: ListeningProfileDetailMode
    let summaryText: String
    let footnoteText: String
    let genres: [HomeHotGenrePresentationItem]
    let sources: [HomeHotSourcePresentationItem]
    let accentKey: HomeHotAccentKey?
    var onSelectGenre: ((HomeHotGenrePresentationItem) -> Void)? = nil

    var body: some View {
        ZStack {
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [SonicTheme.background, SonicTheme.background],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: [
                    AmbientOrb(
                        color: (genres.first?.accentKey ?? accentKey ?? .tide).solidColor.opacity(0.32),
                        size: 380,
                        blur: 90,
                        opacity: 0.65,
                        offsetFrom: CGSize(width: -120, height: -160),
                        offsetTo: CGSize(width: -40, height: -80),
                        duration: 16
                    ),
                    AmbientOrb(
                        color: (genres.first?.accentKey ?? accentKey ?? .tide).solidColor.opacity(0.18),
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
                topHeaderControlBar

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(mode.title)
                                .font(.system(size: 24, weight: .bold, design: .rounded))
                                .foregroundStyle(SonicTheme.textPrimary)
                            Text(summaryText)
                                .font(.system(size: 13, weight: .medium))
                                .foregroundStyle(SonicTheme.textSecondary)
                                .fixedSize(horizontal: false, vertical: true)
                            Text(footnoteText)
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(SonicTheme.textSecondary)
                                .fixedSize(horizontal: false, vertical: true)
                        }

                        switch mode {
                        case .genre:
                            ListeningProfileDetailSection(
                                title: "口味偏好完整数据",
                                summary: "点按列表项可继续查看对应流派及其精选专辑。",
                                slices: genreSlices,
                                accentKey: genres.first?.accentKey ?? accentKey,
                                trailingText: { compactCount($0.count) },
                                onItemTap: { slice in
                                    if let target = genres.first(where: { $0.id == slice.id || $0.title == slice.title }) {
                                        dismiss()
                                        onSelectGenre?(target)
                                    }
                                }
                            )
                        case .source:
                            ListeningProfileDetailSection(
                                title: "播放渠道完整数据",
                                summary: "查看所有关联播放渠道的累计量度分布。",
                                slices: sourceSlices,
                                accentKey: sources.first?.accentKey ?? accentKey,
                                trailingText: { "\(percentLabel($0.share)) · \(compactCount($0.count))" }
                            )
                        }
                    }
                    .padding(.horizontal, 18)
                    .padding(.bottom, 24)
                }
            }
        }
        .frame(
            minWidth: 480, idealWidth: 540, maxWidth: 580,
            minHeight: 460, idealHeight: 540, maxHeight: 620
        )
    }

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
                Image(systemName: mode == .genre ? "sparkles" : "waveform.path")
                    .font(.system(size: 12, weight: .semibold))
                Text(mode.title)
                    .font(.system(size: 12, weight: .semibold))
            }
            .foregroundStyle((genres.first?.accentKey ?? accentKey ?? .tide).solidColor)
            .padding(.horizontal, 12)
            .padding(.vertical, 5)
            .background(
                Capsule()
                    .fill((genres.first?.accentKey ?? accentKey ?? .tide).solidColor.opacity(0.12))
            )
        }
        .padding(.horizontal, 18)
        .padding(.top, 14)
        .padding(.bottom, 10)
    }

    private var genreSlices: [ProfileDonutSlice] {
        let total = max(genres.reduce(0) { $0 + $1.count }, 1)
        return genres.map { item in
            ProfileDonutSlice(
                id: item.id,
                title: item.title,
                count: item.count,
                share: Double(item.count) / Double(total),
                rank: item.rank,
                accentKey: item.accentKey,
                symbolName: "sparkles"
            )
        }
    }

    private var sourceSlices: [ProfileDonutSlice] {
        sources.map { item in
            ProfileDonutSlice(
                id: item.id,
                title: item.title,
                count: item.count,
                share: item.share,
                rank: item.rank,
                accentKey: item.accentKey,
                symbolName: item.symbolName
            )
        }
    }
}

private struct ListeningProfileDetailSection: View {
    let title: String
    let summary: String
    let slices: [ProfileDonutSlice]
    let accentKey: HomeHotAccentKey?
    let trailingText: (ProfileDonutSlice) -> String
    var onItemTap: ((ProfileDonutSlice) -> Void)? = nil

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 16) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: title,
                    subtitle: summary,
                    accentKey: accentKey
                )

                ProfileDonutChart(
                    slices: slices,
                    centerBadge: slices.first.map { (rank: $0.rank, accentKey: $0.accentKey) },
                    centerTitle: slices.first?.title ?? title,
                    centerSubtitle: slices.first.map { "\(percentLabel($0.share))" } ?? "等待更多数据",
                    chartSize: 160,
                    lineWidth: 14,
                    showsCenterContent: true
                )
                .frame(maxWidth: .infinity)

                VStack(spacing: 8) {
                    ForEach(slices) { slice in
                        ProfileLegendRow(
                            slice: slice,
                            trailingText: trailingText(slice),
                            accentKey: accentKey,
                            onTap: onItemTap != nil ? { onItemTap?(slice) } : nil
                        )
                    }
                }
            }
        }
    }
}

struct ListeningProfileSummarySidebar: View {
    let summaryText: String
    let footnoteText: String
    let genres: [HomeHotGenrePresentationItem]
    let sources: [HomeHotSourcePresentationItem]
    let accentKey: HomeHotAccentKey?
    var onSelectGenre: ((HomeHotGenrePresentationItem) -> Void)? = nil
    var onOpenDetailMode: ((ListeningProfileDetailMode) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HotModuleSectionHeader(
                title: "聆听画像",
                subtitle: summaryText,
                accentKey: accentKey
            )

            VStack(spacing: 10) {
                ListeningProfileGenrePanel(
                    items: Array(genres.prefix(4)),
                    accentKey: accentKey,
                    style: .compactSummary,
                    onSelectGenre: onSelectGenre,
                    onOpenDetail: { onOpenDetailMode?(.genre) }
                )
                ListeningProfileSourcePanel(
                    items: Array(sources.prefix(3)),
                    accentKey: accentKey,
                    style: .compactSummary,
                    onOpenDetail: { onOpenDetailMode?(.source) }
                )
            }.padding(.top, 22)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }
}

struct GenreDigestCard: View {
    let summaryText: String
    let items: [HomeHotGenrePresentationItem]
    var accentKey: HomeHotAccentKey? = nil
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "口味流派",
                    subtitle: summaryText,
                    actionTitle: actionTitle,
                    onAction: onAction,
                    accentKey: accentKey
                )

                ListeningProfileGenrePanel(items: items, accentKey: accentKey)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

struct SourceMixCard: View {
    let summaryText: String
    let footnoteText: String
    let items: [HomeHotSourcePresentationItem]
    var accentKey: HomeHotAccentKey? = nil

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "播放渠道",
                    subtitle: summaryText,
                    accentKey: accentKey
                )

                ListeningProfileSourcePanel(items: items, accentKey: accentKey)

                Text(footnoteText)
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(SonicTheme.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct ListeningProfileGenrePanel: View {
    enum Style {
        case full
        case compactSummary
    }

    let items: [HomeHotGenrePresentationItem]
    let accentKey: HomeHotAccentKey?
    var style: Style = .full
    var onSelectGenre: ((HomeHotGenrePresentationItem) -> Void)? = nil
    var onOpenDetail: (() -> Void)? = nil

    var body: some View {
        ListeningProfilePanelShell(
            title: "口味流派",
            systemImage: "sparkles",
            accentKey: accentKey ?? items.first?.accentKey,
            onOpenDetail: onOpenDetail
        ) {
            if items.isEmpty {
                HotModuleEmptyState(
                    title: "还没有流派画像",
                    message: "等更多播放记录同步进来，这里会出现更清晰的口味轮廓。"
                )
            } else {
                switch style {
                case .full:
                    HStack(alignment: .top, spacing: 14) {
                        ProfileDonutChart(
                            slices: genreSlices,
                            centerBadge: items.first.map { (rank: $0.rank, accentKey: $0.accentKey) },
                            centerTitle: items.first?.title ?? "口味流派",
                            centerSubtitle: items.first.map { "\(percentLabel(share(for: $0, in: items))) 偏好占比" } ?? "等待更多数据"
                        )
                        .frame(width: 130)
                        .onTapGesture {
                            if let first = items.first {
                                onSelectGenre?(first)
                            }
                        }

                        VStack(alignment: .leading, spacing: 8) {
                            ForEach(genreSlices) { slice in
                                ProfileLegendRow(
                                    slice: slice,
                                    trailingText: compactCount(slice.count),
                                    accentKey: accentKey,
                                    onTap: {
                                        if let target = items.first(where: { $0.id == slice.id || $0.title == slice.title }) {
                                            onSelectGenre?(target)
                                        }
                                    }
                                )
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                case .compactSummary:
                    HStack(alignment: .center, spacing: 12) {
                        ProfileDonutChart(
                            slices: genreSlices,
                            centerBadge: items.first.map { (rank: $0.rank, accentKey: $0.accentKey) },
                            centerTitle: items.first?.title ?? "口味流派",
                            centerSubtitle: items.first.map { "\(percentLabel(share(for: $0, in: items))) 偏好占比" } ?? "等待更多数据",
                            chartSize: 96,
                            lineWidth: 12
                        )
                        .frame(width: 96, height: 96)
                        .onTapGesture {
                            if let first = items.first {
                                onSelectGenre?(first)
                            }
                        }

                        VStack(alignment: .leading, spacing: 7) {
                            ForEach(Array(genreSlices.prefix(3))) { slice in
                                CompactProfileLegendRow(
                                    slice: slice,
                                    trailingText: compactCount(slice.count),
                                    onTap: {
                                        if let target = items.first(where: { $0.id == slice.id || $0.title == slice.title }) {
                                            onSelectGenre?(target)
                                        }
                                    }
                                )
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private var genreSlices: [ProfileDonutSlice] {
        let total = max(items.reduce(0) { $0 + $1.count }, 1)
        return items.map { item in
            ProfileDonutSlice(
                id: item.id,
                title: item.title,
                count: item.count,
                share: Double(item.count) / Double(total),
                rank: item.rank,
                accentKey: item.accentKey,
                symbolName: "sparkles"
            )
        }
    }

    private func share(for item: HomeHotGenrePresentationItem, in items: [HomeHotGenrePresentationItem]) -> Double {
        let total = max(items.reduce(0) { $0 + $1.count }, 1)
        return Double(item.count) / Double(total)
    }
}

private struct ListeningProfileSourcePanel: View {
    enum Style {
        case full
        case compactSummary
    }

    let items: [HomeHotSourcePresentationItem]
    let accentKey: HomeHotAccentKey?
    var style: Style = .full
    var onOpenDetail: (() -> Void)? = nil

    var body: some View {
        ListeningProfilePanelShell(
            title: "播放渠道",
            systemImage: "waveform.path",
            accentKey: accentKey ?? items.first?.accentKey,
            onOpenDetail: onOpenDetail
        ) {
            if items.isEmpty {
                HotModuleEmptyState(
                    title: "还没有来源分布",
                    message: "等更多播放记录同步进来，这里会汇总主要播放渠道。"
                )
            } else {
                switch style {
                case .full:
                    HStack(alignment: .top, spacing: 14) {
                        ProfileDonutChart(
                            slices: sourceSlices,
                            centerBadge: items.first.map { (rank: $0.rank, accentKey: $0.accentKey) },
                            centerTitle: items.first?.title ?? "播放渠道",
                            centerSubtitle: items.first.map { "\(percentLabel($0.share)) 当前主力" } ?? "等待更多数据"
                        )
                        .frame(width: 130)

                        VStack(alignment: .leading, spacing: 8) {
                            ForEach(sourceSlices) { slice in
                                ProfileLegendRow(
                                    slice: slice,
                                    trailingText: "\(percentLabel(slice.share)) · \(compactCount(slice.count))",
                                    accentKey: accentKey
                                )
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                case .compactSummary:
                    HStack(alignment: .center, spacing: 12) {
                        ProfileDonutChart(
                            slices: sourceSlices,
                            centerBadge: items.first.map { (rank: $0.rank, accentKey: $0.accentKey) },
                            centerTitle: items.first?.title ?? "播放渠道",
                            centerSubtitle: items.first.map { "\(percentLabel($0.share)) 当前主力" } ?? "等待更多数据",
                            chartSize: 96,
                            lineWidth: 12
                        )
                        .frame(width: 96, height: 96)

                        VStack(alignment: .leading, spacing: 7) {
                            ForEach(Array(sourceSlices.prefix(3))) { slice in
                                CompactProfileLegendRow(
                                    slice: slice,
                                    trailingText: "\(percentLabel(slice.share)) · \(compactCount(slice.count))"
                                )
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private var sourceSlices: [ProfileDonutSlice] {
        items.map { item in
            ProfileDonutSlice(
                id: item.id,
                title: item.title,
                count: item.count,
                share: item.share,
                rank: item.rank,
                accentKey: item.accentKey,
                symbolName: item.symbolName
            )
        }
    }
}

private struct ListeningProfilePanelShell<Content: View>: View {
    let title: String
    let systemImage: String
    let accentKey: HomeHotAccentKey?
    var onOpenDetail: (() -> Void)? = nil
    let content: Content

    init(
        title: String,
        systemImage: String,
        accentKey: HomeHotAccentKey?,
        onOpenDetail: (() -> Void)? = nil,
        @ViewBuilder content: () -> Content
    ) {
        self.title = title
        self.systemImage = systemImage
        self.accentKey = accentKey
        self.onOpenDetail = onOpenDetail
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .center) {
                Label {
                    Text(title)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                } icon: {
                    Image(systemName: systemImage)
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle((accentKey ?? .tide).solidColor)
                }

                Spacer(minLength: 8)

                if let onOpenDetail {
                    Button(action: onOpenDetail) {
                        HStack(spacing: 4) {
                            Text("查看全部")
                            Image(systemName: "chevron.right")
                                .font(.system(size: 9, weight: .bold))
                        }
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(SonicTheme.primary)
                    }
                    .buttonStyle(.plain)
                }
            }

            content
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
        .padding(14)
        .background(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .fill(SonicTheme.card)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
        )
    }
}

struct ArtistLadderCard: View {
    static let homeMinimumWidth: CGFloat = 300

    let items: [HomeHotArtistPresentationItem]
    let artworkBaseURL: URL?
    var collectionCount: Int64? = nil
    var accentKey: HomeHotAccentKey? = nil
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "热门艺术家",
                    subtitle: "汇总显示最有存在感的创作者。",
                    metricTag: collectionCount.map { "总：\(compactCount($0))" },
                    actionTitle: actionTitle,
                    onAction: onAction,
                    accentKey: accentKey
                )

                content
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    @ViewBuilder
    private var content: some View {
        if items.isEmpty {
            HotModuleEmptyState(
                title: "还没有艺术家热度",
                message: "播放记录再积累一点，这里会把最有存在感的创作者推上来。"
            )
        } else {
            VStack(spacing: 10) {
                ForEach(items) { item in
                    artistRow(item)
                }

                Spacer(minLength: 0)
            }
        }
    }

    private func artistRow(_ item: HomeHotArtistPresentationItem) -> some View {
        let emphasized = item.rank == 1
        return HStack(spacing: 12) {
            ArtistAvatarView(
                title: item.title,
                artworkPath: item.artworkPath,
                artworkBaseURL: artworkBaseURL,
                accentKey: item.accentKey,
                size: emphasized ? 44 : 38,
                monogramSize: emphasized ? 15 : 13
            )
            .frame(width: emphasized ? 44 : 38, height: emphasized ? 44 : 38)

            VStack(alignment: .leading, spacing: 5) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    RankBadge(rank: item.rank, accentKey: item.accentKey)
                    Text(item.title)
                        .font(.system(size: emphasized ? 17 : 14, weight: emphasized ? .bold : .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                        .lineLimit(1)
                }

                Capsule()
                    .fill(SonicTheme.dynamicColor(light: .sonicWhite(0, alpha: 0.08), dark: .sonicWhite(1, alpha: 0.08)))
                    .frame(height: 7)
                    .overlay(alignment: .leading) {
                        Capsule()
                            .fill(item.accentKey.gradient)
                            .frame(maxWidth: CGFloat(max(28.0, 164.0 * item.relativeWeight)), maxHeight: 7)
                    }
            }

            Spacer(minLength: 10)

            VStack(alignment: .trailing, spacing: 3) {
                Text(compactCount(item.count))
                    .font(.system(size: emphasized ? 17 : 14, weight: .bold, design: .rounded))
                    .foregroundStyle(SonicTheme.textPrimary)
                if let secondaryCount = item.secondaryCount, secondaryCount != item.count {
                    Text("\(compactCount(secondaryCount)) 首")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, emphasized ? 12 : 10)
        .background(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .fill(item.accentKey.tintedSurface(opacity: emphasized ? 0.12 : 0.06))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(item.accentKey.tintedSurface(opacity: emphasized ? 0.18 : 0.10), lineWidth: 1)
        )
    }
}

private struct ArtistAvatarView: View {
    let title: String
    let artworkPath: String?
    let artworkBaseURL: URL?
    let accentKey: HomeHotAccentKey
    let size: CGFloat
    let monogramSize: CGFloat

    private var resolvedArtworkURL: URL? {
        guard let resolved = ArtworkURLResolver.resolveArtworkPath(artworkPath, artworkBaseURL: artworkBaseURL) else {
            return nil
        }
        return URL(string: resolved)
    }

    var body: some View {
        Group {
            if let resolvedArtworkURL {
                AsyncImage(url: resolvedArtworkURL) { phase in
                    if let image = phase.image {
                        image
                            .resizable()
                            .scaledToFill()
                    } else {
                        fallbackAvatar
                    }
                }
            } else {
                fallbackAvatar
            }
        }
        .frame(width: size, height: size)
        .clipShape(Circle())
        .overlay(
            Circle()
                .stroke(accentKey.tintedSurface(opacity: 0.14), lineWidth: 1)
        )
    }

    private var fallbackAvatar: some View {
        ZStack {
            Circle()
                .fill(accentKey.gradient)
            Text(artistMonogram(for: title))
                .font(.system(size: monogramSize, weight: .bold, design: .rounded))
                .foregroundStyle(.white.opacity(0.92))
        }
    }
}

struct AlbumShelfCard: View {
    enum Style {
        case featureGrid
        case rail
        case compactGrid
    }

    static let homeMinimumWidth: CGFloat = 360

    let items: [HomeHotAlbumPresentationItem]
    let artworkBaseURL: URL?
    let style: Style
    var collectionCount: Int64? = nil
    var accentKey: HomeHotAccentKey? = nil
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil
    let onAlbumTap: (Int64) -> Void

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "热门专辑",
                    subtitle: "统计最近30天的专辑",
                    metricTag: collectionCount.map { "总：\(compactCount($0))" },
                    actionTitle: actionTitle,
                    onAction: onAction,
                    accentKey: accentKey
                )

                content
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    @ViewBuilder
    private var content: some View {
        if items.isEmpty {
            HotModuleEmptyState(
                title: "还没有热门专辑",
                message: "等更多专辑进入播放循环，这里会优先展示最值得点开的封面。"
            )
        } else {
            switch style {
            case .featureGrid:
                featureGrid
            case .rail:
                rail
            case .compactGrid:
                compactGrid
            }
        }
    }

    private var featureGrid: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let hero = items.first {
                albumFeatureCard(hero, artworkSize: 152, isHero: true)
            }

            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                ForEach(Array(items.dropFirst().prefix(4))) { item in
                    albumFeatureCard(item, artworkSize: 72, isHero: false)
                }
            }

            Spacer(minLength: 0)
        }
    }

    private var rail: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(alignment: .top, spacing: 12) {
                ForEach(items) { item in
                    Button {
                        guard item.id > 0 else { return }
                        onAlbumTap(item.id)
                    } label: {
                        VStack(alignment: .leading, spacing: 9) {
                            rankArtwork(
                                rank: item.rank,
                                accentKey: item.accentKey,
                                size: item.rank == 1 ? 132 : 112,
                                cornerRadius: 18
                            ) {
                                ArtworkSquareView(
                                    artworkURL: resolvedArtworkURL(for: item),
                                    fallbackTitle: item.title,
                                    size: item.rank == 1 ? 132 : 112,
                                    cornerRadius: 18,
                                    style: .vivid
                                )
                            }
                            VStack(alignment: .leading, spacing: 3) {
                                Text(item.title)
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(SonicTheme.textPrimary)
                                    .lineLimit(2)
                                Text(item.subtitle)
                                    .font(.system(size: 12, weight: .medium))
                                    .foregroundStyle(SonicTheme.textSecondary)
                                    .lineLimit(1)
                                Text(compactCount(item.count))
                                    .font(.system(size: 11, weight: .semibold))
                                    .foregroundStyle(SonicTheme.textSecondary)
                            }
                        }
                        .frame(width: item.rank == 1 ? 144 : 122, alignment: .leading)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private var compactGrid: some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
            ForEach(items) { item in
                Button {
                    guard item.id > 0 else { return }
                    onAlbumTap(item.id)
                } label: {
                    VStack(alignment: .leading, spacing: 8) {
                        rankArtwork(
                            rank: item.rank,
                            accentKey: item.accentKey,
                            size: 148,
                            cornerRadius: 20
                        ) {
                            ArtworkSquareView(
                                artworkURL: resolvedArtworkURL(for: item),
                                fallbackTitle: item.title,
                                size: 148,
                                cornerRadius: 20,
                                style: .vivid
                            )
                        }
                        Text(item.title)
                            .font(.system(size: 14, weight: item.rank == 1 ? .bold : .semibold))
                            .foregroundStyle(SonicTheme.textPrimary)
                            .lineLimit(2)
                        Text(item.subtitle)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(SonicTheme.textSecondary)
                            .lineLimit(1)
                        Text(compactCount(item.count))
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(SonicTheme.textSecondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func albumFeatureCard(_ item: HomeHotAlbumPresentationItem, artworkSize: CGFloat, isHero: Bool) -> some View {
        Button {
            guard item.id > 0 else { return }
            onAlbumTap(item.id)
        } label: {
            HStack(alignment: .center, spacing: 12) {
                rankArtwork(
                    rank: item.rank,
                    accentKey: item.accentKey,
                    size: artworkSize,
                    cornerRadius: isHero ? 24 : 16
                ) {
                    ArtworkSquareView(
                        artworkURL: resolvedArtworkURL(for: item),
                        fallbackTitle: item.title,
                        size: artworkSize,
                        cornerRadius: isHero ? 24 : 16,
                        style: .vivid
                    )
                }

                VStack(alignment: .leading, spacing: 7) {
                    Text(item.title)
                        .font(.system(size: isHero ? 20 : 15, weight: isHero ? .bold : .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                        .lineLimit(isHero ? 2 : 1)
                    Text(item.subtitle)
                        .font(.system(size: isHero ? 13 : 12, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .lineLimit(1)
                    Text("\(compactCount(item.count)) 次播放")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(SonicTheme.textSecondary)
                }

                Spacer(minLength: 0)
            }
            .padding(12)
            .background(
                RoundedRectangle(cornerRadius: isHero ? 26 : 18, style: .continuous)
                    .fill(item.accentKey.tintedSurface(opacity: isHero ? 0.12 : 0.06))
            )
            .overlay(
                RoundedRectangle(cornerRadius: isHero ? 26 : 18, style: .continuous)
                    .stroke(item.accentKey.tintedSurface(opacity: isHero ? 0.18 : 0.10), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }

    private func resolvedArtworkURL(for item: HomeHotAlbumPresentationItem) -> String? {
        ArtworkURLResolver.resolveArtworkPath(item.artworkPath, artworkBaseURL: artworkBaseURL)
    }
}

struct TrackShelfCard: View {
    enum Style {
        case featureGrid
        case rail
        case compactGrid
    }

    static let homeMinimumWidth: CGFloat = 360

    let items: [HomeHotTrackPresentationItem]
    let artworkBaseURL: URL?
    let style: Style
    var visibleItemCount: Int = 5
    var totalTracksCount: Int64? = nil
    var accentKey: HomeHotAccentKey? = nil
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil
    let onTrackTap: (TopTrack) -> Void

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "热门曲目",
                    subtitle: "统计最近30天的曲目",
                    metricTag: totalTracksCount.map { "总：\(compactCount($0))" },
                    actionTitle: actionTitle,
                    onAction: onAction,
                    accentKey: accentKey
                )

                content
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    @ViewBuilder
    private var content: some View {
        if items.isEmpty {
            HotModuleEmptyState(
                title: "还没有热门曲目",
                message: "等更多单曲进入循环回放，这里会把最常回到耳边的歌推上来。"
            )
        } else {
            switch style {
            case .featureGrid:
                featureGrid
            case .rail:
                rail
            case .compactGrid:
                compactGrid
            }
        }
    }

    @ViewBuilder
    private var featureGrid: some View {
        let displayItems = Array(items.prefix(max(visibleItemCount, 0)))

        VStack(alignment: .leading, spacing: 12) {
            if let hero = displayItems.first {
                trackFeatureCard(hero, artworkSize: 152, isHero: true)
            }

            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                ForEach(Array(displayItems.dropFirst())) { item in
                    trackFeatureCard(item, artworkSize: 72, isHero: false)
                }
            }

            Spacer(minLength: 0)
        }
    }

    private var rail: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(alignment: .top, spacing: 12) {
                ForEach(items) { item in
                    Button {
                        guard item.sourceTrack.trackID > 0 else { return }
                        onTrackTap(item.sourceTrack)
                    } label: {
                        VStack(alignment: .leading, spacing: 9) {
                            rankArtwork(
                                rank: item.rank,
                                accentKey: item.accentKey,
                                size: item.rank == 1 ? 132 : 112,
                                cornerRadius: 18
                            ) {
                                ArtworkSquareView(
                                    artworkURL: resolvedArtworkURL(for: item),
                                    fallbackTitle: item.title,
                                    size: item.rank == 1 ? 132 : 112,
                                    cornerRadius: 18,
                                    style: .vivid
                                )
                            }
                            VStack(alignment: .leading, spacing: 3) {
                                Text(item.title)
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(SonicTheme.textPrimary)
                                    .lineLimit(2)
                                Text(item.subtitle)
                                    .font(.system(size: 12, weight: .medium))
                                    .foregroundStyle(SonicTheme.textSecondary)
                                    .lineLimit(1)
                                Text(compactCount(item.count))
                                    .font(.system(size: 11, weight: .semibold))
                                    .foregroundStyle(SonicTheme.textSecondary)
                            }
                        }
                        .frame(width: item.rank == 1 ? 144 : 122, alignment: .leading)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private var compactGrid: some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
            ForEach(items) { item in
                Button {
                    guard item.sourceTrack.trackID > 0 else { return }
                    onTrackTap(item.sourceTrack)
                } label: {
                    VStack(alignment: .leading, spacing: 8) {
                        rankArtwork(
                            rank: item.rank,
                            accentKey: item.accentKey,
                            size: 148,
                            cornerRadius: 20
                        ) {
                            ArtworkSquareView(
                                artworkURL: resolvedArtworkURL(for: item),
                                fallbackTitle: item.title,
                                size: 148,
                                cornerRadius: 20,
                                style: .vivid
                            )
                        }
                        Text(item.title)
                            .font(.system(size: 14, weight: item.rank == 1 ? .bold : .semibold))
                            .foregroundStyle(SonicTheme.textPrimary)
                            .lineLimit(2)
                        Text(item.subtitle)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(SonicTheme.textSecondary)
                            .lineLimit(1)
                        Text(compactCount(item.count))
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(SonicTheme.textSecondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func trackFeatureCard(_ item: HomeHotTrackPresentationItem, artworkSize: CGFloat, isHero: Bool) -> some View {
        Button {
            guard item.sourceTrack.trackID > 0 else { return }
            onTrackTap(item.sourceTrack)
        } label: {
            HStack(alignment: .center, spacing: 12) {
                rankArtwork(
                    rank: item.rank,
                    accentKey: item.accentKey,
                    size: artworkSize,
                    cornerRadius: isHero ? 24 : 16
                ) {
                    ArtworkSquareView(
                        artworkURL: resolvedArtworkURL(for: item),
                        fallbackTitle: item.title,
                        size: artworkSize,
                        cornerRadius: isHero ? 24 : 16,
                        style: .vivid
                    )
                }

                VStack(alignment: .leading, spacing: 7) {
                    Text(item.title)
                        .font(.system(size: isHero ? 20 : 15, weight: isHero ? .bold : .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                        .lineLimit(isHero ? 2 : 1)
                    Text(item.subtitle)
                        .font(.system(size: isHero ? 13 : 12, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .lineLimit(1)
                    if isHero, let tertiaryText = item.tertiaryText, tertiaryText.isEmpty == false {
                        Text(tertiaryText)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(SonicTheme.textSecondary.opacity(0.88))
                            .lineLimit(1)
                    }
                    Text("\(compactCount(item.count)) 次播放")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(SonicTheme.textSecondary)
                }

                Spacer(minLength: 0)
            }
            .padding(12)
            .background(
                RoundedRectangle(cornerRadius: isHero ? 26 : 18, style: .continuous)
                    .fill(item.accentKey.tintedSurface(opacity: isHero ? 0.12 : 0.06))
            )
            .overlay(
                RoundedRectangle(cornerRadius: isHero ? 26 : 18, style: .continuous)
                    .stroke(item.accentKey.tintedSurface(opacity: isHero ? 0.18 : 0.10), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }

    private func resolvedArtworkURL(for item: HomeHotTrackPresentationItem) -> String? {
        ArtworkURLResolver.resolveArtworkPath(item.artworkPath, artworkBaseURL: artworkBaseURL)
    }
}

private func compactCount(_ value: Int) -> String {
    switch value {
    case 1_000_000...:
        return String(format: "%.1fm", Double(value) / 1_000_000)
    case 1_000...:
        return String(format: "%.1fk", Double(value) / 1_000)
    default:
        return "\(value)"
    }
}

private func compactCount(_ value: Int64) -> String {
    compactCount(Int(value))
}

private func percentLabel(_ value: Double) -> String {
    "\(Int((value * 100).rounded()))%"
}

private func rankLabel(for rank: Int) -> String {
    String(format: "#%02d", rank)
}

private struct ProfileDonutSlice: Identifiable {
    let id: String
    let title: String
    let count: Int
    let share: Double
    let rank: Int
    let accentKey: HomeHotAccentKey
    let symbolName: String
}

private struct ProfileLegendRow: View {
    let slice: ProfileDonutSlice
    let trailingText: String
    let accentKey: HomeHotAccentKey?
    var onTap: (() -> Void)? = nil

    var body: some View {
        Button {
            onTap?()
        } label: {
            HStack(spacing: 10) {
                Image(systemName: slice.symbolName)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(slice.accentKey.solidColor)
                    .frame(width: 28, height: 28)
                    .background(slice.accentKey.tintedSurface(opacity: 0.08), in: RoundedRectangle(cornerRadius: 9, style: .continuous))

                VStack(alignment: .leading, spacing: 4) {
                    HStack(alignment: .firstTextBaseline, spacing: 6) {
                        Text(slice.title)
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(SonicTheme.textPrimary)
                            .lineLimit(1)
                        RankBadge(rank: slice.rank, accentKey: slice.accentKey)
                    }

                    Capsule()
                        .fill(SonicTheme.progressTrack)
                        .frame(height: 5)
                        .overlay(alignment: .leading) {
                            Capsule()
                                .fill(slice.accentKey.gradient)
                                .frame(maxWidth: CGFloat(max(20.0, 104.0 * max(slice.share, 0.18))), maxHeight: 5)
                        }
                }

                Spacer(minLength: 8)

                Text(trailingText)
                    .font(.system(size: 12, weight: .bold, design: .rounded))
                    .foregroundStyle(SonicTheme.textPrimary)

                if onTap != nil {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 10, weight: .bold))
                        .foregroundStyle(SonicTheme.textSecondary.opacity(0.6))
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(slice.accentKey.tintedSurface(opacity: 0.05))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(slice.accentKey.tintedSurface(opacity: 0.08), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .disabled(onTap == nil)
    }
}

private struct CompactProfileLegendRow: View {
    let slice: ProfileDonutSlice
    let trailingText: String
    var onTap: (() -> Void)? = nil

    var body: some View {
        Button {
            onTap?()
        } label: {
            HStack(spacing: 8) {
                Image(systemName: slice.symbolName)
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(slice.accentKey.solidColor)
                    .frame(width: 22, height: 22)
                    .background(
                        slice.accentKey.tintedSurface(opacity: 0.08),
                        in: RoundedRectangle(cornerRadius: 7, style: .continuous)
                    )

                Text(slice.title)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(SonicTheme.textPrimary)
                    .lineLimit(1)

                Spacer(minLength: 4)

                Text(trailingText)
                    .font(.system(size: 11, weight: .bold, design: .rounded))
                    .foregroundStyle(SonicTheme.textSecondary)

                if onTap != nil {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(SonicTheme.textSecondary.opacity(0.5))
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .fill(slice.accentKey.tintedSurface(opacity: 0.04))
            )
        }
        .buttonStyle(.plain)
        .disabled(onTap == nil)
    }
}

struct RankBadge: View {
    enum Style {
        case subtle
        case light
        case prominent
        case inline
        case artwork
    }

    let rank: Int
    let accentKey: HomeHotAccentKey
    var style: Style = .subtle

    var body: some View {
        Text("#\(String(format: "%02d", rank))")
            .font(.system(size: style == .prominent ? 11 : (style == .light || style == .inline ? 10 : 9), weight: .bold, design: .monospaced))
            .foregroundStyle(textColor)
            .padding(.horizontal, style == .prominent ? 8 : (style == .light || style == .artwork ? 6 : 5))
            .padding(.vertical, style == .prominent ? 4 : 2)
            .background(backgroundColor, in: Capsule())
    }

    private var textColor: Color {
        switch style {
        case .subtle, .inline:
            return accentKey.solidColor
        case .light, .prominent, .artwork:
            return .white
        }
    }

    private var backgroundColor: Color {
        switch style {
        case .subtle, .inline:
            return accentKey.tintedSurface(opacity: 0.12)
        case .light:
            return Color.white.opacity(0.20)
        case .prominent, .artwork:
            return accentKey.solidColor.opacity(0.88)
        }
    }
}



private struct ProfileDonutChart: View {
    let slices: [ProfileDonutSlice]
    let centerBadge: (rank: Int, accentKey: HomeHotAccentKey)?
    let centerTitle: String
    let centerSubtitle: String
    var chartSize: CGFloat = 132
    var lineWidth: CGFloat = 14
    var showsCenterContent: Bool = true

    var body: some View {
        ZStack {
            ZStack {
                Circle()
                    .stroke(SonicTheme.progressTrack, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))

                ForEach(chartSegments) { segment in
                    Circle()
                        .trim(from: segment.start, to: segment.end)
                        .stroke(
                            segment.slice.accentKey.gradient,
                            style: StrokeStyle(lineWidth: lineWidth, lineCap: .round)
                        )
                }
            }
            .rotationEffect(.degrees(-90))

            if showsCenterContent {
                VStack(spacing: 6) {
                    if let centerBadge {
                        RankBadge(rank: centerBadge.rank, accentKey: centerBadge.accentKey, style: .inline)
                    }
                    Text(centerTitle)
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(SonicTheme.textPrimary)
                        .lineLimit(2)
                        .multilineTextAlignment(.center)
                    Text(centerSubtitle)
                        .font(.system(size: 10, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.horizontal, max(12, chartSize * 0.14))
            }
        }
        .frame(maxWidth: .infinity)
        .frame(width: chartSize, height: chartSize)
    }

    private var chartSegments: [ProfileDonutChartSegment] {
        let maxDisplaySlices = 8
        let visibleSlices = slices.filter { $0.share > 0 }
        let primarySlices = Array(visibleSlices.prefix(maxDisplaySlices))
        let total = max(visibleSlices.reduce(0.0) { $0 + $1.share }, 0.0001)
        let gap = primarySlices.count > 1 ? min(0.012, 0.06 / Double(primarySlices.count)) : 0.0
        var offset = 0.0

        var segments: [ProfileDonutChartSegment] = []
        for slice in primarySlices {
            let normalizedShare = slice.share / total
            guard normalizedShare > 0.005 else { continue }
            let segmentStart = offset + gap / 2
            let segmentEnd = min(offset + max(normalizedShare - gap / 2, 0.008), 0.999)
            offset += normalizedShare
            if segmentStart < segmentEnd {
                segments.append(ProfileDonutChartSegment(slice: slice, start: segmentStart, end: segmentEnd))
            }
        }

        if visibleSlices.count > maxDisplaySlices && offset < 0.98 {
            let remainingShare = max(1.0 - offset, 0.0)
            if remainingShare > 0.02 {
                let segmentStart = offset + gap / 2
                let segmentEnd = min(offset + max(remainingShare - gap / 2, 0.008), 0.999)
                if segmentStart < segmentEnd {
                    let otherSlice = ProfileDonutSlice(
                        id: "other_genres",
                        title: "其他",
                        count: 0,
                        share: remainingShare,
                        rank: maxDisplaySlices + 1,
                        accentKey: .slate,
                        symbolName: "ellipsis"
                    )
                    segments.append(ProfileDonutChartSegment(slice: otherSlice, start: segmentStart, end: segmentEnd))
                }
            }
        }

        return segments
    }
}

private struct ProfileDonutChartSegment: Identifiable {
    let slice: ProfileDonutSlice
    let start: Double
    let end: Double

    var id: String { slice.id }
}

@ViewBuilder
private func rankArtwork<Content: View>(
    rank: Int,
    accentKey: HomeHotAccentKey,
    size: CGFloat,
    cornerRadius: CGFloat,
    @ViewBuilder content: () -> Content
    ) -> some View {
        content()
        .frame(width: size, height: size)
        .overlay(alignment: .topLeading) {
            RankBadge(rank: rank, accentKey: accentKey, style: .artwork)
                .padding(4)
        }
        .overlay {
            RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                .stroke(accentKey.tintedSurface(opacity: rank == 1 ? 0.26 : 0.12), lineWidth: 1)
        }
}

private func artistMonogram(for value: String) -> String {
    value.trimmingCharacters(in: .whitespacesAndNewlines).first.map { String($0).uppercased() } ?? "A"
}

struct HotModuleEmptyState: View {
    let title: String
    let message: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(SonicTheme.textPrimary)
            Text(message)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(SonicTheme.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .fill(SonicTheme.dynamicColor(light: .sonicWhite(1, alpha: 0.55), dark: .sonicWhite(0.08, alpha: 0.72)))
        )
    }
}

extension HomeHotAccentKey {
    var solidColor: Color {
        switch self {
        case .tide:
            return SonicTheme.dynamicColor(
                light: .sonicRGBA(0.18, 0.44, 0.76, 1),
                dark: .sonicRGBA(0.34, 0.60, 0.92, 1)
            )
        case .amber:
            return SonicTheme.dynamicColor(
                light: .sonicRGBA(0.78, 0.54, 0.16, 1),
                dark: .sonicRGBA(0.90, 0.67, 0.26, 1)
            )
        case .mint:
            return SonicTheme.dynamicColor(
                light: .sonicRGBA(0.17, 0.58, 0.50, 1),
                dark: .sonicRGBA(0.33, 0.74, 0.64, 1)
            )
        case .coral:
            return SonicTheme.dynamicColor(
                light: .sonicRGBA(0.76, 0.36, 0.31, 1),
                dark: .sonicRGBA(0.90, 0.50, 0.43, 1)
            )
        case .indigo:
            return SonicTheme.dynamicColor(
                light: .sonicRGBA(0.33, 0.39, 0.73, 1),
                dark: .sonicRGBA(0.49, 0.56, 0.90, 1)
            )
        case .slate:
            return SonicTheme.dynamicColor(
                light: .sonicRGBA(0.38, 0.44, 0.52, 1),
                dark: .sonicRGBA(0.55, 0.61, 0.70, 1)
            )
        }
    }

    var gradient: LinearGradient {
        LinearGradient(
            colors: [solidColor.opacity(0.92), solidColor.opacity(0.68)],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }

    func tintedSurface(opacity: Double) -> Color {
        solidColor.opacity(opacity)
    }
}
