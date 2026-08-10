import CoreGraphics
import Foundation
import SwiftUI

#if os(iOS)
import UIKit
#endif

/// 表示正文流中单个微观排版/行节点 (Node)
struct ShareFlowItemNode: Identifiable, Hashable {
    let id: String
    let height: CGFloat
    let view: AnyView
    let previewText: String?

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
        hasher.combine(height)
    }

    static func == (lhs: ShareFlowItemNode, rhs: ShareFlowItemNode) -> Bool {
        lhs.id == rhs.id && lhs.height == rhs.height
    }
}

/// 连续切片描述 (保证 Y_{k+1, start} == Y_{k, end} 绝对零字节丢失)
struct ShareContinuousPageSlice: Identifiable, Hashable {
    let id: Int // 页码 (1-indexed)
    let startIndex: Int
    let endIndex: Int
    let yStart: CGFloat
    let yEnd: CGFloat
    let isFirstPage: Bool
    let continuationLabel: String?
    let pageText: String // 例如 "第 1 / 3 页"
    let continuationPreview: String? // 上一页末尾的文字预览（接上语义）
}

/// 满屏连续分页计划
struct ShareContinuousPaginationPlan {
    let targetPageHeight: CGFloat
    let totalContentHeight: CGFloat
    let slices: [ShareContinuousPageSlice]
    let totalPages: Int
}

@MainActor
enum SharePayloadPaginator {
    static let posterWidth: CGFloat = 390
    #if os(iOS)
    static let contentMeasurementWidth: CGFloat = 310
    static var defaultTargetPageHeight: CGFloat {
        let screenHeight = UIScreen.main.bounds.height
        return max(screenHeight, 844)
    }
    #else
    /// macOS Right Canvas is ~1432px wide. We split into two columns of ~704px each.
    static let contentMeasurementWidth: CGFloat = 704
    static var defaultTargetPageHeight: CGFloat {
        return 1080
    }
    #endif

