import SwiftUI

/// macOS 专属连续切片横屏渲染视图 (Mac Horizontal Paginated Poster View)
struct MacSharePaginatedPosterView: View {
    let header: ShareHeaderPayload
    let footer: ShareFooterPayload
    let slice: ShareContinuousPageSlice
    let scene: ShareScene
    let nodes: [ShareFlowItemNode]
    let targetPageHeight: CGFloat
    let renderedImage: PlatformShareImage?

    var body: some View {
        MacSharePosterShell(
            header: header,
            footer: footer,
            renderedImage: renderedImage,
            continuationLabel: slice.continuationLabel,
            pageText: slice.pageText,
            showsFooter: true,
            showsTrafficLights: true,
            topCaptionText: "SonicLens Bridge · macOS Ultra HD 16:9"
        ) {
            VStack(alignment: .leading, spacing: 14) {
                // 卡片正文标题
                HStack {
                    Text(sectionTitle)
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)

                    Spacer()

                    if let label = slice.continuationLabel {
                        Text(label)
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(Color.white.opacity(0.60))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(Color.white.opacity(0.08), in: Capsule())
                    }
                }
                .padding(.bottom, 6)

                // 节点内容：双栏流式呈现 (Bento Grid / Dual Column)
                HStack(alignment: .top, spacing: 24) {
                    VStack(alignment: .leading, spacing: 12) {
                        ForEach(leftNodes) { node in
                            node.view
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .topLeading)

                    VStack(alignment: .leading, spacing: 12) {
                        ForEach(rightNodes) { node in
                            node.view
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .topLeading)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
        .frame(width: 1920, height: 1080)
    }

    private var leftNodes: [ShareFlowItemNode] {
        splitNodes().left
    }

    private var rightNodes: [ShareFlowItemNode] {
        splitNodes().right
    }

    private func splitNodes() -> (left: [ShareFlowItemNode], right: [ShareFlowItemNode]) {
        let budget = (targetPageHeight - 156 - 28) // usableHeightPerColumn
        var left: [ShareFlowItemNode] = []
        var right: [ShareFlowItemNode] = []
        var leftHeight: CGFloat = 0

        for node in nodes {
            // spacing is 12
            let spacing: CGFloat = left.isEmpty ? 0 : 12
            if leftHeight + node.height + spacing <= budget {
                left.append(node)
                leftHeight += node.height + spacing
            } else {
                right.append(node)
            }
        }
        return (left, right)
    }

    private var sectionTitle: String {
        switch scene {
        case .trackInsight: return "音眸鉴赏剖析"
        case .trackLyrics: return "歌词与对齐"
        case .trackInfo: return "曲目元数据"
        case .albumInfo: return "专辑详细档案"
        case .albumInsight: return "专辑深度音眸"
        }
    }
}

/// macOS 单页全景海报视图工厂
@MainActor
enum MacSharePosterViewFactory {
    static func makeView(
        payload: SharePayload,
        renderedImage: PlatformShareImage? = nil,
        pageText: String? = nil
    ) -> AnyView {
        let footerPayload = footer(for: payload)

        let view = MacSharePosterShell(
            header: payload.header,
            footer: footerPayload,
            renderedImage: renderedImage,
            pageText: pageText,
            showsFooter: true,
            showsTrafficLights: true
        ) {
            VStack(alignment: .leading, spacing: 16) {
                switch payload {
                case let .insight(p):
                    InsightLongPosterContent(document: p.document)
                case let .lyrics(p):
                    LyricsLongPosterContent(blocks: p.blocks)
                case let .info(p):
                    InfoPosterContent(fields: p.fields)
                case let .albumInfo(p):
                    InfoPosterContent(fields: p.fields)
                case let .albumInsight(p):
                    InsightLongPosterContent(document: p.document)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
        .frame(width: 1920, height: 1080)

        return AnyView(view)
    }

    private static func footer(for payload: SharePayload) -> ShareFooterPayload {
        switch payload {
        case let .insight(p): return p.footer
        case let .lyrics(p): return p.footer
        case let .info(p): return p.footer
        case let .albumInfo(p): return p.footer
        case let .albumInsight(p): return p.footer
        }
    }
}
