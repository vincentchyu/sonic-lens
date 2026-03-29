import Foundation
import OSLog

final class WebSocketClient {
    private let url: URL
    private let session: URLSession
    private var task: URLSessionWebSocketTask?
    private var reconnectDelay: TimeInterval = 1
    private var reconnectWorkItem: DispatchWorkItem?
    private var heartbeatTimer: DispatchSourceTimer?
    private var isManuallyDisconnected: Bool = false
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "WebSocket")

    var onMessage: ((URLSessionWebSocketTask.Message) -> Void)?
    var onStateChange: ((Bool) -> Void)?

    init(url: URL, session: URLSession = .shared) {
        self.url = url
        self.session = session
    }

    func connect() {
        isManuallyDisconnected = false
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        heartbeatTimer?.cancel()
        heartbeatTimer = nil

        task?.cancel(with: .goingAway, reason: nil)
        task = session.webSocketTask(with: url)
        task?.resume()
        logger.debug("connect websocket url=\(self.url.absoluteString, privacy: .public)")
        onStateChange?(true)
        reconnectDelay = 1
        startHeartbeat()
        receive()
    }

    func disconnect() {
        isManuallyDisconnected = true
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        heartbeatTimer?.cancel()
        heartbeatTimer = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        logger.debug("disconnect websocket url=\(self.url.absoluteString, privacy: .public)")
        onStateChange?(false)
    }

    private func receive() {
        task?.receive { [weak self] result in
            switch result {
            case .success(let message):
                self?.logger.debug("websocket receive message=\(self?.messagePreview(message) ?? "<unknown>", privacy: .public)")
                self?.onMessage?(message)
                self?.receive()
            case .failure(let error):
                self?.logger.error("websocket receive failed error=\(error.localizedDescription, privacy: .public)")
                self?.onStateChange?(false)
                self?.scheduleReconnect()
            }
        }
    }

    private func scheduleReconnect() {
        guard !isManuallyDisconnected else { return }

        let baseDelay = min(reconnectDelay, 10)
        reconnectDelay = min(reconnectDelay * 2, 10)
        let jitteredDelay = baseDelay * Double.random(in: 0.85...1.15)
        logger.debug("schedule websocket reconnect delay=\(jitteredDelay, privacy: .public)")

        let workItem = DispatchWorkItem { [weak self] in
            self?.connect()
        }
        reconnectWorkItem?.cancel()
        reconnectWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + jitteredDelay, execute: workItem)
    }

    private func startHeartbeat() {
        let timer = DispatchSource.makeTimerSource(queue: DispatchQueue.global(qos: .utility))
        timer.schedule(deadline: .now() + 10, repeating: 10)
        timer.setEventHandler { [weak self] in
            guard let self, let task = self.task, !self.isManuallyDisconnected else { return }
            task.sendPing { error in
                if error != nil {
                    self.logger.error("websocket heartbeat ping failed")
                    self.onStateChange?(false)
                    self.scheduleReconnect()
                }
            }
        }
        heartbeatTimer = timer
        timer.resume()
    }

    private func messagePreview(_ message: URLSessionWebSocketTask.Message) -> String {
        switch message {
        case .string(let text):
            return text.count > 400 ? String(text.prefix(400)) + "..." : text
        case .data(let data):
            let text = String(decoding: data.prefix(400), as: UTF8.self)
            return data.count > 400 ? text + "..." : text
        @unknown default:
            return "<unknown>"
        }
    }
}
