# 记忆索引

## 2026-04-11

- **日期**: 2026-04-11
    - **特性摘要**: Bridge 端为 `now_playing` 快照补充 `receivedAt` 新鲜度事实源，暂停后即便缺失新的 WS 事件，也不会再把旧快照重新当成活跃播放来推进进度或允许进入正在播放页
    - **链接**: [Bridge 暂停静默播放态守卫修复清单](memory/2026-04-11/bridge_pause_stale_now_playing_guard_manifest.md)

- **日期**: 2026-04-11
    - **特性摘要**: Bridge macOS mini 播放条正式改为窗口级 AppKit 玻璃宿主，`NSVisualEffectView` 不再作为 SwiftUI 背景补丁层使用，前后台切换时的实色退化问题转由稳定窗口宿主闭环解决
    - **链接**: [Bridge macOS Mini 播放条窗口级玻璃宿主闭环](memory/2026-04-11/bridge_mac_mini_player_window_overlay_manifest.md)

## 2026-04-10

- **日期**: 2026-04-10
    - **特性摘要**: 专辑原始发布日期正式拆分为 `album.original_release_date` 独立事实字段，MusicBrainz、Audirvana/exiftool、手动维护与资料库同步链路统一透传但不参与专辑身份匹配
    - **链接**: [专辑原始发布日期落库特性清单](memory/2026-04-10/original_release_date_lakehouse_manifest.md)

## 2026-04-10

- **日期**: 2026-04-10
    - **特性摘要**: 专辑与曲目副标题身份索引正式补齐，`album`、`track`、`track_favorite_event`、`track_play_records` 以及资料库/最近播放 fallback join 统一改为 subtitle-aware，同时保留 `release_date` 作为专辑版本身份的一部分，避免不同版本继续串绑
    - **链接**: [subtitle 身份索引闭环特性清单](memory/2026-04-10/subtitle_identity_index_closure_manifest.md)

## 2026-04-09

- **日期**: 2026-04-09
    - **特性摘要**: 专辑副标题从原始标题字符串里正式拆出并贯通到资料库、播放流水、待归因专辑、排行榜、D1 同步与 Bridge 展示，`album_subtitle` / `name_subtitle` 成为稳定数据契约
    - **链接**: [专辑副标题贯通闭环特性清单](memory/2026-04-09/album_subtitle_end_to_end_closure_manifest.md)

## 2026-04-08

- **日期**: 2026-04-08
    - **特性摘要**: 音眸曲目、专辑与历史版本统一返回 `recommended_insight_id`，客户端与 Web 改为显式按该字段选中默认推荐版本，不再依赖第一条的隐式顺序
    - **链接**: [音眸推荐版本 ID 统一闭环特性清单](memory/2026-04-08/recommended_insight_id_unification_manifest.md)

## 2026-04-07

- **日期**: 2026-04-07
    - **特性摘要**: 音眸反馈原因枚举正式下沉到后端 `common`，`reason_codes` 白名单、`top_reason_codes` 汇总和 prompt 负反馈上下文统一改为基于同一套枚举归一
    - **链接**: [音眸反馈原因枚举对齐特性清单](memory/2026-04-07/insight_feedback_reason_enum_alignment_manifest.md)

## 2026-04-05

- **日期**: 2026-04-05
    - **特性摘要**: 单用户音眸反馈升级为“历史反馈 + 结构化问题 + 再生成纠偏”闭环，Bridge 列表/详情统一展示 `未评价 / 已认可 / 待修正` 语义
    - **链接**: [单用户音眸反馈历史与再生成闭环](memory/2026-04-05/insight_feedback_single_user_history_manifest.md)

- **日期**: 2026-04-05
    - **特性摘要**: 专辑音眸新增独立反馈闭环，`album_insight_feedbacks` 与曲目反馈分表，专辑分析会注入历史差评上下文，API 补齐专辑反馈写入与读取入口
    - **链接**: [专辑音眸反馈闭环特性清单](memory/2026-04-05/album_insight_feedback_closure_manifest.md)

## 2026-04-03

- **日期**: 2026-04-03
    - **特性摘要**: 曲目 MB 深度维护与精选回写链路统一做繁简与标点归一，避免 `track` 表在 MusicBrainz 写回时再次落入繁体标题，并为旧有简体存量行补了回退匹配
    - **链接**: [曲目 MB 深度维护简体写入闭环特性清单](memory/2026-04-03/track_mb_simplified_storage_closure_manifest.md)

