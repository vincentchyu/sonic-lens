import Foundation
import OSLog

final class RecentServerStore {
    private let key = "soniclens.recentServers"
    private let maxCount = 5
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "RecentServerStore")

    func load() -> [ServerConfig] {
        let startedAt = CFAbsoluteTimeGetCurrent()
        guard let data = UserDefaults.standard.data(forKey: key) else { return [] }
        let decoded = (try? JSONDecoder().decode([ServerConfig].self, from: data)) ?? []
        let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
        logger.debug("读取最近连接服务端完成，数量 \(decoded.count, privacy: .public)，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
        return decoded
    }

    func add(_ server: ServerConfig) {
        let startedAt = CFAbsoluteTimeGetCurrent()
        var list = load().filter { $0.host != server.host || $0.port != server.port }
        list.insert(server, at: 0)
        if list.count > maxCount {
            list = Array(list.prefix(maxCount))
        }
        if let data = try? JSONEncoder().encode(list) {
            UserDefaults.standard.set(data, forKey: key)
        }
        let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
        logger.debug("写入最近连接服务端完成，数量 \(list.count, privacy: .public)，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
    }
}
