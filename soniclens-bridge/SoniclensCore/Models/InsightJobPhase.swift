import Foundation

enum InsightJobPhase: String, Codable, Hashable {
    case queued
    case running
    case completed
    case failed
    case canceled

    var isTerminal: Bool {
        switch self {
        case .completed, .failed, .canceled:
            return true
        default:
            return false
        }
    }

    var statusText: String {
        switch self {
        case .queued:
            return "准备启动"
        case .running:
            return "分析中"
        case .completed:
            return "已完成"
        case .failed:
            return "失败"
        case .canceled:
            return "已取消"
        }
    }
}
