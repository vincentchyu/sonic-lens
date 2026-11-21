import AppKit
import SwiftUI
import XCTest

@MainActor
enum SnapshotAppearance {
    case light
    case dark

    var nsAppearanceName: NSAppearance.Name {
        switch self {
        case .light:
            return .aqua
        case .dark:
            return .darkAqua
        }
    }

    var colorScheme: ColorScheme {
        switch self {
        case .light:
            return .light
        case .dark:
            return .dark
        }
    }
}

@MainActor
enum SnapshotRenderer {
    static func outputDirectory(callerFilePath: String = #filePath) -> URL {
        if let customPath = ProcessInfo.processInfo.environment["SONICLENS_UI_SNAPSHOT_DIR"],
           !customPath.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            let url = URL(fileURLWithPath: customPath, isDirectory: true)
            try? FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
            return url
        }

        // .../soniclens-bridge/SoniclensBridgeMacTests/SnapshotRenderer.swift -> 仓库根目录
        let fileURL = URL(fileURLWithPath: callerFilePath)
        let repoRoot = fileURL
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        let dir = repoRoot
            .appendingPathComponent(".derivedData", isDirectory: true)
            .appendingPathComponent("ui_snapshots", isDirectory: true)
            .appendingPathComponent("macos", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }

    @discardableResult
    static func render<V: View>(
        _ view: V,
        named filename: String,
        size: CGSize,
        appearance: SnapshotAppearance,
        settleInterval: TimeInterval = 0.28,
        file: StaticString = #filePath,
        line: UInt = #line
    ) throws -> URL {
        let wrapped = view
            .frame(width: size.width, height: size.height)
            .environment(\.colorScheme, appearance.colorScheme)

        let hostingView = NSHostingView(rootView: wrapped)
        hostingView.frame = NSRect(origin: .zero, size: size)
        hostingView.appearance = NSAppearance(named: appearance.nsAppearanceName)

        let window = NSWindow(
            contentRect: NSRect(x: -12000, y: -12000, width: size.width, height: size.height),
            styleMask: [.borderless],
            backing: .buffered,
            defer: false
        )
        window.isReleasedWhenClosed = false
        window.appearance = NSAppearance(named: appearance.nsAppearanceName)
        window.backgroundColor = .clear
        window.isOpaque = false
        window.contentView = hostingView
        window.orderBack(nil)

        // 先走一轮布局，让 onAppear / measureDebouncedWidth 与异步 Canvas 完成首次渲染
        hostingView.layoutSubtreeIfNeeded()
        RunLoop.main.run(until: Date().addingTimeInterval(settleInterval))
        hostingView.layoutSubtreeIfNeeded()
        hostingView.displayIfNeeded()
        RunLoop.main.run(until: Date().addingTimeInterval(0.05))

        let bounds = hostingView.bounds
        guard let bitmap = hostingView.bitmapImageRepForCachingDisplay(in: bounds) else {
            window.close()
            XCTFail("无法创建离屏位图缓存: \(filename)", file: file, line: line)
            throw NSError(domain: "SnapshotRenderer", code: 1)
        }
        hostingView.cacheDisplay(in: bounds, to: bitmap)
        window.close()

        guard bitmap.pixelsWide > 0, bitmap.pixelsHigh > 0,
              let pngData = bitmap.representation(using: .png, properties: [:]) else {
            XCTFail("PNG 编码失败: \(filename)", file: file, line: line)
            throw NSError(domain: "SnapshotRenderer", code: 2)
        }

        let targetURL = outputDirectory().appendingPathComponent(filename)
        try pngData.write(to: targetURL, options: .atomic)
        return targetURL
    }
}
