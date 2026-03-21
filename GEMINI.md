# ⚡ SonicLens Quick Memory

这是一份关于 SonicLens 项目的长期记忆清单，整合了项目架构、核心逻辑、开发规范及关键“陷阱”规避方案。AI Agent 在进行任何代码变更前应阅读本清单。

---

## 0. 长期记忆管理协议 (Memory Protocol)

- **核心索引文件**：[memory_index.md](./memory_index.md) 是全站开发的特性历史索引。
- **更新守则**：**AI Agent 在完成重大特性开发、架构重构或核心逻辑修复后，必须执行以下操作**：
    1. 在 `memory/YYYY-MM-DD/` 线下创建详细的 `feature_manifest.md` 特性清单。
    2. 将该清单挂载到 `memory_index.md` 的顶部。
    3. 同步审查并更新本 `GEMINI.md` 文件，确保“核心业务记忆”章节反映最新的逻辑现状。

---

## 1. 项目架构蓝图与模块索引

### 1.1 核心模块树 (Module Tree)

- **`main.go`**: 应用总入口。负责初始化配置、日志、数据库连接及启动 API Server。
- **`cmd/`**: 独立命令行工具集，用于同步、回放和维护任务。
- **`common/`**: 通用枚举、转换与基础工具，供全项目复用。
- **`core/`**: 基础设施与外部能力适配层，负责日志、数据库、缓存、WebSocket、歌词、AI、播放器/第三方服务集成等底座能力。
- **`internal/`**: 核心领域层，负责 DAO、业务编排、播放器适配与后台同步任务。
- **`api/`**: Gin 接口层，负责参数绑定、权限判断、响应编码与 WebSocket 接口暴露。
- **`soniclens-bridge/`**: SwiftUI 客户端工作区，包含三端 target、共享核心层、ViewModels、Views 与客户端文档。
- **`templates/`**: Web 模板层，承载历史 Web UI 与页面脚本入口。
- **`static/`**: Web 静态资源层，存放 CSS、图片与前端素材。

**快速定位梗概**

- 数据库 CRUD 与事务入口看 `internal/model/`；不要在 `api/` 或 `logic/` 直接散落 GORM 细节。
- 业务流程编排看 `internal/logic/`；播放器状态接入看 `internal/scrobbler/`；批处理/回放/补写任务看 `internal/sync/`。
- 实时协议与播放态广播看 `core/websocket/`；歌词解析与 LRC 同步判定看 `core/lyrics/`；AI 结构化 schema 看 `core/ai/`。
- Web 端页面逻辑主要落在 `templates/*.html`；Bridge 共享能力看 `soniclens-bridge/SoniclensCore/`，端侧容器与交互看 `soniclens-bridge/SoniclensBridge/ViewModels` 和 `soniclens-bridge/SoniclensBridge/Views`。
- 涉及客户端边界、API 映射与构建验证时，优先查 `soniclens-bridge/Docs/`，不要只凭页面代码反推架构。

### 1.2 模块调用拓扑
`Main` -> `API` -> `Internal/Logic` -> `Internal/Model` -> `Core/DB`
`Scrobbler` -> `Internal/Logic` -> `Internal/Model`
`Bridge App` -> `ViewModels` -> `SoniclensCore` -> `API/WebSocket`

### 1.3 前端双轨制说明
- **当前现状**: Web 端大量核心功能仍承载于 `templates/*.html` (Go Templates) + Vanilla JS；Bridge 是独立的 SwiftUI 客户端体系。修改 `dashboard.html` 等历史页面时，需预期其代码体量大且存在冗余定义。
- **Bridge 资料库红线**: `soniclens-bridge` 的专辑/曲目列表必须坚持“本地 SQLite 轻量索引 + FTS5 搜索 + `/api/library/sync` 增量同步 + `library_updated(version)` WebSocket 推送 + 详情页懒加载”模式，禁止回退到“远端分页 + 本地数组过滤/排序”的混合设计。
- **Bridge 模块边界红线**: `soniclens-bridge` 当前包含 `SoniclensBridgeMac`、`SoniclensBridgePad`、`SoniclensBridgePhone` 三个产品线。新增能力时应优先下沉到 `SoniclensCore` / `ViewModels` 共享层，再按端实现容器与交互差异，禁止将 macOS `AppKit` 窗口语义泄漏到 iPad/iPhone。详细边界见 `soniclens-bridge/Docs/CLIENT_MODULE_BOUNDARY.md`。
- **Bridge 音眸渲染红线**: 音眸的数据契约与标签语义必须以 `core/ai/agent.go` 的 `GetTrackInsightSchema()` 和 `templates/lyrics_live.html` 为唯一事实标准；`analysis_by_section`、`<original>/<translation>/<explain>` 解析、主 insight 选择与富渲染树必须收口到共享层，端差异只允许体现在外层容器与排版。
- **细节归档原则**: 具体 UI 规格、排序规则、专辑详情布局、多碟展示与高密度滚动性能策略，不再堆叠在本核心记忆中；应优先维护在 `soniclens-bridge/Docs/` 下的专题文档，以及 `memory/2026-03-15/library_index_sync_bridge_feature_manifest.md`、`memory/2026-03-18/bridge_dual_platform_product_line_manifest.md`、`memory/2026-03-20/bridge_album_library_three_platform_closure_manifest.md`、`memory/2026-03-20/bridge_three_platform_insight_closure_manifest.md` 等特性清单。

