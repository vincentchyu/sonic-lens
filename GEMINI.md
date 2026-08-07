# ⚡ SonicLens Quick Memory

这是一份关于 SonicLens 项目的长期记忆清单，整合了项目架构、核心逻辑、开发规范及关键“陷阱”规避方案。AI Agent 在进行任何代码变更前应阅读本清单。

---

## 0. 长期记忆管理协议 (Memory Protocol)

- **核心索引文件**：[memory_index.md](./memory_index.md) 是全站开发的特性历史索引。
- **更新守则**：**AI Agent 在完成重大特性开发、架构重构或核心逻辑修复后，必须执行以下操作**：
    1. 在 `memory/YYYY-MM-DD/` 线下创建详细的 `feature_manifest.md` 特性清单。
    2. 将该清单挂载到 `memory_index.md` 的顶部。
    3. 同步审查并更新本 `GEMINI.md` 文件，确保“核心业务记忆”章节反映最新的逻辑现状。
- **公众号文章生成规约**: 详情见 [wechat_article_generation_constraint.md](./output/wechat_article_generation_constraint.md) 

---

## 0.1 约束抽象母规则

以下四条是当前 SonicLens 的高频母规则；后续新增细则时，优先判断自己是在补充哪一条，而不是继续堆叠零散规则。

- **结构化事实优先**：专辑/曲目身份、展示文案、反馈状态、分析结果默认版本等，都必须优先消费结构化字段或显式契约；禁止回退到从标题字符串、隐式顺序或页面状态里二次猜测事实。
- **高频状态专仓化**：高频播放态、收藏态、资料库查询态、连接恢复态必须收口到共享 store / projection / coordinator；`AppStore`、页面容器和密集列表只消费派生值，不直接承载高频广播。
- **异步事件主链 + 查询对账**：WebSocket / 推送 / 异步任务是前台主通道，GET 查询是恢复和对账通道；长任务、播放态、连接态都要避免重新退回“单次长请求即真相”的旧模型。
- **生成物目录化 + 兼容入口化**：文章、分享物料、导出资源等生成物应落到独立目录，根目录只保留稳定入口、约束文件和兼容层；不要再把真实产物直接散落到仓库公共出口目录。

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
- Web 端页面逻辑主要落在 `templates/*.html`；后台总入口已轻量拆分为 `templates/admin.html` shell、`templates/admin/*.html` partial、`static/admin/admin.css` 和 `static/admin/*.js`，`/` 与 `/admin` 共用该入口。其中 `/api/dashboard/*` 仍表示“仪表盘统计域”，不要和后台入口命名混淆。Bridge 共享能力看 `soniclens-bridge/SoniclensCore/`，端侧容器与交互看 `soniclens-bridge/SoniclensBridge/ViewModels` 和 `soniclens-bridge/SoniclensBridge/Views`。
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
- **测试与 SQLite 限制规约**: `GlobalDBForSqlLite` 已被废弃。禁止在任何新生产代码中引入或使用对 `GlobalDBForSqlLite` 的依赖；仅为旧测试编译与运行兼容，允许 `GetDB()` 在全局 MySQL 实例为 `nil` 且全局 SQLite 实例不为 `nil` 时透明回退。未来编写测试或修改 DAO 时应逐步使用 Mock 数据库，避免污染全局状态。

