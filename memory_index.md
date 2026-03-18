# 记忆索引

## 2026-03-18

- **日期**: 2026-03-18
    - **特性摘要**: `soniclens-bridge` 扩展为 macOS 与 iPadOS 双产品线，新增独立 iPad target、平台根路由与沉浸式播放容器，并明确共享层与平台壳边界
    - **链接**: [Bridge 双端产品线特性清单](memory/2026-03-18/bridge_dual_platform_product_line_manifest.md)

## 2026-03-15

- **日期**: 2026-03-15
    - **特性摘要**: 梳理播放流水补归因的现网 MySQL 迁移 SQL、手动 replay 顺序、验数 SQL 与人工联调清单
    - **链接**: [播放流水补归因 MySQL 迁移与人工测试清单](memory/2026-03-15/play_replay_mysql_rollout_checklist.md)

- **日期**: 2026-03-15
    - **特性摘要**: 引入播放归因置信度分层，禁止 Apple Music 流媒体与 Roon 等弱来源继续新建专辑结构并收紧 `track` 身份解析
    - **链接**: [播放归因置信度与弱来源止血](memory/2026-03-15/play_resolution_confidence_guardrails.md)

- **日期**: 2026-03-15
    - **特性摘要**: 修复专辑被曲目发布日期拆分的问题，新增重复专辑合并清洗命令并统一迁移专辑关联
    - **链接**: [重复专辑清洗与专辑身份修复](memory/2026-03-15/duplicate_album_cleanup_and_album_identity_fix.md)

- **日期**: 2026-03-15
    - **特性摘要**: 按当前 MySQL 实库同步 SQL DDL，清理 DML 历史数据快照，并重建初始化脚本语义
    - **链接**: [SQL 结构与初始化脚本同步清单](memory/2026-03-15/sql_schema_sync_manifest.md)

- **日期**: 2026-03-15
    - **特性摘要**: 引入 model 层统一事务入口与 tx-aware DAO，完成 MusicBrainz 与 API 分层收口，并为 model 包建立 MySQL 方言 sqlmock 测试基座
    - **链接**: [DAO 边界与 MusicBrainz 事务收口特性清单](memory/2026-03-15/dao_boundary_musicbrainz_refactor_manifest.md)

- **日期**: 2026-03-15
    - **特性摘要**: 为 Bridge 资料库引入本地 SQLite 索引、增量同步接口与变更日志删除 tombstone，统一专辑/曲目列表的本地搜索排序分页语义
    - **链接**: [资料库本地索引与增量同步桥接特性清单](memory/2026-03-15/library_index_sync_bridge_feature_manifest.md)
    - **设计文档**: [BroadcastLibraryUpdate 客户端交互流程](docs/broadcast_library_update_flow.md)

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
