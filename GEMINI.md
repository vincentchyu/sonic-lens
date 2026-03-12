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
- **`cmd/`**: 独立命令行工具集。包含同步记录脚本（`sync_records.go`）及项目维护工具。
- **`common/`**: 全局通用基础设施。
    - `enum.go`: 统一定义播放器类型、数据库类型等枚举。
    - `utils.go`: 通用验证（如 `ValidateTrackInfo`）与字符串处理助手。
- **`core/`**: 系统“基座”模块（核心基础设施）。
    - `log/`: 基于 Zap 的结构化日志实现，支持多级别输出与 Context 关联。
    - `db/`: GORM 数据库初始化与连接池管理，适配 SQLite/MySQL。
    - `redis/`: Redis 客户端封装与缓存访问抽象。
    - `websocket/`: 实时通信枢纽，负责 `now_playing` 等消息的广播与连接生命周期。
    - `telemetry/`: 基于 OpenTelemetry 的分布式链路追踪与性能监控指标上报。
    - `ai/`: AI 模型驱动适配器（支持 OpenAI, DeepSeek, 豆包等），处理 Prompt 注入。
    - `applemusic/`: Apple Music API 交互封装。
    - `lastfm/`: Last.fm API 交互封装，支持记录上报（Scrobble）与状态检查。
    - `musicbrainz/`: MusicBrainz 元数据查询，用于专辑/曲目信息的深度补全。
    - `lyrics/`: 统一歌词搜索与解析引擎。
    - `musixmatch/`: Musixmatch 歌词服务适配（辅助歌词源）。
    - `audirvana/`: Audirvana 播放器底层状态获取逻辑。
    - `roon/`: Roon 播放器 API 监听与控制。
    - `applesciprt/` (拼写固定): AppleScript 执行封装，用于 macOS 本地播放器（如 AM）的自动化控制。
    - `env/`: 环境变量解析与全局配置加载。
    - `cache/`: 通用内存缓存工具类。
    - `exec/`: 系统外部命令调用助手，提供超时控制与日志捕获。
    - `fx/`: 基于 Uber Fx 的依赖注入定义集合，解决模块间的生命周期管理。
- **`internal/`**: 业务黑盒（核心领域逻辑）。
    - **`model/`**: **数据持久层 (DAO)**。
        - 准则：所有直接操作数据库的代码必须在此文件夹内，且按表分文件。
    - **`logic/`**: **业务服务层 (Service)**。
        - 准则：处理具体的业务流程（如 AI 赏析生成流程、MusicBrainz 补全逻辑）。
    - **`scrobbler/`**: **播放器适配层 (Drivers)**。
        - 准则：负责与外部播放器（Apple Music, Audirvana 等）通信获取状态。
- **`api/`**: **通信门户层 (Interface)**。
    - 职责：Gin 路由定义、HTTP 处理函数（Handlers）、WebSocket 接口定义。
- **`templates/`**: **视觉展示层 (Views)**。
    - 存放 `.html` 模板。注意：所有逻辑控制应尽量留在 JS 函数中。
- **`static/`**: **静态资源库**。
    - 存放 CSS、图片、Logo。

### 1.2 模块调用拓扑
`Main` -> `API` -> `Internal/Logic` -> `Internal/Model` -> `Core/DB`
`Scrobbler` -> `Internal/Logic` -> `Internal/Model`

### 1.2 前端双轨制说明
- **当前现状**: 大量核心功能承载于 `templates/*.html` (Go Templates) 配合 Vanilla JS。
- **开发建议**: 在修改 `dashboard.html` 等文件时，需注意代码体量巨大且存在冗余定义（详见 3.1）。

---

## 2. 开发规范与最佳实践 (依 .cursor/rules 对齐)

### 2.1 后端编码规约 (Go)
- **命名与风格**: 严格遵循 Uber Go 风格。变量/函数使用 `PascalCase` 或 `camelCase`，包名全小写。
- **注释要求**: **所有注释必须使用中文**。导出的函数、类型必须有阐述“为什么”的注释。
- **日志记录**: 使用结构化日志 `log.Info/Error(ctx, "msg", zap.Field)`。错误日志必须携带 `zap.Error(err)`。
- **错误处理**: 禁止忽略错误。使用 `%w` 进行错误包装以保留调用链。

### 2.2 数据库设计指南 (GORM)
- **代码位置规范**：**所有数据库 CRUD 操作必须定义在 `internal/model/` 下对应的表 `go` 文件中**。严禁将原生的数据库查询/更新逻辑散落在各个业务模块（Logic 层）里。
- **复用性原则**：优先封装可复用的模型方法（如 `GetOrCreateAlbum`），减少重复的 SQL 逻辑，确保数据访问层（DAO）的纯粹性。
- **上下文绑定**: 所有数据库操作必须使用 `.WithContext(ctx)` 确保链路可追踪。
- **并发控制**: 重要更新（如 `PlayCount` 增加）应实现基于 `version` 字段的**乐观锁**机制。
- **索引原则**: 复合索引遵循最左前缀原则。新系统必须包含 `created_at` 和 `updated_at`。

---

## 3. 核心模型参考清单 (internal/model)

该清单映射了 `internal/model/` 目录下的核心实体及其关键职责，AI Agent 在涉及数据变更时应参考对应的模型文件：

- **[track.go](./internal/model/track.go)**: 
    - **核心索引**: `uidx_t_aatdntn` (Artist, Album, Track, DiscNumber, TrackNumber)。
    - **功能**: 曲目元数据、播放次数统计、乐观锁版本控制。
- **[album.go](./internal/model/album.go)**: 
    - **核心索引**: `uidx_album_artist_name_release_date`。
    - **功能**: 专辑元数据、同步状态 (SyncStatus) 管理。
- **[track_album.go](./internal/model/track_album.go)**: 
    - **功能**: 维护曲目与专辑的多对多关联，支持碟号和轨道号的物理映射。
- **[track_play_record.go](./internal/model/track_play_record.go)**: 
    - **功能**: 详尽的听歌流水历史，用于统计周/月榜单及同步 Last.fm。
- **[track_insight.go](./internal/model/track_insight.go)**: 
    - **功能**: AI 生成的歌曲赏析细节（背景、歌词翻译、时代背景）。
- **[track_lyrics.go](./internal/model/track_lyrics.go)**: 
    - **功能**: 原始歌词与翻译歌词的持久化。
- **[genre.go](./internal/model/genre.go)**: 
    - **功能**: 音乐流派库。
- **[dashboard_stat.go](./internal/model/dashboard_stat.go)**: 
    - **功能**: 复杂聚合统计逻辑（Top 艺术家、流派占比、年度统计）。
- **[init.go](./internal/model/init.go)**: 
    - **功能**: 数据库初始化与 AutoMigrate 配置。

---
*最后更新日期：2026-03-12 | 文档版本: v2.0*
AI MUST READ THIS FILE BEFORE MODIFYING CODE.
