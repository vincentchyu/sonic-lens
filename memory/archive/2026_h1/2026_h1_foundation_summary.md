# 2026H1 基础能力演进摘要

## 日期

2026-06-30 (归档于 2026-08-07)

## 定位

本文件用于压缩 2026 年上半年的历史记忆（涵盖 2026-02-19 至 2026-04-24），只保留对当前系统演进与分析仍有参考价值的底层设计和基础事实。

若需追溯逐字开发日志和具体实现细节，可回看 `memory/archive/2026_h1/` 下各个日期的原始清单；但在新需求开发和日常设计推理中，应优先参考本摘要，避免被中途演进的局部方案误导。

---

## 2026H1 真正沉淀下来的基础能力

### 1. 视觉体验与 UI 架构重构
- **多端视觉风格升级**：针对后台 Dashboard 与 Cloudflare 镜像视图进行了玻璃拟态和渐变重构，统一了系统 Logo 与水印标识。
- **iOS ShareKit 框架收口**：在 Bridge iPhone 端新增 ShareKit 模块，将卡片生成、长图 PNG 导出、Photos 保存及系统分享闭环；引入了 `SharePosterShell` 玻璃拟态公共外壳，统一了顶部 header（左封面、右元数据、左下指标、右下收藏态）和底部品牌布局，实现了各内容视图的排版解耦。
- **窗口级玻璃宿主**：macOS mini 播放条彻底放弃在 SwiftUI 视图层直接修饰背景模糊，改为通过 AppKit 窗口级 `NSVisualEffectView` 宿主承载，彻底解决了前后台切换时实色退化的难题。
- **Admin 后台轻量化拆分**：将 Web Admin 后台巨大的 dashboard 模板轻量化拆分为 Go template partials、`static/admin/admin.css` 与按顺序加载 Vanilla JS 文件，维持现有的 DOM 结构与全局 API 契约不变。

### 2. 核心领域元数据与多维身份升级
- **MusicBrainz 深度集成与多碟对齐**：实现了专辑管理与 MusicBrainz 精准补全、轨道校正；曲目身份主键升级为 `(Artist, Album, AlbumSubtitle, Track, DiscNumber, TrackNumber)` 复合五元组索引，关联写入与已听曲目对齐改为“碟号 + 轨道号优先”，修复了多碟专辑匹配错乱的问题。
- **专辑副标题（Subtitle）贯通**：从原始标题字符串中拆出 Deluxe/Remaster/Anniversary 等结构化版本说明，贯通至播放流水、待归因、排行榜、D1 同步与 Bridge；核心数据表（`album`、`track`、`track_favorite_event`、`track_play_records` 等）均升级为 subtitle-aware，避免了同名不同版本的专辑串绑。
- **原始发布日期落库**：新增 `album.original_release_date` 独立字段作为物理事实，保留发行年份，但不参与专辑的主身份匹配。

### 3. 待归因与后台维护工作台
- **手动维护与回填机制**：为待归因专辑工作台新增手动维护路径，在 MusicBrainz 缺失时可直接由管理员提交整专结构，并复用 replay 与收藏回填逻辑；引入了实时对比与显式上下文刷新。
- **简体写入与繁简归一**：曲目 MB 深度维护与精选回写链路统一执行繁简、标点和大小写归一，防止 `track` 表在 MB 深度维护写回时混入繁体，并为旧有简体存量行补齐了回退匹配。

### 4. 音眸 AI 异步生成与反馈闭环
- **异步分析与 Live Activity 联动**：将 iPhone 端音眸分析改为“POST 创建异步任务 + WebSocket 进度推送 + GET 状态对账 + iOS Live Activity 灵动岛进度条 + 深链回流”的混合链路，封面图走本地异步下载以突破 Widget 网络限制。
- **多版本推荐与负反馈纠偏**：音眸曲目与专辑统一返回 `recommended_insight_id`，客户端按此字段显式选中默认版本。反馈历史（未评价/已认可/待修正）和原因枚举下沉至 `common` 包；分析时自动注入历史负反馈上下文，实现再生成纠偏。

