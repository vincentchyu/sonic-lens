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
- 读接口缓存优先在 `api/` 层按路由挂载 Redis middleware，默认 TTL 5 分钟，可按接口覆盖；空结果走 3 秒短 TTL 负缓存，命中时要回写 `ETag` / `Cache-Control`，Redis 不可用时必须透明降级。
- Web 端页面逻辑主要落在 `templates/*.html`；Bridge 共享能力看 `soniclens-bridge/SoniclensCore/`，端侧容器与交互看 `soniclens-bridge/SoniclensBridge/ViewModels` 和 `soniclens-bridge/SoniclensBridge/Views`。
- iPhone 分享能力已收口到 `soniclens-bridge/SoniclensBridge/ShareKit/`：`Builder` 负责 payload 装配，`Template/iPhone` 负责版式，`Render` 负责长图/分页导出，`Action` 负责相册保存与系统分享，禁止继续在详情页里直接拼接快照导出逻辑。
- iPhone 分享预览的公共外壳已收口到 `SharePosterShell`：顶部公共标题、背景、玻璃拟态容器、封面式 hero、底部居中品牌信息统一复用，`TrackInfoPosterView`、`LyricsLongPosterView`、`InsightLongPosterView` 只允许保留各自正文内容。
- `SharePosterHeader` 的 hero 样式固定为“左侧封面、右侧歌名/艺人专辑、左下位置/指标、右下收藏态”结构，位置标签不再放在顶部。
- 涉及客户端边界、API 映射与构建验证时，优先查 `soniclens-bridge/Docs/`，不要只凭页面代码反推架构。

### 1.2 模块调用拓扑
`Main` -> `API` -> `Internal/Logic` -> `Internal/Model` -> `Core/DB`
`Scrobbler` -> `Internal/Logic` -> `Internal/Model`
`Bridge App` -> `ViewModels` -> `SoniclensCore` -> `API/WebSocket`

---

## 2. 开发规范与最佳实践 

### 2.1 后端编码规约 (Go)
- **命名与风格**: 严格遵循 Uber Go 风格。变量/函数使用 `PascalCase` 或 `camelCase`，包名全小写。
- **枚举规范**: 对外可复用的枚举必须统一放在 `common` 包（优先 `common/enum.go`）；枚举需声明独立类型并使用类型化常量，底层类型限定为 `string`、`int8` 或 `uint8`，禁止在业务文件内分散定义裸常量枚举。
- **注释要求**: **所有注释必须使用中文**。导出的函数、类型必须有阐述“为什么”的注释。
- **日志记录**: 使用结构化日志 `log.Info/Error(ctx, "msg", zap.Field)`。错误日志必须携带 `zap.Error(err)`。
- **日志打印要求**: **所有日志必须使用中文**。打印不同级别的日志（具体使用什么级别看紧急程度，不要滥用）, 关键的函数要出入口要打印。
- **异步协程红线**: 禁止直接写裸 `go func` / `go xxx(...)`；统一使用 `core/telemetry.GoSafe`、`GoSafeDetached` 或 `GoOnlySafe`，分别处理“新建异步 span”“脱离取消的异步 span”“仅 recover 不起长期 span”的场景，避免 panic 打崩进程并避免长循环 trace 失真。
- **可观测性红线**: 面向 SigNoz 的 tracing / metrics 统一走 `core/telemetry` 提供的全局 tracer/meter provider；HTTP 入站走 `otelgin`，Redis 走 `redisotel`，GORM 走 `gorm opentelemetry tracing plugin`，`database/sql` 连接池指标走 `otelsql.RegisterDBStatsMetrics`，不要再在这些标准链路旁边叠加手写重复 span。
- **错误处理**: 禁止忽略错误。使用 `%w` 进行错误包装以保留调用链。
- **测试边界**: 默认 `go test ./...` 必须可在无本地音乐文件、无私有配置、无外部服务凭据环境下稳定运行；依赖真实文件系统、真实第三方 API 或本地私有配置的测试统一使用 `integration` build tag 隔离。


