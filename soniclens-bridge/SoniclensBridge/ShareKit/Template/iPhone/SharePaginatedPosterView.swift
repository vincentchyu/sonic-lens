import SwiftUI

/// 渲染固定占满手机屏幕高度的单页高清分享海报
struct SharePaginatedPosterView: View {
    let header: ShareHeaderPayload
    let footer: ShareFooterPayload
    let slice: ShareContinuousPageSlice
    let scene: ShareScene
    let nodes: [ShareFlowItemNode]
    let targetPageHeight: CGFloat
    var renderedImage: PlatformShareImage? = nil

    var body: some View {
        SharePosterBackground {
            VStack(alignment: .leading, spacing: 0) {
                // 1. 顶部 Header Chrome
                VStack(alignment: .leading, spacing: 5) {
                    Text("SonicLens Bridge · iPhone")
                        .font(.system(size: 8, weight: .medium))
                        .foregroundStyle(Color.white.opacity(0.72))
                        .frame(maxWidth: .infinity, alignment: .center)
                        .padding(.bottom, 12)

                    SharePosterHeader(
                        header: header,
                        continuationLabel: slice.continuationLabel,
                        continuationPreview: slice.continuationPreview,
                        renderedImage: renderedImage
                    )
                }
                .padding(.bottom, 14)

                // 2. 中间满屏正文展示区 Card (连续流式填充)
                SharePosterCard(cornerRadius: 24) {
                    VStack(alignment: .leading, spacing: 12) {
                        Text(sectionTitle)
                            .font(.system(size: 18, weight: .bold, design: .rounded))
                            .foregroundStyle(.white)

                        VStack(alignment: .leading, spacing: 10) {
                            ForEach(nodes) { node in
                                node.view
                            }
                        }
                        .clipped() // 防止高度估算微小误差导致视觉溢出
                    }
                    .frame(maxWidth: .infinity, alignment: .topLeading)
                }

                Spacer(minLength: 0)

                // 3. 底部固定 Footprint 水印区
                SharePosterFooter(footer: footer, pageText: slice.pageText)
                    .padding(.top, 14)
            }
            .padding(20)
            .frame(width: SharePayloadPaginator.posterWidth, height: targetPageHeight, alignment: .topLeading)
        }
        .frame(width: SharePayloadPaginator.posterWidth, height: targetPageHeight)
        .clipped()
    }

    private var sectionTitle: String {
        let suffix = slice.isFirstPage ? "" : " (续)"
        switch scene {
        case .trackInfo, .albumInfo:
            return "内容提要\(suffix)"
        case .trackLyrics:
            return "歌词全文\(suffix)"
        case .trackInsight, .albumInsight:
            return "音眸全文\(suffix)"
        }
    }
}
