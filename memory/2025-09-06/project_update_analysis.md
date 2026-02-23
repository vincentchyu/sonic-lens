# 项目更新分析报告

## 1. 核心功能更新

### 1.1 播放记录追踪

- 实现了 `TrackPlayRecord` 数据模型，用于存储每次播放的详细信息
- 在 `internal/scrobbler/track_check_playing.go` 中集成了数据库存储逻辑
- 当曲目播放进度超过55%时，将播放记录保存到本地数据库

### 1.2 播放统计

- 实现了 `TrackPlayCount` 数据模型，用于统计每首曲目的播放次数
- 使用乐观锁机制保证并发安全
- 在每次成功scrobble后更新对应曲目的播放次数

### 1.3 数据同步

- 实现了未同步到Last.fm的播放记录的同步功能
- 通过 `GetUnscrobbledRecords` 方法获取未同步的记录
- 同步成功后更新记录的 `Scrobbled` 状态

## 2. 代码结构优化

### 2.1 模块重构

- 将核心模块移至 `core/` 目录下，包括 `applesciprt/`, `audirvana/`, `db/`, `exec/`, `lastfm/`, `log/`, `musixmatch/`,
  `roon/`, `telemetry/`
- 业务逻辑移至 `internal/` 目录下，包括 `logic/`, `model/`, `scrobbler/`

### 2.2 数据库集成

- 使用 GORM 实现数据库操作
- 实现了自定义日志记录器集成 zap 和 OpenTelemetry

## 3. 新增功能模块

### 3.1 分析报告功能

- 新增 `cmd/analysis_cmd.go` 实现分析报告命令
- 提供播放统计、推荐曲目等分析功能
- 通过Web界面展示分析结果

### 3.2 内存管理工具

- 新增 `cmd/memory_tool.go` 实现内存管理功能
- 提供特性清单和记忆索引管理