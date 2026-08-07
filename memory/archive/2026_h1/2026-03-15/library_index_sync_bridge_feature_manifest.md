# 资料库本地索引与增量同步桥接特性清单

## 日期
2026-03-15

## 特性摘要
- 为 `soniclens-bridge` 资料库页引入本地 SQLite 索引层，专辑/曲目列表的搜索、排序、筛选、分页统一在本地完成。
- 新增 `/api/library/sync` 轻量同步接口，桥接端启动时先读本地索引，再后台同步远端索引数据。
- 为专辑与曲目模型增加 `library_change_log` 变更日志，资料库同步游标改为日志版本号，支持增量 upsert 与 tombstone 删除。
- 收藏状态在 Bridge 端完成本地即时回写，前台切回应用时执行增量刷新，避免列表状态滞后。
- WebSocket 新增 `library_updated(version)` 事件，Bridge 端收到后 debounce 触发增量同步。
- Bridge 本地索引新增 FTS5 虚表，专辑/曲目搜索改为 FTS 优先、LIKE 兜底，提升中文搜索体验。

## 后端改动
- 新增 `internal/model/library_change_log.go`，定义资料库变更日志表与查询方法。
- 在 `Album`、`Track` 的 GORM `AfterCreate/AfterUpdate/AfterDelete` hook 中自动追加变更事件。
- `internal/model/library_index.go` 从“基于 updated_at 的快照同步”改为“基于 change log version 的增量同步”。
- `api/server.go` 的 `/api/library/sync` 返回：
  - `sync_version`
  - `albums`
  - `tracks`
  - `deleted_album_ids`
  - `deleted_track_ids`
- `core/websocket/websocket.go` 新增 `BroadcastLibraryUpdate`，在资料库变更日志写入后向 Bridge 广播最新版本号。

## Bridge 改动
- 新增 `LibraryIndexStore.swift`，使用本地 SQLite 持久化 `album_index` 与 `track_index`。
- `LibraryIndexStore.swift` 新增 `album_index_fts` 与 `track_index_fts` FTS5 虚表，查询路径改为全文检索优先。
- 新增 `LibrarySyncService.swift`，管理全量/增量同步与 schema 版本迁移。
- `LibraryViewModel.swift` 改为查询本地索引，并在收藏变更、前台激活时刷新当前查询结果。
- `AppStore.swift` 与 `NowPlayingService.swift` 接入 `library_updated(version)` 事件，向列表层派发增量同步通知。
- `AlbumGridView.swift`、`LibraryView.swift` 不再对内存页做二次过滤排序，而是直接消费本地查询结果。

## 设计约束
- 列表页只依赖轻量索引，不承担详情字段。
- 详情页继续走原有详情接口。
- 首次同步或 schema 升级时执行全量替换，平时使用增量同步。

## 已验证
- `go test ./api ./internal/model`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -configuration Debug build`

## 后续建议
- 如果后端后续需要进一步降低广播噪音，可将 `library_change_log` 的即时广播改为批量聚合或短窗口合并。
- 若 Bridge 本地索引继续扩容，可进一步评估 FTS5 查询排序权重与中文分词 tokenization 策略。
