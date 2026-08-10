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
        let plan = SharePayloadPaginator.makePlan(payload: payload)

        switch mode {
        case .singleLongImage:
            return try renderSingleResult(
                payload: payload,
                artworkImage: artworkImage,
                pageText: nil
            )
        case .automatic where plan.slices.count == 1:
            return try renderSingleResult(
                payload: payload,
                artworkImage: artworkImage,
                pageText: plan.slices[0].pageText
            )
        case .automatic, .pagedImages:
            break
        }

        let footerPayload = footer(for: payload)
        let allNodes = SharePayloadPaginator.extractFlowNodes(from: payload)
        var urls: [URL] = []

        for (index, slice) in plan.slices.enumerated() {
            let sliceNodes: [ShareFlowItemNode]
            if slice.startIndex <= slice.endIndex && slice.endIndex < allNodes.count {
                sliceNodes = Array(allNodes[slice.startIndex...slice.endIndex])
            } else {
                sliceNodes = []
            }

            let pageView = SharePaginatedPosterView(
                header: payload.header,
                footer: footerPayload,
                slice: slice,
                scene: payload.scene,
                nodes: sliceNodes,
                targetPageHeight: plan.targetPageHeight,
                renderedImage: artworkImage
            )

            let image = try renderSingleImage(
                from: AnyView(pageView),
                size: CGSize(width: LongPosterPaginator.posterWidth, height: plan.targetPageHeight)
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

        return ShareRenderResult(
            fileURLs: urls,
            logicalSize: CGSize(width: LongPosterPaginator.posterWidth, height: plan.targetPageHeight)
        )
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

private enum ShareRendererError: LocalizedError {
    case renderFailed

    var errorDescription: String? {
        switch self {
        case .renderFailed:
            return "分享图片渲染失败"
        }
    }
}
#else
import SwiftUI
import AppKit

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
        let plan = SharePayloadPaginator.makePlan(payload: payload, targetPageHeight: 1080)

        let footerPayload = footer(for: payload)
        let allNodes = SharePayloadPaginator.extractFlowNodes(from: payload)
        var urls: [URL] = []

        for (index, slice) in plan.slices.enumerated() {
            let sliceNodes: [ShareFlowItemNode]
            if slice.startIndex <= slice.endIndex && slice.endIndex < allNodes.count {
                sliceNodes = Array(allNodes[slice.startIndex...slice.endIndex])
            } else {
                sliceNodes = []
            }

            let pageView = MacSharePaginatedPosterView(
                header: payload.header,
                footer: footerPayload,
                slice: slice,
                scene: payload.scene,
                nodes: sliceNodes,
                targetPageHeight: 1080,
                renderedImage: artworkImage
            )

            let image = try renderSingleImage(
                from: AnyView(pageView),
                size: CGSize(width: 1920, height: 1080)
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

        return ShareRenderResult(
            fileURLs: urls,
            logicalSize: CGSize(width: 1920, height: 1080)
        )
    }

    private func footer(for payload: SharePayload) -> ShareFooterPayload {
        switch payload {
        case let .insight(p): return p.footer
        case let .lyrics(p): return p.footer
        case let .info(p): return p.footer
        case let .albumInfo(p): return p.footer
        case let .albumInsight(p): return p.footer
        }
    }

    private func loadArtworkImage(from urlString: String?) async -> NSImage? {
        guard let urlString, let url = URL(string: urlString) else { return nil }
        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            return NSImage(data: data)
        } catch {
            return nil
        }
    }

    private func renderSingleImage(from view: AnyView, size: CGSize) throws -> NSImage {
        let framed = view.frame(width: size.width, height: size.height, alignment: .top).clipped()
        let renderer = ImageRenderer(content: framed)
        renderer.scale = 2.0
        guard let nsImage = renderer.nsImage else {
            throw ShareRendererError.renderFailed
        }
        return nsImage
    }

    private func imageData(from image: NSImage) throws -> EncodedImageData {
        guard let tiffData = image.tiffRepresentation,
              let bitmapRep = NSBitmapImageRep(data: tiffData),
              let pngData = bitmapRep.representation(using: .png, properties: [:]) else {
            throw ShareRendererError.renderFailed
        }
        return EncodedImageData(data: pngData, fileExtension: "png")
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
#endif
