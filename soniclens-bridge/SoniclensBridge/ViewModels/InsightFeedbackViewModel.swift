import Foundation
import OSLog

@MainActor
final class InsightFeedbackViewModel: ObservableObject {
    @Published var summary: InsightFeedbackSummary?
    @Published var history: [InsightFeedbackRecord] = []
    @Published var isLoading: Bool = false
    @Published var isSubmitting: Bool = false
    @Published var errorMessage: String?

    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "InsightFeedbackViewModel")
    private var loadToken = UUID()
    private var cache: [CacheKey: CachedFeedback] = [:]
    private var loadTask: Task<CachedFeedback, Error>?

    func reset() {
        cancelLoad()
        loadToken = UUID()
        summary = nil
        history = []
        isLoading = false
        isSubmitting = false
        errorMessage = nil
    }

    func load(using server: ServerConfig, insightID: Int64, targetType: InsightTargetType, limit: Int = 10) async {
        let cacheKey = CacheKey(
            serverID: server.id,
            insightID: insightID,
            targetType: targetType,
            limit: limit
        )
        if let cached = cache[cacheKey] {
            cancelLoad()
            loadToken = UUID()
            summary = cached.summary
            history = cached.history
            isLoading = false
            errorMessage = nil
            return
        }

        cancelLoad()
        let currentLoadToken = UUID()
        loadToken = currentLoadToken
        isLoading = true
        errorMessage = nil
        let client = APIClient(baseURL: server.baseURL)
        let task = Task<CachedFeedback, Error> {
            async let summaryRequest: InsightFeedbackSummary = client.getJSON(
                path: APIPath.insightFeedbackSummary(id: insightID),
                queryItems: [URLQueryItem(name: "analysis_target_type", value: targetType.rawValue)]
            )
            async let historyRequest: InsightFeedbackHistoryResponse = client.getJSON(
                path: APIPath.insightFeedbackHistory(id: insightID),
                queryItems: [
                    URLQueryItem(name: "analysis_target_type", value: targetType.rawValue),
                    URLQueryItem(name: "limit", value: String(limit))
                ]
            )

            let (loadedSummary, loadedHistory) = try await (summaryRequest, historyRequest)
            try Task.checkCancellation()
            return CachedFeedback(summary: loadedSummary, history: loadedHistory.feedbacks ?? [])
        }
        loadTask = task

        do {
            let loadedFeedback = try await task.value
            guard loadToken == currentLoadToken else { return }
            summary = loadedFeedback.summary
            history = loadedFeedback.history
            cache[cacheKey] = loadedFeedback
        } catch is CancellationError {
            guard loadToken == currentLoadToken else { return }
        } catch {
            guard loadToken == currentLoadToken else { return }
            logger.error("加载音眸反馈失败 \(error.localizedDescription, privacy: .public)")
            errorMessage = "反馈加载失败"
        }

        if loadToken == currentLoadToken {
            loadTask = nil
            isLoading = false
        }
    }

    func submitHelpful(using server: ServerConfig, insightID: Int64, targetType: InsightTargetType) async {
        await submit(
            using: server,
            insightID: insightID,
            targetType: targetType,
            request: .helpful()
        )
    }

    func submitIssue(
        using server: ServerConfig,
        insightID: Int64,
        targetType: InsightTargetType,
        draft: InsightIssueDraft
    ) async {
        let request = InsightFeedbackSubmitRequest(
            score: -1,
            comment: draft.normalizedComment,
            reasonCodes: draft.reasonCodes,
            sectionKey: draft.sectionKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : draft.sectionKey.trimmingCharacters(in: .whitespacesAndNewlines),
            sourcePlatform: InsightFeedbackPlatform.current
        )
        await submit(using: server, insightID: insightID, targetType: targetType, request: request)
    }

    private func submit(
        using server: ServerConfig,
        insightID: Int64,
        targetType: InsightTargetType,
        request: InsightFeedbackSubmitRequest
    ) async {
        cancelLoad()
        loadToken = UUID()
        isSubmitting = true
        errorMessage = nil
        let client = APIClient(baseURL: server.baseURL)
        struct Ok: Decodable { let status: String }

        do {
            let path = targetType == .album ? APIPath.albumInsightFeedback(id: insightID) : APIPath.trackInsightFeedback(id: insightID)
            _ = try await client.postJSON(path: path, body: request) as Ok
            invalidateCache(for: server.id, insightID: insightID, targetType: targetType)
            await load(using: server, insightID: insightID, targetType: targetType)
        } catch {
            logger.error("提交音眸反馈失败 \(error.localizedDescription, privacy: .public)")
            errorMessage = "反馈提交失败"
        }

        isSubmitting = false
    }

    private func invalidateCache(for serverID: UUID, insightID: Int64, targetType: InsightTargetType) {
        cache = cache.filter { key, _ in
            !(key.serverID == serverID && key.insightID == insightID && key.targetType == targetType)
        }
    }

    private func cancelLoad() {
        loadTask?.cancel()
        loadTask = nil
    }
}

private struct CacheKey: Hashable {
    let serverID: UUID
    let insightID: Int64
    let targetType: InsightTargetType
    let limit: Int
}

private struct CachedFeedback {
    let summary: InsightFeedbackSummary
    let history: [InsightFeedbackRecord]
}
