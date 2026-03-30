#if os(iOS)
import SwiftUI
import UIKit

@MainActor
final class ShareRenderer {
    static let shared = ShareRenderer()

    private let tempFileStore = ShareTempFileStore()

    private init() {}

    enum RenderMode {
        case automatic
        case singleLongImage
        case pagedImages
    }

    private struct EncodedImageData {
        let data: Data
        let fileExtension: String
    }

    func render(payload: SharePayload, mode: RenderMode = .automatic) async throws -> ShareRenderResult {
        let artworkImage = await loadArtworkImage(from: payload.header.artworkURL)
        let measurementView = SharePosterViewFactory.makeView(
            payload: payload,
            renderedImage: artworkImage,
            showsFooter: false
        )
        let measuredHeight = measureHeight(of: measurementView, width: LongPosterPaginator.posterWidth)
        let paginationPlan = LongPosterPaginator.makePlan(contentHeight: measuredHeight)

        switch mode {
        case .singleLongImage:
            return try renderSingleResult(
                payload: payload,
                artworkImage: artworkImage,
                pageText: nil
            )
        case .automatic where paginationPlan.slices.count == 1:
            return try renderSingleResult(
                payload: payload,
                artworkImage: artworkImage,
                pageText: paginationPlan.slices[0].pageText
            )
        case .automatic, .pagedImages:
            break
        }

        let footer = footer(for: payload)
        let continuationHeaderRenderer = ContinuationChromeRenderer(payload: payload, renderedImage: artworkImage)

        var urls: [URL] = []
        for (index, slice) in paginationPlan.slices.enumerated() {
            let image = try renderPaginatedImage(
                contentView: measurementView,
                slice: slice,
                footer: footer,
                continuationHeaderRenderer: continuationHeaderRenderer
            )
            let encodedData = try imageData(from: image)
            urls.append(
                try tempFileStore.writeImageData(
                    encodedData.data,
                    suggestedFilename: payload.filename,
                    pageIndex: index,
                    fileExtension: encodedData.fileExtension
                )
            )
        }

        return ShareRenderResult(fileURLs: urls, logicalSize: paginationPlan.logicalSize)
    }

    private func renderSingleResult(
        payload: SharePayload,
        artworkImage: UIImage?,
        pageText: String?
    ) throws -> ShareRenderResult {
        let singleView = SharePosterViewFactory.makeView(
            payload: payload,
            renderedImage: artworkImage,
            pageText: pageText
        )
        let singleMeasuredHeight = measureHeight(of: singleView, width: LongPosterPaginator.posterWidth)
        let image = try renderSingleImage(
            from: singleView,
            size: CGSize(width: LongPosterPaginator.posterWidth, height: singleMeasuredHeight)
        )
        let encodedData = try imageData(from: image)
        let url = try tempFileStore.writeImageData(
            encodedData.data,
            suggestedFilename: payload.filename,
            pageIndex: nil,
            fileExtension: encodedData.fileExtension
        )
        return ShareRenderResult(
            fileURLs: [url],
            logicalSize: CGSize(width: LongPosterPaginator.posterWidth, height: singleMeasuredHeight)
        )
    }

    private func footer(for payload: SharePayload) -> ShareFooterPayload {
        switch payload {
        case let .insight(payload):
            return payload.footer
        case let .lyrics(payload):
            return payload.footer
        case let .info(payload):
            return payload.footer
        case let .albumInfo(payload):
            return payload.footer
        case let .albumInsight(payload):
            return payload.footer
        }
    }

