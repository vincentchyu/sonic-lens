import CryptoKit
import Foundation

struct ResolvedArtworkResource: Codable, Hashable {
    let remoteURL: String?
    let coverArtObjectKey: String?

    init(remoteURL: String?, coverArtObjectKey: String?) {
        self.remoteURL = Self.normalized(remoteURL)
        self.coverArtObjectKey = Self.normalized(coverArtObjectKey)
    }

    var localIdentifier: String? {
        if let coverArtObjectKey {
            return Self.sha256Hex(for: "object:\(coverArtObjectKey)")
        }
        if let remoteURL {
            return Self.sha256Hex(for: "url:\(remoteURL)")
        }
        return nil
    }

    var isEmpty: Bool {
        remoteURL == nil && coverArtObjectKey == nil
    }

    private static func normalized(_ value: String?) -> String? {
        guard let value else { return nil }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func sha256Hex(for value: String) -> String {
        let digest = SHA256.hash(data: Data(value.utf8))
        return digest.map { String(format: "%02x", $0) }.joined()
    }
}
