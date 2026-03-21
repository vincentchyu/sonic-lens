import Foundation

@MainActor
final class AlbumDetailViewModel: ObservableObject {
    @Published var detail: AlbumDetail?
    @Published var candidates: [ReleaseCandidate] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    func load(using server: ServerConfig, albumID: Int64) async {
        isLoading = true
        errorMessage = nil
        let client = APIClient(baseURL: server.baseURL)
        do {
            detail = try await client.getJSON(path: "/api/albums/\(albumID)")
            candidates = (try? await client.getJSON(path: "\(APIPath.musicBrainzCandidates)/\(albumID)")) ?? []
            isLoading = false
        } catch {
            errorMessage = "专辑详情加载失败"
            isLoading = false
        }
    }

    func refreshCandidates(using server: ServerConfig, albumID: Int64) async {
        let client = APIClient(baseURL: server.baseURL)
        do {
            candidates = try await client.getJSON(path: "\(APIPath.musicBrainzCandidates)/\(albumID)")
        } catch {
            // ignore candidate errors
        }
    }

    func searchCandidates(using server: ServerConfig, albumID: Int64) async {
        let client = APIClient(baseURL: server.baseURL)
        do {
            struct Ok: Decodable { let status: String }
            _ = try await client.getJSON(path: "\(APIPath.musicBrainzSearchReleases)/\(albumID)") as Ok
            await refreshCandidates(using: server, albumID: albumID)
        } catch {
            // ignore search errors
        }
    }

    func confirmSelection(using server: ServerConfig, albumID: Int64, candidate: ReleaseCandidate) async {
        let client = APIClient(baseURL: server.baseURL)
        do {
            struct Ok: Decodable { let status: String }
            let req = LinkAlbumRequest(albumID: albumID, releaseMBID: candidate.id, mbid: candidate.mbid)
            _ = try await client.postJSON(path: APIPath.musicBrainzLinkAlbum, body: req) as Ok
            await load(using: server, albumID: albumID)
        } catch {
            // ignore link errors
        }
    }
}
