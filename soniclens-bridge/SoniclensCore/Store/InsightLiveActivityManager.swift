#if os(iOS)
import ActivityKit
import Foundation
import OSLog
import UIKit

@MainActor
final class InsightLiveActivityManager {
    var onPushToken: ((String, String, ServerConfig) -> Void)?

    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "InsightLiveActivity")
    private var currentActivity: Activity<InsightLiveActivityAttributes>?
    private var currentArtworkIdentifier: String?
    private var artworkTask: Task<Void, Never>?
    private var tokenTask: Task<Void, Never>?

    func startOrUpdate(
        for job: InsightAnalysisJob,
        route: InsightAnalysisRouteSnapshot?,
        using server: ServerConfig?
    ) async {
        guard #available(iOS 16.2, *), UIDevice.current.userInterfaceIdiom == .phone else { return }
        guard ActivityAuthorizationInfo().areActivitiesEnabled else {
            logger.debug("live activity disabled by system")
            return
        }
        if job.phase.isTerminal, currentActivity == nil {
            return
        }

        if let activity = currentActivity, activity.attributes.jobID != job.id {
            await end(activity: activity, using: content(for: job), dismissalPolicy: .immediate)
            currentActivity = nil
            currentArtworkIdentifier = nil
            artworkTask?.cancel()
            artworkTask = nil
            tokenTask?.cancel()
            tokenTask = nil
        }

        let cachedArtworkIdentifier = await LiveActivityArtworkStore.shared.cachedLocalIdentifier(for: route?.artworkDescriptor)
        let artworkIdentifier = cachedArtworkIdentifier ?? currentArtworkIdentifier

        if currentActivity == nil {
            do {
                let activity = try requestActivity(for: job, route: route, artworkLocalIdentifier: artworkIdentifier)
                let supportsPushToken = activity.pushToken != nil
                currentActivity = activity
                currentArtworkIdentifier = artworkIdentifier
                logger.info(
                    "start live activity job=\(job.id, privacy: .public) push_token_supported=\(supportsPushToken, privacy: .public) artwork=\(self.describeArtworkIdentifier(artworkIdentifier), privacy: .public)"
                )
                if let server, supportsPushToken {
                    registerCurrentToken(for: activity, jobID: job.id, using: server)
                    listenForTokenUpdates(activity: activity, jobID: job.id, server: server)
                } else if !supportsPushToken {
                    logger.debug("live activity started without push token job=\(job.id, privacy: .public)")
                }
            } catch {
                logger.error("start live activity failed error=\(error.localizedDescription, privacy: .public)")
                return
            }
        }

        guard let activity = currentActivity else { return }
        if job.phase.isTerminal {
            logger.info("终态已到达，保留灵动岛等待用户点进 App 后关闭 id=\(job.id, privacy: .public)")
            currentArtworkIdentifier = artworkIdentifier
            await activity.update(content(for: job, route: route, artworkLocalIdentifier: artworkIdentifier))
        } else {
            currentArtworkIdentifier = artworkIdentifier
            await activity.update(content(for: job, route: route, artworkLocalIdentifier: artworkIdentifier))
            prepareArtworkIfNeeded(job: job, route: route)
        }
    }

    func dismissCurrentActivityIfNeeded(for job: InsightAnalysisJob, route: InsightAnalysisRouteSnapshot?) async {
        guard #available(iOS 16.2, *), UIDevice.current.userInterfaceIdiom == .phone else { return }
        guard job.phase.isTerminal else { return }

        let activity = currentActivity ?? Activity<InsightLiveActivityAttributes>.activities.first { $0.attributes.jobID == job.id }
        guard let activity else {
            logger.debug("没有找到需要关闭的 Live Activity id=\(job.id, privacy: .public)")
            return
        }

        logger.info("用户已进入 App，关闭终态灵动岛 id=\(job.id, privacy: .public)")
        await end(
            activity: activity,
            using: content(for: job, route: route, artworkLocalIdentifier: currentArtworkIdentifier),
            dismissalPolicy: .immediate
        )
        if currentActivity?.attributes.jobID == job.id {
            currentActivity = nil
        }
        currentArtworkIdentifier = nil
        artworkTask?.cancel()
        artworkTask = nil
        tokenTask?.cancel()
        tokenTask = nil
    }

    private func content(
        for job: InsightAnalysisJob,
        route: InsightAnalysisRouteSnapshot? = nil,
        artworkLocalIdentifier: String? = nil
    ) -> ActivityContent<InsightLiveActivityAttributes.ContentState> {
        logger.debug(
            "生成 Live Activity 内容 id=\(job.id, privacy: .public) phase=\(job.phase.rawValue, privacy: .public) 封面=\(self.describeArtworkIdentifier(artworkLocalIdentifier), privacy: .public)"
        )
        return ActivityContent(
            state: InsightLiveActivityAttributes.ContentState(
                title: job.displayTitle,
                artist: job.artist,
                album: job.album,
                artworkLocalIdentifier: artworkLocalIdentifier,
                providerDisplayName: job.providerDisplayName ?? job.provider,
                modelDisplayName: job.modelDisplayName ?? job.model,
                phase: job.phase
            ),
            staleDate: nil
        )
    }

    private func dismissalPolicy(for phase: InsightJobPhase) -> ActivityUIDismissalPolicy {
        switch phase {
        case .completed:
            return .after(Date().addingTimeInterval(180))
        case .failed, .canceled:
            return .after(Date().addingTimeInterval(45))
        default:
            return .default
        }
    }

    private func end(
        activity: Activity<InsightLiveActivityAttributes>,
        using content: ActivityContent<InsightLiveActivityAttributes.ContentState>,
        dismissalPolicy: ActivityUIDismissalPolicy
    ) async {
        await activity.end(content, dismissalPolicy: dismissalPolicy)
    }

    private func registerCurrentToken(
        for activity: Activity<InsightLiveActivityAttributes>,
        jobID: String,
        using server: ServerConfig
    ) {
        guard let token = activity.pushToken else { return }
        onPushToken?(jobID, hexString(from: token), server)
    }

    private func listenForTokenUpdates(
        activity: Activity<InsightLiveActivityAttributes>,
        jobID: String,
        server: ServerConfig
    ) {
        tokenTask?.cancel()
        tokenTask = Task { [weak self] in
            for await token in activity.pushTokenUpdates {
                guard !Task.isCancelled else { return }
                await MainActor.run {
                    self?.onPushToken?(jobID, self?.hexString(from: token) ?? "", server)
                }
            }
        }
    }

    private func hexString(from data: Data) -> String {
        data.map { String(format: "%02x", $0) }.joined()
    }

    private func requestActivity(
        for job: InsightAnalysisJob,
        route: InsightAnalysisRouteSnapshot?,
        artworkLocalIdentifier: String?
    ) throws -> Activity<InsightLiveActivityAttributes> {
        let attributes = InsightLiveActivityAttributes(jobID: job.id, targetType: job.targetType.rawValue)
        let activityContent = content(for: job, route: route, artworkLocalIdentifier: artworkLocalIdentifier)
        do {
            return try Activity<InsightLiveActivityAttributes>.request(
                attributes: attributes,
                content: activityContent,
                pushType: .token
            )
        } catch {
            logger.error(
                "start live activity with push token failed, fallback to local activity error=\(error.localizedDescription, privacy: .public)"
            )
            return try Activity<InsightLiveActivityAttributes>.request(
                attributes: attributes,
                content: activityContent,
                pushType: nil
            )
        }
    }

    private func prepareArtworkIfNeeded(job: InsightAnalysisJob, route: InsightAnalysisRouteSnapshot?) {
        guard currentArtworkIdentifier == nil,
              let descriptor = route?.artworkDescriptor,
              descriptor.localIdentifier != nil else {
            return
        }

        artworkTask?.cancel()
        artworkTask = Task { @MainActor [weak self] in
            guard let self else { return }
            let localIdentifier = await LiveActivityArtworkStore.shared.prepareArtwork(for: descriptor)
            guard !Task.isCancelled,
                  let localIdentifier,
                  let activity = self.currentActivity,
                  activity.attributes.jobID == job.id else {
                return
            }

            self.currentArtworkIdentifier = localIdentifier
            logger.debug(
                "补发 Live Activity 本地封面 id=\(job.id, privacy: .public) artwork=\(self.describeArtworkIdentifier(localIdentifier), privacy: .public)"
            )
            await activity.update(content(for: job, route: route, artworkLocalIdentifier: localIdentifier))
        }
    }

    private func describeArtworkIdentifier(_ artworkIdentifier: String?) -> String {
        guard let artworkIdentifier else { return "空" }
        let trimmed = artworkIdentifier.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "空字符串" }
        return String(trimmed.prefix(12))
    }
}
#endif
