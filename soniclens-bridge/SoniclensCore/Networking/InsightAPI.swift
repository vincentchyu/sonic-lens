import Foundation

enum InsightAPIClient {
    static func fetchTrackInsightDetail(using server: ServerConfig, id: Int64) async throws -> Insight {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.insightDetail(id: id),
            queryItems: [
                URLQueryItem(name: "analysis_target_type", value: InsightTargetType.track.rawValue)
            ]
        )
    }

    static func fetchAlbumInsightDetail(using server: ServerConfig, id: Int64) async throws -> AlbumInsight {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.insightDetail(id: id),
            queryItems: [
                URLQueryItem(name: "analysis_target_type", value: InsightTargetType.album.rawValue)
            ]
        )
    }

    static func fetchTrackInsightHistory(using server: ServerConfig, id: Int64, limit: Int = 20) async throws -> PaginatedInsights {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.insightHistory(id: id),
            queryItems: [
                URLQueryItem(name: "analysis_target_type", value: InsightTargetType.track.rawValue),
                URLQueryItem(name: "limit", value: String(limit))
            ]
        )
    }

    static func fetchAlbumInsightHistory(using server: ServerConfig, id: Int64, limit: Int = 20) async throws -> PaginatedInsights {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.insightHistory(id: id),
            queryItems: [
                URLQueryItem(name: "analysis_target_type", value: InsightTargetType.album.rawValue),
                URLQueryItem(name: "limit", value: String(limit))
            ]
        )
    }
}