### 2.3 客户端规约
- **日志打印要求**: **所有日志必须使用中文**。打印不同级别的日志（具体使用什么级别看紧急程度，不要滥用）, 关键的函数要出入口要打印。
- **当前现状**: Bridge 是独立的 SwiftUI 客户端体系；新增能力优先下沉到 `SoniclensCore`、`ViewModels` 与专题文档，禁止把共享逻辑散落在端侧页面里临时兜底。`soniclens-bridge` 当前包含 `SoniclensBridgeMac`、`SoniclensBridgePad`、`SoniclensBridgePhone` 三个产品线，端差异应停留在容器与交互层，禁止将 macOS `AppKit` 窗口语义泄漏到 iPad/iPhone。详细边界见 `soniclens-bridge/Docs/CLIENT_MODULE_BOUNDARY.md`。
- **Bridge 资料库红线**: 专辑/曲目列表必须坚持“本地 SQLite 轻量索引 + FTS5 搜索 + `/api/library/sync` 增量同步 + `library_updated(version)` WebSocket 推送 + 详情页懒加载”模式，禁止回退到“远端分页 + 本地数组过滤/排序”的混合设计。`LibraryViewModel` 刷新必须保持 single-flight，列表交互遵循“第一页先返回，总数异步补 + 请求 token 丢弃过期结果”模式；控制区反馈优先于统计完成，收藏变更优先 patch 可见行，禁止回退到“每次筛选/收藏都整页重载”。`LibraryIndexStore` 当前以 schema `7` 为基线，`track_index.is_favorited_effective` 及 favorites / unreported / recent 复合索引属于长期约束，收藏筛选语义不得回退到运行时 `OR` 拼接。
- **Bridge 专辑展示红线**: 专辑主名与版本说明必须以结构化字段展示，`album.name + album.name_subtitle` / `now_playing.album + album_subtitle` 是唯一事实源；列表、详情、最近播放、热门专辑和当前播放禁止各自重新从原始标题字符串猜括号后缀。
- **Bridge 状态与交互红线**: `AppStore` 只允许承载低频全局态（连接、最近服务端、任务协调等）；高频播放态、收藏态必须收口到细粒度 `PlaybackStore` / `FavoriteStore`，并通过 Observation/typed environment 下发。密集列表、详情页、播放条禁止直接订阅 `AppStore` 的高频字段；排序、筛选等控制态要在控制区回显，高频状态不要直接扩散到密集列表。
- **Bridge 工程生成红线**: `soniclens-bridge/SoniclensBridge.xcodeproj` 是由 `soniclens-bridge/project.yml` 通过 `xcodegen generate` 生成的产物。任何 target、scheme、Info.plist 生成属性、extension 嵌入关系的改动，都必须先落在 `project.yml`，否则下一次 `xcodegen generate` 会回滚手改工程。
- **Bridge macOS 玻璃宿主红线**: macOS mini 播放条若需要真实 backdrop blur，必须使用窗口级 `NSVisualEffectView` 宿主长期承载，再把 SwiftUI 内容嵌入其中；禁止继续把 `NSVisualEffectView` 塞进 SwiftUI `background`、`clipShape`、按钮样式或普通 overlay 修饰链里，否则前后台切换时容易退化成实色背景。
- **Bridge 正在播放红线**: 三端 `NowPlaying` 必须优先消费 `WS now_playing` 提供的 `apple_music_state`、`lastfm_state`、`favorite_state`，`favorite_pending` / `unfavorite_pending` 不能再被布尔位抹平；共享态应收口到 `SoniclensCore` 的 favorite projection，再由各端 UI 复用同一套状态推导与提示文案。正在播放页与全局播放条的本地进度只允许在最近一次 `now_playing` 更新仍然新鲜时继续自增；如果 WS 静默超过短阈值，就必须自动冻结当前进度，收到新的 `now_playing` 后再恢复推进，禁止在暂停后继续空跑计时器。客户端必须保留每条 `now_playing` 快照的接收时间作为新鲜度事实源，不能在重新进入页面时把旧快照当成“刚同步”的活跃播放；`WS now_playing` 也是“存在播放对象”的最高优先级事实源，静默检测只允许影响进度推进与状态 banner，不允许把仍持有的播放对象直接降级成“无活动播放”空卡片。
- **Bridge 连接与恢复红线**: Bonjour 自动发现若已拿到解析地址，连接链路必须优先直连解析地址；连接过程中必须同时提供顶部阶段反馈、行内反馈、取消能力与全局断开入口，不能让用户处于“点了没反应”的状态。已连接过的服务端在下次启动时应优先做静默健康检查，成功后直接进 dashboard；失败时必须保留当前连接上下文并进入用户决策态，允许用户选择“退出当前连接”或“重新连接”，禁止软件在未告知用户的情况下自动断开。
- **Bridge URL 编码红线**: `soniclens-bridge` 所有 GET 请求的 query 参数必须统一走 `SoniclensCore/Networking/APIClient.swift` 的百分号编码收口，禁止在业务层手写 query string 或依赖 `+` 的隐式语义。曲名、艺人名、专辑名等元数据只允许传原始值，由共享网络层负责把 `+` 编码为 `%2B`，避免后端将 `+` 还原成空格。
- **Bridge 分享与音眸红线**: iPhone 分享首期只从 `TrackDetailView` 进入；可复用的是 ShareKit 的数据装配、渲染和动作层，不是 macOS 快照布局。系统分享只走单张长图；保存图片允许在“长图 / 分页”之间显式选择。音眸分享必须复用现有 `InsightTaggedContentParser` 标签语义并渲染全文 segment，不能退回成摘要卡片或大段纯文本。音眸的数据契约与标签语义必须以 `core/ai/agent_insight_track.go` 的 `GetTrackInsightSchema()`、`core/ai/agent_insight_album.go` 的 `GetAlbumInsightSchema()` 和 `templates/pages/lyrics_live.html` 为唯一事实标准；`analysis_by_section`、`<original>/<translation>/<explain>` 解析、主 insight 选择与富渲染树必须收口到共享层，端差异只允许体现在外层容器与排版，`appreciate_analysis` 的每组原文/翻译/解读必须保持标签完整性，不能被切成一组一个标签标题的碎片卡。iPhone 音眸分析已改为“`/api/insight-jobs` 异步任务 + `WS insight_job_updated` 前台推送 + `GET /api/insight-jobs/:id` 恢复兜底 + `soniclens://insight-job/<id>` 深链回流”；详情页禁止再直接持有长时间 `POST /api/track-insight` / `POST /api/album-insight` 作为主调用链，统一通过 `SoniclensCore/Store/AppStore.swift` 挂载的 `InsightAnalysisCoordinator` 管理单活跃任务、路由快照和 Live Activity。iOS 端长时任务必须关联 Live Activity 进度反馈；封面渲染必须走 `LiveActivityArtworkStore` 异步下载至本地，禁止在 Widget 侧直接触发网络请求。
- **Bridge 视觉热区红线**: 首页、正在播放页和其他常驻热区的动态背景、模糊材质、阴影与常驻动画必须提供性能模式或紧凑降级路径；`APIClient` 默认复用共享 `URLSession`，`PlayerViewModel` 这类热点 ViewModel 必须优先并行可并行请求并丢弃过期结果。
- **Bridge macOS 开发红线**:
  - **AppKit 与 SwiftUI 边界隔离**: macOS 专属代码（如 `NSWindow` 操作、`NSVisualEffectView` 窗口级宿主、窗口控制按钮控制）严禁泄漏到 iPad/iPhone 目标代码中；跨平台代码需使用 `#if os(macOS)` 或抽象平台适配层 (`PlatformUI.swift`) 隔离。
  - **NSVisualEffectView 玻璃宿主**: macOS mini 播放条或高阶透明材质必须由窗口级/宿主级 `NSVisualEffectView` 长期承载后再嵌入 SwiftUI 内容，严禁直接在 SwiftUI 修饰链 (`background`/`clipShape`) 中简单放置以避免前后台切换时退化为实色。
  - **XcodeGen 工程驱动**: 任何 target/scheme/Info.plist 更改必须更新 `project.yml` 并运行 `xcodegen generate`，严禁直接在 Xcode 界面上手修改 `.xcodeproj` 文件，避免被生成工具覆盖。
  - **高频状态 Observation 隔离**: 密集列表和播放常驻组件禁止直接订阅全局 `AppStore` 的高频更新，高频状态必须收口到细粒度的 `PlaybackStore` / `FavoriteStore`；复杂列表更新需保持 single-flight 并使用请求 token 丢弃过期异步结果。
  - **响应式 UI 性能模式**: 常驻热区（如正在播放沉浸背景、歌词滚动与高密度网格）必须包含紧凑/低性能设备降级路径，避免开销较大的实时模糊与大范围动画阻塞主线程。