### 2.2 数据库设计指南 (GORM)
- **代码位置规范**：**所有数据库 CRUD 操作必须定义在 `internal/model/` 下对应的表 `go` 文件中**。严禁将原生的数据库查询/更新逻辑散落在各个业务模块（Logic 层）里。
- **复用性原则**：优先封装可复用的模型方法（如 `GetOrCreateAlbum`），减少重复的 SQL 逻辑，确保数据访问层（DAO）的纯粹性。
- **事务边界**：多表事务允许由 Logic 层负责编排，但**事务入口必须由 `internal/model/` 提供**（例如 `InTx`）；Logic 层在事务闭包中只能调用 DAO，不能直接书写 `tx.Where/First/Save/Create/Updates`。
- **DAO 形态**：需要参与事务的 DAO 优先提供 `ctx` 入口与 `Tx` 入口两套能力，公开接口复用事务内核，避免同一 SQL 在上下文版和事务版之间漂移。
- **接口文档约束**：任何新增或修改 `/api/*` 路由都必须同步维护 `api/api.md`，至少更新对应的功能表和关键调用链；如果是重大接口能力变更，还要同时检查相关 `memory/YYYY-MM-DD/*.md` 特性清单是否需要补充。
- **上下文绑定**: 所有数据库操作必须使用 `.WithContext(ctx)` 确保链路可追踪。
- **并发控制**: 重要更新（如 `PlayCount` 增加）应实现基于 `version` 字段的**乐观锁**机制。
- **索引原则**: 复合索引遵循最左前缀原则。新系统必须包含 `created_at` 和 `updated_at`。

