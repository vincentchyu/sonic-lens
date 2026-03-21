import Foundation

enum InsightGenerationState: Equatable {
    case idle
    case loadingModels
    case selectingModel
    case generating
    case success
    case error
}

@MainActor
final class TrackDetailViewModel: ObservableObject {
    @Published var resolvedArtworkURL: String?
    @Published var lyrics: TrackLyricsResponse?
    @Published var lyricLines: [LyricLine] = []
    @Published var insights: [Insight] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?
    @Published var insightGenerationState: InsightGenerationState = .idle
    @Published var availableAIPlatforms: [AIPlatformOption] = []
    @Published var availableAIModels: [AIModelOption] = []
    @Published var selectedAIPlatform: String = ""
    @Published var selectedAIModel: String = ""
    @Published var isModelPickerPresented: Bool = false
    @Published var generationStatusMessage: String?

    private let artworkResolveService = ArtworkResolveService.shared
    private var artworkRequestKey: String = ""
    private static var aiPlatformCache: [String: [AIPlatformOption]] = [:]
    private static var aiModelCache: [String: [String: [AIModelOption]]] = [:]
    private let preferredPlatformStoragePrefix = "soniclens.bridge.preferred_ai_platform."
    private let preferredModelStoragePrefix = "soniclens.bridge.preferred_ai_model."
    private let legacyPreferredModelStoragePrefix = "soniclens.bridge.preferred_ai_model_legacy."

    func load(
        using server: ServerConfig,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) async {
        isLoading = true
        errorMessage = nil
        let client = APIClient(baseURL: server.baseURL)
        let requestKey = [artist, album ?? "", track, String(trackNumber ?? 0), String(discNumber ?? 0)].joined(separator: "•")
        artworkRequestKey = requestKey
        do {
            resolvedArtworkURL = nil
            resolveArtworkInBackground(
                using: server,
                requestKey: requestKey,
                artist: artist,
                album: album
            )

            lyrics = try await client.getJSON(
                path: APIPath.trackLyrics,
                queryItems: [
                    URLQueryItem(name: "artist", value: artist),
                    URLQueryItem(name: "album", value: album ?? ""),
                    URLQueryItem(name: "track", value: track),
                    URLQueryItem(name: "trackNumber", value: trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: discNumber.map(String.init))
                ]
            )
            lyricLines = LRCParser.parseLyrics(lyrics?.lyrics ?? "", hasLRC: lyrics?.hasLRC ?? false)
            insights = try await fetchTrackInsights(
                using: client,
                artist: artist,
                album: album,
                track: track,
                trackNumber: trackNumber,
                discNumber: discNumber
            )
            isLoading = false
        } catch {
            errorMessage = "曲目详情加载失败"
            isLoading = false
        }
    }

    func beginInsightGeneration(
        using server: ServerConfig,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) async {
        guard insightGenerationState != .loadingModels, insightGenerationState != .generating else { return }
        generationStatusMessage = nil
        insightGenerationState = .loadingModels

        do {
            let platforms = try await fetchAIPlatforms(using: server)
            guard !platforms.isEmpty else {
                insightGenerationState = .error
                generationStatusMessage = "当前服务器没有可用平台"
                return
            }

            availableAIPlatforms = platforms
            let preferredPlatform = resolvePreferredPlatform(using: server, platforms: platforms)
            try await selectAIPlatform(preferredPlatform, using: server)
            guard !availableAIModels.isEmpty else {
                insightGenerationState = .error
                generationStatusMessage = "当前平台没有可用模型"
                return
            }
            isModelPickerPresented = true
            insightGenerationState = .selectingModel
        } catch {
            insightGenerationState = .error
            generationStatusMessage = "加载模型列表失败"
        }
    }