- **日期**: 2026-04-03
    - **特性摘要**: 首页最近播放补齐封面路径落库与封面优先渲染，`track_play_records` 新增 `cover_art_path`，Bridge 端最近播放头部与时间标签一并收敛到更统一的首页样式
    - **链接**: [最近播放封面路径与首页视觉对齐特性清单](memory/2026-04-03/recent_play_cover_path_home_alignment_manifest.md)

## 2026-04-02

- **日期**: 2026-04-02
    - **特性摘要**: Bridge 连接链路新增“静默恢复 + 周期健康检查 + 失效待决策”策略，启动后默认尝试恢复上次成功连接，失败时由用户决定退出或重连
    - **链接**: [Bridge 连接恢复与静默健康检查策略特性清单](memory/2026-04-02/bridge_connection_restore_silent_healthcheck_manifest.md)

- **日期**: 2026-04-02
    - **特性摘要**: Bridge 实现音眸异步任务闭环与 iOS Live Activity 支持；Web Dashboard 新增艺人管理、待处理专辑手动维护及上下文陈旧检测
    - **链接**: [Bridge 音眸异步任务、Live Activity 与 Web Dashboard 增强特性清单](memory/2026-04-02/bridge_live_activity_web_dashboard_manifest.md)

## 2026-04-01

- **日期**: 2026-04-01
    - **特性摘要**: 首页“热门艺术家”接入 `artist_profile` 头像映射，`top-artists` 响应补齐头像字段，Bridge 三端优先展示对象存储头像并保留首字母兜底
    - **链接**: [首页热门艺术家头像映射闭环特性清单](memory/2026-04-01/home_artist_avatar_profile_manifest.md)

## 2026-03-31

- **日期**: 2026-03-31
    - **特性摘要**: Bridge 完成一轮共享层性能治理，收口高频状态广播、资料库 single-flight/页优先加载、本地索引查询结构、Bonjour 直连和连接阶段反馈，并补齐三端断开入口
    - **链接**: [Bridge 性能治理与连接链路收敛特性清单](memory/2026-03-31/bridge_performance_governance_manifest.md)

- **日期**: 2026-03-31
    - **特性摘要**: iPhone 音眸改为“异步任务 + WS 主通道 + GET 对账 + Live Activity/灵动岛”混合链路，详情页、深链回流与 Widget Extension 已完成接线，并补齐 `result_insight_id` 结果闭环
    - **链接**: [iPhone 音眸灵动岛与 WS/任务混合闭环特性清单](memory/2026-03-31/iphone_insight_dynamic_island_ws_job_manifest.md)

## 2026-03-30

- **日期**: 2026-03-30
    - **特性摘要**: 待归因专辑工作台新增手动维护兜底路径，MusicBrainz 缺失时可直接提交整专结构并复用 replay/收藏回填闭环
    - **链接**: [待归因专辑手动维护兜底闭环](memory/2026-03-30/pending_album_manual_maintenance_fallback_manifest.md)

## 2026-03-29

- **日期**: 2026-03-29
    - **特性摘要**: Bridge 共享网络层统一改为 GET query 百分号编码，修复曲名中的 `+` 被后端误解为空格的问题，并补齐客户端 URL 编码红线
    - **链接**: [Bridge Query 百分号编码修复清单](memory/2026-03-29/bridge_query_percent_encoding_manifest.md)

- **日期**: 2026-03-29
    - **特性摘要**: 播放流水新增 `trace_id/root_span_id/trace_sampled`，当前歌曲根 span 标识已能随 scrobble 阈值写入 `track_play_record`，支持从业务记录反查观测链路
    - **链接**: [播放流水 Trace 关联闭环](memory/2026-03-29/track_play_record_trace_link_closure_manifest.md)

- **日期**: 2026-03-29
    - **特性摘要**: Scrobbler 正在播放链路完成单首歌根 span 收口、收藏探测降频、favorite projection 会话缓存和 Last.fm 热缓存退避，显著降低稳态轮询 trace 噪音
    - **链接**: [Scrobbler 收藏探测与 Trace 降噪闭环](memory/2026-03-29/scrobbler_favorite_probe_trace_noise_reduction_manifest.md)