---

## 2. 开发规范与最佳实践 (依 .cursor/rules 对齐)

### 2.1 后端编码规约 (Go)
- **命名与风格**: 严格遵循 Uber Go 风格。变量/函数使用 `PascalCase` 或 `camelCase`，包名全小写。
- **枚举规范**: 对外可复用的枚举必须统一放在 `common` 包（优先 `common/enum.go`）；枚举需声明独立类型并使用类型化常量，底层类型限定为 `string`、`int8` 或 `uint8`，禁止在业务文件内分散定义裸常量枚举。
- **注释要求**: **所有注释必须使用中文**。导出的函数、类型必须有阐述“为什么”的注释。
- **日志记录**: 使用结构化日志 `log.Info/Error(ctx, "msg", zap.Field)`。错误日志必须携带 `zap.Error(err)`。
- **错误处理**: 禁止忽略错误。使用 `%w` 进行错误包装以保留调用链。
- **测试边界**: 默认 `go test ./...` 必须可在无本地音乐文件、无私有配置、无外部服务凭据环境下稳定运行；依赖真实文件系统、真实第三方 API 或本地私有配置的测试统一使用 `integration` build tag 隔离。

### 2.2 数据库设计指南 (GORM)
- **代码位置规范**：**所有数据库 CRUD 操作必须定义在 `internal/model/` 下对应的表 `go` 文件中**。严禁将原生的数据库查询/更新逻辑散落在各个业务模块（Logic 层）里。
- **复用性原则**：优先封装可复用的模型方法（如 `GetOrCreateAlbum`），减少重复的 SQL 逻辑，确保数据访问层（DAO）的纯粹性。
- **事务边界**：多表事务允许由 Logic 层负责编排，但**事务入口必须由 `internal/model/` 提供**（例如 `InTx`）；Logic 层在事务闭包中只能调用 DAO，不能直接书写 `tx.Where/First/Save/Create/Updates`。
- **DAO 形态**：需要参与事务的 DAO 优先提供 `ctx` 入口与 `Tx` 入口两套能力，公开接口复用事务内核，避免同一 SQL 在上下文版和事务版之间漂移。
- **上下文绑定**: 所有数据库操作必须使用 `.WithContext(ctx)` 确保链路可追踪。
- **并发控制**: 重要更新（如 `PlayCount` 增加）应实现基于 `version` 字段的**乐观锁**机制。
- **索引原则**: 复合索引遵循最左前缀原则。新系统必须包含 `created_at` 和 `updated_at`。

---

## 3. 核心模型参考清单 (internal/model)

该清单映射了 `internal/model/` 目录下的核心实体及其关键职责，AI Agent 在涉及数据变更时应参考对应的模型文件：

- **[track.go](./internal/model/track.go)**: 
    - **核心索引**: `uidx_t_aatdntn` (Artist, Album, Track, DiscNumber, TrackNumber)。
    - **功能**: 曲目元数据、播放次数统计、乐观锁版本控制。
    - **补充**: `Source` / `BundleID` / `UniqueID` / `ReleaseDate` 只能作为弱线索，不能再被视为稳定主键；低置信来源（如 Apple Music 流媒体、Roon 简化播放态）只允许命中既有曲目并增加播放次数，不允许新建 `album` / `track_album`。
- **[album.go](./internal/model/album.go)**: 
    - **核心索引**: `uidx_album_artist_name_release_date`。
    - **功能**: 专辑元数据、同步状态 (SyncStatus) 管理。
    - **补充**: `Album` 的 GORM Hook 会写入 `library_change_log`，用于 Bridge 资料库索引增量同步。
    - **补充**: `GetOrCreateAlbum` 不可再把播放器/文件上报的曲目级 `release_date` 视为稳定专辑身份；精确命中失败后必须回退复用同 `artist + name` 的现有专辑，缺失日期时优先复用 `sync_status=3` 的已深度维护专辑。
- **[track_album.go](./internal/model/track_album.go)**: 
    - **功能**: 维护曲目与专辑的多对多关联，支持碟号和轨道号的物理映射。
    - **规则**: 任何占位符匹配、专辑内曲目绑定、MusicBrainz 对齐都必须优先按 `(album_id, disc_number, track_number)` 处理，`track` 名称只能作为兜底条件。
    - **补充**: `DeepingMaintenance` 处理已听曲目时，`track_album` 的物理位置优先级高于 `track` 主表坐标；名称匹配必须先做繁简、括号和大小写归一。
    - **补充**: 对 `sync_status=3` 专辑补全未听曲目时，应优先创建真实 `track`（`play_count=0`）并建立 `track_album.track_id>0` 关联，不再新增 `track_id=0` 占位符。
