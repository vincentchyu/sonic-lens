import Foundation
import OSLog

@MainActor
final class InsightAnalysisCoordinator: ObservableObject {
    @Published private(set) var activeJob: InsightAnalysisJob?
    @Published private(set) var lastInsightJobEventAt: Date?
    @Published private(set) var pendingRoute: InsightAnalysisRouteSnapshot?

    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "InsightAnalysisCoordinator")
    private let persistenceKey = "soniclens.bridge.insight_analysis.state"
    private let staleInterval: TimeInterval = 12
    private var currentRoute: InsightAnalysisRouteSnapshot?
    #if os(iOS)
    private let liveActivityManager = InsightLiveActivityManager()
    #endif

    init() {
        restorePersistedState()
        #if os(iOS)
        liveActivityManager.onPushToken = { [weak self] jobID, token, server in
            Task { @MainActor [weak self] in
                await self?.registerLiveActivityToken(jobID: jobID, token: token, using: server)
            }
        }
        #endif
    }

    func restorePersistedState() {
        guard let data = UserDefaults.standard.data(forKey: persistenceKey) else { return }
        do {
            let decoded = try JSONDecoder().decode(PersistedInsightAnalysisState.self, from: data)
            activeJob = decoded.job
            currentRoute = decoded.route
            if let timestamp = decoded.lastEventTimestamp {
                lastInsightJobEventAt = Date(timeIntervalSince1970: timestamp)
            }
        } catch {
            logger.error("恢复音眸任务持久化状态失败 error=\(error.localizedDescription, privacy: .public)")
        }
    }

    func startTrackInsightJob(
        using server: ServerConfig,
        track: Track,
        artworkResource: ResolvedArtworkResource?,
        provider: String,
        model: String
    ) async throws -> InsightAnalysisJob {
        let request = InsightJobCreateRequest(
            targetType: .track,
            artist: track.artist,
            album: track.album,
            track: track.track,
            trackNumber: track.trackNumber,
            discNumber: track.discNumber,
            albumID: nil,
            provider: provider,
            model: model,
            clientPlatform: "iphone"
        )
        let response: InsightJobResponse = try await APIClient(baseURL: server.baseURL).postJSON(path: APIPath.insightJobs, body: request)
        let route = InsightAnalysisRouteSnapshot.track(jobID: response.job.id, track: track, artworkDescriptor: artworkResource)
        logger.info(
            "启动曲目音眸任务 id=\(response.job.id, privacy: .public) 曲目=\(track.track, privacy: .public) 封面=\(self.describeArtworkURL(artworkResource?.remoteURL), privacy: .public)"
        )
        apply(job: response.job, route: route, eventDate: Date(), shouldQueueRoute: false)
        #if os(iOS)
        await liveActivityManager.startOrUpdate(for: response.job, route: route, using: server)
        #endif
        return response.job
    }

    func startAlbumInsightJob(
        using server: ServerConfig,
        albumID: Int64,
        artist: String,
        album: String,
        artworkResource: ResolvedArtworkResource?,
        provider: String,
        model: String
    ) async throws -> InsightAnalysisJob {
        let request = InsightJobCreateRequest(
            targetType: .album,
            artist: artist,
            album: album,
            track: nil,
            trackNumber: nil,
            discNumber: nil,
            albumID: albumID,
            provider: provider,
            model: model,
            clientPlatform: "iphone"
        )
        let response: InsightJobResponse = try await APIClient(baseURL: server.baseURL).postJSON(path: APIPath.insightJobs, body: request)
        let route = InsightAnalysisRouteSnapshot.album(jobID: response.job.id, albumID: albumID, artworkDescriptor: artworkResource)
        logger.info(
            "启动专辑音眸任务 id=\(response.job.id, privacy: .public) 专辑=\(album, privacy: .public) 封面=\(self.describeArtworkURL(artworkResource?.remoteURL), privacy: .public)"
        )
        apply(job: response.job, route: route, eventDate: Date(), shouldQueueRoute: false)
        #if os(iOS)
        await liveActivityManager.startOrUpdate(for: response.job, route: route, using: server)
        #endif
        return response.job
    }

    func handleWebSocketJobUpdate(_ job: InsightAnalysisJob, using server: ServerConfig?) {
        guard activeJob?.id == nil || activeJob?.id == job.id else { return }
        let route = resolvedRoute(for: job)
        apply(job: job, route: route, eventDate: Date(), shouldQueueRoute: false)
        #if os(iOS)
        Task {
            await liveActivityManager.startOrUpdate(for: job, route: route, using: server)
        }
        #endif
    }

    func reconcileIfNeeded(using server: ServerConfig, force: Bool = false) async {
        guard let activeJob, !activeJob.phase.isTerminal else { return }
        if !force, let lastInsightJobEventAt, Date().timeIntervalSince(lastInsightJobEventAt) < staleInterval {
            return
        }

        do {
            let response: InsightJobResponse = try await APIClient(baseURL: server.baseURL).getJSON(path: APIPath.insightJob(id: activeJob.id))
            let route = resolvedRoute(for: response.job)
            apply(job: response.job, route: route, eventDate: Date(), shouldQueueRoute: false)
            #if os(iOS)
            await liveActivityManager.startOrUpdate(for: response.job, route: route, using: server)
            #endif
        } catch {
            logger.error("对账音眸任务状态失败 id=\(activeJob.id, privacy: .public) error=\(error.localizedDescription, privacy: .public)")
        }
    }

    func handleDeepLink(_ url: URL, using server: ServerConfig?) async {
        guard url.scheme == "soniclens" else { return }
        let parts = url.pathComponents.filter { $0 != "/" }
        guard url.host == "insight-job", let jobID = parts.first, !jobID.isEmpty else { return }

        if let activeJob, activeJob.id == jobID {
            pendingRoute = resolvedRoute(for: activeJob)
            await dismissTerminalLiveActivityIfNeeded(for: activeJob, route: pendingRoute)
            return
        }

        if let data = UserDefaults.standard.data(forKey: persistenceKey),
            let persisted = try? JSONDecoder().decode(PersistedInsightAnalysisState.self, from: data),
           persisted.job.id == jobID {
            apply(job: persisted.job, route: persisted.route, eventDate: lastInsightJobEventAt, shouldQueueRoute: true)
            await dismissTerminalLiveActivityIfNeeded(for: persisted.job, route: persisted.route)
            return
        }

        guard let server else { return }
        do {
            let response: InsightJobResponse = try await APIClient(baseURL: server.baseURL).getJSON(path: APIPath.insightJob(id: jobID))
            let route = resolvedRoute(for: response.job)
            apply(job: response.job, route: route, eventDate: Date(), shouldQueueRoute: true)
            await dismissTerminalLiveActivityIfNeeded(for: response.job, route: route)
        } catch {
            logger.error("处理音眸深链失败 id=\(jobID, privacy: .public) error=\(error.localizedDescription, privacy: .public)")
        }
    }

    func consumePendingRoute() {
        pendingRoute = nil
        persist()
    }

    private func apply(
        job: InsightAnalysisJob,
        route: InsightAnalysisRouteSnapshot?,
        eventDate: Date?,
        shouldQueueRoute: Bool
    ) {
        activeJob = job
        if let eventDate {
            lastInsightJobEventAt = eventDate
        }
        currentRoute = route ?? currentRoute
        pendingRoute = shouldQueueRoute ? (route ?? currentRoute) : nil
        logger.debug(
            "更新音眸任务状态 id=\(job.id, privacy: .public) phase=\(job.phase.rawValue, privacy: .public) 封面=\(self.describeArtworkURL((route ?? self.currentRoute)?.artworkDescriptor?.remoteURL), privacy: .public)"
        )
        persist(routeOverride: route)
    }

    private func registerLiveActivityToken(jobID: String, token: String, using server: ServerConfig) async {
        do {
            let _: InsightJobResponse = try await APIClient(baseURL: server.baseURL).postJSON(
                path: APIPath.insightJobLiveActivityToken(id: jobID),
                body: InsightJobLiveActivityTokenRequest(token: token)
            )
        } catch {
            logger.error("上报 Live Activity token 失败 id=\(jobID, privacy: .public) error=\(error.localizedDescription, privacy: .public)")
        }
    }

    private func dismissTerminalLiveActivityIfNeeded(
        for job: InsightAnalysisJob,
        route: InsightAnalysisRouteSnapshot?
    ) async {
        #if os(iOS)
        await liveActivityManager.dismissCurrentActivityIfNeeded(for: job, route: route)
        #endif
    }

    private func persist(routeOverride: InsightAnalysisRouteSnapshot? = nil) {
        guard let activeJob else {
            UserDefaults.standard.removeObject(forKey: persistenceKey)
            return
        }

        let payload = PersistedInsightAnalysisState(
            job: activeJob,
            route: routeOverride ?? currentRoute,
            lastEventTimestamp: lastInsightJobEventAt?.timeIntervalSince1970
        )
        do {
            let data = try JSONEncoder().encode(payload)
            UserDefaults.standard.set(data, forKey: persistenceKey)
        } catch {
            logger.error("持久化音眸任务状态失败 error=\(error.localizedDescription, privacy: .public)")
        }
    }

    private func resolvedRoute(for job: InsightAnalysisJob) -> InsightAnalysisRouteSnapshot? {
        let fallback = InsightAnalysisRouteSnapshot.fallback(from: job)
        let candidates = [pendingRoute, currentRoute, fallback]
            .compactMap { $0 }
            .filter { $0.jobID == job.id }

        guard let base = candidates.first ?? fallback else { return nil }
        let artworkDescriptor = candidates.lazy
            .compactMap(\.artworkDescriptor)
            .first { !$0.isEmpty }

        switch base.targetType {
        case .track:
            guard let track = base.track ?? fallback?.track else { return nil }
            return .track(jobID: job.id, track: track, artworkDescriptor: artworkDescriptor)
        case .album:
            guard let albumID = base.albumID ?? fallback?.albumID else { return nil }
            return .album(jobID: job.id, albumID: albumID, artworkDescriptor: artworkDescriptor)
        }
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
}