    private func loadArtworkImage(from urlString: String?) async -> UIImage? {
        guard let urlString,
              let url = URL(string: urlString)
        else {
            return nil
        }

        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            return UIImage(data: data)
        } catch {
            return nil
        }
    }

    private func measureHeight(of view: AnyView, width: CGFloat) -> CGFloat {
        let controller = UIHostingController(rootView: view)
        controller.view.backgroundColor = .clear
        let measured = controller.sizeThatFits(in: CGSize(width: width, height: CGFloat.greatestFiniteMagnitude))
        return ceil(measured.height)
    }

    private func renderSingleImage(from view: AnyView, size: CGSize) throws -> UIImage {
        if size.height > LongPosterPaginator.singleImageMaxHeight {
            return try renderTiledSingleImage(from: view, size: size)
        }

        let framed = view
            .frame(width: size.width, height: size.height, alignment: .top)
            .clipped()
        let renderer = ImageRenderer(content: framed)
        renderer.scale = renderScale(for: size.height)
        guard let image = renderer.uiImage else {
            throw ShareRendererError.renderFailed
        }
        return image
    }

    private func renderTiledSingleImage(from view: AnyView, size: CGSize) throws -> UIImage {
        let scale = renderScale(for: size.height)
        let tileHeight = LongPosterPaginator.pagedImageMaxHeight
        let totalHeight = ceil(size.height)
        let totalWidth = ceil(size.width)

        var renderedTiles: [(image: UIImage, offsetY: CGFloat)] = []
        var offsetY: CGFloat = 0

        while offsetY < totalHeight {
            let currentTileHeight = min(tileHeight, totalHeight - offsetY)
            let tileView = view
                .frame(width: totalWidth, height: totalHeight, alignment: .top)
                .offset(y: -offsetY)
                .frame(width: totalWidth, height: currentTileHeight, alignment: .top)
                .clipped()

            let tileRenderer = ImageRenderer(content: tileView)
            tileRenderer.scale = scale
            guard let tileImage = tileRenderer.uiImage else {
                throw ShareRendererError.renderFailed
            }
            renderedTiles.append((tileImage, offsetY))
            offsetY += currentTileHeight
        }

        let format = UIGraphicsImageRendererFormat.default()
        format.scale = scale
        let stitchedRenderer = UIGraphicsImageRenderer(
            size: CGSize(width: totalWidth, height: totalHeight),
            format: format
        )

        return stitchedRenderer.image { _ in
            for tile in renderedTiles {
                tile.image.draw(at: CGPoint(x: 0, y: tile.offsetY))
            }
        }
    }

    private func renderPaginatedImage(
        contentView: AnyView,
        slice: LongPosterPaginationPlan.Slice,
        footer: ShareFooterPayload,
        continuationHeaderRenderer: ContinuationChromeRenderer
    ) throws -> UIImage {
        let headerInset = slice.continuationLabel == nil ? CGFloat(0) : LongPosterPaginator.continuationHeaderHeight
        let footerInset: CGFloat = 64
        let canvasHeight = headerInset + slice.contentHeight + footerInset
        let pageView = SharePaginatedRenderView(
            contentView: contentView,
            slice: slice,
            footer: footer,
            continuationHeaderRenderer: continuationHeaderRenderer,
            canvasHeight: canvasHeight,
            headerInset: headerInset,
            footerInset: footerInset
        )

        let renderer = ImageRenderer(content: pageView)
        renderer.scale = renderScale(for: canvasHeight)
        guard let image = renderer.uiImage else {
            throw ShareRendererError.renderFailed
        }
        return image
    }

    private func imageData(from image: UIImage) throws -> EncodedImageData {
        if let pngData = image.pngData() {
            return EncodedImageData(data: pngData, fileExtension: "png")
        }

        guard let jpegData = image.jpegData(compressionQuality: 0.96) else {
            throw ShareRendererError.renderFailed
        }
        return EncodedImageData(data: jpegData, fileExtension: "jpg")
    }

    private func renderScale(for height: CGFloat) -> CGFloat {
        switch height {
        case ..<5000:
            return 3
        case ..<9000:
            return 2
        default:
            return 1.5
        }
    }
}

private struct SharePaginatedRenderView: View {
    let contentView: AnyView
    let slice: LongPosterPaginationPlan.Slice
    let footer: ShareFooterPayload
    let continuationHeaderRenderer: ContinuationChromeRenderer
    let canvasHeight: CGFloat
    let headerInset: CGFloat
    let footerInset: CGFloat

    var body: some View {
        ZStack(alignment: .topLeading) {
            Color.black

            contentView
                .offset(y: headerInset - slice.contentOffsetY)

            if let label = slice.continuationLabel,
               let headerImage = continuationHeaderRenderer.renderHeader(label: label) {
                Image(uiImage: headerImage)
                    .resizable()
                    .frame(
                        width: LongPosterPaginator.posterWidth - 40,
                        height: LongPosterPaginator.continuationHeaderHeight - 20
                    )
                    .offset(x: 20, y: 16)
            }

            if let footerImage = continuationHeaderRenderer.renderFooter(footer: footer, pageText: slice.pageText) {
                Image(uiImage: footerImage)
                    .resizable()
                    .frame(width: LongPosterPaginator.posterWidth - 40, height: 40)
                    .offset(x: 20, y: canvasHeight - footerInset + 10)
            }
        }
        .frame(width: LongPosterPaginator.posterWidth, height: canvasHeight, alignment: .topLeading)
        .clipped()
    }
}

private enum ShareRendererError: LocalizedError {
    case renderFailed

    var errorDescription: String? {
        switch self {
        case .renderFailed:
            return "分享图片渲染失败"
        }
    }
}

@MainActor
private final class ContinuationChromeRenderer {
    private let payload: SharePayload
    private let renderedImage: UIImage?

    init(payload: SharePayload, renderedImage: UIImage?) {
        self.payload = payload
        self.renderedImage = renderedImage
    }

    func renderHeader(label: String) -> UIImage? {
        let view = SharePosterHeader(header: payload.header, continuationLabel: label, renderedImage: renderedImage)
            .frame(width: LongPosterPaginator.posterWidth - 40)
        let renderer = ImageRenderer(content: view)
        renderer.scale = 3
        return renderer.uiImage
    }

    func renderFooter(footer: ShareFooterPayload, pageText: String) -> UIImage? {
        let view = SharePosterFooter(footer: footer, pageText: pageText)
            .frame(width: LongPosterPaginator.posterWidth - 40)
        let renderer = ImageRenderer(content: view)
        renderer.scale = 3
        return renderer.uiImage
    }
}
#else
import Foundation

@MainActor
final class ShareRenderer {
    static let shared = ShareRenderer()

    private init() {}

    func render(payload: SharePayload) async throws -> ShareRenderResult {
        _ = payload
        return ShareRenderResult(fileURLs: [], logicalSize: .zero)
    }
}
#endif
