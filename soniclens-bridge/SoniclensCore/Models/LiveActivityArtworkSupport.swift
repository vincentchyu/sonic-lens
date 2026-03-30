import Foundation

enum LiveActivityArtworkSupport {
    static let appGroupIdentifier = "group.com.vincentchyu.soniclens-bridge.shared"
    static let directoryName = "LiveActivityArtwork"
    static let fileExtension = "jpg"

    static func containerDirectory(fileManager: FileManager = .default) -> URL? {
        fileManager.containerURL(forSecurityApplicationGroupIdentifier: appGroupIdentifier)?
            .appendingPathComponent(directoryName, isDirectory: true)
    }

    static func fileURL(for identifier: String, fileManager: FileManager = .default) -> URL? {
        let normalized = identifier.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty else { return nil }
        return containerDirectory(fileManager: fileManager)?
            .appendingPathComponent(normalized)
            .appendingPathExtension(fileExtension)
    }
}
