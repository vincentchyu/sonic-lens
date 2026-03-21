import CoreGraphics

struct LongPosterPaginationPlan {
    struct Slice {
        let contentOffsetY: CGFloat
        let contentHeight: CGFloat
        let continuationLabel: String?
        let pageText: String
    }

    let logicalSize: CGSize
    let slices: [Slice]
}

enum LongPosterPaginator {
    static let posterWidth: CGFloat = 390
    static let singleImageMaxHeight: CGFloat = 4800
    static let pagedImageMaxHeight: CGFloat = 3200
    static let continuationHeaderHeight: CGFloat = 78

    static func makePlan(contentHeight: CGFloat) -> LongPosterPaginationPlan {
        let width = posterWidth
        guard contentHeight > singleImageMaxHeight else {
            return LongPosterPaginationPlan(
                logicalSize: CGSize(width: width, height: contentHeight),
                slices: [
                    LongPosterPaginationPlan.Slice(
                        contentOffsetY: 0,
                        contentHeight: contentHeight,
                        continuationLabel: nil,
                        pageText: "第 1 / 1 页"
                    )
                ]
            )
        }

        let subsequentContentHeight = pagedImageMaxHeight - continuationHeaderHeight - 20
        var slices: [LongPosterPaginationPlan.Slice] = []
        var offsetY: CGFloat = 0
        var pageIndex = 0

        while offsetY < contentHeight {
            let isFirstPage = pageIndex == 0
            let visibleContentHeight = isFirstPage
                ? min(pagedImageMaxHeight, contentHeight - offsetY)
                : min(subsequentContentHeight, contentHeight - offsetY)

            slices.append(
                LongPosterPaginationPlan.Slice(
                    contentOffsetY: offsetY,
                    contentHeight: visibleContentHeight,
                    continuationLabel: isFirstPage ? nil : "续页 · \(pageIndex + 1)",
                    pageText: ""
                )
            )

            offsetY += visibleContentHeight
            pageIndex += 1
        }

        let total = max(slices.count, 1)
        let finalized = slices.enumerated().map { index, slice in
            LongPosterPaginationPlan.Slice(
                contentOffsetY: slice.contentOffsetY,
                contentHeight: slice.contentHeight,
                continuationLabel: slice.continuationLabel,
                pageText: "第 \(index + 1) / \(total) 页"
            )
        }

        return LongPosterPaginationPlan(
            logicalSize: CGSize(width: width, height: pagedImageMaxHeight),
            slices: finalized
        )
    }
}
