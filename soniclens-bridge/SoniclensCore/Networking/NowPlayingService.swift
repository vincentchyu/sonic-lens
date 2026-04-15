import Foundation
import OSLog

final class NowPlayingService {
    private let client: WebSocketClient
    private let server: ServerConfig
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "NowPlayingService")

    var onUpdate: ((NowPlaying?, String?) -> Void)?
    var onLibraryUpdate: ((Int64) -> Void)?
    var onInsightJobUpdate: ((InsightAnalysisJob) -> Void)?
    var onRecentPlaysUpdate: (() -> Void)?

    init(server: ServerConfig) {
        self.server = server
        client = WebSocketClient(url: server.webSocketURL)
        client.onMessage = { [weak self] message in
            self?.handle(message)
        }
    }

    func start() {
        client.connect()
    }

    func stop() {
        client.disconnect()
    }

    private func handle(_ message: URLSessionWebSocketTask.Message) {
        switch message {
        case .string(let text):
            guard let data = text.data(using: .utf8) else { return }
            decode(data)
        case .data(let data):
            decode(data)
        @unknown default:
            break
        }
    }

    private func decode(_ data: Data) {
        guard let envelope = try? JSONDecoders.defaultDecoder().decode(WebSocketEnvelope.self, from: data) else {
            logger.error("decode websocket envelope failed body=\(String(decoding: data.prefix(400), as: UTF8.self), privacy: .public)")
            return
        }
        logger.debug("decode websocket envelope type=\(envelope.type, privacy: .public)")
        if envelope.type == "now_playing" || envelope.type == "stop" {
            guard let msg = try? JSONDecoders.defaultDecoder().decode(NowPlayingMessage.self, from: data) else {
                logger.error("decode now_playing message failed")
                return
            }
            if msg.type == "now_playing" {
                logger.debug("dispatch now_playing source=\(msg.source ?? "", privacy: .public)")
                onUpdate?(normalizedNowPlaying(from: msg.data), msg.source)
            } else if msg.type == "stop" {
                logger.debug("dispatch stop event")
                onUpdate?(nil, nil)
            }
        } else if envelope.type == "library_updated" {
            guard let msg = try? JSONDecoders.defaultDecoder().decode(LibraryUpdatedMessage.self, from: data) else {
                logger.error("decode library_updated message failed")
                return
            }
            logger.debug("dispatch library_updated version=\(msg.version, privacy: .public)")
            onLibraryUpdate?(msg.version)
        } else if envelope.type == "insight_job_updated" {
            guard let msg = try? JSONDecoders.defaultDecoder().decode(InsightJobUpdatedEnvelope.self, from: data) else {
                logger.error("decode insight_job_updated message failed")
                return
            }
            logger.debug("dispatch insight_job_updated id=\(msg.data.id, privacy: .public) phase=\(msg.data.phase.rawValue, privacy: .public)")
            onInsightJobUpdate?(msg.data)
        } else if envelope.type == "recent_plays_updated" {
            guard (try? JSONDecoders.defaultDecoder().decode(RecentPlaysUpdatedMessage.self, from: data)) != nil else {
                logger.error("decode recent_plays_updated message failed")
                return
            }
            logger.debug("dispatch recent_plays_updated")
            onRecentPlaysUpdate?()
        }
    }

    private func normalizedNowPlaying(from nowPlaying: NowPlaying?) -> NowPlaying? {
        guard let nowPlaying else { return nil }
        let artwork = resolveArtworkURL(nowPlaying.artwork)
        return NowPlaying(
            artist: nowPlaying.artist,
            album: nowPlaying.album,
            albumSubtitle: nowPlaying.albumSubtitle,
            track: nowPlaying.track,
            duration: nowPlaying.duration,
            position: nowPlaying.position,
            positionMs: nowPlaying.positionMs,
            sampleRate: nowPlaying.sampleRate,
            artwork: artwork,
            isAppleMusicFav: nowPlaying.isAppleMusicFav,
            isLastFmFav: nowPlaying.isLastFmFav,
            appleMusicState: nowPlaying.appleMusicState,
            lastfmState: nowPlaying.lastfmState,
            favoriteState: nowPlaying.favoriteState,
            trackNumber: nowPlaying.trackNumber,
            discNumber: nowPlaying.discNumber,
            receivedAt: Date()
        )
    }

    private func resolveArtworkURL(_ raw: String?) -> String? {
        guard let raw, !raw.isEmpty else { return nil }
        guard let url = URL(string: raw) else { return nil }
        if url.scheme != nil {
            return raw
        }
        return server.artworkBaseURL.appending(path: raw.hasPrefix("/") ? String(raw.dropFirst()) : raw).absoluteString
    }
}
