# 📱 iOS 端户外/旅行 Apple Music 听歌离线捕获与握手补录特性清单

## 1. 背景与目标

在户外通勤或旅行场景下，用户通常使用 iPhone / iPad 原生 Apple Music 听歌，此时设备不在家庭局域网内且无法连接服务端，导致该时段听歌数据处于“真空”状态。本次特性在不依赖苹果付费开发者证书（MusicKit Capability）及后台常驻保活的前提下，构建了 **iOS 本地双引擎离线采集 + 持久化队列 + 服务端统一接入与幂等补录** 的完整闭环：

1. **客户端 MediaPlayer 双引擎采集（iOS / iPadOS）**：
   - **引擎一（`MPMediaLibrary` 差量对账）**：利用系统原生维护的 `MPMediaItem.playCount` 与 `lastPlayedDate` 增量对账。首次建立静默水位基线（Watermark Baseline）防止历史存量误报，之后每次 App 启动、回前台或连接页出现时扫描资料库增量，即便户外全程未打开 SonicLens，回家后也能自动补录所有入库曲目的新增播放与推算时间戳。
   - **引擎二（`MPMusicPlayerController.systemMusicPlayer` 实时监听 + 离线队列）**：监听系统播放器曲目与播放态变更，覆盖未加入媒体库的试听流媒体曲目；满足时长 $\ge 30\text{s}$ 且播放进度 $\ge \min(\text{duration} \times 0.5, 240\text{s})$ 时即时入队本地 SQLite 离线队列，并与引擎一在时间窗内自动协同去重。
2. **服务端高内聚复用与批量幂等补录**：
   - 抽离 `BuildAppleMusicPlaybackEventInput` 及 Apple Music 规范化清洗函数至 `internal/logic/track`，使桌面端 `scrobbler` 与移动端批量接入 100% 共享同一套字符清洗、版本副标题（`AlbumSubtitle`）、发行后缀（`- EP/Single/LP`）解析与 `HandleTrackPlaybackThreshold` 调度中枢。
   - 新增 `POST /api/scrobbler/mobile/batch` 路由（单批上限 200 条），支持按原始 `played_at` 升序重放、60 秒时间窗防重去重（已存在流水直接幂等返回 `accepted_keys`，不重复推送 Last.fm）、Last.fm 历史时间戳 Scrobble 补推、`track_play_records` 落库、曲库/专辑归因与流派统计累加及 WebSocket 广播。

---

## 2. 核心设计要点

### 2.1 iOS 端独立持久化存储 (`MobileScrobbleStore`)
- 独立于轻量化曲库索引 `LibraryIndexStore`，数据落盘于 `Application Support/SonicLens/data/db/soniclens-mobile-scrobble.sqlite`。
- 维护 `scrobble_meta`（`baseline_initialized`、`baseline_initialized_at`、`last_scan_at`、`last_sync_at`）、`media_library_baseline`（`persistent_id` 水位表）、`pending_scrobble_queue`（待补录离线队列）及 `scrobble_sync_history`（最近 200 条已同步明细，含入库与 Last.fm 上报状态）。
- 差量推算每首曲目单次上限 20 条并结合 `±max(duration, 120)` 秒时间窗对已入队记录去重；仅在服务端明确返回 `accepted_keys`（含新入库及后端判重已存在的记录）后才从本地队列移除并原子写入 `scrobble_sync_history`；用户点击状态卡片即可通过 `MobileScrobbleDetailSheet` 查看基线曲目数、待同步队列及最近同步历史明细。

### 2.2 服务端安全边界与防误操作护栏
- `BuildAppleMusicPlaybackEventInput` 显式置 `ControllerFavoriteKnown: false`，严禁离线批量补录调用宿主机 macOS `Music.app` AppleScript 收藏探针。
- `IngestAppleMusicPlaybackBatch` 在调用 `HandleTrackPlaybackThreshold` 前先执行 `modelIsDuplicateTrackPlayRecord(..., 60*time.Second)`，防止重复批次请求二次推送 Last.fm。
- 移动采集默认置信度为 `Medium`（`track_number > 0 && duration > 0`）或 `Low`，遵循“低置信源仅匹配已有曲库累加播放数，不盲目新建专辑”的域规则。