- **细节归档原则**: 具体 UI 规格、排序规则、专辑详情布局、多碟展示与高密度滚动性能策略，不再堆叠在本核心记忆中；应优先维护在 `soniclens-bridge/Docs/` 下的专题文档。

### 2.4 Web 端规约
- **日志打印要求**: **所有日志必须使用中文**。打印不同级别的日志（具体使用什么级别看紧急程度，不要滥用）, 关键的函数要出入口要打印。
- **当前现状**: Web 端大量核心功能仍承载于 Go Templates + Vanilla JS。Admin 后台总入口已拆分为 shell + partial + 静态 CSS/JS，后续修改 `admin.html` 时只允许维护页面骨架和 `{{ template ... }}` 入口，大段 HTML 应进入 `templates/admin/`，样式进入 `static/admin/admin.css`，脚本进入 `static/admin/*.js`；列表页 loading / empty / error 状态优先走 `static/admin/ui-state.js` 的 `renderAdminLoading/Empty/Error`；`dashboard` 命名只保留给仪表盘统计子域和既有统计 API。
- **Web Dashboard 维护红线**: 待处理专辑维护必须具备“实时 vs 冻结”对账能力，检测到 context stale 时强制提示刷新；手动维护路径必须支持位置重排与归因证据回填。

---

