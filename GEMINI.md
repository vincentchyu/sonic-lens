# ⚡ SonicLens Quick Memory

SonicLens 核心架构蓝图、领域规则、开发规范与防坑记忆。AI Agent 在进行代码变更前必须阅读并遵循。

---

## 0. 长期记忆与抽象母规则

### 0.1 记忆管理协议 (Memory Protocol)
- **历史索引**：[memory_index.md](./memory_index.md) 是全站开发的特性历史索引。
- **更新守则**：重大特性开发、架构重构或核心逻辑修复后，必须执行：
  1. 在 `memory/YYYY-MM-DD/` 下创建详细的 `feature_manifest.md` 特性清单；
  2. 将清单挂载到 `memory_index.md` 顶部；
  3. 同步更新本 `GEMINI.md` 核心业务契约。
- **公众号文章生成规约**：详见 [wechat_article_generation_constraint.md](./output/wechat_article_generation_constraint.md)。

### 0.2 四大约束母规则
1. **结构化事实优先**：专辑/曲目身份、版本说明 (`name_subtitle`)、发行格式 (`release_type`) 等统一消费结构化字段，严禁从标题字符串或页面状态二次猜测。
2. **高频状态专仓化**：高频播放态、收藏态收口至独立 Store/Projection/Coordinator，禁止在全局 `AppStore` 或密集列表中直接广播与订阅高频变化。
3. **异步主链 + 查询对账**：WebSocket/推送/异步任务为主通道，GET 查询为恢复和对账通道；禁止退回单次长阻塞请求。
4. **生成物目录化**：文章、分享物料等落入独立目录，根目录仅保留入口与约束。

---

## 1. 项目架构蓝图与模块索引

### 1.1 核心模块职责
- **`main.go` / `cmd/`**: 应用总入口与独立 CLI 工具。新增 CLI 必须通过 `cmd.RegisterCommands(rootCmd)` 统一挂载，严禁在 `cmd/` 中写裸 SQL。
- **`internal/model/`**: **唯一数据访问层 (DAO)**。所有 GORM CRUD、事务包装（`model.InTx`）必须收口于此。
- **`internal/logic/`**: 核心业务编排层。仅通过 DAO 访问数据库，严禁直接散落 GORM 查询。
- **`internal/scrobbler/` & `internal/sync/`**: 播放器状态接入与 D1/后台数据增量同步。
- **`api/`**: Gin 接口层、参数校验、中间件（Redis 缓存/负缓存）与 WebSocket 广播。修改/新增路由必须同步维护 `api/api.md`。
- **`core/`**: 底座基础设施（`telemetry` OTel/SigNoz、`ai` 平台抽象、`lyrics` 歌词解析、`websocket`、`redis`、`musicbrainz`、`objectstorage`）。
- **`soniclens-bridge/`**: SwiftUI 三端客户端（Mac/Pad/Phone）。共享逻辑收口于 `SoniclensCore`，端差异仅限容器交互。
- **`templates/` & `static/`**: Web UI。后台入口为 `templates/admin.html` (Shell) + `templates/admin/*.html` (Partial) + `static/admin/`。

### 1.2 模块调用拓扑
`Main/CLI` -> `API` -> `Internal/Logic` -> `Internal/Model` (DAO/Tx) -> `Core/DB`  
`Scrobbler` -> `Logic` -> `Model` | `Bridge App` -> `ViewModels` -> `SoniclensCore` -> `API/WebSocket`

---

## 2. 开发规范与最佳实践

### 2.1 后端编码与基础设施规范 (Go)
- **命名与风格**：严格遵循 Uber Go 风格。**所有注释与日志必须使用中文**；导出的函数和类型必须阐述“为什么”。
- **枚举规范**：对外可复用枚举统一收口在 `common/`（如 `common/enum.go`），使用类型化常量（`string`/`uint8`/`int8`），严禁散落裸常量。
- **并发与异步红线**：禁止使用裸 `go func`。必须使用 `core/telemetry` 提供的 `GoSafe`（带 span）、`GoSafeDetached`（脱离父 span）或 `GoOnlySafe`（仅 recover）。
- **可观测性红线**：链路追踪与指标全量收口至 `core/telemetry`（SigNoz OTLP gRPC）。HTTP 走 `otelgin`，Redis 走 `redisotel`，DB 走 GORM tracing 插件与 `otelsql` 连接池指标；禁止业务代码手写重复 span。
- **错误处理**：禁止忽略错误，统一使用 `%w` 进行错误包装以保留调用链。
- **测试边界规范**：默认 `go test -count=1 ./...` 必须 100% 独立、无实体库、无外部网络、秒级稳定运行；依赖真实外部资源/GUI 的测试用 `//go:build integration` 隔离；DAO/Service 单测统一使用 `internal/testutil` 内存库（SQLite 内存库/sqlmock），参考技能 `.agents/skills/soniclens-unit-test/`。