### 5. 正在播放与客户端状态一致性
- **收藏投影视图（Favorite Projection）**：推出 favorite projection 机制，统一输出稳定收藏态与待归因/Pending 收藏态，Bridge 正在播放页全面接入，消除了播放条对 pending 收藏的感知延迟。
- **WS 播放存在性与静默冻守**：Bridge 客户端确立 `WS now_playing` 为播放存在性的最高事实源；新增静默检测逻辑，暂停后自动冻结本地进度递增；快照引入 `receivedAt` 新鲜度守卫，避免重新进入页面时将旧快照误判为活跃播放。
- **百分号编码收口**：Bridge 网络层统一收口 GET query 百分号编码，彻底解决了曲名中 `+` 被误还原为空格的元数据传输漏洞。

### 6. 基础设施、可观测性与云端同步
- **封面对象存储**：接入 S3 兼容对象存储（MinIO/R2），由 scrobbler 优先处理对象命中、回退本地并写入 album 封面字段，实现资源闭环。
- **D1 增量同步单飞保护**：CF D1 同步引入 single-flight 与启动串行化，防止定时器重入与全量同步滚动耗尽额度。
- **协程 Trace 与 Panic 安全**：禁止使用裸写 `go func`，统一收口至 `telemetry.GoSafe`、`GoSafeDetached` 与 `GoOnlySafe` 保护 Panic 并透传 Trace 上下文。
- **OTLP 全链路监控与日志采集**：集成 SigNoz OTLP exporter，打通 HTTP/Redis/DB 自动 tracing，对 scrobbler 探测进行降频和缓存退避以减少 noise trace，并追加 SigNoz filelog 本地轮转日志采集。
- **MySQL DDL 审计与 Schema 退场**：对 MySQL 运行时 `ensure*Schema` 自动建表进行了 DDL 对照审计，归档补齐字段后制定了稳定期自动建表退场的顺序。

---

## 对今天仍然有用的历史结论

- **身份判定三要素**：曲目/专辑判断身份时，**必须以“主元数据 + Subtitle 副标题 + 发行日期”三者结合作为唯一事实源**；严禁退回至解析原始标题括号后缀的方式。
- **状态发布主链**：客户端正在播放以 **`WS now_playing` 作为播放活动状态 the 最高优先级判据**，本地进度应具备静默冻结与新鲜度守卫逻辑，防止过度推进。
- **异步长时任务模型**：耗时较长的 AI 任务（如音眸分析）统一走“`/api/insight-jobs` 异步提交 + WebSocket 进度反馈”设计；客户端需结合 Live Activity 展现，不可再用单次长请求阻塞前台。
- **可观测性红线**：异步协程与后台循环任务统一使用 `core/telemetry` 安全包装，拒绝任何裸写 `go`。

---

## 明确标记为历史背景、不要直接照搬的内容

- **手动/裸 API 操作 D1**：早期的 D1 增量同步由于缺乏单飞保护常有重入问题，现已全部统一到 single-flight 守护管理器。
- **DOM 内拼装分享卡片**：早期在 Web 页中直接解析 DOM 截屏的逻辑已被 iPhone 端 ShareKit (基于 `SharePosterShell`) 取代。
- **裸 HTML Admin 页面**：早期的巨型 `admin.html` 已经在 2026-04-24 拆分为 partial 与静态 JS/CSS，禁止在新功能中继续向根后台模板文件堆积 HTML/JS 代码。

---

## 归档说明

- `memory/archive/2026_h1/` 下的原始文件夹已移动并保留，作为历史细部档案，不再在索引中逐条展开。
- 后续开发若需对比或回溯 2026 年上半年工作，应以本摘要的“底层能力画像”为依据。