## 3. 核心模型与相关基础设施参考清单

本节以 `internal/model/` 为主，同时收录与模型事实强绑定的配置、同步与基础设施入口。AI Agent 在涉及数据变更、异步链路、音眸任务或可观测性改动时，应优先参考对应文件，而不是只凭调用方反推。

### 3.1 核心领域模型

- **[track.go](./internal/model/track.go)**:
    - **核心索引**: `uidx_t_aaastdntn` `(Artist, Album, AlbumSubtitle, Track, DiscNumber, TrackNumber)`。
    - **功能**: 曲目元数据、播放次数统计、乐观锁版本控制。
    - **规则**: `Source` / `BundleID` / `UniqueID` / `ReleaseDate` 只能作为弱线索；低置信来源只允许命中既有曲目并增加播放次数，不允许新建 `album` / `track_album`。`GetOrCreateTrackByIdentityTx`、`UpdateTrackCuratedMetadataTx` 与播放写入路径落库前必须统一做 `UnityFixAll + ConversionSimplifiedFx`。`track.album_subtitle` 已成为正式字段，曲目稳定身份主键升级为 `(artist, album, album_subtitle, track, disc_number, track_number)`，不得回退到只按主名匹配；落库前及查询定位前需自动剥离 ` - EP/Single/LP` 后缀，确保与 album 表防增量策略一致。
- **[album.go](./internal/model/album.go)**:
    - **核心索引**: `uidx_album_artist_name_subtitle_release_date`。
    - **功能**: 专辑元数据、同步状态 `SyncStatus` 管理。
    - **规则**: `Album` 的 GORM Hook 会写入 `library_change_log`。`GetOrCreateAlbum` 必须以 `artist + name + name_subtitle + release_date` 共同约束专辑身份；缺失日期时可优先复用 `sync_status=3` 的已深度维护专辑，但不得把不同 subtitle 的版本硬合并。`original_release_date` 是独立事实字段，不参与身份匹配。`album` 维护 `cover_art_url` / `cover_art_mime` / `cover_art_object_key` 的对象存储闭环；`album.name_subtitle` 是 Deluxe / Remaster / Anniversary 等结构化版本说明的唯一正式承载字段；`album.release_type` 用于承载从后缀中剥离出来的 `ep`/`single`/`lp` 类型枚举，落库前必须统一经由 `normalizeAlbumReleaseTypeSuffix` 自动规整并剥除后缀以保存干净专辑名。
