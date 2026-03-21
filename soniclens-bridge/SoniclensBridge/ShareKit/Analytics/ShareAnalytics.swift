import Foundation
import OSLog

enum ShareAnalyticsEvent: String {
    case previewOpened = "preview_opened"
    case renderSucceeded = "render_succeeded"
    case renderFailed = "render_failed"
    case saveSucceeded = "save_succeeded"
    case saveFailed = "save_failed"
    case shareTriggered = "share_triggered"
}

protocol ShareAnalyticsReporting {
    func log(event: ShareAnalyticsEvent, scene: ShareScene, metadata: [String: String])
}

final class ShareAnalytics: ShareAnalyticsReporting {
    static let shared = ShareAnalytics()

    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "ShareAnalytics")

    private init() {}

    func log(event: ShareAnalyticsEvent, scene: ShareScene, metadata: [String: String] = [:]) {
        let metadataText = metadata
            .sorted(by: { $0.key < $1.key })
            .map { "\($0.key)=\($0.value)" }
            .joined(separator: ",")

        logger.info("分享事件 scene=\(scene.rawValue, privacy: .public) event=\(event.rawValue, privacy: .public) metadata=\(metadataText, privacy: .public)")
    }
}
