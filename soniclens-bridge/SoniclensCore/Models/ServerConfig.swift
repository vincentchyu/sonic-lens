import Foundation

struct ServerConfig: Codable, Hashable, Identifiable {
    let id: UUID
    var name: String
    var host: String
    var port: Int
    var scheme: String
    var displayHost: String?
    var resolvedHost: String?

    private enum CodingKeys: String, CodingKey {
        case id
        case name
        case host
        case port
        case scheme
        case displayHost
        case resolvedHost
    }

    init(
        id: UUID = UUID(),
        name: String,
        host: String,
        port: Int,
        scheme: String = "http",
        displayHost: String? = nil,
        resolvedHost: String? = nil
    ) {
        self.id = id
        self.name = name
        self.host = Self.normalizeHost(host)
        self.port = port
        self.scheme = scheme
        self.displayHost = displayHost.map(Self.normalizeHost)
        self.resolvedHost = resolvedHost.map(Self.normalizeHost)
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decodeIfPresent(UUID.self, forKey: .id) ?? UUID()
        name = try container.decode(String.self, forKey: .name)
        host = Self.normalizeHost(try container.decode(String.self, forKey: .host))
        port = try container.decode(Int.self, forKey: .port)
        scheme = try container.decodeIfPresent(String.self, forKey: .scheme) ?? "http"
        displayHost = try container.decodeIfPresent(String.self, forKey: .displayHost).map(Self.normalizeHost)
        resolvedHost = try container.decodeIfPresent(String.self, forKey: .resolvedHost).map(Self.normalizeHost)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(host, forKey: .host)
        try container.encode(port, forKey: .port)
        try container.encode(scheme, forKey: .scheme)
        try container.encodeIfPresent(displayHost, forKey: .displayHost)
        try container.encodeIfPresent(resolvedHost, forKey: .resolvedHost)
    }

    var baseURL: URL {
        URL(string: "\(scheme)://\(urlHostComponent):\(port)")!
    }

    var webSocketURL: URL {
        let wsScheme = scheme == "https" ? "wss" : "ws"
        return URL(string: "\(wsScheme)://\(urlHostComponent):\(port)/ws")!
    }

    var artworkBaseURL: URL {
        URL(string: "\(scheme)://\(urlHostComponent):9000")!
    }

    var displayName: String {
        "\(name) (\(displayHost ?? host):\(port))"
    }

    private var connectionHost: String {
        resolvedHost ?? host
    }

    private var urlHostComponent: String {
        let host = connectionHost.trimmingCharacters(in: .whitespacesAndNewlines)
        guard host.contains(":"),
              !host.hasPrefix("["),
              !host.hasSuffix("]") else {
            return host
        }
        return "[\(host)]"
    }

    static func normalizeHost(_ value: String) -> String {
        var host = value.trimmingCharacters(in: .whitespacesAndNewlines)
        while host.hasSuffix(".") {
            host.removeLast()
        }

        while host.lowercased().hasSuffix(".local.local") {
            host = String(host.dropLast(".local".count))
        }

        return host
    }

    func withResolvedHost(_ resolvedHost: String?) -> ServerConfig {
        ServerConfig(
            id: id,
            name: name,
            host: host,
            port: port,
            scheme: scheme,
            displayHost: displayHost,
            resolvedHost: resolvedHost
        )
    }
}

struct ServerCandidate: Identifiable, Hashable {
    var id: String { normalizedIdentity }
    let name: String
    let host: String
    let port: Int
    let resolvedHost: String?

    func toConfig() -> ServerConfig {
        let normalizedHost = ServerConfig.normalizeHost(host)
        return ServerConfig(
            name: name,
            host: normalizedHost,
            port: port,
            displayHost: normalizedHost,
            resolvedHost: resolvedHost.map(ServerConfig.normalizeHost)
        )
    }

    var detailText: String {
        let normalizedHost = ServerConfig.normalizeHost(host)
        guard let resolvedHost = resolvedHost.map(ServerConfig.normalizeHost),
              !resolvedHost.isEmpty,
              resolvedHost != normalizedHost else {
            return "\(normalizedHost):\(port)"
        }
        return "\(normalizedHost):\(port) · 直连 \(resolvedHost)"
    }

    private var normalizedIdentity: String {
        "\(name)|\(ServerConfig.normalizeHost(host))|\(resolvedHost.map(ServerConfig.normalizeHost) ?? "")|\(port)"
    }
}
