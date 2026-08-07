# D1 Incremental Single-Flight Sync Closure

## 日期
2026-03-23

## 特性摘要
- D1 同步恢复为启动可用，并且不再依赖本地调试时手工注释代码。
- D1 同步链路新增单飞保护，任何时刻只允许一个 `SyncAll` 运行，后续触发会直接跳过，避免定时器和启动同步互相滚动。
- 启动时首次同步改为串行执行，完成后再启动定时器，防止初次全量还未结束时又拉起下一轮同步。
- `track_play_records`、`tracks`、`top_album_stat` 的 D1 schema 与本地模型字段重新对齐，避免因字段漂移导致的回退全量和批量写入失败。

## 后端改动
- `internal/sync/d1_sync.go`
  - `D1Client` 增加同步中的原子锁状态，`SyncAll` 进入后先抢占单飞锁。
  - `track_play_records` 的 D1 初始化和迁移补齐 `album_id`、`music_brainz_id`、`resolved_track_id`、`resolution_status`、`resolution_confidence`、`library_applied`。
  - `top_album_stat` 补齐 `album_id`，并在 D1 初始化时保证索引存在。
  - 批量参数上限与字段数量同步对齐，避免批次计算和 SQL 参数个数漂移。
- `internal/sync/d1_scheduler.go`
  - 启动时先执行一次同步，结束后再启动定时器。
  - 若遇到同步进行中，定时器直接跳过本轮，而不是并发重入。

## D1 结构
- `cloudflare/d1_schema.sql`
  - 更新 `track_play_records`，与本地播放记录模型保持一致。
  - 更新 `top_album_stat`，补齐 `album_id`。

## 验证
- `go test ./internal/sync -run TestD1ClientSyncSingleFlight`
- `go test ./internal/sync/...`
- `go test ./... -run '^$'`