### 2.3 前端客户端规约
- **日志打印要求**: **所有日志必须使用中文**。打印不同级别的日志（具体使用什么级别看紧急程度，不要滥用）, 关键的函数要出入口要打印。
- **当前现状**: Web 端大量核心功能仍承载于 `templates/*.html` (Go Templates) + Vanilla JS；Bridge 是独立的 SwiftUI 客户端体系。修改 `dashboard.html` 等历史页面时，需预期其代码体量大且存在冗余定义。
- **Bridge 资料库红线**: `soniclens-bridge` 的专辑/曲目列表必须坚持“本地 SQLite 轻量索引 + FTS5 搜索 + `/api/library/sync` 增量同步 + `library_updated(version)` WebSocket 推送 + 详情页懒加载”模式，禁止回退到“远端分页 + 本地数组过滤/排序”的混合设计。
- **Bridge 专辑展示红线**: 专辑主名与版本说明必须以结构化字段展示，`album.name + album.name_subtitle` / `now_playing.album + album_subtitle` 是唯一事实源；列表、详情、最近播放、热门专辑和当前播放禁止各自重新从原始标题字符串猜括号后缀。
- **Bridge 模块边界红线**: `soniclens-bridge` 当前包含 `SoniclensBridgeMac`、`SoniclensBridgePad`、`SoniclensBridgePhone` 三个产品线。新增能力时应优先下沉到 `SoniclensCore` / `ViewModels` 共享层，再按端实现容器与交互差异，禁止将 macOS `AppKit` 窗口语义泄漏到 iPad/iPhone。详细边界见 `soniclens-bridge/Docs/CLIENT_MODULE_BOUNDARY.md`。
- **Bridge 性能状态红线**: `AppStore` 只允许承载低频全局态（连接、最近服务端、任务协调等）；高频播放态、收藏态必须收口到细粒度 `PlaybackStore` / `FavoriteStore`，并通过 Observation/typed environment 下发。密集列表、详情页、播放条禁止直接订阅 `AppStore` 的高频字段。
- **Bridge 资料库性能红线**: `LibraryViewModel` 的资料库刷新必须保持 single-flight；列表交互遵循“第一页先返回，总数异步补 + 请求 token 丢弃过期结果”模式。控制区反馈优先于统计完成，收藏变更优先 patch 可见行，禁止回退到“每次筛选/收藏都整页重载”的策略。
- **Bridge 本地索引红线**: `LibraryIndexStore` 当前以 schema `7` 为基线；`track_index.is_favorited_effective` 及 favorites / unreported / recent 复合索引属于长期约束，收藏筛选语义不得回退到运行时 `OR` 拼接。
- **Bridge 工程生成红线**: `soniclens-bridge/SoniclensBridge.xcodeproj` 是由 `soniclens-bridge/project.yml` 通过 `xcodegen generate` 生成的产物。任何 target、scheme、Info.plist 生成属性、extension 嵌入关系的改动，都必须先落在 `project.yml`，否则下一次 `xcodegen generate` 会回滚手改工程。
- **Bridge macOS 玻璃宿主红线**: macOS mini 播放条若需要真实 backdrop blur，必须使用窗口级 `NSVisualEffectView` 宿主长期承载，再把 SwiftUI 内容嵌入其中；禁止继续把 `NSVisualEffectView` 塞进 SwiftUI `background`、`clipShape`、按钮样式或普通 overlay 修饰链里，否则前后台切换时容易退化成实色背景。
- **Bridge 正在播放收藏红线**: 三端 `NowPlaying` 必须优先消费 `WS now_playing` 提供的 `apple_music_state`、`lastfm_state`、`favorite_state`。`favorite_pending` / `unfavorite_pending` 不能再被布尔位抹平；共享态应收口到 `SoniclensCore` 的 favorite projection，再由各端 UI 复用同一套状态推导与提示文案。
- **Bridge 正在播放进度红线**: 三端正在播放页与全局播放条的本地进度只允许在最近一次 `now_playing` 更新仍然新鲜时继续自增；如果 WS 静默超过短阈值，就必须自动冻结当前进度，收到新的 `now_playing` 后再恢复推进，禁止在暂停后继续空跑计时器。Bridge 客户端必须保留每条 `now_playing` 快照的接收时间作为新鲜度事实源，不能在重新进入页面时把旧快照当成“刚同步”的活跃播放。
- **Bridge 连接链路红线**: Bonjour 自动发现若已拿到解析地址，连接链路必须优先直连解析地址；连接过程中必须同时提供顶部阶段反馈、行内反馈、取消能力与全局断开入口，不能让用户处于“点了没反应”的状态。
- **Bridge 连接恢复红线**: 已连接过的服务端在下次启动时应优先做静默健康检查，成功后直接进 dashboard；失败时必须保留当前连接上下文并进入用户决策态，允许用户选择“退出当前连接”或“重新连接”，禁止软件在未告知用户的情况下自动断开。
- **Bridge URL 编码红线**: `soniclens-bridge` 所有 GET 请求的 query 参数必须统一走 `SoniclensCore/Networking/APIClient.swift` 的百分号编码收口，禁止在业务层手写 query string 或依赖 `+` 的隐式语义。曲名、艺人名、专辑名等元数据只允许传原始值，由共享网络层负责把 `+` 编码为 `%2B`，避免后端将 `+` 还原成空格。
- **Bridge 分享红线**: iPhone 分享首期只从 `TrackDetailView` 进入；可复用的是 ShareKit 的数据装配、渲染和动作层，不是 macOS 快照布局。系统分享只走单张长图；保存图片允许在“长图 / 分页”之间显式选择。音眸分享必须复用现有 `InsightTaggedContentParser` 标签语义并渲染全文 segment，不能退回成摘要卡片或大段纯文本。
- **Bridge 交互红线**: 排序、筛选等控制态要在控制区回显；高频状态不要直接扩散到密集列表。
- **Bridge 视觉热区红线**: 首页、正在播放页和其他常驻热区的动态背景、模糊材质、阴影与常驻动画必须提供性能模式或紧凑降级路径；`APIClient` 默认复用共享 `URLSession`，`PlayerViewModel` 这类热点 ViewModel 必须优先并行可并行请求并丢弃过期结果。
- **Bridge 音眸渲染红线**: 音眸的数据契约与标签语义必须以 `core/ai/agent.go` 的 `GetTrackInsightSchema()` 和 `templates/lyrics_live.html` 为唯一事实标准；`analysis_by_section`、`<original>/<translation>/<explain>` 解析、主 insight 选择与富渲染树必须收口到共享层，端差异只允许体现在外层容器与排版。`appreciate_analysis` 的每组原文/翻译/解读必须保持标签完整性，不能被切成一组一个标签标题的碎片卡。
- **Bridge 音眸长任务红线**: iPhone 音眸分析已改为“`/api/insight-jobs` 异步任务 + `WS insight_job_updated` 前台推送 + `GET /api/insight-jobs/:id` 恢复兜底 + `soniclens://insight-job/<id>` 深链回流”。详情页禁止再直接持有长时间 `POST /api/track-insight` / `POST /api/album-insight` 作为主调用链；统一通过 `SoniclensCore/Store/AppStore.swift` 挂载的 `InsightAnalysisCoordinator` 管理单活跃任务、路由快照和 Live Activity。
- **Bridge Live Activity 红线**: iOS 端长时任务必须关联 Live Activity 进度反馈；封面渲染必须走 `LiveActivityArtworkStore` 异步下载至本地，禁止在 Widget 侧直接触发网络请求。
- **Web Dashboard 维护红线**: 待处理专辑维护必须具备“实时 vs 冻结”对账能力，检测到 context stale 时强制提示刷新；手动维护路径必须支持位置重排与归因证据回填。
- **细节归档原则**: 具体 UI 规格、排序规则、专辑详情布局、多碟展示与高密度滚动性能策略，不再堆叠在本核心记忆中；应优先维护在 `soniclens-bridge/Docs/` 下的专题文档。