### 2.3 iOS / iPadOS 离线浏览模式与在线态无感自愈
- `AppStore` 新增 `isOfflineBrowsing` 与 `isConnectionSheetPresented`：iOS/iPadOS 启动静默恢复失败时不再阻塞在全屏连接失败决策页，而是保留 `recentServers.first` 并自动进入离线浏览模式（连接页及决策页亦提供手动进入离线模式入口）。
- 离线进入主界面后，`HomeViewModel.loadOfflineCache()` 恢复 `SnapshotCache` 缓存看板，`LibraryViewModel.loadLocalOnly()` 直接加载本地 SQLite 曲库索引，`OfflineModeBanner` 提供一键重连与连接 Sheet 入口；音眸、未上报与远程歌词/详情优雅提示离线不可用。
- 应用切回前台（`scenePhase == .active`）时通过 `attemptAutoReconnectIfOffline()` 静默探活，回家连上局域网后自动切回在线态并无缝触发增量同步与离线听歌补录。

### 2.4 跨平台编译回归治理
- 顺带修复 `AppLayoutView.swift`、`AlbumGridView.swift` 与 `LibraryView.swift` 中由历史 `NavigationCoordinator`（macOS 专属）引发的 iOS 目标跨平台编译与详情跳转缺失问题：`SidebarDestination` 移出 `#if os(macOS)` 供导航模型复用，共享专辑网格与曲目列表在 iOS/iPadOS 恢复原生 `.navigationDestination(item:)` 下钻跳转。

### 2.5 跨重启防重与双表幂等去重加固
- **实时播放态严格校验与检查点持久化**：`AppleMusicScrobbleCoordinator.evaluateSystemPlayerNowPlaying()` 严格要求 `player.playbackState == .playing` 才评估实时入队，避免 App 冷启动时系统播放器停留在暂停态且进度过半的残留 `nowPlayingItem` 再次推算伪造时间戳重复捕获；新增 `RealtimeCaptureCheckpoint` 持久化到 `scrobble_meta`（`realtime_capture_checkpoint`），跨重启保留当前曲目的 `thresholdReached`、`lastPosition` 与 `capturedAt`，仅在切实切歌或同一首歌显式从头重播时才开启新一轮记录。
- **双表联查防重**：`MobileScrobbleStore.hasRecentRecordDuplicate` 同时检索 `pending_scrobble_queue` 与 `scrobble_sync_history`（新增 `idx_scrobble_sync_history_dedupe` 索引），防止记录同步出队后被后台重启或 `MPMediaLibrary` 延迟更新再次重复采集。
- **状态栏动态回写与启动恢复**：扫描无新增且无待同步时动态刷新为监控状态提示，同步完成后精确展示是否已清空队列，并在 `refreshLocalState()` 启动加载时从 `scrobble_meta.last_sync_at` 自动恢复 `lastSyncDate`。

### 2.6 接入层全角标点与流派统计兜底 + 客户端在线/离线显式区分
- **接入层清洗补齐**：`ParseAppleMusicAlbumMetadata` 补齐 `common.UnityFixAll` 全角符号与 `feat.` 归一；`HandleTrackPlaybackThreshold` 统计流派时优先采用已落库/归因后的 `playRecord.Genre`，再回退 `input.Metadata.Genre` 并经 `model.NormalizeGenre` 兜底规整后调用 `IncrementGenrePlayCount`。
- **并发重入锁与状态区分**：`AppleMusicScrobbleCoordinator.evaluateSystemPlayerNowPlaying()` 增加串行重入门控（`isEvaluatingNowPlaying` + `needsReevaluateNowPlaying`）并确保读取 checkpoint 后再标记 `hasLoadedCheckpoint`；`MobileScrobbleStatusCard` 与 `MobileScrobbleDetailSheet` 显式展示当前是否已连接服务端，并将“已同步”指标明确为“累计已同步”。

