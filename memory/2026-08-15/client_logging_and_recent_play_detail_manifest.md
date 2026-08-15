# 客户端曲目详情流派/播放数解耦、流水快照承载与客户端日志调试文档特性清单

- **日期**: 2026-08-15
- **模块**: `soniclens-bridge` / `internal/model` / `Docs`
- **状态**: ✅ 已完成 (Verified)

---

## 1. 业务背景与问题

1. **曲目详情流派与元数据缺失**：
   - 首页“最近播放”消费的是 `/api/recent-plays` 流水。此前 `RecentPlayRecord` 未解析流水中的 `genre`, `duration`, `track_number`, `disc_number`，且点击时硬编码构造空 `Track(playCount: 0)`，导致进入详情页时流派空白、播放次数显示为 0。
2. **盲目请求曲库导致报错**：
   - 对于尚未添加到曲库的试听歌曲，详情页此前请求 `GET /api/track`，服务端返回 `{ "play_count": 0 }` 导致客户端反序列化 `Track` 失败并产生 `GET decode failed` 错误日志。
3. **曲目轻量索引与详情懒加载边界**：
   - 本地 SQLite `track_index` 仅服务于快速列表展示与 FTS5 搜索，不应膨胀非必需的重度宽表字段。
4. **同屏流派重复展示**：
   - 顶部 Header 胶囊与下方“基础信息”卡片同时展示了流派，造成界面冗余。
5. **客户端运行日志与沙盒调试文档缺失**：
   - 缺少对 Apple `os.Logger` 统一日志系统、`log stream` 命令与沙盒 SQLite 路径的系统性说明。

---

## 2. 核心改动与架构实现

### 2.1 流水快照独立承载与本地资料库匹配
- **`RecentPlayRecord` 模型补齐**：完整解析 `genre`, `duration`, `track_number`, `disc_number`, `resolved_track_id`，并提供 `bridgeTrack` 转换属性；
- **播放次数多层保障**：
  - 进入详情页时在本地 SQLite（`LibraryIndexStore`）中毫秒级匹配，已入库单曲自动展示真实累计播放次数；
  - 未入库单曲保底展示 1 次（最近播放事实），彻底消除 `播放次数 0`。

### 2.2 彻底移除多余的曲库详情请求
- **`TrackDetailViewModel.load` 纯粹化**：曲目详情页直接消费传入的完整实体/流水快照，仅并发加载 `GET /api/track-lyrics`（歌词）与 `GET /api/track-insight`（音眸），彻底根除 `GET /api/track` 解码报错日志。

### 2.3 保持轻量索引纯粹性
- **回退索引表冗余字段**：从后端 `TrackIndexRow` 与客户端本地 `track_index` DDL / ALTER 逻辑中清理掉 `genre` 冗余列，坚守轻量索引边界。

### 2.4 UI 视觉层级去重
- **`TrackDetailView` 规整**：流派统一收口于顶部 Header 胶囊（`DetailMetaChip`）醒目呈现，移除了下方“基础信息”列表中的重复流派行。

### 2.5 客户端日志与沙盒调试文档沉淀
- 在 `Docs/BUILD_AND_VERIFY.md` 与 `Docs/PACKAGING_AND_LAUNCH.md` 中新增专门章节：
  - 阐述 Apple `os.Logger`（`subsystem: com.vincentchyu.soniclens-bridge`）机制；
  - 详细提供 `log stream`、`log show`、`Console.app` 查看命令及特定分类（`APIClient`、`WebSocket`、`LibrarySync`）过滤语法；
  - 详细标注 macOS 沙盒主目录与本地 SQLite 数据库路径。

---

## 3. 修改文件清单

| 文件路径 | 变更类型 | 说明 |
| :--- | :--- | :--- |
| `soniclens-bridge/SoniclensCore/Models/DashboardModels.swift` | 修改 | `RecentPlayRecord` 补齐流水字段解析与 `bridgeTrack` |
| `soniclens-bridge/SoniclensBridge/ViewModels/TrackDetailViewModel.swift` | 修改 | 移除 `GET /api/track` 请求，新增本地资料库毫秒级匹配 |
| `soniclens-bridge/SoniclensBridge/Views/TrackDetailView.swift` | 修改 | 流派收口至顶部胶囊，接入 `effectivePlayCount` |
| `soniclens-bridge/SoniclensBridge/Views/HomeView.swift` | 修改 | `recentPlayRow` 点击使用 `item.bridgeTrack` |
| `soniclens-bridge/SoniclensCore/Store/LibraryIndexStore.swift` | 修改 | 新增 `findTrack` 本地查询，保持 `track_index` 轻量 |
| `internal/model/library_index.go` | 修改 | 移除 `TrackIndexRow` 的 `genre` 字段 |
| `GEMINI.md` | 修改 | 同步更新轻量索引与流水快照架构规范 |
| `soniclens-bridge/Docs/BUILD_AND_VERIFY.md` | 修改 | 补充客户端日志与沙盒调试指南 |
| `soniclens-bridge/Docs/PACKAGING_AND_LAUNCH.md` | 修改 | 补充客户端日志与沙盒数据查看章节 |
