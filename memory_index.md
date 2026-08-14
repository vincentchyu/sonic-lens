# 记忆索引

## 2026-08-14

- **日期**: 2026-08-14
    - **聚合摘要**: 【当日聚合】待归因全链路版本说明/发行格式闭环、热门流派 Top 50 端到端扩容、网络/存储防穿透治理与单元测试工程化建设。聚合了 4 项核心演进与治理：
        1. 彻底打通 DTO/物料解析/模态交互/事务落库的全链路 `album_subtitle` 与 `release_type` 闭环并补齐原子播放量累加与对账；
        2. 实现热门流派预聚合保底 50、DAO 自适应降级与补齐、API 默认 limit=50 与 Bridge 客户端圆环图鲜明聚焦/50 项详情列表滚动的端到端打通；
        3. 根治客户端 URL 二次转义（`%2520`）、服务端未防御解码直接拼入 SQL 与本地 SQLite `genre` 漏存缺陷，实现防御性反转义与真实 HTTP 路由单测覆盖；
        4. 隔离宿主进程与公网依赖至 `//go:build integration` 标签，封装 `internal/testutil` 内存测试库（SQLite 内存库 + sqlmock）并沉淀专属单测 Skill，实现全项目默认 `go test -count=1 ./...` 100% 独立秒级通过。
    - **聚合链接**: [2026-08-14 每日架构演进与特性聚合清单](memory/2026-08-14/daily_aggregation_manifest.md)
        - [冗余 CMD 工具、无效补数据定时器、遗留配置与历史迁移代码清理特性清单](memory/2026-08-14/legacy_cmd_and_schedulers_cleanup_manifest.md)
        - [流派权威源收口至 GenreCache 与防污染重构特性清单](memory/2026-08-14/genre_cache_authority_refactor_manifest.md)
        - [热门流派关联专辑 0 匹配与 URL 编码穿透/SQLite 字段缺失修复特性清单](memory/2026-08-14/genre_url_escaping_and_sqlite_index_fix_manifest.md)
        - [待归因专辑与全链路 album_subtitle & release_type 闭环修复特性清单](memory/2026-08-14/pending_album_subtitle_release_type_closure_manifest.md)
        - [热门流派 Top 50 扩容与端到端闭环特性清单](memory/2026-08-14/top_genres_50_limit_closure_manifest.md)
        - [单元测试零外部依赖改造与项目单测专属 Skill 建设特性清单](memory/2026-08-14/unit_test_isolation_and_skill_manifest.md)

## 2026-08-13

- **日期**: 2026-08-13
    - **聚合摘要**: 【当日聚合】播放事实源归一 (SST)、统计与流派全链路闭环对账、数据自愈修补引擎与高并发/表结构稳定性治理。聚合了 6 项核心演进与治理：
        1. 确定 `track_play_records` 为全局唯一播放事实源，落地 `album.play_count` 物理列与 SQL 下推排序，并在 Web 后台暴露可视化对账卡片；
        2. 构建 4 层历史听歌流水自动修补引擎（`replay-track-play-records --repair --reconcile`），成功修补 7203 条残缺记录并精确复原全表专辑物理播放量；
        3. 补全流水归因与修补引擎中的 `release_type` 实体双向回填，历史 53 条残缺流水纠偏归零；
        4. 落地流派多标签解包、主体提取与精准词界 SQL 匹配器，实现热门流派统计与专辑检索 100% 精准对齐及 Bridge 查看全部流派导航；
        5. 彻底移除 8 个 `*_stat` 统计快照表的自增 `id` 并重构为自然复合主键，消除自增 ID 飙升（22971+）与写放大开销；
        6. 实施客户端入口即时锁定与服务端 60s 内窗防重拦截的双层防护，根除 Scrobbler 并发重复上报缺陷。
    - **聚合链接**: [2026-08-13 每日架构演进与特性聚合清单](memory/2026-08-13/daily_aggregation_manifest.md)
    - **细分清单**:
        - [播放统计闭环统一重构与 Web 可视化对账特性清单](memory/2026-08-13/playback_statistics_unified_reconcile_manifest.md)
        - [听歌流水历史残缺数据自动修补与全表播放量纠偏特性清单](memory/2026-08-13/track_play_records_repair_and_reconcile_manifest.md)
        - [听歌流水 release_type 字段丢失与修补归因对账特性清单](memory/2026-08-13/track_play_records_release_type_backfill_manifest.md)
        - [Top热门流派与流派专辑播放统计闭环对账重构特性清单](memory/2026-08-13/top_genre_stat_alignment_manifest.md)
        - [*_stat 统计表移除自增 ID 与复合主键重构清单](memory/2026-08-13/top_stat_tables_composite_pk_refactor_manifest.md)
        - [播放完成并发重复上报与双层防重拦截特性清单](memory/2026-08-13/concurrency_scrobble_duplication_prevention_manifest.md)

