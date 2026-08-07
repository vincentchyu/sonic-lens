# 最近播放封面路径与首页视觉对齐特性清单

## 特性摘要

首页“最近播放”模块新增 `track_play_records.cover_art_path`，由播放阈值上报链路在落库时直接保存封面路径，Bridge 端在最近播放列表中优先展示封面图片，并统一使用和其他首页模块一致的金黄色标题条与更紧凑的时间表达。

## 变更范围

- 后端 `track_play_records` 表新增 `cover_art_path` 字段
- `HandleTrackPlaybackThreshold` 在写入播放流水时同步保存封面路径
- `GetRecentPlayRecords` / `GetRecentPlayRecordsByDays` 支持读取封面路径并兼容旧数据回退
- Bridge 首页最近播放改为封面优先展示，去掉重复的“最近播放”副标题文案
- `api/api.md` 补充最近播放响应字段说明

## 关键约束

- 播放封面路径以对象键优先，统一收敛为客户端可消费的 `cover_art_path`
- 最近播放列表应优先显示曲目、专辑、作者，时间只保留为短标签或可省略信息
- 首页最近播放头部应复用和其他热门模块一致的金黄色标题条，保持视觉节奏统一

## 验证

- `go test ./internal/model -run 'TestBuildTrackPlayRecordArtworkPath|TestGetRecentPlayRecordsReturnsCoverArtPath'`
- `go test ./internal/model -run 'TestProcessTrackPlayRecordResolvesPendingFavoriteEvent|TestProcessTrackPlayRecordResolvesPendingLastFmFavoriteEvent|TestReplayTrackPlayRecordsDryRunDoesNotModifyData|TestReplayTrackPlayRecordsAppliesPendingRecord|TestReplayTrackPlayRecordsFiltersByRecordID|TestReplayTrackPlayRecordsDefaultIgnoresArchivedHistoricalRows'`
- `go test ./internal/logic/track/...`
- `go test ./internal/logic/pendingalbum/...`
