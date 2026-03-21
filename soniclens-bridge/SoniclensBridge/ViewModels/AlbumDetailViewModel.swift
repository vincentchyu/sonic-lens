import Foundation

struct AlbumDiscGroup: Identifiable, Equatable {
    let discNumber: Int?
    let tracks: [Track]

    var id: String {
        discNumber.map { "disc-\($0)" } ?? "disc-unknown"
    }

    var title: String {
        if let discNumber {
            return "光盘 \(discNumber)"
        }
        return "未标记光盘"
    }
}

struct AlbumTrackPresentation: Equatable {
    let discGroups: [AlbumDiscGroup]
    let trackCount: Int
    let totalDuration: Int64

    static let empty = AlbumTrackPresentation(discGroups: [], trackCount: 0, totalDuration: 0)

    static func build(from tracks: [Track]) -> AlbumTrackPresentation {
        let grouped = Dictionary(grouping: tracks) { track in
            track.discNumber
        }

        let discGroups = grouped
            .map { discNumber, tracks in
                AlbumDiscGroup(discNumber: discNumber, tracks: tracks.sorted(by: trackOrder))
            }
            .sorted { lhs, rhs in
                switch (lhs.discNumber, rhs.discNumber) {
                case let (l?, r?):
                    return l < r
                case (_?, nil):
                    return true
                case (nil, _?):
                    return false
                case (nil, nil):
                    return false
                }
            }

        let totalDuration = tracks.compactMap(\.duration).reduce(0, +)
        return AlbumTrackPresentation(discGroups: discGroups, trackCount: tracks.count, totalDuration: totalDuration)
    }

    private static func trackOrder(_ lhs: Track, _ rhs: Track) -> Bool {
        let lhsTrack = lhs.trackNumber ?? Int.max
        let rhsTrack = rhs.trackNumber ?? Int.max
        if lhsTrack != rhsTrack {
            return lhsTrack < rhsTrack
        }
        return lhs.id < rhs.id
    }
}

@MainActor
final class AlbumDetailViewModel: ObservableObject {
    @Published var detail: AlbumDetail?
    @Published var resolvedArtworkURL: String?
    @Published var candidates: [ReleaseCandidate] = []
    @Published var albumInsights: [AlbumInsight] = []
    @Published var favoriteTrackIDs: Set<Int64> = []
    @Published var trackPresentation: AlbumTrackPresentation = .empty
    @Published var isLoading: Bool = false
    @Published var isSearchingCandidates: Bool = false
    @Published var errorMessage: String?
    @Published var albumInsightGenerationState: InsightGenerationState = .idle
    @Published var availableAIPlatforms: [AIPlatformOption] = []
    @Published var availableAIModels: [AIModelOption] = []
    @Published var selectedAIPlatform: String = ""
    @Published var selectedAIModel: String = ""
    @Published var isModelPickerPresented: Bool = false
    @Published var generationStatusMessage: String?
    private let artworkResolveService = ArtworkResolveService.shared
    private var artworkRequestAlbumID: Int64 = 0
    private var favoriteObserver: NSObjectProtocol?
    private static var aiPlatformCache: [String: [AIPlatformOption]] = [:]
    private static var aiModelCache: [String: [String: [AIModelOption]]] = [:]
    private let preferredPlatformStoragePrefix = "soniclens.bridge.preferred_album_ai_platform."
    private let preferredModelStoragePrefix = "soniclens.bridge.preferred_album_ai_model."
    private let legacyPreferredModelStoragePrefix = "soniclens.bridge.preferred_album_ai_model_legacy."

    init() {
        favoriteObserver = NotificationCenter.default.addObserver(
            forName: .libraryFavoriteDidChange,
            object: nil,
            queue: .main
        ) { [weak self] notification in
            guard let change = notification.object as? LibraryFavoriteChange else { return }
            Task { @MainActor [weak self] in
                self?.handleFavoriteChange(change)
            }
        }
    }

    deinit {
        if let favoriteObserver {
            NotificationCenter.default.removeObserver(favoriteObserver)
        }
    }

    func load(using server: ServerConfig, albumID: Int64, favoriteKeys: Set<String> = []) async {
        isLoading = true
        errorMessage = nil
        generationStatusMessage = nil
        artworkRequestAlbumID = albumID
        let client = APIClient(baseURL: server.baseURL)
        do {
            let loadedDetail: AlbumDetail = try await client.getJSON(path: "/api/albums/\(albumID)")
            detail = loadedDetail
            favoriteTrackIDs = Self.makeFavoriteTrackIDs(detail: loadedDetail, favoriteKeys: favoriteKeys)
            trackPresentation = AlbumTrackPresentation.build(from: loadedDetail.tracks)
            resolvedArtworkURL = ArtworkURLResolver.resolveArtworkPath(loadedDetail.coverArtURL, artworkBaseURL: server.artworkBaseURL)
            resolveArtworkInBackground(using: server, detail: loadedDetail)
            async let candidateRequest: [ReleaseCandidate] = (try? await client.getJSON(path: "\(APIPath.musicBrainzCandidates)/\(albumID)")) ?? []
            async let insightRequest: [AlbumInsight] = (try? await fetchAlbumInsights(using: client, albumID: albumID)) ?? []
            candidates = await candidateRequest
            albumInsights = await insightRequest
            isLoading = false
        } catch {
            errorMessage = "专辑详情加载失败"
            trackPresentation = .empty
            albumInsights = []
            isLoading = false
        }
    }