- **[track_album.go](./internal/model/track_album.go)**:
    - **功能**: 维护曲目与专辑的多对多关联以及碟号/轨道号物理映射。
    - **规则**: 占位符匹配、专辑内曲目绑定、MusicBrainz 对齐都必须优先按 `(album_id, disc_number, track_number)`，`track` 名称只能做兜底。`DeepingMaintenance` 处理已听曲目时，`track_album` 物理位置优先级高于 `track` 主表坐标，名称匹配必须先做繁简、括号和大小写归一。对 `sync_status=3` 专辑补全未听曲目时，应优先创建真实 `track`（`play_count=0`）并建立 `track_album.track_id>0` 关联，不再新增 `track_id=0` 占位符；在 `upsertTrackAlbumTx` 关系写入时设计有 `force` 强制覆盖标志，在重复专辑合并清洗时需指定 `force = true` 强行绕过 `sync_status=3` 的精选锁定拦截，避免关系静默丢失。
- **[album_cleanup.go](./internal/model/album_cleanup.go)**:
    - **功能**: 清洗 `(artist, name)` 维度的重复专辑，统一迁移 `track_album` 与 `album_release_mb` 关联并删除冗余专辑。
    - **规则**: 主专辑优先保留 `sync_status` 更高、已确认 MusicBrainz 关联更多、挂载曲目更多的记录；同名同作者下由曲目日期裂变出的多行默认视为脏数据。`CleanupReleaseTypeSuffixes` 支持一键扫出历史带有连字符发行格式的专辑，剥离后缀，自动对齐目标干净专辑并安全合并其曲目与 MB 关联。
- **[tx.go](./internal/model/tx.go)**:
    - **功能**: 提供 model 层统一事务入口。
    - **规则**: 所有跨表事务都应优先走 `model.InTx(...)`，不要再让 Logic 直接持有裸 GORM 事务细节。
- **[track_play_record.go](./internal/model/track_play_record.go)**:
    - **功能**: 听歌流水历史，用于统计、同步 Last.fm、资料库归因与排障。
    - **规则**: 回填 `album_id` 时不可再使用低置信三元组兜底。实时 scrobble 与后台 replay 都应优先复用 `ProcessTrackPlayRecord` / `ReplayTrackPlayRecords`，不要在命令层手工串联“查记录 -> 增播放 -> 回填状态”。`track_play_record` 现已显式记录 `resolved_track_id`、`resolution_status`、`resolution_confidence`、`library_applied`、`trace_id`、`root_span_id`、`trace_sampled`、`cover_art_path`、`album_subtitle` 与 `release_type`；最近播放、D1 镜像、待归因上下文与排障都应优先消费这些结构化字段，不能再退回时间/标题模糊匹配或 `artist + album` 兜底。其中 `release_type` 用于在播放落库时完整归档裁剪出的发行类型后缀，保证 unresolved 未归因状态前的流水原始线索完整不丢失。
- **[track_favorite_event.go](./internal/model/track_favorite_event.go)**:
    - **功能**: 收藏事件表，用于“先记意图，再归因回填”。
    - **规则**: `track_favorite_event` 只表示待归因收藏意图，不得替代 `track` 表中的稳定收藏事实；事件表会显式记录 `provider_favorite`、`resolved_track_id`、`resolution_status`、`resolution_confidence` 与 `applied`，对外读取必须统一走 logic 层 favorite projection 合成稳定态与 pending 态。`POST /api/favorite` 与 `WS now_playing` 必须同时输出 `apple_music_state`、`lastfm_state`、`favorite_state`；兼容布尔位 `apple_music` / `lastfm` 表示有效收藏态，`favorite_pending` 也应表现为 `true`。收藏身份查找已纳入 `album_subtitle`，实时探测必须优先复用 projection 缓存与版本失效机制。