## 2026-08-12

- **日期**: 2026-08-12
    - **特性摘要**: 实现 macOS 侧边栏折叠状态下的顶部标题 Quick Switcher Menu 与 Option (`⌥1`~`⌥6`) 系统快捷键导航，解决侧边栏收起后模块切换阻尼大问题，保持大屏沉浸体验的同时实现一击与键盘秒级切页。
    - **链接**: [macOS 侧边栏折叠状态 Quick Switcher 导航与 ⌥1~⌥6 快捷键特性清单](memory/2026-08-12/collapsed_sidebar_quick_switcher_navigation_manifest.md)

## 2026-08-10

- **日期**: 2026-08-10
    - **特性摘要**: ShareKit 引擎全面覆盖 macOS 大屏 16:9 画幅与 Bento Grid 双栏流式分页 (Dual-Column)，解决切片窄框偏小与溢出截断问题；实现多页海报剪贴板批量写入、多图自动编号导出与横向画廊式滚动预览；完成 `AlbumDetailView` 专辑海报分享全覆盖，支持无音眸时优雅降级卡片提示；清理历史 `SnapshotExport.swift` 快照代码，并保持 iOS iPhone 端的长图导出与系统分享逻辑 100% 独立与验证通过。
    - **链接**: [ShareKit macOS 大屏 Bento Grid 双栏分页海报与专辑分享全覆盖特性清单](memory/2026-08-10/sharekit_mac_bento_grid_export_manifest.md)

## 2026-08-09

- **日期**: 2026-08-09
    - **特性摘要**: macOS 正在播放沉浸界面重构为主副卡槽栈式平滑平移动效 (`Secondary & Primary Stack Layout`)，音眸模式下左侧副屏保留 LRC 实时歌词同步，完美辅助外语歌曲对标赏析与翻译；同时移除外层硬 `clipped()`，解决封面 32px 弥散阴影在 380px 边界被裁切产生黑色折线的问题。
    - **链接**: [macOS 正在播放主副卡槽栈式交互重构与阴影裁剪修复特性清单](memory/2026-08-09/mac_now_playing_stack_layout_manifest.md)

## 2026-08-08

- **日期**: 2026-08-08
    - **特性摘要**: AI 模型多步切分流式分析（Multi-Step Analysis）与 JobID 全链路追踪，实现了 Translation (Step 1)、Appreciate (Step 2)、Deep Summary (Step 3) 的无状态多步编排与精简 Prompt 控制；引入单步退避重试 (`executeStepWithRetry`) 与保留早期已成功结果的优雅降级策略；落地纯音乐/无歌词曲目的 Step 1 & Step 2 双步骤智能跳过；引入 `llm_call_logs.job_id` 自动打标、跨协程 Context 传导、控制台按 JobID 聚合重排及步骤徽章与文本对比度修复。
    - **链接**: [AI 模型多步切分流式分析与 JobID 全链路追踪特性清单](memory/2026-08-08/ai_multi_step_analysis_manifest.md)

## 2026-08-07

- **日期**: 2026-08-07
    - **特性摘要**: 专辑发行格式后缀（- EP/Single/LP）在 scrobbler 上报、DAO 自动规一、MusicBrainz 统一两阶段搜索与待归因工作台中全面适配，完成一键 CLI 历史清洗及锁定绕过曲目合并闭环
    - **链接**: [专辑发行格式后缀（- EP/Single/LP）适配与清洗特性清单](memory/2026-08-07/album_release_type_suffix_cleanup_manifest.md)

- **日期**: 2026-08-07
    - **特性摘要**: 正式专辑与待归因工单实现 100% 公共预审、草稿微调与落库闭环，修复 `track_album.mb_recording_id` 遗漏与前端 POST 缺少 `Content-Type` 头/入口拦截，重构待归因模态框布局并将手动创建表单默认折叠
    - **链接**: [正式专辑与待归因工单 MB 精选维护、草稿闭环及 UI 布局重构特性清单](memory/2026-08-07/unified_album_mb_maintenance_and_draft_closure_manifest.md)

## 2026H1

- **日期**: 2026H1
    - **特性摘要**: 汇总 2026 年上半年基础能力演进，提炼视觉重构、专辑副标题与五元组身份、待归因工作台与草稿闭环、音眸 AI 异步反馈、Bridge 正在播放与 OTLP 可观测性治理等沉淀
    - **链接**: [2026H1 基础能力演进摘要](memory/archive/2026_h1/2026_h1_foundation_summary.md)

## 2025

- **日期**: 2025-12-31
    - **特性摘要**: 汇总 2025 年的基础能力演进，提炼播放数据主干、Track 模型统一、多播放源接入、收藏同步、分析展示、基础设施治理与已过时约束
    - **链接**: [2025 基础能力演进摘要](memory/2025-12-31/2025_foundation_summary.md)