- **[album_cleanup.go](./internal/model/album_cleanup.go)**:
    - **功能**: 清洗 `(artist, name)` 维度的重复专辑，统一迁移 `track_album` 与 `album_release_mb` 关联并删除冗余专辑。
    - **规则**: 主专辑优先保留 `sync_status` 更高、已确认 MusicBrainz 关联更多、挂载曲目更多的记录；默认将同名同作者下由曲目日期裂变出的多行视为脏数据。
- **[tx.go](./internal/model/tx.go)**:
    - **功能**: 提供 model 层统一事务入口，供 Logic 层组合多个 DAO 时使用。
    - **规则**: 后续所有跨表事务都应优先走 `model.InTx(...)`，不要再让 Logic 直接持有裸 GORM 事务细节。
- **[track_play_record.go](./internal/model/track_play_record.go)**: 
    - **功能**: 详尽的听歌流水历史，用于统计周/月榜单及同步 Last.fm。
    - **补充**: 播放流水回填 `album_id` 时不可再使用低置信三元组兜底，避免弱来源播放记录误绑定到错误专辑。
    - **补充**: 播放流水现已显式记录 `resolved_track_id`、`resolution_status`、`resolution_confidence` 与 `library_applied`；实时 scrobble 应优先走 `ProcessTrackPlayRecord`，统一完成资料库写入与归因回填。
    - **补充**: 后台补归因与补写资料库时，应优先复用 `ReplayTrackPlayRecords` / `ProcessTrackPlayRecord`，不要再在命令层手工串联“查记录 -> 增播放 -> 回填状态”。
- **[config.go](./config/config.go)**:
    - **补充**: `playReplay` 用于控制播放流水自动补归因调度，默认应保持关闭，待手动 replay 验数稳定后再开启。
- **[library_change_log.go](./internal/model/library_change_log.go)**:
    - **功能**: 记录专辑与曲目的增删改事件，为 Bridge `/api/library/sync` 提供版本游标、upsert 集合与删除 tombstone。
- **[track_insight.go](./internal/model/track_insight.go)**: 
    - **功能**: AI 生成的歌曲赏析细节（背景、歌词翻译、时代背景）。
    - **补充**: 结构化 insight 的事实标准不是 Bridge 自行猜测的扁平字段，而是 `core/ai/agent.go` 中 `GetTrackInsightSchema()` 定义的 JSON Schema；`analysis_by_section` 是核心字段，`appreciate_analysis` 内允许出现 `<original>/<translation>/<explain>` 标签串。
    - **补充**: Bridge 客户端当前只展示后端排序后的第一条 insight；共享解码与富渲染规则位于 `soniclens-bridge/SoniclensCore/Models/LibraryModels.swift` 与 `soniclens-bridge/SoniclensBridge/Views/InsightDetailView.swift`。
- **[track_lyrics.go](./internal/model/track_lyrics.go)**: 
    - **功能**: 原始歌词与翻译歌词的持久化。
    - **补充**: `track_lyrics.synced` 只表示“可解析出至少一个合法 LRC 时间标签”，`[Verse]`/`[ar:...]` 这类标签不能再单独触发同步歌词状态。
- **[genre.go](./internal/model/genre.go)**: 
    - **功能**: 音乐流派库。
- **[dashboard_stat.go](./internal/model/dashboard_stat.go)**: 
    - **功能**: 复杂聚合统计逻辑（Top 艺术家、流派占比、年度统计）。
- **[init.go](./internal/model/init.go)**: 
    - **功能**: 数据库初始化与 AutoMigrate 配置。
- **[sql/ddl](./internal/model/sql/ddl)**:
    - **功能**: 维护当前 MySQL 实库对应的表结构 SQL 档案。
    - **规则**: 这里应与最新 MySQL 表结构保持一致，不再混入历史 SQLite 片段或过期补丁。
- **[sql/dml](./internal/model/sql/dml)**:
    - **功能**: 维护数据库初始化动作脚本。
    - **规则**: 这里只保留初始化语义，不保留历史业务数据快照；静态样本数据应使用独立导入方案。
- **[model_test.go](./internal/model/model_test.go)**:
    - **功能**: model 包 MySQL 方言单元测试基座入口（基于 GORM MySQL dialector + sqlmock）。
    - **规则**: DAO 层重构与事务治理优先补 model 级测试，优先校验 MySQL SQL 与事务顺序，避免再为 SQLite 兼容性让路。

---
*最后更新日期：2026-03-20 | 文档版本: v2.5*
AI MUST READ THIS FILE BEFORE MODIFYING CODE.