- **[pending_album_work_item.go](./internal/model/pending_album_work_item.go)**:
    - **功能**: 待归因专辑工作项的冻结上下文、实时对比、显式刷新与手动维护，同时作为待归因工单与正式专辑（Album）精选维护草稿 `staging_draft_json` 的统一持久化载体。
    - **规则**: `GetPendingAlbumWorkItemDetail` 必须同时返回冻结记录、实时 `live_group` 与 `context_stale`；冻结上下文刷新必须走显式 `POST /api/pending-albums/work-items/:id/refresh-context`，查询接口只允许读。待归因工单与正式专辑（Album）在“重新搜索 => 选定 MB => 精选维护”流程中**100% 复用同一个预审与差异比对弹窗 (`pendingAlbumDiffPreviewModal`)、同一套微调草稿保存 (`staging_draft_json`) 与草稿自动加载/拉取刷新机制**。已完成的工单也可随时点开预审弹窗进行只读审查；正式专辑的草稿通过 `SaveAlbumStagingDraftDB` 绑定至对应的 `resolved_album_id` 工单。手动维护路径 `POST /api/pending-albums/work-items/:id/manual-maintenance` 与正式专辑 `POST /api/albums/:id/musicbrainz/apply-maintenance` 均通过事务级校验落库。
- **[library_change_log.go](./internal/model/library_change_log.go)**:
    - **功能**: 记录专辑与曲目的增删改事件，为 Bridge `/api/library/sync` 提供版本游标、upsert 集合与删除 tombstone。
- **[track_insight.go](./internal/model/track_insight.go)**:
    - **功能**: 曲目级音眸结果与反馈。
    - **规则**: 曲目 insight 的事实标准是 `core/ai/agent_insight_track.go` 中 `GetTrackInsightSchema()` 定义的 JSON Schema，`analysis_by_section` 是核心字段，`appreciate_analysis` 允许 `<original>/<translation>/<explain>` 标签串。Bridge 默认应优先消费后端返回的 `recommended_insight_id`。`track_insight_feedbacks` 只承载曲目反馈；单用户 Bridge 反馈语义固定为“我的反馈 / 历史反馈 / 待修正”，`reason_codes` 必须与 `common.InsightFeedbackReason` 对齐并按白名单归一。
- **[album_insight.go](./internal/model/album_insight.go)**:
    - **功能**: 专辑级音眸结果与反馈。
    - **规则**: 专辑 insight 必须基于 `track_album` 曲序和已存在的 `track_insight` 二次聚合，不是重新逐曲分析；每首歌只允许带入“总分最高、同分最新”的那条曲目 insight。契约以 `core/ai/agent_insight_album.go` 中的 `GetAlbumInsightSchema()` 为准，`metadata` 必须保留 `album_id`、`total_tracks`、`analyzed_tracks`、`selected_track_insight_ids`。`album_insight_feedbacks` 已独立建表，再生成时应优先消费归一后的 `reason_codes`、`section_key`、`source_platform` 与最近负反馈摘要。
- **[insight_job.go](./internal/model/insight_job.go)**:
    - **功能**: 曲目/专辑音眸异步任务状态、客户端平台、Live Activity push token 与结果可用性，是长调用恢复事实源。
    - **规则**: `/api/insight-jobs*` 是 iPhone 音眸主调用链；`track-insight` / `album-insight` 只承担结果读取与兼容旧入口。任务表真实表名为 `insight_job`，包含 `target_key`、`provider/model`、display name、`client_platform`、`live_activity_push_token`、`result_insight_id`、`result_available` 等字段；MySQL 结构以 `sql/ddl/insight_job.sql` 与 model struct 为准，稳定期不再依赖运行时 `ensure*Schema()` 自动补库。任务状态广播统一走 `core/websocket` 的 `insight_job_updated`；Bridge 前台优先消费 WS，回前台或深链回流时再用 `GET /api/insight-jobs/:id` 对账。终态应尽量写回 `result_insight_id`；本地 Live Activity 创建不能强依赖 push token，必要时必须降级为本地 Live Activity。`SoniclensActivities.appex` 也必须真正嵌入安装包，否则真机不会显示灵动岛。