### 2.2 数据库设计与 DAO 规范 (GORM / MySQL)
- **DAO 边界与事务**：所有 CRUD 必须在 `internal/model/`；跨表事务使用 `model.InTx(ctx, func(tx *gorm.DB) error)`，DAO 内部传递 `tx`。
- **乐观锁与上下文**：所有 DB 操作必须绑定 `.WithContext(ctx)`；计数更新等使用 `version` 乐观锁。
- **统计快照表无自增 ID**：`*_stat` 排行与统计派生表严禁使用自增 ID，统一使用业务复合主键（如 `(period_days, rank)`），杜绝自增锁与索引碎片。
- **字符清洗与统一规整**：曲目/专辑落库前统一执行 `UnityFixAll + ConversionSimplifiedFx` 繁简与标点归一；自动剥离 ` - EP/Single/LP` 等连字符发行格式后缀。

### 2.3 客户端规范 (SwiftUI / Bridge)
*详细 UI 布局、多碟规范与性能模式见 [soniclens-bridge/Docs/](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/Docs/)*
- **工程管理**：Xcode 工程由 `project.yml` 驱动，任何 Target/配置修改必须改 `project.yml` 并执行 `xcodegen generate`，严禁手改 `.xcodeproj`。
- **平台隔离与材质**：macOS 专属代码与 AppKit 必须通过 `#if os(macOS)` 隔离；macOS 玻璃拟态效果必须由窗口级 `NSVisualEffectView` 长期承载，禁止在 SwiftUI 修饰链中裸用。
- **资料库架构与轻量索引**：
  - 坚持“本地 SQLite (schema 7+) + FTS5 搜索 + `/api/library/sync` 增量同步 + WS 推送 + 详情页懒加载”；`LibraryViewModel` 单飞刷新，首页优先返回，禁止整页重载。
  - **轻量索引边界红线**：`track_index` / `LibraryIndexStore` 严格保持轻量（仅保留列表展示与 FTS5 搜索核心列：`artist`, `album`, `track`, `play_count`, `duration`, `track_number`, `disc_number`, 收藏态），严禁塞入非列表必需的重度字段（如歌词、深层归因元数据）。
  - **详情页异步装载**：曲目详情页直接消费传入的完整实体/快照，仅异步并发加载歌词 (`/api/track-lyrics`) 与音眸 (`/api/track-insight`)，避免盲目重查曲库曲目实体。
- **流水快照独立承载**：
  - **流水表 (`TrackPlayRecord`) vs 曲库实体 (`Track`) 严格解耦**：最近播放（`RecentPlayRecord`）属于时序播放事实，自带播放发生时的快照元数据（`genre`, `duration`, `track_number`, `disc_number` 等）。对于从未加入曲库的歌曲，点击详情时必须完整承载流水自带的元数据；杜绝因曲库无实体而导致流派与元数据空白。
- **状态与高频隔离**：`AppStore` 仅承载全局低频态（连接/配置）；高频播放态、收藏态收口到 `PlaybackStore` / `FavoriteStore`；密集列表/详情页禁止直接订阅高频广播。
- **播放与网络契约**：
  - GET 请求 query 参数统一经由 `APIClient` 百分号编码（将 `+` 编码为 `%2B`），路径拼接严禁使用 `appendingPathComponent` 二次转义。
  - 正在播放消费 `WS now_playing` 结构化状态（`apple_music_state`、`lastfm_state`、`favorite_state`）；WS 静默超时自动冻结本地进度。
- **音眸与分享 (ShareKit)**：
  - 音眸契约以 `core/ai/agent_insight_*.go` JSON Schema 为准；任务统一走 `/api/insight-jobs` 异步主链 + WS 推送 + 深链回流；iOS 关联 Live Activity 进度。
  - 分享统一走 `ShareKit`（iPhone 390pt 长图 / macOS 16:9 Bento Grid 双栏流式分页），全面废弃直接 View 截图。

### 2.4 Web 端规范
- **后台架构**：`templates/admin.html` (Shell) + `templates/admin/*.html` (Partial) + `static/admin/` (CSS/JS)；列表状态统一使用 `ui-state.js`。
- **待处理专辑与流派维护**：必须支持实时 vs 冻结对账与 context stale 强制刷新；全链路保持 `album_subtitle` 与 `release_type` 闭环。

---

## 3. 核心领域模型与底座契约