---

## 3. 核心模型参考清单 (internal/model)

该清单映射了 `internal/model/` 目录下的核心实体及其关键职责，AI Agent 在涉及数据变更时应参考对应的模型文件：

- **[track.go](./internal/model/track.go)**: 
    - **核心索引**: `uidx_t_aaastdntn` (Artist, Album, AlbumSubtitle, Track, DiscNumber, TrackNumber)。
    - **功能**: 曲目元数据、播放次数统计、乐观锁版本控制。
    - **补充**: `Source` / `BundleID` / `UniqueID` / `ReleaseDate` 只能作为弱线索，不能再被视为稳定主键；低置信来源（如 Apple Music 流媒体、Roon 简化播放态）只允许命中既有曲目并增加播放次数，不允许新建 `album` / `track_album`。
    - **补充**: `GetOrCreateTrackByIdentityTx`、`UpdateTrackCuratedMetadataTx` 和播放写入路径在落库前必须统一做 `UnityFixAll + ConversionSimplifiedFx`，保证 MusicBrainz 深度维护不会把繁体标题、艺人、专辑、专辑艺人和流派原样写回 `track`，同时保留对旧有简体存量行的回退匹配。
    - **补充**: `track.album_subtitle` 已成为正式字段；曲目资料库、播放归因与收藏待归因都要透传该字段，曲目稳定身份主键已升级为 `(artist, album, album_subtitle, track, disc_number, track_number)`，不要再回退到只按主名匹配。
- **[album.go](./internal/model/album.go)**: 
    - **核心索引**: `uidx_album_artist_name_subtitle_release_date`。
    - **功能**: 专辑元数据、同步状态 (SyncStatus) 管理。
    - **补充**: `Album` 的 GORM Hook 会写入 `library_change_log`，用于 Bridge 资料库索引增量同步。
    - **补充**: `GetOrCreateAlbum` 不可再把播放器/文件上报的曲目级 `release_date` 视为唯一稳定专辑身份；专辑身份现在以 `artist + name + name_subtitle + release_date` 共同约束，缺失日期时仍可优先复用 `sync_status=3` 的已深度维护专辑，但不能把不同 subtitle 的版本硬合并。
    - **补充**: `original_release_date` 现在是独立事实字段，只能在来源明确提供 `release-group.first-release-date` / 文件 `OriginalDate` 时填入；它不参与专辑身份匹配，也不能用当前 `release_date` 直接冒充。
    - **补充**: 专辑封面已升级为对象存储闭环：`album` 维护 `cover_art_url` / `cover_art_mime` / `cover_art_object_key`，实时播放链路优先复用对象存储 URL，失败时才回退内存封面缓存。
    - **补充**: `album.name_subtitle` 已用于承载 Deluxe / Remaster / Anniversary 等结构化版本说明；资料库同步、详情页和排行榜都应优先读取该字段，而不是再把 `name` 当成带括号的完整展示串。
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
    - **补充**: `track_play_record` 现已记录 `trace_id`、`root_span_id`、`trace_sampled`，用于把业务播放流水直接锚定到当前歌曲根 span；后续排障、replay 和调试页应优先复用这组字段反查 Jaeger/SigNoz，而不是再靠播放时间和曲名模糊匹配。
    - **补充**: 播放阈值达到时写入的 `track_play_record.cover_art_path` 是最近播放封面的客户端消费字段；`HandleTrackPlaybackThreshold` 必须在落库时同步收敛封面路径，最近播放首页应优先展示该路径对应的封面图片，时间只保留短标签或省略。
    - **补充**: `track_play_records.album_subtitle` 是最近播放、D1 镜像和待归因专辑上下文的专辑版本说明来源；播放入库时必须同步保留，最近播放与资料库索引的 fallback join 也必须按 `album_subtitle` 进行匹配，不能再只按 `artist + album` 兜底。
