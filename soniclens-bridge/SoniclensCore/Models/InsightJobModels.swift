import Foundation

struct InsightAnalysisJob: Codable, Identifiable, Hashable {
    let id: String
    let targetType: InsightTargetType
    let albumID: Int64?
    let artist: String
    let album: String
    let track: String
    let trackNumber: Int?
    let discNumber: Int?
    let provider: String
    let model: String
    let providerDisplayName: String?
    let modelDisplayName: String?
    let clientPlatform: String?
    let resultInsightID: Int64?
    let phase: InsightJobPhase
    let resultAvailable: Bool
    let errorMessage: String?
    let startedAt: String?
    let finishedAt: String?
    let updatedAt: String?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case targetType = "analysis_target_type"
        case albumID = "album_id"
        case artist
        case album
        case track
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case provider
        case model
        case providerDisplayName = "provider_display_name"
        case modelDisplayName = "model_display_name"
        case clientPlatform = "client_platform"
        case resultInsightID = "result_insight_id"
        case phase = "status"
        case resultAvailable = "result_available"
        case errorMessage = "error_message"
        case startedAt = "started_at"
        case finishedAt = "finished_at"
        case updatedAt = "updated_at"
        case createdAt = "created_at"
    }

    var displayTitle: String {
        targetType == .album ? album : track
    }

    var subtitle: String {
        if targetType == .album {
            return artist
        }
        let parts = [artist, album].filter { !$0.isEmpty }
        return parts.joined(separator: " · ")
    }

    var fallbackTrack: Track? {
        guard targetType == .track, !artist.isEmpty, !track.isEmpty else { return nil }
        return Track(
            id: 0,
            artist: artist,
            album: album,
            track: track,
            playCount: 0,
            trackNumber: trackNumber,
            discNumber: discNumber,
            duration: nil,
            isAppleMusicFav: nil,
            isLastFmFav: nil,
            createdAt: nil,
            updatedAt: nil
        )
    }

    func matches(track candidate: Track) -> Bool {
        guard targetType == .track else { return false }
        return artist == candidate.artist
            && album == candidate.album
            && track == candidate.track
            && trackNumber == candidate.trackNumber
            && discNumber == candidate.discNumber
    }

    func matches(albumID candidate: Int64) -> Bool {
        guard targetType == .album else { return false }
        return albumID == candidate
    }
}

struct InsightJobCreateRequest: Encodable {
    let targetType: InsightTargetType
    let artist: String?
    let album: String?
    let track: String?
    let trackNumber: Int?
    let discNumber: Int?
    let albumID: Int64?
    let provider: String
    let model: String
    let clientPlatform: String

    enum CodingKeys: String, CodingKey {
        case targetType = "target_type"
        case artist
        case album
        case track
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case albumID = "album_id"
        case provider
        case model
        case clientPlatform = "client_platform"
    }
}

struct InsightJobLiveActivityTokenRequest: Encodable {
    let token: String
}

struct InsightJobResponse: Decodable {
    let job: InsightAnalysisJob
    let existing: Bool?
}

struct InsightJobUpdatedEnvelope: Decodable {
    let type: String
    let data: InsightAnalysisJob
}

struct InsightAnalysisRouteSnapshot: Codable, Hashable {
    let jobID: String
    let targetType: InsightTargetType
    let track: Track?
    let albumID: Int64?
    let artworkDescriptor: ResolvedArtworkResource?

    enum CodingKeys: String, CodingKey {
        case jobID
        case targetType
        case track
        case albumID
        case artworkDescriptor
        case legacyArtworkURL = "artworkURL"
    }

    init(
        jobID: String,
        targetType: InsightTargetType,
        track: Track?,
        albumID: Int64?,
        artworkDescriptor: ResolvedArtworkResource?
    ) {
        self.jobID = jobID
        self.targetType = targetType
        self.track = track
        self.albumID = albumID
        self.artworkDescriptor = artworkDescriptor?.isEmpty == true ? nil : artworkDescriptor
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        jobID = try container.decode(String.self, forKey: .jobID)
        targetType = try container.decode(InsightTargetType.self, forKey: .targetType)
        track = try container.decodeIfPresent(Track.self, forKey: .track)
        albumID = try container.decodeIfPresent(Int64.self, forKey: .albumID)
        if let descriptor = try container.decodeIfPresent(ResolvedArtworkResource.self, forKey: .artworkDescriptor) {
            artworkDescriptor = descriptor.isEmpty ? nil : descriptor
        } else if let legacyArtworkURL = try container.decodeIfPresent(String.self, forKey: .legacyArtworkURL) {
            let descriptor = ResolvedArtworkResource(remoteURL: legacyArtworkURL, coverArtObjectKey: nil)
            artworkDescriptor = descriptor.isEmpty ? nil : descriptor
        } else {
            artworkDescriptor = nil
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(jobID, forKey: .jobID)
        try container.encode(targetType, forKey: .targetType)
        try container.encodeIfPresent(track, forKey: .track)
        try container.encodeIfPresent(albumID, forKey: .albumID)
        try container.encodeIfPresent(artworkDescriptor, forKey: .artworkDescriptor)
    }

    static func track(jobID: String, track: Track, artworkDescriptor: ResolvedArtworkResource?) -> InsightAnalysisRouteSnapshot {
        InsightAnalysisRouteSnapshot(jobID: jobID, targetType: .track, track: track, albumID: nil, artworkDescriptor: artworkDescriptor)
    }

    static func album(jobID: String, albumID: Int64, artworkDescriptor: ResolvedArtworkResource?) -> InsightAnalysisRouteSnapshot {
        InsightAnalysisRouteSnapshot(jobID: jobID, targetType: .album, track: nil, albumID: albumID, artworkDescriptor: artworkDescriptor)
    }

    static func fallback(from job: InsightAnalysisJob) -> InsightAnalysisRouteSnapshot? {
        switch job.targetType {
        case .track:
            guard let track = job.fallbackTrack else { return nil }
            return .track(jobID: job.id, track: track, artworkDescriptor: nil)
        case .album:
            guard let albumID = job.albumID else { return nil }
            return .album(jobID: job.id, albumID: albumID, artworkDescriptor: nil)
        }
    }
}

struct PersistedInsightAnalysisState: Codable {
    let job: InsightAnalysisJob
    let route: InsightAnalysisRouteSnapshot?
    let lastEventTimestamp: Double?
}
