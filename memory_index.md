# 记忆索引

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