- **[track_favorite_event.go](./internal/model/track_favorite_event.go)**:
    - **功能**: 收藏事件表，用于“先记意图，再归因回填”。
    - **补充**: `track_favorite_event` 只表示待归因收藏意图，不得替代 `track` 表中的稳定收藏事实；对外读取必须统一走 logic 层 favorite projection 合成稳定态与 pending 态。
    - **补充**: `POST /api/favorite` 与 `WS now_playing` 必须同时输出 `apple_music_state`、`lastfm_state`、`favorite_state`；兼容布尔位 `apple_music` / `lastfm` 表示“有效收藏态”，`favorite_pending` 也应表现为 `true`。
    - **补充**: `track_favorite_event` 的身份查找已经纳入 `album_subtitle`；实时 scrobbler 的收藏探测必须优先复用 logic 层 favorite projection 缓存与版本失效机制，同歌稳态且 probe 未变化时，不要每轮重查 `track` / `track_favorite_event`，本地 `SetTrackFavorite` 写入后需主动使缓存失效。
- **[pending_album_work_item.go](./internal/model/pending_album_work_item.go)**:
    - **功能**: 待归因专辑工作项的冻结上下文、实时对比和显式刷新。
    - **补充**: `GetPendingAlbumWorkItemDetail` 会同时返回冻结的播放/点赞记录、实时 `live_group` 和 `context_stale`，前端可以据此提示用户是否刷新。
    - **补充**: 冻结上下文刷新必须走显式 `POST /api/pending-albums/work-items/:id/refresh-context`，查询接口只允许读，不允许隐式改写工作项。
    - **补充**: 待归因专辑维护现已支持手动兜底路径：`POST /api/pending-albums/work-items/:id/manual-maintenance` 可直接提交整专元数据与曲目表，并与 MusicBrainz 路径复用同一套“专辑结构维护 -> Replay 播放 -> 应用收藏 -> 工单完成”骨架。
    - **补充**: 手动维护成功后同样视为稳定专辑，`album.sync_status=3`、`pending_album_work_item.status=completed`；但手动路径不得伪造 `release_mb` / `album_release_mb` 关系。
    - **补充**: 手动维护时，`manual_tracks[].title` 必须优先于冻结上下文证据标题；`evidence_titles` 仅用于 replay 归因，不得反向覆盖 curated 曲名。若复用了历史 `resolved_track_id`，也必须允许把 `track.artist/album/track` 身份字段修正为本次手填值。
    - **补充**: v1 手动维护曲目表必须实时接收输入值，不能依赖 `change` 失焦事件；当 `disc_number/track_number` 变化时，前端需立即按位置重排，且唯一约束只在 `(disc_number, track_number)`，不限制同名曲。
    - **补充**: 当冻结上下文中的播放/收藏标题与手填曲名不一致时，pending album 维护流程必须在落库阶段显式把该工单冻结的 `track_play_records` / `track_favorite_event` 绑定到目标 `track.id`；不能只依赖后续 replay 再按旧标题找库，否则会出现 `track_album` 已修正但 `track` 表与回填状态仍停留在旧标题的错位。
    - **补充**: `pending_album_work_item.album_subtitle` 必须和冻结的播放/收藏上下文一起持久化，避免 Deluxe / Anniversary 专辑在待归因列表、详情和手动维护之间丢失版本说明。
