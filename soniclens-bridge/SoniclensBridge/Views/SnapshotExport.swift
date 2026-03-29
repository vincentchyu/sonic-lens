import SwiftUI

#if os(macOS)
import AppKit

@MainActor
func exportSnapshotPNG<V: View>(_ view: V, suggestedFilename: String) {
    let renderer = ImageRenderer(content: view)
    renderer.scale = NSScreen.main?.backingScaleFactor ?? 2

    guard let nsImage = renderer.nsImage,
          let tiff = nsImage.tiffRepresentation,
          let rep = NSBitmapImageRep(data: tiff),
          let pngData = rep.representation(using: .png, properties: [:])
    else {
        return
    }

    let panel = NSSavePanel()
    panel.allowedContentTypes = [.png]
    panel.nameFieldStringValue = suggestedFilename.hasSuffix(".png") ? suggestedFilename : "\(suggestedFilename).png"
    panel.canCreateDirectories = true

    panel.begin { response in
        guard response == .OK, let url = panel.url else { return }
        do {
            try pngData.write(to: url, options: .atomic)
        } catch {
            // ignore export errors
        }
    }
}
#else
@MainActor
func exportSnapshotPNG<V: View>(_ view: V, suggestedFilename: String) {
    _ = view
    _ = suggestedFilename
}
#endif
