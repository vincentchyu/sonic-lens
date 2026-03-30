#if os(iOS)
import ActivityKit
import Foundation

@available(iOS 16.2, *)
struct InsightLiveActivityAttributes: ActivityAttributes {
    struct ContentState: Codable, Hashable {
        let title: String
        let artist: String
        let album: String
        let artworkLocalIdentifier: String?
        let providerDisplayName: String
        let modelDisplayName: String
        let phase: InsightJobPhase
    }

    let jobID: String
    let targetType: String
}
#endif