- **[config.go](./config/config.go)**:
    - **补充**: `playReplay` 用于控制播放流水自动补归因调度，默认应保持关闭，待手动 replay 验数稳定后再开启。
- **[internal/sync/d1_sync.go](./internal/sync/d1_sync.go)**:
    - **补充**: D1 同步必须走“单飞 + 启动串行 + 按表更新时间戳增量”闭环，`SyncAll` 进入后要先抢占运行锁，定时器触发时若已有同步在跑应直接跳过，避免首轮全量和定时重入同时刷 D1 额度。
    - **补充**: D1 直连 `database/sql` 链路统一走 `otelsql.Open(...)` + `RegisterDBStatsMetrics(...)`，不要再让同步侧 SQL 调用游离在主库 tracing/metrics 体系之外。
    - **补充**: D1 镜像表 `track_play_records` 与 `top_album_stat` 现已同步维护 `album_subtitle`；新增镜像字段时要同时补齐建表、补列、迁移复制和 batch upsert 四条链路，不能只改单侧 schema。
- **[library_change_log.go](./internal/model/library_change_log.go)**:
    - **功能**: 记录专辑与曲目的增删改事件，为 Bridge `/api/library/sync` 提供版本游标、upsert 集合与删除 tombstone。
- **[track_insight.go](./internal/model/track_insight.go)**: 
    - **功能**: AI 生成的歌曲赏析细节（背景、歌词翻译、时代背景）。
    - **补充**: 结构化 insight 的事实标准不是 Bridge 自行猜测的扁平字段，而是 `core/ai/agent.go` 中 `GetTrackInsightSchema()` 定义的 JSON Schema；`analysis_by_section` 是核心字段，`appreciate_analysis` 内允许出现 `<original>/<translation>/<explain>` 标签串。
    - **补充**: Bridge 客户端现在优先消费后端返回的 `recommended_insight_id` 选中默认 insight；共享解码与富渲染规则位于 `soniclens-bridge/SoniclensCore/Models/LibraryModels.swift` 与 `soniclens-bridge/SoniclensBridge/Views/InsightDetailView.swift`。
    - **补充**: `track_insight_feedbacks` 只承载曲目反馈，不再和专辑反馈混表；`/api/insights/:id/feedbacks` 需要按 `analysis_target_type` 路由到对应表。
    - **补充**: 单用户 Bridge 反馈语义统一为“我的反馈 / 历史反馈 / 待修正”，列表态只读取摘要，不展示所谓“社区评论”。
    - **补充**: `reason_codes` 必须与 `common.InsightFeedbackReason` 保持一致，后端只接受白名单值并按该枚举顺序归一；`TopReasonCodes` 与 prompt 注入都基于这套归一后的原因标签。
- **[album_insight.go](./internal/model/album_insight.go)**:
    - **功能**: AI 生成的专辑级深度分析结果（整专主题、文学解读、作者动机、哲学反思、时代语境）。
    - **补充**: 专辑 insight 不是重新逐曲分析歌词，而是基于 `track_album` 曲序和已存在的 `track_insight` 做二次聚合；每首歌只允许带入“总分最高、同分最新”的那条曲目 insight，并通过 `recommended_insight_id` 显式标记默认推荐版本。
    - **补充**: 结构化专辑契约以 `core/ai/agent.go` 中 `GetAlbumInsightSchema()` 为准，`metadata` 需保留 `album_id`、`total_tracks`、`analyzed_tracks` 与 `selected_track_insight_ids`，便于前端和后台追踪来源。
    - **补充**: `album_insight_feedbacks` 已独立建表；专辑分析会注入该表中的历史差评作为 `FeedbackContext`，`RecordAlbumFeedback` 负责回写点赞/点踩计数。
    - **补充**: 曲目/专辑反馈事件已补齐 `reason_codes`、`section_key`、`source_platform`；再生成时要优先整理成高频问题标签、重点分区和最近负反馈摘要，而不是直接无序拼接 comment。