    func loadAlbumInsights(using server: ServerConfig, albumID: Int64) async {
        let client = APIClient(baseURL: server.baseURL)
        do {
            albumInsights = try await fetchAlbumInsights(using: client, albumID: albumID)
        } catch {
            albumInsights = []
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
        guard !isSearchingCandidates else { return }
        isSearchingCandidates = true
        defer { isSearchingCandidates = false }
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

    func beginAlbumInsightGeneration(using server: ServerConfig) async {
        guard albumInsightGenerationState != .loadingModels, albumInsightGenerationState != .generating else { return }
        generationStatusMessage = nil
        albumInsightGenerationState = .loadingModels

        do {
            let platforms = try await fetchAIPlatforms(using: server)
            guard !platforms.isEmpty else {
                albumInsightGenerationState = .error
                generationStatusMessage = "当前服务器没有可用平台"
                return
            }

            availableAIPlatforms = platforms
            let preferredPlatform = resolvePreferredPlatform(using: server, platforms: platforms)
            try await selectAIPlatform(preferredPlatform, using: server)
            guard !availableAIModels.isEmpty else {
                albumInsightGenerationState = .error
                generationStatusMessage = "当前平台没有可用模型"
                return
            }
            isModelPickerPresented = true
            albumInsightGenerationState = .selectingModel
        } catch {
            albumInsightGenerationState = .error
            generationStatusMessage = "加载模型列表失败"
        }
    }

    func dismissModelPicker() {
        isModelPickerPresented = false
        if albumInsightGenerationState == .selectingModel {
            albumInsightGenerationState = .idle
        }
    }

    func selectAIPlatform(_ platformID: String, using server: ServerConfig) async throws {
        let normalized = platformID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty else { return }

        let models = try await fetchAIModels(using: server, platformID: normalized)
        selectedAIPlatform = normalized
        availableAIModels = models
        selectedAIModel = resolvePreferredModel(using: server, platformID: normalized, models: models)
    }

    func confirmAlbumInsightGeneration(using server: ServerConfig, albumID: Int64) async {
        guard !selectedAIPlatform.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            albumInsightGenerationState = .error
            generationStatusMessage = "请选择平台后再生成"
            return
        }
        guard !selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            albumInsightGenerationState = .error
            generationStatusMessage = "请选择模型后再生成"
            return
        }
        guard albumInsightGenerationState != .generating else { return }

        generationStatusMessage = nil
        isModelPickerPresented = false
        albumInsightGenerationState = .generating

        let client = APIClient(baseURL: server.baseURL)
        do {
            let response: AlbumInsightGenerateResponse = try await client.postJSON(
                path: APIPath.albumInsight,
                body: AlbumInsightGenerateRequest(albumID: albumID, provider: selectedAIPlatform, model: selectedAIModel),
                timeout: 20 * 60
            )

            if !response.insights.isEmpty {
                albumInsights = response.insights
            } else {
                albumInsights = try await fetchAlbumInsights(using: client, albumID: albumID)
            }
            savePreferredSelections(platformID: selectedAIPlatform, modelID: selectedAIModel, for: server)
            albumInsightGenerationState = .success
            generationStatusMessage = response.cached == true ? "已返回缓存解析结果" : "专辑音眸已完成"
        } catch {
            albumInsightGenerationState = .error
            generationStatusMessage = "专辑音眸解析失败，请稍后重试"
        }
    }

    private func resolveArtworkInBackground(using server: ServerConfig, detail: AlbumDetail) {
        guard resolvedArtworkURL == nil else { return }
        let requestedAlbumID = detail.id
        Task { [weak self] in
            guard let self else { return }
            let resolved = await artworkResolveService.resolveArtworkURL(
                using: server,
                albumID: detail.id,
                albumArtist: detail.artist,
                artist: detail.artist,
                album: detail.name,
                artworkKey: detail.coverArtObjectKey
            )
            guard self.artworkRequestAlbumID == requestedAlbumID else { return }
            self.resolvedArtworkURL = resolved
        }
    }

