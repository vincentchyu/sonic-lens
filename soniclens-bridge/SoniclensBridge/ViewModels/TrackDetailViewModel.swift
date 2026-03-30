import Foundation
import OSLog

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
    @Published var resolvedArtworkResource: ResolvedArtworkResource?
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
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "TrackDetailViewModel")
    private let preferredPlatformStoragePrefix = "soniclens.bridge.preferred_ai_platform."
    private let preferredModelStoragePrefix = "soniclens.bridge.preferred_ai_model."
    private let legacyPreferredModelStoragePrefix = "soniclens.bridge.preferred_ai_model_legacy."
    private var lastHandledJobPhaseKey: String?

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
            applyResolvedArtwork(nil)
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
        coordinator: InsightAnalysisCoordinator,
        track: Track
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

        do {
            logger.info("确认曲目音眸生成 曲目=\(track.track, privacy: .public) 封面=\(self.describeArtworkURL(self.resolvedArtworkURL), privacy: .public)")
            let job = try await coordinator.startTrackInsightJob(
                using: server,
                track: track,
                artworkResource: resolvedArtworkResource,
                provider: selectedAIPlatform,
                model: selectedAIModel
            )
            savePreferredSelections(platformID: selectedAIPlatform, modelID: selectedAIModel, for: server)
            await syncInsightJob(job, using: server, track: track, forceRefresh: false)
        } catch {
            insightGenerationState = .error
            generationStatusMessage = "音眸任务启动失败，请稍后重试"
        }
    }

    func syncInsightJob(
        _ job: InsightAnalysisJob?,
        using server: ServerConfig,
        track: Track,
        forceRefresh: Bool = false
    ) async {
        guard let job, job.matches(track: track) else {
            if insightGenerationState == .generating {
                insightGenerationState = .idle
            }
            return
        }

        let phaseKey = "\(job.id)::\(job.phase.rawValue)"
        switch job.phase {
        case .queued, .running:
            insightGenerationState = .generating
            generationStatusMessage = "音眸分析已进入后台任务，切到桌面后可在灵动岛查看进度。"
        case .completed:
            insightGenerationState = .success
            generationStatusMessage = job.resultAvailable ? "音眸解析已完成" : "音眸任务已完成，当前暂无可展示内容"
            guard forceRefresh || lastHandledJobPhaseKey != phaseKey else { return }
            do {
                let client = APIClient(baseURL: server.baseURL)
                if let resultInsightID = job.resultInsightID {
                    let detail = try await fetchTrackInsightDetail(using: client, id: resultInsightID)
                    insights = [detail]
                } else {
                    insights = try await fetchTrackInsights(
                        using: client,
                        artist: track.artist,
                        album: track.album,
                        track: track.track,
                        trackNumber: track.trackNumber,
                        discNumber: track.discNumber
                    )
                }
                lastHandledJobPhaseKey = phaseKey
            } catch {
                generationStatusMessage = "音眸任务已完成，但刷新内容失败"
            }
        case .failed, .canceled:
            insightGenerationState = .error
            generationStatusMessage = job.errorMessage?.isEmpty == false ? job.errorMessage : "音眸任务未完成"
            lastHandledJobPhaseKey = phaseKey
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
            let resolved = await artworkResolveService.resolveArtworkResource(
                using: server,
                albumID: nil,
                albumArtist: artist,
                artist: artist,
                album: album,
                artworkKey: nil
            )
            guard self.artworkRequestKey == requestKey else { return }
            self.applyResolvedArtwork(resolved)
            self.logger.debug("解析曲目封面完成 艺人=\(artist, privacy: .public) 专辑=\(album, privacy: .public) 封面=\(self.describeArtworkURL(resolved?.remoteURL), privacy: .public)")
        }
    }

    private func applyResolvedArtwork(_ resource: ResolvedArtworkResource?) {
        let normalized = resource?.isEmpty == true ? nil : resource
        resolvedArtworkResource = normalized
        resolvedArtworkURL = normalized?.remoteURL
        #if os(iOS)
        Task {
            _ = await LiveActivityArtworkStore.shared.prefetch(resource: normalized)
        }
        #endif
    }

    private func describeArtworkURL(_ artworkURL: String?) -> String {
        guard let artworkURL else { return "空" }
        let trimmed = artworkURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "空字符串" }
        guard let url = URL(string: trimmed) else { return "非法地址" }
        let host = url.host ?? "无主机"
        let file = url.lastPathComponent.isEmpty ? "无文件名" : url.lastPathComponent
        return "\(url.scheme ?? "无协议")://\(host)/\(file)"
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

    private func fetchTrackInsightDetail(using client: APIClient, id: Int64) async throws -> Insight {
        try await client.getJSON(
            path: APIPath.insightDetail(id: id),
            queryItems: [
                URLQueryItem(name: "analysis_target_type", value: InsightTargetType.track.rawValue)
            ]
        )
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