    /// 根据 Payload 和屏高预算计算 100% 文本无丢弃的连续满屏分页计划
    static func makePlan(
        payload: SharePayload,
        targetPageHeight: CGFloat? = nil
    ) -> ShareContinuousPaginationPlan {
        let pageHeight = targetPageHeight ?? defaultTargetPageHeight
        let nodes = extractFlowNodes(from: payload)

        // ─── 固定外壳高度精确估算 ───────────────────────────────────
        // 布局层级（SharePaginatedPosterView）：
        //   SharePosterBackground
        //   └── VStack(spacing:0) .padding(20)
        //       ├── VStack(spacing:5)  [Header Chrome]
        //       │   ├── Caption Text: font(8) + .padding(.bottom,12) ≈ 28 pt
        //       │   └── SharePosterHeader:
        //       │       - 第1页 Hero (minHeight 274 + Card.padding(20)*2) ≈ 314 pt
        //       │       - 续页 Compact Card (约 86 + padding 40)         ≈ 80 pt
        //       │   └── .padding(.bottom, 14)
        //       ├── SharePosterCard(cornerRadius:24) [正文区]
        //       │   └── .padding(20) 上下各 20
        //       │       ├── VStack(spacing:12)
        //       │       │   ├── Text(sectionTitle) font(18bold) ≈ 26 pt
        //       │       │   └── VStack(spacing:10) [节点列表]
        //       ├── Spacer(minLength:0)
        //       └── SharePosterFooter .padding(.top,14)
        //           └── 约 5 行文字 ≈ 80 pt
        //
        // 注意：outerPaddingV = 20(top) + 20(bottom) = 40
        //        cardPaddingV  = 20(top) + 20(bottom) = 40
        //        headerSpacing = 14 (header VStack .padding(.bottom,14))
        //        footerTopPad  = 14
        // ────────────────────────────────────────────────────────────

        let topCaptionHeight: CGFloat = 28
        let heroHeaderHeight: CGFloat = 314
        let compactHeaderHeight: CGFloat = 125
        let footerHeight: CGFloat = 84
        let outerPaddingV: CGFloat = 40
        let cardPaddingV: CGFloat = 40
        let headerBottomSpacing: CGFloat = 14
        let footerTopPad: CGFloat = 14
        let sectionTitleH: CGFloat = 26
        let sectionTitleSpacing: CGFloat = 12

        #if os(iOS)
        let page1WindowHeight = pageHeight - outerPaddingV - topCaptionHeight - heroHeaderHeight - headerBottomSpacing - cardPaddingV - sectionTitleH - sectionTitleSpacing - footerTopPad - footerHeight
        let pageNWindowHeight = pageHeight - outerPaddingV - topCaptionHeight - compactHeaderHeight - headerBottomSpacing - cardPaddingV - sectionTitleH - sectionTitleSpacing - footerTopPad - footerHeight
        let safetyMargin: CGFloat = 28
        let usablePage1Window = max(page1WindowHeight - safetyMargin, 150)
        let usablePageNWindow = max(pageNWindowHeight - safetyMargin, 250)
        #else
        // macOS layout: Sidebar on left, Content on right.
        // Top toolbar is ~16pt + spacing 14pt = 30pt. Padding is 24pt.
        // Total outer vertical consumed = 24*2 (padding) + 30 = 78pt.
        // Inside Right Canvas SharePosterCard, padding is 20*2 = 40pt.
        // Section title is 26pt + 12pt = 38pt.
        // Usable height per column = pageHeight - 78 - 40 - 38 = pageHeight - 156.
        // Since we use dual column, we have 2x vertical budget.
        let usableHeightPerColumn = max(pageHeight - 156 - 28, 500)
        let usablePage1Window = usableHeightPerColumn * 2 // Dual column capacity
        let usablePageNWindow = usableHeightPerColumn * 2
        #endif

        // 计算所有节点及累加纵向 Y 坐标
        var nodeIntervals: [(node: ShareFlowItemNode, yStart: CGFloat, yEnd: CGFloat)] = []
        var currentAccumulatedY: CGFloat = 0

        for node in nodes {
            let start = currentAccumulatedY
            let end = currentAccumulatedY + node.height
            nodeIntervals.append((node: node, yStart: start, yEnd: end))
            currentAccumulatedY = end
        }

        let totalContentHeight = currentAccumulatedY

        // 求解严格无缝的 Slice 划分 (Y_{k+1, start} == Y_{k, end})
        var slices: [ShareContinuousPageSlice] = []
        var currentIndex = 0
        var pageIndex = 0
        var lastPageFinalPreview: String? = nil

        if nodeIntervals.isEmpty {
            return ShareContinuousPaginationPlan(
                targetPageHeight: pageHeight,
                totalContentHeight: 0,
                slices: [
                    ShareContinuousPageSlice(
                        id: 1,
                        startIndex: 0,
                        endIndex: -1,
                        yStart: 0,
                        yEnd: 0,
                        isFirstPage: true,
                        continuationLabel: nil,
                        pageText: "",
                        continuationPreview: nil
                    )
                ],
                totalPages: 1
            )
        }

        while currentIndex < nodeIntervals.count {
            let isFirst = pageIndex == 0
            let maxBudget = isFirst ? usablePage1Window : usablePageNWindow
            let pageYStart = nodeIntervals[currentIndex].yStart

            var accumulatedHeight: CGFloat = 0
            var lastIncludedIndex = currentIndex

            for idx in currentIndex..<nodeIntervals.count {
                let nodeH = nodeIntervals[idx].node.height
                // spacing 与 SharePaginatedPosterView 内 VStack(spacing:10) 严格一致
                let spacing: CGFloat = (idx == currentIndex) ? 0 : 10

                if accumulatedHeight + nodeH + spacing <= maxBudget || idx == currentIndex {
                    accumulatedHeight += nodeH + spacing
                    lastIncludedIndex = idx
                } else {
                    // 超预算，在上一节点封包
                    break
                }
            }

            let pageYEnd = nodeIntervals[lastIncludedIndex].yEnd

            slices.append(
                ShareContinuousPageSlice(
                    id: pageIndex + 1,
                    startIndex: currentIndex,
                    endIndex: lastIncludedIndex,
                    yStart: pageYStart,
                    yEnd: pageYEnd,
                    isFirstPage: isFirst,
                    continuationLabel: isFirst ? nil : "续页 · \(pageIndex + 1)",
                    pageText: "",
                    continuationPreview: lastPageFinalPreview
                )
            )

            // 保存当前页最后一个节点的文字预览供下一页续页 Header 展示
            lastPageFinalPreview = nodeIntervals[lastIncludedIndex].node.previewText

            // 下一页索引严格为包含节点的下一个 (100% 绝对零漏字！)
            currentIndex = lastIncludedIndex + 1
            pageIndex += 1
        }

        let totalPages = max(slices.count, 1)
        let finalizedSlices = slices.map { slice in
            ShareContinuousPageSlice(
                id: slice.id,
                startIndex: slice.startIndex,
                endIndex: slice.endIndex,
                yStart: slice.yStart,
                yEnd: slice.yEnd,
                isFirstPage: slice.isFirstPage,
                continuationLabel: slice.continuationLabel,
                pageText: "第 \(slice.id) / \(totalPages) 页",
                continuationPreview: slice.continuationPreview
            )
        }

        return ShareContinuousPaginationPlan(
            targetPageHeight: pageHeight,
            totalContentHeight: totalContentHeight,
            slices: finalizedSlices,
            totalPages: totalPages
        )
    }