- **日期**: 2026-03-29
    - **特性摘要**: Bridge 正在播放页与全局播放条增加 WS 静默冻结逻辑，暂停后不再继续自增进度，收到新的 `now_playing` 后自动恢复
    - **链接**: [Bridge 正在播放进度静默冻结修复清单](memory/2026-03-29/bridge_now_playing_progress_silence_freeze_manifest.md)

## 2026-03-28

- **日期**: 2026-03-28
    - **特性摘要**: Bridge 三端正在播放接入 favorite projection，统一识别 `favorite_pending` / `unfavorite_pending` 并补齐收藏状态提示闭环
    - **链接**: [Bridge 正在播放收藏投影接入闭环](memory/2026-03-28/bridge_now_playing_favorite_projection_manifest.md)

- **日期**: 2026-03-28
    - **特性摘要**: 收藏链路新增 favorite projection，统一输出稳定收藏态与待归因收藏态，修复当前播放页无法感知 pending 收藏的问题
    - **链接**: [收藏投影视图与待归因收藏态闭环](memory/2026-03-28/favorite_projection_pending_state_manifest.md)

- **日期**: 2026-03-28
    - **特性摘要**: 新增 SigNoz `filelog` 日志采集模板，明确应用日志本地轮转与 SigNoz 1 天 retention 解耦，形成 trace/metrics/logs 分层闭环
    - **链接**: [SigNoz filelog 日志采集闭环](memory/2026-03-28/signoz_filelog_log_collection_closure_manifest.md)

- **日期**: 2026-03-28
    - **特性摘要**: 接入 SigNoz OTLP exporter，补齐 HTTP/Redis/DB tracing 与 metrics 闭环，并统一移除重复手写 span
    - **链接**: [SigNoz OTLP 可观测性闭环](memory/2026-03-28/signoz_otlp_observability_closure_manifest.md)

## 2026-03-27

- **日期**: 2026-03-27
    - **特性摘要**: 异步协程统一走 `telemetry.GoSafe` / `GoSafeDetached` / `GoOnlySafe`，为异步任务补齐 span 与 panic 保护，并避免长循环 trace 失真
    - **链接**: [异步协程 trace 与 panic 保护闭环](memory/2026-03-27/goroutine_trace_panic_guard_manifest.md)

- **日期**: 2026-03-27
    - **特性摘要**: AI 模型选择完成平台/模型拆分，新增平台目录与模型目录接口、Redis 目录缓存，以及 Web/Bridge 双级选择闭环
    - **链接**: [AI 模型平台与模型目录闭环特性清单](memory/2026-03-27/ai_model_platform_catalog_closure_manifest.md)

## 2026-03-26

- **日期**: 2026-03-26
    - **特性摘要**: 新增专辑级音眸后端闭环，支持按曲序聚合高质量 `track_insight`、生成 `album_insight` 并暴露专辑分析 API
    - **链接**: [专辑音眸后端闭环特性清单](memory/2026-03-26/album_insight_backend_closure_manifest.md)

## 2026-03-24

- **日期**: 2026-03-24
    - **特性摘要**: Bridge iPhone ShareKit 公共壳层收口完成，分享预览统一使用背景/头图/正文/底部的公共闭环，三类场景只保留各自内容
    - **链接**: [iPhone ShareKit 公共壳层收口特性清单](memory/2026-03-24/iphone_sharekit_public_shell_closure_manifest.md)

- **日期**: 2026-03-24
    - **特性摘要**: Bridge 新增 iPhone ShareKit，一期完成曲目信息/歌词/音眸分享预览、长图 PNG 导出、Photos 保存、系统分享与本地测试 target
    - **链接**: [iPhone ShareKit 一期闭环特性清单](memory/2026-03-24/iphone_sharekit_phase1_manifest.md)

## 2026-03-23

- **日期**: 2026-03-23
    - **特性摘要**: API 层新增 Redis 响应缓存 middleware，支持路由级 TTL、空结果 3 秒负缓存、`ETag` 回写与 304 语义，Redis 不可用时自动降级
    - **链接**: [HTTP Redis 缓存中间件特性清单](memory/2026-03-23/http_redis_cache_middleware_manifest.md)

## 2026-03-23

