# 记忆索引

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

## 2025-12-06

- **日期**: 2025-12-06
    - **特性摘要**: 实现 Cloudflare D1 数据同步、Workers API 及静态 Web 展示页面，支持无服务器环境下的数据查询与可视化
    - **链接**: [Cloudflare D1 数据同步与静态页面展示特性清单](memory/2025-12-06/cloudflare_d1_web_feature_manifest.md)

## 2025-09-19

- **日期**: 2025-09-19
    - **特性摘要**: 为Redis客户端实现完整的链路跟踪和日志记录功能，包括自定义日志钩子、OpenTelemetry集成、慢查询检测等，与数据库模块保持一致的设计和实现
    - **链接**: [Redis链路跟踪和日志记录功能特性清单](memory/2025-09-19/redis_tracing_and_logging_feature_manifest.md)

## 2025-09-18

- **isDev 配置项特性清单**: 记录了 isDev
  配置项的作用和使用场景，用于控制数据库表结构的初始化行为。 [链接](./memory/2025-09-18/isDev_configuration_feature_manifest.md)

## 2025-09-15

- **仪表板增强和流派自定义特性清单**:
  为仪表板添加了流派自定义功能，用户可以自定义流派映射规则。 [链接](./memory/2025-09-15/dashboard_enhancement_and_genre_customization_feature_manifest.md)

## 2025-09-13

- **曲目模型增强和参数优化特性清单**:
  增强了曲目模型，支持更多音乐元数据，并优化了相关参数。 [链接](./memory/2025-09-13/track_model_enhancement_and_parameter_optimization_feature_manifest.md)

## 2025-09-11

- **日期**: 2025-09-11
    - **特性摘要**: 实现曲目收藏功能，允许用户通过前端界面收藏正在播放的曲目，并在Apple Music和Last.fm上同步收藏状态
    - **链接**: [曲目收藏功能特性清单](memory/2025-09-11/track_favorite_feature_manifest.md)

## 2025-09-10

- **日期**: 2025-09-10
    - **特性摘要**: 实现Last.fm音乐收藏（喜爱）功能，包括检查和设置歌曲收藏状态
    - **链接**: [Last.fm收藏功能特性清单](memory/2025-09-10/lastfm_favorite_feature_manifest.md)

- **日期**: 2025-09-10
    - **特性摘要**: 为TrackPlayRecord模型增加分页查询未同步记录的功能，并实现批量同步到Last.fm的逻辑
    - **链接**: [播放记录同步功能特性清单](memory/2025-09-10/track_play_record_sync_feature_manifest.md)

- **日期**: 2025-09-10
    - **特性摘要**: 定义Go后端开发的分层架构规范，确保业务逻辑、数据访问和API接口的职责分离
    - **链接**: [Go后端开发规范](memory/2025-09-10/golang_backend_development_rules.md)

- **日期**: 2025-09-10
    - **特性摘要**: 重构TrackPlayCount模型为新的Track模型，并增加对Apple Music和Last.fm喜欢状态的支持
    - **链接
      **: [Track模型重构和喜欢状态功能特性清单](memory/2025-09-10/track_model_refactor_and_favorite_feature_manifest.md)

## 2025-09-07

- **日期**: 2025-09-07
    - **特性摘要**: 实现Apple Music播放跟踪支持，包括播放状态检查、记录保存和Last.fm同步
    - **链接**: [Apple Music支持功能特性清单](memory/2025-09-07/apple_music_support_feature_manifest.md)

- **日期**: 2025-09-07
    - **特性摘要**: 实现Apple Music的AppleScript接口支持，替换原有的nowplaying-cli实现
    - **链接**: [Apple Music AppleScript支持](memory/2025-09-07/applemusic_applescript_support.md)

## 2025-09-06

- **特性摘要**: 实现WebSocket实时播放信息推送功能，包括后端WebSocket服务和前端实时展示
- **链接**: [WebSocket实时播放信息功能特性清单](memory/2025-09-06/websocket_realtime_playback_feature_manifest.md)

## 2025-09-06

- **特性摘要**: 实现音乐分析功能，包括音乐偏好报告、定时报告生成和智能音乐推荐
- **链接**: [音乐分析功能特性清单](memory/2025-09-06/music_analysis_feature_manifest.md)

## 2025-09-06

- **特性摘要**: 分析项目更新内容，包括播放记录追踪、播放统计和数据同步功能
- **链接**: [项目更新分析](memory/2025-09-06/project_update_analysis.md)

## 2025-09-04

- **特性摘要**: 分析项目更新内容，包括播放状态检查优化和播放记录统计增强
- **链接**: [项目更新分析](memory/2025-09-04/project_update_analysis.md)

## 2025-09-03

- **特性摘要**: 实现播放统计功能，包括统计表设计、乐观锁机制和HTTP API接口
- **链接**: [播放统计功能特性清单](memory/2025-09-03/playback_statistics_feature_manifest.md)

## 2025-09-03

- **特性摘要**: 实现播放记录功能，包括播放记录表设计和手动同步功能
- **链接**: [播放记录功能特性清单](memory/2025-09-03/track_play_record_feature_manifest.md)