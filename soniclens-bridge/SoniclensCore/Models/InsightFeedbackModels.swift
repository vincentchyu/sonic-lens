import Foundation
#if os(iOS)
import UIKit
#endif

enum InsightFeedbackReason: String, CaseIterable, Identifiable, Codable, Hashable {
    case inaccurate = "不准确"
    case vague = "太空泛"
    case notRelevant = "不贴合歌曲/专辑"
    case missingInfo = "缺少关键信息"
    case messyStructure = "结构混乱"
    case other = "其他"

    var id: String { rawValue }
}

struct InsightFeedbackRecord: Codable, Identifiable, Hashable {
    let id: Int64
    let insightID: Int64
    let score: Int
    let comment: String
    let reasonCodes: [String]?

    let sectionKey: String?
    let sourcePlatform: String?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case insightID = "insight_id"
        case score
        case comment
        case reasonCodes = "reason_codes"
        case sectionKey = "section_key"
        case sourcePlatform = "source_platform"
        case createdAt = "created_at"
    }

    var isNegative: Bool {
        score < 0
    }

    var isPositive: Bool {
        score > 0
    }
}

struct InsightFeedbackSummary: Codable, Hashable {
    let insightID: Int64
    let analysisTargetType: InsightTargetType
    let likeCount: Int
    let dislikeCount: Int
    let hasFeedback: Bool
    let latestFeedback: InsightFeedbackRecord?
    let latestNegativeFeedback: InsightFeedbackRecord?
    let topReasonCodes: [String]?

    enum CodingKeys: String, CodingKey {
        case insightID = "insight_id"
        case analysisTargetType = "analysis_target_type"
        case likeCount = "like_count"
        case dislikeCount = "dislike_count"
        case hasFeedback = "has_feedback"
        case latestFeedback = "latest_feedback"
        case latestNegativeFeedback = "latest_negative_feedback"
        case topReasonCodes = "top_reason_codes"
    }

    var displayStatus: String {
        if latestFeedback?.score == -1 {
            return "待修正"
        }
        if likeCount > 0 || latestFeedback?.score == 1 {
            return "已认可"
        }
        return "未评价"
    }
}

struct InsightFeedbackHistoryResponse: Codable {
    let feedbacks: [InsightFeedbackRecord]?
    let analysisTargetType: InsightTargetType?

    enum CodingKeys: String, CodingKey {
        case feedbacks
        case analysisTargetType = "analysis_target_type"
    }
}

struct InsightFeedbackSubmitRequest: Encodable {
    let score: Int
    let comment: String
    let reasonCodes: [String]?

    let sectionKey: String?
    let sourcePlatform: String

    enum CodingKeys: String, CodingKey {
        case score
        case comment
        case reasonCodes = "reason_codes"
        case sectionKey = "section_key"
        case sourcePlatform = "source_platform"
    }

    static func helpful(sourcePlatform: String = InsightFeedbackPlatform.current) -> InsightFeedbackSubmitRequest {
        InsightFeedbackSubmitRequest(
            score: 1,
            comment: "",
            reasonCodes: [],
            sectionKey: nil,
            sourcePlatform: sourcePlatform
        )
    }
}

struct InsightIssueDraft: Equatable {
    var selectedReasons: Set<InsightFeedbackReason> = []
    var comment: String = ""
    var sectionKey: String = ""

    var reasonCodes: [String] {
        InsightFeedbackReason.allCases
            .filter { selectedReasons.contains($0) }
            .map(\.rawValue)
    }

    var normalizedComment: String {
        comment.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}

enum InsightFeedbackPlatform {
    static var current: String {
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
        return "bridge"
        #endif
    }
}
