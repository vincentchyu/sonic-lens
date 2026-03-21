# 2026-03-15 至 2026-03-20 核心收敛摘要

## 日期范围

2026-03-15 ~ 2026-03-20

## 定位

本文件用于压缩 2026 年 3 月中旬这一轮连续重构，只保留对当前分析仍有高价值的核心变化。`memory_index.md` 只索引本摘要；若需要实现细节、SQL、回放步骤或端侧页面约束，再下钻到原始特性清单。

## 这一阶段真正沉淀下来的核心变化

### 1. 后端数据治理从“能跑”收口到“可归因、可维护、可测试”

- 播放归因增加了置信度分层，弱来源不再允许继续新建专辑结构，`track` / `album` 身份收紧为更保守的解析策略。
- 专辑身份修复补上了重复专辑清洗、专辑关联迁移与曲目物理位置优先的约束，避免再被曲目发布日期或弱线索撕裂。
- DAO、事务与 API 分层在这一轮被明确收口：数据库 CRUD 与事务入口回归 `internal/model/`，业务编排集中到 `internal/logic/`，默认测试面也切回可自动化执行。
- SQL DDL / DML 与 MySQL 实库语义重新对齐，播放流水 replay 的迁移顺序与人工验数路径也被留档。

### 2. Bridge 从“单端客户端”演进为“本地优先的三端产品线”

- Bridge 资料库主路径正式固化为“本地 SQLite 轻量索引 + `/api/library/sync` 增量同步 + WebSocket 推送 + 详情懒加载”，不再接受回退到远端分页加本地数组筛选的混合模式。
- 客户端产品线从 macOS 扩展到 iPadOS、iPhone，形成共享 `SoniclensCore` / `ViewModels` 与端私有容器并存的三端结构。
- 专辑库、专辑详情与音眸渲染在这一轮补齐三端闭环，但这些具体页面规则属于专题文档和特性清单，不再作为索引层主叙事。

### 3. 对今天仍然最有用的判断标准

- 只要需求涉及数据库写入、事务、MusicBrainz 编排、播放归因或 replay，先按“`api -> logic -> model`”分层与弱来源止血约束思考。
- 只要需求涉及 Bridge 资料库、排序、详情、Insight、播放页或多端适配，先按“共享层优先、本地索引优先、端容器后分叉”的思路思考。
- 3 月中旬之后的默认上下文应理解为：后端主线强调数据身份与事务治理，客户端主线强调本地优先与三端共享边界。

## 2026-03-19

- **日期**: 2026-03-19
    - **特性摘要**: 修复歌词同步链路仅有整秒精度的问题，新增 `position_ms` 广播、统一 LRC 判定规则，并收口 Web/Bridge 两端歌词解析与高亮时钟
    - **链接**: [歌词毫秒级同步链路修复清单](./../2026-03-19/lyrics_millisecond_sync_fix_manifest.md)

## 2026-03-18

- **日期**: 2026-03-18
    - **特性摘要**: `soniclens-bridge` 扩展为 macOS 与 iPadOS 双产品线，新增独立 iPad target、平台根路由与沉浸式播放容器，并明确共享层与平台壳边界
    - **链接**: [Bridge 双端产品线特性清单](./../2026-03-18/bridge_dual_platform_product_line_manifest.md)


## 下钻文档

### 后端治理与数据主线

- [播放流水补归因 MySQL 迁移与人工测试清单](./../2026-03-15/play_replay_mysql_rollout_checklist.md)
- [播放归因置信度与弱来源止血](./../2026-03-15/play_resolution_confidence_guardrails.md)
- [重复专辑清洗与专辑身份修复](./../2026-03-15/duplicate_album_cleanup_and_album_identity_fix.md)
- [SQL 结构与初始化脚本同步清单](./../2026-03-15/sql_schema_sync_manifest.md)
- [DAO 边界与 MusicBrainz 事务收口特性清单](./../2026-03-15/dao_boundary_musicbrainz_refactor_manifest.md)

### Bridge 资料库与三端闭环

- [资料库本地索引与增量同步桥接特性清单](./../2026-03-15/library_index_sync_bridge_feature_manifest.md)
- [Bridge 双端产品线特性清单](./../2026-03-18/bridge_dual_platform_product_line_manifest.md)
- [Bridge 专辑库三端闭环与性能收敛记忆清单](./bridge_album_library_three_platform_closure_manifest.md)
- [Bridge 三端产品线与音眸闭环记忆清单](./bridge_three_platform_insight_closure_manifest.md)
- [BroadcastLibraryUpdate 客户端交互流程](../../docs/broadcast_library_update_flow.md)

## 不再在索引层平铺的信息

- replay 的具体命令顺序、验数 SQL 与人工联调步骤
- SQL 初始化脚本调整明细
- 专辑卡片尺寸、排序菜单、详情布局、多碟展示、高密度滚动性能等页面级规则
- 三端音眸富渲染的具体页面承载方式

这些内容仍保留在原始特性清单中，索引层不再逐条展开。