- **[insight_list.go](./internal/model/insight_list.go)**:
    - **功能**: 音眸后台列表聚合，统一承载曲目与专辑两类解析摘要。
    - **规则**: `/api/insights/all` 必须显式支持 `analysis_target_type`；默认曲目 `track`，专辑 `album` 必须走独立查询与独立 UI tab，不能混成同一个列表语义。
- **[llm_call_log.go](./internal/model/llm_call_log.go)**:
    - **功能**: 大模型调用流水审计与恢复现场。
    - **规则**: `analysis_target_type` 与 `target_key` 是主查询维度，`target_metadata` 专门保存对象元数据，`track_info` 只允许作为兼容展示字段。AI 模型选择链路已升级为“平台 + 模型”双字段，`llm_call_logs.provider/model` 与 `target_metadata` 必须同时保留 requested/effective 信息；新增 `job_id` 关联字段，支持通过 `job_id` 强关联聚合多轮分析流水并进行排序展示，在后台展示时当 `job_id` 为空时自动 fallback 至默认名字关联。稳定期结构以 DDL 与 model struct 为准。
- **[track_lyrics.go](./internal/model/track_lyrics.go)**:
    - **功能**: 原始歌词与翻译歌词持久化。
    - **规则**: `track_lyrics.synced` 只表示“可解析出至少一个合法 LRC 时间标签”，`[Verse]` / `[ar:...]` 这类标签不能单独触发同步歌词状态。
- **[genre.go](./internal/model/genre.go)**:
    - **功能**: 音乐流派库。
- **[dashboard_stat.go](./internal/model/dashboard_stat.go)**:
    - **功能**: Top 艺术家、流派占比、年度统计等复杂聚合。
    - **规则**: `top-artists` 响应中的头像应通过 `artist_profile` 补齐，统计层只负责排名与计数；热门专辑统计必须显式输出 `album_subtitle`，优先从 `album.name_subtitle` 回填，缺失时再回退播放流水。
- **[init.go](./internal/model/init.go)**:
    - **功能**: 数据库初始化、AutoMigrate 与 model 链路可观测性挂载。
    - **规则**: GORM tracing 统一走 `gorm.io/plugin/opentelemetry/tracing`，数据库连接池指标统一通过 `otelsql.RegisterDBStatsMetrics` 注册；不要再在 GORM logger 里手写 SQL span。
- **[sql/ddl](./internal/model/sql/ddl)**:
    - **功能**: 维护当前 MySQL 实库对应的表结构 SQL 档案。
    - **规则**: 原则上应与最新 MySQL 实库结构保持一致，不再混入历史 SQLite 片段或过期补丁；稳定期以 `model struct + sql/ddl` 为主事实源，新增字段应优先补正式 migration / DDL，避免重新回到运行时自动补库模式。
- **[sql/dml](./internal/model/sql/dml)**:
    - **功能**: 维护数据库初始化动作脚本。
    - **规则**: 这里只保留初始化语义，不保留历史业务数据快照；静态样本数据应使用独立导入方案。
- **[model_test.go](./internal/model/model_test.go)**:
    - **功能**: model 包 MySQL 方言单元测试基座入口（GORM MySQL dialector + sqlmock）。
    - **规则**: DAO 层重构与事务治理优先补 model 级测试，优先校验 MySQL SQL 与事务顺序，避免再为 SQLite 兼容性让路。

### 3.2 相关配置与同步