- **日期**: 2026-03-23
    - **特性摘要**: D1 同步恢复闭环，增加单飞保护与启动串行化，避免首轮全量与定时器重入滚动耗尽额度
    - **链接**: [D1 增量同步单飞闭环](memory/2026-03-23/d1_incremental_singleflight_sync_closure_manifest.md)

## 2026-03-22

- **日期**: 2026-03-22
    - **特性摘要**: 待归因专辑工作项详情新增实时对比与显式刷新，避免冻结上下文被误当作实时数据
    - **链接**: [待归因专辑工作项上下文刷新特性清单](memory/2026-03-22/pending_album_work_item_context_refresh_manifest.md)

## 2026-03-21

- **日期**: 2026-03-21
    - **特性摘要**: 封面链路接入 S3 兼容对象存储（MinIO/R2），scrobbler 优先对象命中并回退内存缓存，播放落库后回填 album 封面字段
    - **链接**: [专辑封面对象存储闭环](memory/2026-03-21/album_artwork_object_storage_closure_manifest.md)

## 2026-03-15 ~ 2026-03-20

- **日期范围**: 2026-03-15 ~ 2026-03-20
    - **特性摘要**: 后端完成数据归因、专辑身份、DAO/事务边界与 SQL 语义治理；Bridge 完成本地优先资料库、共享层边界与三端产品线闭环
    - **链接**: [2026-03-15 至 2026-03-20 核心收敛摘要](memory/2026-03-20/2026-03-15_to_2026-03-20_foundation_summary.md)

## 2026-03-14

- **日期**: 2026-03-14
    - **特性摘要**: 修复 MusicBrainz 深度维护在多碟专辑中错误复用第一碟录音的问题，已听曲目对齐改为 `track_album` 物理位置优先
    - **链接**: [MusicBrainz 多碟已听曲目对齐修复](memory/2026-03-14/musicbrainz_multi_disc_heard_track_alignment_fix.md)

- **日期**: 2026-03-14
    - **特性摘要**: 完成曲目五元组身份升级后的专辑闭环修复，统一 `track_album` 与 MusicBrainz 维护链路为按碟号和曲号优先匹配
    - **链接**: [曲目五元组与专辑闭环修复特性清单](memory/2026-03-14/track_identity_album_closure_feature_manifest.md)

## 2026-03-12

- **日期**: 2026-03-12
    - **特性摘要**: 实现专辑占位曲目灰度展示、歌词页收藏星星三态呈现（SVG 半星逻辑）及后端双平台状态同步修复
    - **链接**: [专辑占位展示与收藏三态记录](memory/2026-03-12/dashboard_placeholder_and_favorite_enhancement.md)


## 2026-03-09

- **日期**: 2026-03-09
    - **特性摘要**: 专辑管理系统与 MusicBrainz 深度集成，实现数据精准补全、轨道序号校正及详情页样式重构
    - **链接**: [专辑管理与 MusicBrainz 深度集成特性清单](memory/2026-03-09/album_and_musicbrainz_integration_feature_manifest.md)

## 2026-02-21

- **日期**: 2026-02-21
    - **特性摘要**: 仪表板和Cloudflare视图进行统一视觉重构，引入渐变背景与玻璃拟态效果，并整合适配全站Logo和水印
    - **链接**: [UI与视觉整体升级特性清单](memory/2026-02-21/visual_enhancement_and_ui_redesign_feature_manifest.md)

## 2026-02-20

- **日期**: 2026-02-20
    - **特性摘要**: 为曲目详情模态框增加独立的歌词分页视图，实现对所选曲目的歌词按需加载和展示
    - **链接**: [曲目详情页增加歌词视图特性清单](memory/2026-02-20/track_details_lyrics_tab_feature_manifest.md)

## 2026-02-19

- **日期**: 2026-02-19
    - **特性摘要**: 实现音眸功能，提供 AI 分析结果的分享图片功能及模型选择交互，允许将生成的歌词或者洞察转为美观的卡片图片分享
    - **链接**: [音眸: AI分析结果分享图片特性清单](memory/2026-02-19/share_ai_insight_image_feature_manifest.md)

## 2025

- **日期**: 2025-12-31
    - **特性摘要**: 汇总 2025 年的基础能力演进，提炼播放数据主干、Track 模型统一、多播放源接入、收藏同步、分析展示、基础设施治理与已过时约束
    - **链接**: [2025 基础能力演进摘要](memory/2025-12-31/2025_foundation_summary.md)
