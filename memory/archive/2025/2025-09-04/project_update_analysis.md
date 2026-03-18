# 项目更新分析

## 更新概览

项目最近的更新主要集中在增强播放记录和播放统计功能，优化检查播放状态的逻辑，以及集成链路跟踪功能。

## 主要更新内容

1. **播放状态检查优化**:
    - 在 `scrobbler/track_check_playing.go` 中，增加了检查播放状态的优化逻辑。
    - 当连续100次检查未播放时，将检查间隔从3秒延长到60秒，以减少资源消耗。
    - 当检测到播放时，检查间隔会恢复到3秒。

2. **播放记录和统计增强**:
    - 在播放记录中标记歌曲完成时，增加了对播放统计的更新。
    - 使用乐观锁机制保证并发安全地更新播放统计。

3. **数据存储**:
    - 播放记录和播放统计都存储在本地 SQLite 数据库中，使用 GORM 作为 ORM 框架。

4. **链路跟踪集成**:
    - 集成了 OpenTelemetry 用于链路跟踪。
    - 在 `telemetry` 包中初始化了 tracer provider 和 exporter。
    - 在 `scrobbler/track_check_playing.go` 中，为每次检查播放状态的操作创建了新的 span，以便跟踪执行过程。

## 代码变更细节

### `scrobbler/track_check_playing.go`

- 增加了 `longSleep` 常量，值为60秒。
- 增加了 `checkCount` 常量，值为100次。
- 增加了 `isLong` 和 `isLong2` 变量，用于标记是否处于长间隔检查状态。
- 在 `AudirvanaCheckPlayingTrack` 和 `RoonCheckPlayingTrack` 函数中，增加了检查间隔调整的逻辑。
- 在标记歌曲完成时，调用 `model.IncrementTrackPlayCount` 更新播放统计。
- 使用 `telemetry.StartSpanForTracerName` 为每次检查创建新的 span。

### `telemetry/telemetry.go`

- 初始化了 OpenTelemetry 的 tracer provider 和 trace exporter。
- 提供了 `StartSpanForTracerName` 函数用于创建 span。

## 总结

本次更新通过优化播放状态检查逻辑，减少了在无播放时的资源消耗。同时，增强了播放记录和统计功能，并集成了链路跟踪功能，使项目更加完善和易于调试。