- **[insight_job.go](./internal/model/insight_job.go)**:
    - **功能**: 承载曲目/专辑音眸的异步任务状态、客户端平台、Live Activity push token 与结果可用性，作为长调用恢复事实源。
    - **补充**: `/api/insight-jobs*` 是 iPhone 音眸的主调用链；`track-insight` / `album-insight` 继续只承担结果读取与兼容旧入口，不再承担 iPhone 首选长任务调度。
    - **补充**: 任务状态广播统一走 `core/websocket` 的 `insight_job_updated`；Bridge 前台优先消费 WS，App 回前台或深链回流时再用 `GET /api/insight-jobs/:id` 做短链路对账。
    - **补充**: 任务终态应尽量写回 `result_insight_id`；Bridge 详情页优先通过 `GET /api/insights/:id?analysis_target_type=...` 闭环读取本次任务产物，避免再按 artist/album/track 回查漂移到历史记录。
    - **补充**: iPhone 本地 Live Activity 创建不能强依赖 push token；若 `Activity.request(..., pushType: .token)` 因 capability/系统环境失败，应自动降级为本地 Live Activity，先保证灵动岛可见，再单独补 APNs 远程终态链路。
    - **补充**: 仅有 `SoniclensActivities/` 源码不够，`soniclens-bridge/SoniclensBridge.xcodeproj` 必须实际包含并嵌入 `SoniclensActivities.appex` Widget Extension target；若安装包里没有 `PlugIns/SoniclensActivities.appex`，真机不会显示灵动岛，即使主 App 已带 `NSSupportsLiveActivities`。
- **[llm_call_log.go](./internal/model/llm_call_log.go)**:
    - **功能**: 大模型调用流水审计与恢复现场。
    - **补充**: 日志必须区分分析对象类型，`analysis_target_type` 与 `target_key` 是主查询维度，`target_metadata` 专门保存对象元数据，`track_info` 只允许作为兼容展示字段，不能再承载元数据。
    - **补充**: AI 模型选择链路已升级为“平台 + 模型”双字段；`llm_call_logs.provider/model` 必须记录实际调用的平台与模型，`target_metadata` 需补充 `requested_provider`、`requested_model`、`effective_provider`、`effective_model`，用于回放客户端真实选择与后端最终落点。
    - **补充**: MySQL 老库补列由 `internal/model/schema_llm_call_log.go` 兜底，`internal/model/init.go` 初始化阶段会自动执行，避免表结构缺失导致 1146。
- **[core/ai/](./core/ai/)**:
    - **补充**: AI provider 抽象已拆分为“平台工厂 + 运行时 LLMProvider”，平台枚举统一以 `common.AIModelPlatform` 为准；`/api/ai-models` 只返回平台列表，`/api/ai-models/:platform/models` 返回模型目录。
    - **补充**: 模型目录只走 Redis 读穿缓存，不新增 MySQL 模型目录表；Redis 不可用时必须透明降级为实时查询。
    - **补充**: `internal/logic/insight` 的 provider cache key 必须按 `platform + model` 组合，禁止继续只按 provider 复用实例，否则会串模型。
    - **补充**: Gemini SDK 统一通过 `genai.ClientConfig.HTTPClient` 注入 `core/telemetry.WrapHTTPClient(...)`；若配置了 `GeminiConfig.BaseURL`，需同步传给 SDK `HTTPOptions.BaseURL`，不要再让 baseUrl 变成悬空配置。
