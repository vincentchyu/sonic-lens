#if os(iOS)
import SwiftUI
import UIKit
import UniformTypeIdentifiers

struct ShareSheetPresenter: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

enum ShareActivityItemFactory {
    static func makeItems(fileURLs: [URL]) -> [Any] {
        fileURLs.map { url in
            if let image = UIImage(contentsOfFile: url.path) {
                return ShareImageActivityItemSource(
                    image: image,
                    filename: url.deletingPathExtension().lastPathComponent
                )
            }
            return url
        }
    }
}

private final class ShareImageActivityItemSource: NSObject, UIActivityItemSource {
    private let image: UIImage
    private let filename: String

    init(image: UIImage, filename: String) {
        self.image = image
        self.filename = filename
        super.init()
    }

    func activityViewControllerPlaceholderItem(_ activityViewController: UIActivityViewController) -> Any {
        image
    }

    func activityViewController(
        _ activityViewController: UIActivityViewController,
        itemForActivityType activityType: UIActivity.ActivityType?
    ) -> Any? {
        image
    }

    func activityViewController(
        _ activityViewController: UIActivityViewController,
        dataTypeIdentifierForActivityType activityType: UIActivity.ActivityType?
    ) -> String {
        UTType.png.identifier
    }

    func activityViewController(
        _ activityViewController: UIActivityViewController,
        subjectForActivityType activityType: UIActivity.ActivityType?
    ) -> String {
        filename
    }
}
#else
import SwiftUI
import AppKit

@MainActor
enum MacShareActionHelper {
    @discardableResult
    static func copyImagesToPasteboard(fileURLs: [URL]) -> Bool {
        let images = fileURLs.compactMap { NSImage(contentsOf: $0) }
        guard !images.isEmpty else { return false }
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        return pasteboard.writeObjects(images)
    }

    static func showSharingPicker(fileURLs: [URL], relativeTo rect: NSRect, of view: NSView) {
        let items: [Any] = fileURLs.compactMap { NSImage(contentsOf: $0) }
        guard !items.isEmpty else { return }
        let picker = NSSharingServicePicker(items: items)
        picker.show(relativeTo: rect, of: view, preferredEdge: .minY)
    }
}

struct ShareSheetPresenter: View {
    let items: [Any]

    var body: some View {
        EmptyView()
    }
}
#endif
