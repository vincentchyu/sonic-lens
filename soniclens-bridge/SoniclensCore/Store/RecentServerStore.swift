import Foundation

final class RecentServerStore {
    private let key = "soniclens.recentServers"
    private let maxCount = 5

    func load() -> [ServerConfig] {
        guard let data = UserDefaults.standard.data(forKey: key) else { return [] }
        let decoded = (try? JSONDecoder().decode([ServerConfig].self, from: data)) ?? []
        return decoded
    }

    func add(_ server: ServerConfig) {
        var list = load().filter { $0.host != server.host || $0.port != server.port }
        list.insert(server, at: 0)
        if list.count > maxCount {
            list = Array(list.prefix(maxCount))
        }
        if let data = try? JSONEncoder().encode(list) {
            UserDefaults.standard.set(data, forKey: key)
        }
    }
}
