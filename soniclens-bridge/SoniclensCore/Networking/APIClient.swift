import Foundation
import OSLog
#if os(iOS)
import UIKit
#endif

final class APIClient {
    private let baseURL: URL
    private let session: URLSession
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "APIClient")

    init(baseURL: URL, session: URLSession? = nil) {
        self.baseURL = baseURL
        if let session = session {
            self.session = session
        } else {
            let config = URLSessionConfiguration.default
            config.timeoutIntervalForRequest = 5
            config.timeoutIntervalForResource = 8
            self.session = URLSession(configuration: config)
        }
    }

    func get(path: String, queryItems: [URLQueryItem] = []) async throws -> Data {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)
        if !queryItems.isEmpty {
            components?.queryItems = queryItems
        }
        guard let url = components?.url else {
            throw URLError(.badURL)
        }

        logger.log("GET \(url.absoluteString, privacy: .public)")
        let request = makeRequest(url: url, method: "GET")
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            logger.error("GET failed url=\(url.absoluteString, privacy: .public) response=\(String(describing: response), privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw URLError(.badServerResponse)
        }
        logger.log("GET response status=\(http.statusCode, privacy: .public) url=\(url.absoluteString, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
        return data
    }

    func getJSON<T: Decodable>(path: String, queryItems: [URLQueryItem] = []) async throws -> T {
        let data = try await get(path: path, queryItems: queryItems)
        do {
            return try await decodeJSON(T.self, from: data)
        } catch {
            logger.error("GET decode failed type=\(String(describing: T.self), privacy: .public) error=\(error.localizedDescription, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw error
        }
    }

    func post<Body: Encodable>(path: String, body: Body, timeout: TimeInterval? = nil) async throws -> Data {
        let url = baseURL.appendingPathComponent(path)
        var request = makeRequest(url: url, method: "POST")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        if let timeout {
            request.timeoutInterval = timeout
        }

        logger.log("POST \(url.absoluteString, privacy: .public) body=\(self.responsePreview(from: request.httpBody ?? Data()), privacy: .public)")
        let activeSession = sessionForRequest(timeout: timeout)
        let (data, response) = try await activeSession.data(for: request)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            logger.error("POST failed url=\(url.absoluteString, privacy: .public) response=\(String(describing: response), privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw URLError(.badServerResponse)
        }
        logger.log("POST response status=\(http.statusCode, privacy: .public) url=\(url.absoluteString, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
        return data
    }

    func postJSON<T: Decodable, Body: Encodable>(path: String, body: Body, timeout: TimeInterval? = nil) async throws -> T {
        let data = try await post(path: path, body: body, timeout: timeout)
        do {
            return try await decodeJSON(T.self, from: data)
        } catch {
            logger.error("POST decode failed type=\(String(describing: T.self), privacy: .public) error=\(error.localizedDescription, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw error
        }
    }

    private func sessionForRequest(timeout: TimeInterval?) -> URLSession {
        guard let timeout else { return session }
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = timeout
        config.timeoutIntervalForResource = timeout
        return URLSession(configuration: config)
    }

    private func makeRequest(url: URL, method: String) -> URLRequest {
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue(Self.terminalIdentifier, forHTTPHeaderField: Self.terminalHeaderField)
        return request
    }

    private func decodeJSON<T: Decodable>(_ type: T.Type, from data: Data) async throws -> T {
        try await Task.detached(priority: .userInitiated) {
            try JSONDecoders.defaultDecoder().decode(T.self, from: data)
        }.value
    }

    private func responsePreview(from data: Data) -> String {
        guard !data.isEmpty else { return "<empty>" }
        let text = String(decoding: data.prefix(800), as: UTF8.self)
        return data.count > 800 ? text + "..." : text
    }

    private static let terminalHeaderField = "X-SonicLens-Terminal"

    private static var terminalIdentifier: String {
        #if os(macOS)
        return "mac"
        #elseif os(iOS)
        switch UIDevice.current.userInterfaceIdiom {
        case .pad:
            return "ipad"
        case .phone:
            return "iphone"
        default:
            return "ios"
        }
        #else
        return "web"
        #endif
    }
}

enum ArtworkURLResolver {
    static func resolve(_ raw: String?, baseURL: URL?) -> String? {
        guard let raw else { return nil }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        guard let parsed = URL(string: trimmed) else { return nil }
        if parsed.scheme != nil {
            return trimmed
        }
        guard let baseURL else { return nil }
        return baseURL.appending(path: trimmed.hasPrefix("/") ? String(trimmed.dropFirst()) : trimmed).absoluteString
    }

    static func resolveArtworkPath(_ raw: String?, artworkBaseURL: URL?) -> String? {
        resolve(raw, baseURL: artworkBaseURL)
    }
}