### 3.1 核心领域模型契约表
| 领域模型 / 文件 | 核心身份键 / 索引 | 业务规则与不变量 |
| :--- | :--- | :--- |
| **Track**<br>[`track.go`](./internal/model/track.go) | `(Artist, Album, AlbumSubtitle, Track, DiscNumber, TrackNumber)` | 1. 六元组唯一身份，查询与落库自动剥离发行格式后缀；<br>2. 落库必须做繁简与标点归一（`UnityFixAll`）；<br>3. 低置信源仅累加播放数，不自动新建专辑。 |
| **Album**<br>[`album.go`](./internal/model/album.go) | `(Artist, Name, NameSubtitle, ReleaseDate)` | 1. 四元组唯一身份，`original_release_date` 仅作事实字段不参与身份匹配；<br>2. `name_subtitle` 为版本说明（Deluxe等）唯一正式字段，`release_type`（ep/single/lp）统一清洗入库；<br>3. GORM Hook 自动触发 `library_change_log`。 |
| **TrackAlbum**<br>[`track_album.go`](./internal/model/track_album.go) | `(AlbumID, DiscNumber, TrackNumber)` | 1. 物理位置优先于曲名匹配；<br>2. 深度维护（`sync_status=3`）补全曲目时创建真实曲目（`play_count=0`），不再写占位符；<br>3. 合并清洗时通过 `force=true` 覆盖保护。 |
| **TrackPlayRecord**<br>[`track_play_record.go`](./internal/model/track_play_record.go) | 播放历史流水 (SST) | 1. **全站唯一播放事实源**，承载归因状态、`release_type`、`album_subtitle` 与 Trace 上下文；<br>2. 提供 4 层自动数据修补引擎与 `/api/admin/stats/reconcile` 全表播放量纠偏对账。 |
| **Favorite**<br>[`track_favorite_event.go`](./internal/model/track_favorite_event.go) | 收藏意图事件 | 1. 事件表仅记意图，对外读取统一由 Logic 层 Favorite Projection 合成稳定态与 pending 态；<br>2. 结构化输出 `apple_music_state`、`lastfm_state`、`favorite_state`。 |
| **PendingAlbum**<br>[`pending_album_work_item.go`](./internal/model/pending_album_work_item.go) | 待归因工单 / 维护草稿 | 1. 维护冻结上下文与实时比对，支持 `context_stale` 刷新；<br>2. 待归因工单与正式专辑维护 100% 复用预审比对弹窗与 `staging_draft_json` 草稿机制。 |
| **Genre**<br>[`genre.go`](./internal/model/genre.go) | 流派权威源 | 1. `internal/cache/genre_cache.go`（基于 DB 表）为全局唯一权威源，彻底废弃静态字典；<br>2. 对账时严禁自动创建未认证流派，未匹配项进 `unmatchedGenres` 供人工映射；<br>3. 热门流派支持 Top 50，专辑检索采用精确 Token 词界匹配。 |
| **Insight / Job**<br>[`insight_job.go`](./internal/model/insight_job.go)<br>[`track_insight.go`](./internal/model/track_insight.go) | 音眸异步任务与解析 | 1. JSON Schema 以 `core/ai/agent_insight_*.go` 为唯一契约；<br>2. iPhone 端通过 `/api/insight-jobs` 异步执行 + WS `insight_job_updated` 广播 + 降级轮询；<br>3. 专辑音眸基于已有最高分曲目音眸二次聚合，不重复逐曲请求 LLM。 |
| **LibrarySync**<br>[`library_change_log.go`](./internal/model/library_change_log.go) | 增量版本日志 | 记录增删改 Tombstone，驱动 Bridge 端的增量同步游标。 |

### 3.2 核心底座与外部集成
- **AI 抽象 (`core/ai/`)**：平台工厂 + `LLMProvider` 架构，统一对齐 OMLX 规范。`MultiStepAnalyzeTrack` 按翻译 (Step 1) -> 解读 (Step 2) -> 深度总结 (Step 3) 流式多步推进，支持纯音乐自动跳过与单步指数退避重试；上下文透传 `ContextKeyJobID` 便于全链路排障。
- **可观测性 (`core/telemetry/`)**：全局 SigNoz OTLP gRPC 接入；Scrobbler 坚持单曲长根 span + 阶段 event；HTTP Client 出站统一使用 `WrapHTTPClient` 注入追踪。
- **D1 增量同步 (`internal/sync/d1_sync.go`)**：基于运行锁与时间戳增量同步；直连 `database/sql` 必须挂载 `otelsql` 指标。
- **外部集成**：MusicBrainz 统一在包装层建 span（禁止反射）；Redis 走 `redisotel`；S3/MinIO/R2 走 smithy 原生 OTel adapter。

---

*文档版本: v4.0 | 维护原则: 保持高密度架构事实与规约，细节下沉至专项文档或 Memory。*