    func dismissModelPicker() {
        isModelPickerPresented = false
        if insightGenerationState == .selectingModel {
            insightGenerationState = .idle
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

    func confirmInsightGeneration(
        using server: ServerConfig,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) async {
        guard !selectedAIPlatform.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            insightGenerationState = .error
            generationStatusMessage = "请选择平台后再生成"
            return
        }
        guard !selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            insightGenerationState = .error
            generationStatusMessage = "请选择模型后再生成"
            return
        }
        guard insightGenerationState != .generating else { return }

        generationStatusMessage = nil
        isModelPickerPresented = false
        insightGenerationState = .generating

        let client = APIClient(baseURL: server.baseURL)
        do {
            let response: TrackInsightGenerateResponse = try await client.postJSON(
                path: APIPath.trackInsight,
                body: TrackInsightGenerateRequest(
                    artist: artist,
                    album: album ?? "",
                    track: track,
                    trackNumber: trackNumber,
                    discNumber: discNumber,
                    provider: selectedAIPlatform,
                    model: selectedAIModel
                ),
                timeout: 20 * 60
            )

            if !response.insights.isEmpty {
                insights = response.insights
            } else {
                insights = try await fetchTrackInsights(
                    using: client,
                    artist: artist,
                    album: album,
                    track: track,
                    trackNumber: trackNumber,
                    discNumber: discNumber
                )
            }
            savePreferredSelections(platformID: selectedAIPlatform, modelID: selectedAIModel, for: server)
            insightGenerationState = .success
            generationStatusMessage = response.cached == true ? "已返回缓存解析结果" : "音眸解析已完成"
        } catch {
            insightGenerationState = .error
            generationStatusMessage = "音眸解析失败，请稍后重试"
        }
    }

    func currentLineID(forPreviewTime previewTime: TimeInterval) -> UUID? {
        guard lyricLines.contains(where: { $0.time != nil }) else { return nil }
        var current: LyricLine?
        for line in lyricLines {
            guard let time = line.time else { continue }
            if time <= previewTime {
                current = line
            } else {
                break
            }
        }
        return current?.id
    }

    private func resolveArtworkInBackground(
        using server: ServerConfig,
        requestKey: String,
        artist: String,
        album: String?
    ) {
        guard let album, !album.isEmpty else { return }

        Task { [weak self] in
            guard let self else { return }
            let resolved = await artworkResolveService.resolveArtworkURL(
                using: server,
                albumID: nil,
                albumArtist: artist,
                artist: artist,
                album: album,
                artworkKey: nil
            )
            guard self.artworkRequestKey == requestKey else { return }
            self.resolvedArtworkURL = resolved
        }
    }

    private func fetchTrackInsights(
        using client: APIClient,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int?,
        discNumber: Int?
    ) async throws -> [Insight] {
        let response: TrackInsightResponse = try await client.getJSON(
            path: APIPath.trackInsight,
            queryItems: [
                URLQueryItem(name: "artist", value: artist),
                URLQueryItem(name: "album", value: album ?? ""),
                URLQueryItem(name: "track", value: track),
                URLQueryItem(name: "trackNumber", value: trackNumber.map(String.init)),
                URLQueryItem(name: "discNumber", value: discNumber.map(String.init))
            ]
        )
        return response.insights
    }

    private func fetchAIPlatforms(using server: ServerConfig) async throws -> [AIPlatformOption] {
        let cacheKey = serverCacheKey(server)
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
        let cacheKey = serverCacheKey(server)
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

    private func serverCacheKey(_ server: ServerConfig) -> String {
        server.baseURL.absoluteString
    }

    private func preferredPlatformStorageKey(for server: ServerConfig) -> String {
        preferredPlatformStoragePrefix + serverCacheKey(server)
    }

    private func preferredModelStorageKey(for server: ServerConfig) -> String {
        preferredModelStoragePrefix + serverCacheKey(server)
    }

    private func legacyPreferredModelStorageKey(for server: ServerConfig) -> String {
        legacyPreferredModelStoragePrefix + serverCacheKey(server)
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