- **[config.go](./config/config.go)**:
    - **规则**: `playReplay` 用于控制播放流水自动补归因调度，默认应保持关闭，待手动 replay 验数稳定后再开启。
- **[d1_sync.go](./internal/sync/d1_sync.go)**:
    - **规则**: D1 同步必须走“单飞 + 启动串行 + 按表更新时间戳增量”闭环；`SyncAll` 进入后先抢占运行锁，定时器触发时若已有同步在跑应直接跳过。D1 直连 `database/sql` 链路统一走 `otelsql.Open(...)` + `RegisterDBStatsMetrics(...)`；`track_play_records` 与 `top_album_stat` 镜像表新增字段时，必须同时补齐建表、补列、迁移复制和 batch upsert 四条链路。

### 3.3 相关基础设施

- **[core/ai/](./core/ai/)**:
    - **规则**: AI provider 抽象已拆分为“平台工厂 + 运行时 `LLMProvider`”，已彻底删除 `openai` 支持，Custom 已改名并对齐 OMLX 规范。新增 `RawChatClient` 统一底座接口以支持 `MultiStepAnalyzeTrack`（多步流式切分分析）：按 Step 1 (歌词翻译) -> Step 2 (分段解读) -> Step 3 (深度总结与指标) 无状态多轮推进；集成 `executeStepWithRetry` 单步指数退避重试，以及步骤失败时保留已成功早期结果的优雅降级策略；落地纯音乐/无歌词曲目（通过 `IsInstrumentalOrEmptyLyrics`）Step 1 与 Step 2 双步骤自动跳过机制，显著省 Token 并缩短延迟。支持从 context 提取 `ContextKeyJobID` 注入日志以跨协程追踪调试。平台枚举统一以 `common.AIModelPlatform` 为准；`/api/ai-models` 只返回平台列表，`/api/ai-models/:platform/models` 返回模型目录。模型目录只走 Redis 读穿缓存，不新增 MySQL 模型目录表；`internal/logic/insight` 的 provider cache key 必须按 `platform + model` 组合。Gemini SDK 统一通过 `genai.ClientConfig.HTTPClient` 注入 `core/telemetry.WrapHTTPClient(...)`，配置了 `GeminiConfig.BaseURL` 时也必须同步传入 SDK。
- **[core/telemetry/](./core/telemetry/)**:
    - **规则**: telemetry 已升级为 SigNoz OTLP gRPC exporter + MeterProvider 闭环；未配置 OTLP endpoint 时回退 stdout trace exporter。自建出站 `http.Client` 应优先通过 `WrapHTTPClient(...)` 包装。Telemetry 初始化完成后必须输出启动自检日志；日志接入 SigNoz 时优先用本地 Collector `filelog` receiver 采集本地日志文件，不要在业务代码里叠第二套 logs exporter。Scrobbler 正在播放链路坚持“单首歌一个长生命周期根 span + 按变化触发阶段 span / event”；收藏探测必须先用进程内热缓存和退避策略吸收稳态轮询，避免 Jaeger / SigNoz 被重复探测淹没。
- **[core/musicbrainz/](./core/musicbrainz/)**:
    - **规则**: `musicbrainzws2` 未暴露底层 `http.Client` 注入点；项目统一在包装层为 `SearchReleases`、`LookupRelease` 创建客户端 span，禁止通过反射改第三方库私有字段。
- **[core/redis/](./core/redis/)**:
    - **规则**: Redis tracing / metrics 统一走 `redisotel`，自定义 hook 只保留日志，不再重复创建 Redis client span。
- **[core/objectstorage/](./core/objectstorage/)**:
    - **规则**: S3 / MinIO / R2 这类基于 AWS SDK v2 的对象存储链路，统一走 smithy 原生 OTel adapter 接到全局 tracer/meter provider，避免在对象存储调用外层再叠手写 span。

*最后更新日期：2026-08-08 | 文档版本: v3.5*
AI MUST READ THIS FILE BEFORE MODIFYING CODE.
