import Foundation
import OSLog

struct LibrarySyncResult {
    let hasChanges: Bool
    let updatedAlbumsCount: Int
    let updatedTracksCount: Int
}

final class LibrarySyncService {
    private let indexStore: LibraryIndexStore
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "LibrarySync")

    init(indexStore: LibraryIndexStore) {
        self.indexStore = indexStore
    }

    @discardableResult
    func sync(using server: ServerConfig, forceFullSync: Bool = false) async throws -> LibrarySyncResult {
        let client = APIClient(baseURL: server.baseURL)
        let requiresResync = try await indexStore.requiresFullResync()
        let shouldForceFullSync = forceFullSync || requiresResync
        let currentVersion = shouldForceFullSync ? 0 : (try await indexStore.currentSyncVersion())
        logger.log(
            "start library sync forceFull=\(forceFullSync, privacy: .public) requiresResync=\(requiresResync, privacy: .public) currentVersion=\(currentVersion, privacy: .public)"
        )

        let response: LibrarySyncResponse
        var hasChanges = false

        if shouldForceFullSync {
            response = try await client.getJSON(
                path: APIPath.librarySync,
                queryItems: [URLQueryItem(name: "since_version", value: "0")]
            )
            logger.log(
                "fetched full sync response version=\(response.syncVersion, privacy: .public) albums=\(response.albums.count, privacy: .public) tracks=\(response.tracks.count, privacy: .public)"
            )
            try await indexStore.resetForFullResync()
            try await indexStore.replaceAll(with: response)
            hasChanges = true
        } else {
            response = try await client.getJSON(
                path: APIPath.librarySync,
                queryItems: [URLQueryItem(name: "since_version", value: "\(currentVersion)")]
            )
            hasChanges = !response.albums.isEmpty || !response.tracks.isEmpty || !response.deletedAlbumIDs.isEmpty || !response.deletedTrackIDs.isEmpty
            do {
                try await indexStore.apply(response)
            } catch {
                logger.error("incremental apply failed, fallback to full sync: \(error.localizedDescription, privacy: .public)")
                let fullResponse: LibrarySyncResponse = try await client.getJSON(
                    path: APIPath.librarySync,
                    queryItems: [URLQueryItem(name: "since_version", value: "0")]
                )
                logger.log(
                    "fetched fallback full sync response version=\(fullResponse.syncVersion, privacy: .public) albums=\(fullResponse.albums.count, privacy: .public) tracks=\(fullResponse.tracks.count, privacy: .public)"
                )
                try await indexStore.resetForFullResync()
                try await indexStore.replaceAll(with: fullResponse)
                hasChanges = true
            }
        }

        // 某些本地状态异常时，防止同步成功后游标仍停留在 0，导致后续始终全量拉取。
        let latestVersion = try await indexStore.currentSyncVersion()
        let targetVersion = latestVersion > 0 ? latestVersion : response.syncVersion
        if latestVersion != targetVersion {
            try await indexStore.updateSyncVersion(targetVersion)
        }

        let persistedSchemaVersion = try await indexStore.currentSyncSchemaVersion()
        if persistedSchemaVersion != LibraryIndexStore.syncSchemaVersion {
            try await indexStore.updateSyncSchemaVersion(LibraryIndexStore.syncSchemaVersion)
        }

        let finalVersion = try await indexStore.currentSyncVersion()
        let finalSchemaVersion = try await indexStore.currentSyncSchemaVersion()
        logger.log(
            "finish library sync persistedVersion=\(finalVersion, privacy: .public) persistedSchema=\(finalSchemaVersion, privacy: .public) hasChanges=\(hasChanges, privacy: .public)"
        )

        return LibrarySyncResult(
            hasChanges: hasChanges,
            updatedAlbumsCount: response.albums.count,
            updatedTracksCount: response.tracks.count
        )
    }
}