    // MARK: - 节点提取 (不漏一个字符的微观拆分)

    static func extractFlowNodes(from payload: SharePayload) -> [ShareFlowItemNode] {
        switch payload {
        case let .info(p):
            return p.fields.map { field in
                makeNode(id: "field_\(field.id)", view: SharePosterInfoFieldCard(field: field))
            }
        case let .albumInfo(p):
            return p.fields.map { field in
                makeNode(id: "field_\(field.id)", view: SharePosterInfoFieldCard(field: field))
            }
        case let .lyrics(p):
            var nodes: [ShareFlowItemNode] = []
            for block in p.blocks {
                let blockNodes = extractLyricsBlockNodes(block)
                nodes.append(contentsOf: blockNodes)
            }
            return nodes
        case let .insight(p):
            return extractInsightDocumentNodes(document: p.document)
        case let .albumInsight(p):
            return extractInsightDocumentNodes(document: p.document)
        }
    }

    private static func extractLyricsBlockNodes(_ block: ShareTextBlock) -> [ShareFlowItemNode] {
        let lines = block.text.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
        return lines.enumerated().map { idx, lineText in
            let lineView = VStack(alignment: .leading, spacing: 4) {
                if idx == 0, let title = block.title {
                    Text(title)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(Color.white.opacity(0.56))
                }
                Text(lineText)
                    .font(.system(size: 17, weight: .medium))
                    .foregroundStyle(.white.opacity(0.94))
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .frame(maxWidth: .infinity, alignment: .leading)

            return makeNode(id: "lyric_\(block.id)_\(idx)", view: lineView, previewText: lineText)
        }
    }

    private static func extractInsightDocumentNodes(document: InsightShareDocument) -> [ShareFlowItemNode] {
        var nodes: [ShareFlowItemNode] = []
        for card in document.cards {
            switch card {
            case let .text(id, title, text):
                // ─── .text card：按段落拆分，防止长文字整体超高 ───
                let paragraphs = text
                    .components(separatedBy: "\n\n")
                    .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                    .filter { !$0.isEmpty }

                if paragraphs.count <= 1 {
                    nodes.append(makeNode(
                        id: "insight_text_\(id)",
                        view: SharePosterInsightTextCard(title: title, text: text),
                        previewText: text
                    ))
                } else {
                    for (pIdx, para) in paragraphs.enumerated() {
                        let paraTitle = pIdx == 0 ? title : ""
                        nodes.append(makeNode(
                            id: "insight_text_\(id)_p\(pIdx)",
                            view: SharePosterInsightTextCard(title: paraTitle, text: para),
                            previewText: para
                        ))
                    }
                }

            case let .tagged(id, title, groups, text):
                let titleOnlyNode = makeNode(
                    id: "insight_tagged_title_\(id)",
                    view: Text(title)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(Color.white.opacity(0.64))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 4),
                    previewText: title
                )
                nodes.append(titleOnlyNode)

                if let text {
                    let textParas = text
                        .components(separatedBy: "\n\n")
                        .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                        .filter { !$0.isEmpty }
                    for (pIdx, para) in textParas.enumerated() {
                        let paraView = Text(para)
                            .font(.system(size: 15, weight: .medium))
                            .lineSpacing(6)
                            .foregroundStyle(.white.opacity(0.9))
                            .fixedSize(horizontal: false, vertical: true)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 4)
                        nodes.append(makeNode(id: "insight_tagged_text_\(id)_p\(pIdx)", view: paraView, previewText: para))
                    }
                }

                for (gIdx, group) in groups.enumerated() {
                    for (rIdx, row) in group.rows.enumerated() {
                        let isLastRow = rIdx == group.rows.count - 1
                        let rowView = VStack(alignment: .leading, spacing: 4) {
                            if !row.original.isEmpty {
                                Text(row.original)
                                    .font(.system(size: 14, weight: .medium))
                                    .foregroundStyle(Color.white.opacity(0.58))
                                    .fixedSize(horizontal: false, vertical: true)
                            }
                            if let translation = row.translation, !translation.isEmpty {
                                Text(translation)
                                    .font(.system(size: 16, weight: .medium))
                                    .italic()
                                    .foregroundStyle(Color.white.opacity(0.72))
                                    .fixedSize(horizontal: false, vertical: true)
                            }
                        }
                        .padding(.horizontal, 12)
                        .padding(.top, rIdx == 0 ? 10 : 4)
                        .padding(.bottom, (isLastRow && group.explain == nil) ? 10 : 4)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        nodes.append(makeNode(id: "insight_tagged_row_\(id)_\(gIdx)_\(rIdx)", view: rowView, previewText: row.original.isEmpty ? row.translation : row.original))
                    }
                    if let explain = group.explain {
                        let explainView = Text(explain)
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(.white.opacity(0.94))
                            .fixedSize(horizontal: false, vertical: true)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(14)
                            .background(Color.white.opacity(0.04), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                        nodes.append(makeNode(id: "insight_tagged_explain_\(id)_\(gIdx)", view: explainView, previewText: explain))
                    }
                }

            case let .section(id, title, sections):
                // ─── 彻底修复：section.text (文学解读/翻译说明/背景介绍等长文章)
                //      必须按段落拆分为独立 Node，绝对不能整篇塞进 headerView！
                for (sIdx, section) in sections.enumerated() {
                    let sectionLabel = sIdx == 0 ? title : "\(title) (续)"

                    // 1. section 标题节点 (仅含标题与小标题)
                    let headerView = VStack(alignment: .leading, spacing: 6) {
                        if sIdx == 0 {
                            Text(sectionLabel)
                                .font(.system(size: 16, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                        }
                        Text(section.title)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(Color.white.opacity(0.64))
                    }
                    .padding(.horizontal, 18)
                    .padding(.top, 18)
                    .padding(.bottom, (section.text == nil && section.groups.isEmpty) ? 18 : 4)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))

                    nodes.append(makeNode(id: "insight_sec_hdr_\(id)_\(sIdx)", view: headerView, previewText: section.title))

                    // 2. section 级别的长文章 (如文学解读、翻译说明等) 按段落独立拆为 Node
                    if let text = section.text, section.id != "appreciate_analysis" {
                        let paragraphs = text
                            .components(separatedBy: "\n\n")
                            .flatMap { $0.components(separatedBy: "\n") }
                            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                            .filter { !$0.isEmpty }

                        for (pIdx, para) in paragraphs.enumerated() {
                            let isLastPara = pIdx == paragraphs.count - 1
                            let paraView = Text(para)
                                .font(.system(size: 15, weight: .medium))
                                .lineSpacing(6)
                                .foregroundStyle(.white.opacity(0.9))
                                .fixedSize(horizontal: false, vertical: true)
                                .padding(.horizontal, 18)
                                .padding(.top, 6)
                                .padding(.bottom, (isLastPara && section.groups.isEmpty) ? 18 : 6)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))

                            nodes.append(makeNode(
                                id: "insight_sec_text_\(id)_\(sIdx)_p\(pIdx)",
                                view: paraView,
                                previewText: para
                            ))
                        }
                    }

                    // 3. 每个 group 的每一行 row 和 explain 各自独立节点
                    for (gIdx, group) in section.groups.enumerated() {
                        let isLastGroup = gIdx == section.groups.count - 1

                        for (rIdx, row) in group.rows.enumerated() {
                            let isLastRow = rIdx == group.rows.count - 1
                            let rowView = VStack(alignment: .leading, spacing: 4) {
                                if !row.original.isEmpty {
                                    Text(row.original)
                                        .font(.system(size: 14, weight: .medium))
                                        .foregroundStyle(Color.white.opacity(0.58))
                                        .fixedSize(horizontal: false, vertical: true)
                                }
                                if let translation = row.translation, !translation.isEmpty {
                                    Text(translation)
                                        .font(.system(size: 16, weight: .medium))
                                        .italic()
                                        .foregroundStyle(Color.white.opacity(0.72))
                                        .fixedSize(horizontal: false, vertical: true)
                                }
                            }
                            .padding(.horizontal, 18)
                            .padding(.top, rIdx == 0 ? 10 : 4)
                            .padding(.bottom, (isLastRow && group.explain == nil && isLastGroup) ? 18 : 4)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
                            nodes.append(makeNode(id: "insight_sec_row_\(id)_\(sIdx)_\(gIdx)_\(rIdx)", view: rowView, previewText: row.original.isEmpty ? row.translation : row.original))
                        }

                        if let explain = group.explain {
                            let explainView = Text(explain)
                                .font(.system(size: 16, weight: .semibold))
                                .foregroundStyle(.white.opacity(0.94))
                                .fixedSize(horizontal: false, vertical: true)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(14)
                                .padding(.bottom, isLastGroup ? 14 : 4)
                                .background(Color.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                                .padding(.horizontal, 18)
                                .padding(.bottom, isLastGroup ? 4 : 0)
                            nodes.append(makeNode(id: "insight_sec_explain_\(id)_\(sIdx)_\(gIdx)", view: explainView, previewText: explain))
                        }
                    }
                }
            }
        }
        return nodes
    }

    private static func makeNode(id: String, view: some View, previewText: String? = nil) -> ShareFlowItemNode {
        let measuredHeight = measureViewHeight(view)
        return ShareFlowItemNode(id: id, height: measuredHeight, view: AnyView(view), previewText: previewText)
    }

    private static func measureViewHeight(_ view: some View) -> CGFloat {
        #if os(iOS)
        let controller = UIHostingController(rootView: view)
        controller.view.backgroundColor = .clear
        let measured = controller.sizeThatFits(
            in: CGSize(width: contentMeasurementWidth, height: CGFloat.greatestFiniteMagnitude)
        )
        return ceil(measured.height)
        #else
        let controller = NSHostingController(rootView: view)
        let measured = controller.sizeThatFits(
            in: CGSize(width: contentMeasurementWidth, height: CGFloat.greatestFiniteMagnitude)
        )
        return max(ceil(measured.height), 30)
        #endif
    }
}