    private func handleFavoriteChange(_ change: LibraryFavoriteChange) {
        guard let detail else { return }
        guard detail.artist == change.artist, detail.name == change.album else { return }

        let isFavorited = change.appleMusic || change.lastfm
        let matchingIDs = detail.tracks.compactMap { track -> Int64? in
            guard track.track == change.track,
                  track.trackNumber == change.trackNumber,
                  track.discNumber == change.discNumber else {
                return nil
            }
            return track.id
        }

        guard !matchingIDs.isEmpty else { return }
        if isFavorited {
            favoriteTrackIDs.formUnion(matchingIDs)
        } else {
            favoriteTrackIDs.subtract(matchingIDs)
        }
    }

    private static func makeFavoriteTrackIDs(detail: AlbumDetail, favoriteKeys: Set<String>) -> Set<Int64> {
        Set(detail.tracks.compactMap { track in
            if track.isFavorited {
                return track.id
            }
            let key = [track.artist, detail.name, track.track, String(track.trackNumber ?? 0), String(track.discNumber ?? 0)]
                .joined(separator: "•")
            return favoriteKeys.contains(key) ? track.id : nil
        })
    }

    private func fetchAlbumInsights(using client: APIClient, albumID: Int64) async throws -> [AlbumInsight] {
        let response: AlbumInsightResponse = try await client.getJSON(
            path: APIPath.albumInsight,
            queryItems: [URLQueryItem(name: "albumID", value: String(albumID))]
        )
        return response.insights
    }

    private func fetchAIPlatforms(using server: ServerConfig) async throws -> [AIPlatformOption] {
        let cacheKey = server.baseURL.absoluteString
        if let cached = Self.aiPlatformCache[cacheKey], !cached.isEmpty {
            return cached
        }

        let client = APIClient(baseURL: server.baseURL)
        let response: AIPlatformListResponse = try await client.getJSON(path: APIPath.aiModels)
        let platforms = response.platforms
        Self.aiPlatformCache[cacheKey] = platforms
        return platforms
    }

    private func fetchAIModels(using server: ServerConfig, platformID: String) async throws -> [AIModelOption] {
        let cacheKey = server.baseURL.absoluteString
        if let cached = Self.aiModelCache[cacheKey]?[platformID], !cached.isEmpty {
            return cached
        }

        let client = APIClient(baseURL: server.baseURL)
        let response: AIModelListResponse = try await client.getJSON(path: APIPath.aiPlatformModels(platformID: platformID))
        let models = response.models
        var scopedCache = Self.aiModelCache[cacheKey] ?? [:]
        scopedCache[platformID] = models
        Self.aiModelCache[cacheKey] = scopedCache
        return models
    }

    private func preferredPlatformStorageKey(for server: ServerConfig) -> String {
        preferredPlatformStoragePrefix + server.baseURL.absoluteString
    }

    private func preferredModelStorageKey(for server: ServerConfig) -> String {
        preferredModelStoragePrefix + server.baseURL.absoluteString
    }

    private func legacyPreferredModelStorageKey(for server: ServerConfig) -> String {
        legacyPreferredModelStoragePrefix + server.baseURL.absoluteString
    }

    private func savePreferredSelections(platformID: String, modelID: String, for server: ServerConfig) {
        UserDefaults.standard.set(platformID, forKey: preferredPlatformStorageKey(for: server))
        UserDefaults.standard.set(modelID, forKey: preferredModelStorageKey(for: server))
    }

    private func resolvePreferredPlatform(using server: ServerConfig, platforms: [AIPlatformOption]) -> String {
        let stored = UserDefaults.standard.string(forKey: preferredPlatformStorageKey(for: server))?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if platforms.contains(where: { $0.id == stored }) {
            return stored
        }

        let legacy = UserDefaults.standard.string(forKey: preferredModelStorageKey(for: server))?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if platforms.contains(where: { $0.id == legacy }) {
            UserDefaults.standard.set(legacy, forKey: preferredPlatformStorageKey(for: server))
            UserDefaults.standard.set("", forKey: preferredModelStorageKey(for: server))
            UserDefaults.standard.set(legacy, forKey: legacyPreferredModelStorageKey(for: server))
            return legacy
        }

        return platforms.first?.id ?? ""
    }

    private func resolvePreferredModel(using server: ServerConfig, platformID: String, models: [AIModelOption]) -> String {
        let stored = UserDefaults.standard.string(forKey: preferredModelStorageKey(for: server))?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if models.contains(where: { $0.id == stored }) {
            return stored
        }

        if let platform = availableAIPlatforms.first(where: { $0.id == platformID }),
           let defaultModel = platform.defaultModel,
           models.contains(where: { $0.id == defaultModel }) {
            return defaultModel
        }
        if let defaultOption = models.first(where: \.isDefault) {
            return defaultOption.id
        }
        return models.first?.id ?? ""
    }
}
