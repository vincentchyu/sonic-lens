import Foundation
import Observation

enum ConnectionRecoveryState: Equatable {
    case idle
    case restoring(server: ServerConfig, detail: String?)
    case needsDecision(server: ServerConfig, message: String, detail: String?)

    var server: ServerConfig? {
        switch self {
        case .idle:
            return nil
        case .restoring(let server, _):
            return server
        case .needsDecision(let server, _, _):
            return server
        }
    }
}

@MainActor
@Observable
final class ConnectionRecoveryStore {
    private(set) var state: ConnectionRecoveryState = .idle

    var server: ServerConfig? {
        state.server
    }

    var isIdle: Bool {
        state == .idle
    }

    var isBootstrapping: Bool {
        if case .restoring = state {
            return true
        }
        return false
    }

    var isRecoveryRequired: Bool {
        if case .needsDecision = state {
            return true
        }
        return false
    }

    func setRestoring(server: ServerConfig, detail: String?) {
        state = .restoring(server: server, detail: detail)
    }

    func setNeedsDecision(server: ServerConfig, message: String, detail: String?) {
        state = .needsDecision(server: server, message: message, detail: detail)
    }

    func clear() {
        state = .idle
    }
}
