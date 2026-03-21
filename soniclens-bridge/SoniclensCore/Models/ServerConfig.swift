import Foundation

struct ServerConfig: Codable, Hashable, Identifiable {
    let id: UUID
    var name: String
    var host: String
    var port: Int
    var scheme: String

    init(id: UUID = UUID(), name: String, host: String, port: Int, scheme: String = "http") {
        self.id = id
        self.name = name
        self.host = host
        self.port = port
        self.scheme = scheme
    }

    var baseURL: URL {
        URL(string: "\(scheme)://\(host):\(port)")!
    }

    var webSocketURL: URL {
        let wsScheme = scheme == "https" ? "wss" : "ws"
        return URL(string: "\(wsScheme)://\(host):\(port)/ws")!
    }

    var artworkBaseURL: URL {
        URL(string: "\(scheme)://\(host):9000")!
    }

    var displayName: String {
        "\(name) (\(host):\(port))"
    }
}

struct ServerCandidate: Identifiable, Hashable {
    var id: String { normalizedIdentity }
    let name: String
    let host: String
    let port: Int

    func toConfig() -> ServerConfig {
        ServerConfig(name: name, host: normalizeHost(host), port: port)
    }

    private var normalizedIdentity: String {
        "\(name)|\(normalizeHost(host))|\(port)"
    }

    private func normalizeHost(_ value: String) -> String {
        var host = value
        if host.hasSuffix(".local.") {
            host = String(host.dropLast(7))
        } else if host.hasSuffix(".local") {
            host = String(host.dropLast(6))
        }
        return host
    }
}
