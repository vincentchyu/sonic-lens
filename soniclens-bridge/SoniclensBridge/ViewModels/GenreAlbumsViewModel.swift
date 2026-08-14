import Foundation
import Combine
import SwiftUI

@MainActor
final class GenreAlbumsViewModel: ObservableObject {
    @Published var rawGenreName: String
    @Published var genreTitle: String
    @Published var rank: Int?
    @Published var playCount: Int?
    @Published var shareText: String?
    @Published var accentKey: HomeHotAccentKey

    @Published private(set) var albums: [Album] = []
    @Published private(set) var totalCount: Int = 0
    @Published private(set) var isLoading: Bool = false
    @Published private(set) var errorMessage: String? = nil

    private let indexStore: LibraryIndexStore
    private var loadTask: Task<Void, Never>? = nil

    init(
        item: HomeHotGenrePresentationItem,
        indexStore: LibraryIndexStore = LibraryIndexStore()
    ) {
        self.rawGenreName = item.rawGenreName
        self.genreTitle = item.title
        self.rank = item.rank
        self.playCount = item.count
        self.accentKey = item.accentKey
        self.indexStore = indexStore
    }

    init(
        rawGenreName: String,
        genreTitle: String,
        rank: Int? = nil,
        playCount: Int? = nil,
        accentKey: HomeHotAccentKey = .tide,
        indexStore: LibraryIndexStore = LibraryIndexStore()
    ) {
        self.rawGenreName = rawGenreName
        self.genreTitle = genreTitle
        self.rank = rank
        self.playCount = playCount
        self.accentKey = accentKey
        self.indexStore = indexStore
    }

    func loadAlbums(server: ServerConfig?) {
        loadTask?.cancel()
        loadTask = Task { [weak self] in
            guard let self else { return }
            self.isLoading = true
            self.errorMessage = nil

            var fetchedFromRemote = false

            if let server {
                do {
                    let client = APIClient(baseURL: server.baseURL)
                    let encodedName = self.rawGenreName.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? self.rawGenreName
                    let path = "/api/genres/\(encodedName)/albums"
                    let queryItems = [
                        URLQueryItem(name: "limit", value: "50"),
                        URLQueryItem(name: "sort", value: "play_count")
                    ]
                    
                    let data = try await client.get(path: path, queryItems: queryItems)
                    let response = try JSONDecoder().decode(GenreAlbumsAPIResponse.self, from: data)
                    
                    if !Task.isCancelled {
                        self.albums = response.albums.map { item in
                            if (item.coverArtURL == nil || item.coverArtURL?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == true) && item.id > 0 {
                                return Album(
                                    id: item.id,
                                    name: item.name,
                                    nameSubtitle: item.nameSubtitle,
                                    artist: item.artist,
                                    releaseDate: item.releaseDate,
                                    coverArtURL: "/api/albums/\(item.id)/cover",
                                    coverArtMime: item.coverArtMime,
                                    coverArtObjectKey: item.coverArtObjectKey,
                                    hasInsight: item.hasInsight,
                                    genre: item.genre,
                                    totalDiscs: item.totalDiscs,
                                    playCount: item.playCount,
                                    createdAt: item.createdAt,
                                    updatedAt: item.updatedAt
                                )
                            }
                            return item
                        }
                        self.totalCount = response.total
                        if let zh = response.genreZh, !zh.isEmpty {
                            self.genreTitle = zh
                        }
                        fetchedFromRemote = true
                    }
                } catch {
                    print("⚠️ 远端获取流派专辑失败，将尝试从本地 SQLite 检索: \(error)")
                }
            }

            // 若离线或 API 失败，优雅回退到本地 SQLite 轻量索引
            if !fetchedFromRemote && !Task.isCancelled {
                do {
                    let localAlbums = try await self.indexStore.fetchAlbumsByGenre(genre: self.rawGenreName, limit: 50, offset: 0)
                    if !Task.isCancelled {
                        self.albums = localAlbums
                        self.totalCount = localAlbums.count
                    }
                } catch {
                    if !Task.isCancelled {
                        self.errorMessage = "加载关联专辑失败，请检查网络或服务连接。"
                    }
                }
            }

            if !Task.isCancelled {
                self.isLoading = false
            }
        }
    }

    deinit {
        loadTask?.cancel()
    }
}

struct GenreAlbumsAPIResponse: Codable {
    let genre: String
    let genreZh: String?
    let total: Int
    let limit: Int
    let offset: Int
    let albums: [Album]

    enum CodingKeys: String, CodingKey {
        case genre
        case genreZh = "genre_zh"
        case total
        case limit
        case offset
        case albums
    }
}
