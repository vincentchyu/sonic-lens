#if os(iOS)
import Foundation
import ImageIO
import OSLog
import UIKit

actor LiveActivityArtworkStore {
    static let shared = LiveActivityArtworkStore()

    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "LiveActivityArtworkStore")
    private let fileManager: FileManager
    private let session: URLSession
    private let maxPixelSize: CGFloat
    private let maxCachedFiles: Int
    private var inFlight: [String: Task<String?, Never>] = [:]

    init(
        fileManager: FileManager = .default,
        session: URLSession? = nil,
        maxPixelSize: CGFloat = 192,
        maxCachedFiles: Int = 48
    ) {
        self.fileManager = fileManager
        self.maxPixelSize = maxPixelSize
        self.maxCachedFiles = maxCachedFiles
        if let session {
            self.session = session
        } else {
            let config = URLSessionConfiguration.default
            config.timeoutIntervalForRequest = 8
            config.timeoutIntervalForResource = 12
            self.session = URLSession(configuration: config)
        }
    }

    func cachedLocalIdentifier(for resource: ResolvedArtworkResource?) -> String? {
        guard Self.isPhoneEnvironment,
              let identifier = resource?.localIdentifier,
              let fileURL = LiveActivityArtworkSupport.fileURL(for: identifier, fileManager: fileManager),
              fileManager.fileExists(atPath: fileURL.path) else {
            return nil
        }

        touch(fileURL)
        return identifier
    }

    func prefetch(resource: ResolvedArtworkResource?) async -> String? {
        await prepareArtwork(for: resource)
    }

    func prepareArtwork(for resource: ResolvedArtworkResource?) async -> String? {
        guard Self.isPhoneEnvironment else { return nil }
        guard let resource else { return nil }

        if let cached = cachedLocalIdentifier(for: resource) {
            return cached
        }

        guard let identifier = resource.localIdentifier,
              let remoteURLString = resource.remoteURL,
              let remoteURL = URL(string: remoteURLString) else {
            return nil
        }

        if let running = inFlight[identifier] {
            return await running.value
        }

        let task = Task<String?, Never> { [fileManager, session, logger, maxPixelSize] in
            guard let targetURL = LiveActivityArtworkSupport.fileURL(for: identifier, fileManager: fileManager),
                  let directoryURL = LiveActivityArtworkSupport.containerDirectory(fileManager: fileManager) else {
                logger.error("解析 Live Activity 共享目录失败 id=\(identifier, privacy: .public)")
                return nil
            }

            do {
                if !fileManager.fileExists(atPath: directoryURL.path) {
                    try fileManager.createDirectory(at: directoryURL, withIntermediateDirectories: true)
                }

                let request = URLRequest(url: remoteURL, cachePolicy: .returnCacheDataElseLoad, timeoutInterval: 8)
                let (data, response) = try await session.data(for: request)
                if let http = response as? HTTPURLResponse, !(200...299).contains(http.statusCode) {
                    logger.error("下载 Live Activity 封面失败 id=\(identifier, privacy: .public) status=\(http.statusCode, privacy: .public)")
                    return nil
                }

                guard let jpegData = Self.makeThumbnailJPEG(from: data, maxPixelSize: maxPixelSize) else {
                    logger.error("缩略 Live Activity 封面失败 id=\(identifier, privacy: .public)")
                    return nil
                }

                try jpegData.write(to: targetURL, options: .atomic)
                try? fileManager.setAttributes([.modificationDate: Date()], ofItemAtPath: targetURL.path)
                logger.debug("写入 Live Activity 本地封面成功 id=\(identifier, privacy: .public) path=\(targetURL.lastPathComponent, privacy: .public)")
                return identifier
            } catch {
                logger.error("准备 Live Activity 本地封面失败 id=\(identifier, privacy: .public) error=\(error.localizedDescription, privacy: .public)")
                return nil
            }
        }
        inFlight[identifier] = task

        let result = await task.value
        inFlight.removeValue(forKey: identifier)

        if let result {
            trimCache(keeping: [result])
        }
        return result
    }

    private func trimCache(keeping identifiers: Set<String>) {
        guard let directoryURL = LiveActivityArtworkSupport.containerDirectory(fileManager: fileManager) else { return }
        guard let contents = try? fileManager.contentsOfDirectory(
            at: directoryURL,
            includingPropertiesForKeys: [.contentModificationDateKey],
            options: [.skipsHiddenFiles]
        ) else {
            return
        }

        let candidates = contents.filter { url in
            let identifier = url.deletingPathExtension().lastPathComponent
            return !identifiers.contains(identifier)
        }

        guard candidates.count >= maxCachedFiles else { return }

        let sorted = candidates.sorted { lhs, rhs in
            let lhsDate = (try? lhs.resourceValues(forKeys: [.contentModificationDateKey]))?.contentModificationDate ?? .distantPast
            let rhsDate = (try? rhs.resourceValues(forKeys: [.contentModificationDateKey]))?.contentModificationDate ?? .distantPast
            return lhsDate > rhsDate
        }

        for url in sorted.dropFirst(maxCachedFiles - identifiers.count) {
            try? fileManager.removeItem(at: url)
        }
    }

    private func touch(_ url: URL) {
        try? fileManager.setAttributes([.modificationDate: Date()], ofItemAtPath: url.path)
    }

    private static var isPhoneEnvironment: Bool {
        UIDevice.current.userInterfaceIdiom == .phone
    }

    private static func makeThumbnailJPEG(from data: Data, maxPixelSize: CGFloat) -> Data? {
        guard let source = CGImageSourceCreateWithData(data as CFData, nil) else {
            return UIImage(data: data)?.jpegData(compressionQuality: 0.88)
        }

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: Int(maxPixelSize),
            kCGImageSourceShouldCacheImmediately: true
        ]
        guard let thumbnail = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return UIImage(data: data)?.jpegData(compressionQuality: 0.88)
        }
        return UIImage(cgImage: thumbnail).jpegData(compressionQuality: 0.88)
    }
}
#endif
