import Foundation
import OSLog

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
        let (data, response) = try await session.data(from: url)
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
            return try JSONDecoders.defaultDecoder().decode(T.self, from: data)
        } catch {
            logger.error("GET decode failed type=\(String(describing: T.self), privacy: .public) error=\(error.localizedDescription, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw error
        }
    }

    func post<Body: Encodable>(path: String, body: Body) async throws -> Data {
        let url = baseURL.appendingPathComponent(path)
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)

        logger.log("POST \(url.absoluteString, privacy: .public) body=\(self.responsePreview(from: request.httpBody ?? Data()), privacy: .public)")
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            logger.error("POST failed url=\(url.absoluteString, privacy: .public) response=\(String(describing: response), privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw URLError(.badServerResponse)
        }
        logger.log("POST response status=\(http.statusCode, privacy: .public) url=\(url.absoluteString, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
        return data
    }

    func postJSON<T: Decodable, Body: Encodable>(path: String, body: Body) async throws -> T {
        let data = try await post(path: path, body: body)
        do {
            return try JSONDecoders.defaultDecoder().decode(T.self, from: data)
        } catch {
            logger.error("POST decode failed type=\(String(describing: T.self), privacy: .public) error=\(error.localizedDescription, privacy: .public) body=\(self.responsePreview(from: data), privacy: .public)")
            throw error
        }
    }

    private func responsePreview(from data: Data) -> String {
        guard !data.isEmpty else { return "<empty>" }
        let text = String(decoding: data.prefix(800), as: UTF8.self)
        return data.count > 800 ? text + "..." : text
    }
}