### 2.7 单曲循环持续监听与去重窗口收敛 + iPhone 导航栏重影修复
- **单曲循环前台持续轮询**：`AppleMusicScrobbleCoordinator.evaluateSystemPlayerNowPlayingOnce()` 取消触发阈值即 `stopPlaybackPolling()` 的限制，改为在 `player.playbackState == .playing && duration >= 30` 期间持续保持 5 秒轻轮询，并将重播重置窗口调优为 `min(duration * 0.35, 30.0)`，确保同一首歌循环回到开头后立即重置阈值并在播过半后继续捕获第 2、3 遍。
- **跨引擎去重时间窗收敛**：`MobileScrobbleStore` 抽取 `dedupeWindowSeconds(forDuration:)`（收敛至 `15~90s`），避免旧的 `max(duration, 120)` 窗口把间隔恰好约 1 个曲长的单曲循环第 2 遍误判为重复记录吞掉。
- **iPhone 重复标题修复**：移除 `PhoneAppLayoutView.phoneNavigationTab` 中重复的 `ToolbarItem(placement: .topBarLeading)`。

---

## 3. 涉及文件清单

### 后端 (`Go`)
- `internal/logic/track/playback.go`：新增 `ExternalPlaybackRecordInput`、`MobileScrobbleBatchResult`、`BuildAppleMusicPlaybackEventInput`、Apple Music 统一清洗方法及 `IngestAppleMusicPlaybackBatch`。
- `internal/logic/track/service.go`：`TrackService` 接口新增 `IngestAppleMusicPlaybackBatch`。
- `internal/scrobbler/scrobbler_player_apple_music.go`：`AppleMusicTrackInfoWrapper` 复用 `track` 包统一清洗方法。
- `api/server.go`：通过 `registerMobileScrobblerRoutes` 注册 `POST /api/scrobbler/mobile/batch`。
- `api/api.md`：更新接口总数与移动端批量补录契约文档。
- `internal/logic/track/playback_test.go`、`api/server_test.go`：补充单元测试与路由测试。

### 客户端 (`soniclens-bridge`)
- `soniclens-bridge/project.yml` & `SoniclensBridge/Info.plist`：补充 `NSAppleMusicUsageDescription` 并通过 `xcodegen generate` 重新生成工程。
- `SoniclensCore/Networking/Endpoints.swift`：新增 `APIPath.mobileScrobbleBatch`。
- `SoniclensCore/Models/MobileScrobbleModels.swift`：定义媒体库快照、离线队列项与批量请求/响应模型。
- `SoniclensCore/Store/MobileScrobbleStore.swift`：独立 SQLite 离线基线对账与待同步队列存储。
- `SoniclensCore/Store/AppleMusicScrobbleCoordinator.swift`：MediaPlayer 双引擎扫描、实时阈值监听与单飞批量同步协调器。
- `SoniclensCore/Store/AppStore.swift` & 三端 App 入口：注入并联动连接握手/前台健康检查自动同步。
- `SoniclensBridge/Views/MobileScrobbleStatusCard.swift`、`ConnectionView.swift`、`PhoneRootView.swift`、`PadRootView.swift`、`PhoneHomeView.swift`、`PadHomeView.swift`：权限引导、待同步胶囊展示与生命周期触发。
- `SoniclensBridge/Views/AppLayoutView.swift`、`AlbumGridView.swift`、`LibraryView.swift`：跨平台导航编译与 iOS 跳转隔离修复。

---

## 4. 验证结果

- **后端全量单元测试**：`/Users/vincent/.asdf/installs/golang/1.27.1/go/bin/go test -count=1 ./...` 全部包通过。
- **客户端三端构建（`xcodegen generate` + `xcodebuild CODE_SIGNING_ALLOWED=NO`）**：
  - `SoniclensBridgePhone`（`generic/platform=iOS`）：**BUILD SUCCEEDED**
  - `SoniclensBridgePad`（`generic/platform=iOS`）：**BUILD SUCCEEDED**
  - `SoniclensBridgeMac`（`macosx`）：**BUILD SUCCEEDED**