- **[core/telemetry/](./core/telemetry/)**:
    - **补充**: telemetry 已升级为 SigNoz OTLP gRPC exporter + MeterProvider 闭环；当未配置 OTLP endpoint 时回退 stdout trace exporter，避免本地无 collector 环境直接失败。
    - **补充**: 自建出站 `http.Client` 应优先通过 `core/telemetry.WrapHTTPClient(...)` 包装，确保 AI、歌词回源等外呼链路也能传播 trace。
    - **补充**: Telemetry 初始化完成后会输出一条启动自检日志，至少包含 exporter 类型、OTLP endpoint、sampler、runtime metrics、db stats metrics 开关；排查 SigNoz 未收数时先看这条日志，不要盲猜配置是否生效。
    - **补充**: 应用日志当前仍保持 `core/log` 本地文件写入策略；若需要把日志接入 SigNoz，优先使用本地 OpenTelemetry Collector `filelog` receiver 采集 `./.logs/go_lastfm-scrobbler.log*`，不要先在业务代码里叠加第二套应用内 logs exporter。
    - **补充**: SigNoz 日志 retention 与应用本地日志轮转是解耦的两套策略；常见做法是应用侧继续保留 7 天本地文件，SigNoz 管理端单独将日志 retention 收紧到 1 天。
    - **补充**: Scrobbler 正在播放链路的 trace 粒度必须坚持“单首歌一个长生命周期根 span + 按变化触发的阶段 span / event”模式；`resolve_now_playing`、`sync_favorite_state`、`handle_playback_threshold` 只应在换歌、收藏态变化或阈值命中时出现，禁止回退到每轮轮询都创建一组业务 span。
    - **补充**: 当前播放收藏态探测必须先用进程内热缓存吸收稳态轮询，再回退 Redis / 外部探测；Apple Music `IsFavorite` 默认 15 秒 TTL，Last.fm 未收藏态应采用递增退避，避免 Jaeger / SigNoz 被重复 `is_favorite`、Redis `GET cache:isFavorite:lastfm:*` 淹没。
- **[core/musicbrainz/](./core/musicbrainz/)**:
    - **补充**: `musicbrainzws2` 当前未暴露底层 `http.Client` 注入点；项目统一在 `core/musicbrainz` 包装层为 `SearchReleases`、`LookupRelease` 创建客户端 span，禁止通过反射去改第三方库私有字段。
- **[internal/model/init.go](./internal/model/init.go)**:
    - **补充**: GORM 链路追踪统一走 `gorm.io/plugin/opentelemetry/tracing`，数据库连接池指标统一通过 `otelsql.RegisterDBStatsMetrics` 注册；不要再在 GORM logger 里手写 SQL span。
- **[core/redis/](./core/redis/)**:
    - **补充**: Redis 标准 tracing / metrics 统一走 `redisotel`，自定义 hook 只保留日志，不再重复创建 Redis client span。
- **[core/objectstorage/](./core/objectstorage/)**:
    - **补充**: S3 / MinIO / R2 这类基于 AWS SDK v2 的对象存储链路，统一走 smithy 原生 OTel adapter 接到全局 tracer/meter provider，避免在对象存储调用外层再叠手写 span。
- **[internal/model/insight_list.go](./internal/model/insight_list.go)**:
    - **功能**: 音眸后台列表聚合，统一承载曲目与专辑两类解析摘要。
    - **补充**: `/api/insights/all` 必须显式支持 `analysis_target_type`，默认曲目 `track`，专辑 `album` 走独立查询与独立 UI tab，不能再把两类对象混在同一个列表语义里。
- **[track_lyrics.go](./internal/model/track_lyrics.go)**: 
    - **功能**: 原始歌词与翻译歌词的持久化。
    - **补充**: `track_lyrics.synced` 只表示“可解析出至少一个合法 LRC 时间标签”，`[Verse]`/`[ar:...]` 这类标签不能再单独触发同步歌词状态。
- **[genre.go](./internal/model/genre.go)**: 
    - **功能**: 音乐流派库。
- **[dashboard_stat.go](./internal/model/dashboard_stat.go)**: 
    - **功能**: 复杂聚合统计逻辑（Top 艺术家、流派占比、年度统计）。
    - **补充**: 首页热门艺术家头像已从统计事实表中拆出，`top-artists` 响应可以通过 `artist_profile` 资料映射补齐 `avatar_url` / `avatar_object_key` / `avatar_mime`；统计层只负责排名与计数，前端优先渲染头像，缺失时回退首字母圆牌。
    - **补充**: 热门专辑统计已显式输出 `album_subtitle`；聚合时应优先从 `album.name_subtitle` 回填，缺失时再回退播放流水里的 `album_subtitle`，保证首页排行榜和专辑详情展示一致。
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

*最后更新日期：2026-04-09 | 文档版本: v3.1*
AI MUST READ THIS FILE BEFORE MODIFYING CODE.
