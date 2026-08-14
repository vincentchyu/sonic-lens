# *_stat 统计表移除自增 ID 与复合主键重构清单

## 概述
针对所有仪表盘/排行榜派生统计表（`*_stat` 结尾表，包含 `top_artist_stat`、`top_album_stat`、`top_genre_stat`、`track_rank_stat` 与 `play_source_stat`）在每次定时刷新时先删除后插入导致 MySQL 自增 `id` 快速飙升（达到 22971+）并带来写放大、主键开销与自增锁开销的问题，进行了彻底的领域模型与数据库主键重构。

## 变更明细

### 1. 全库 8 个 `*_stat` 表覆盖盘点
1. **`top_artist_stat`**: 彻底移除自增 `id` 列，重构为复合主键 `(period_days, metric_type, rank)`。
2. **`top_album_stat`**: 彻底移除自增 `id` 列，重构为复合主键 `(period_days, rank)`。
3. **`top_genre_stat`**: 彻底移除自增 `id` 列，重构为主键 `rank`。
4. **`track_rank_stat`**: 彻底移除自增 `id` 列，重构为复合主键 `(period_type, rank)`。
5. **`play_source_stat`**: 彻底移除自增 `id` 列，重构为主键 `source`。
6. **`dashboard_stat`**: 移除 `autoIncrement` 属性，保持固定单行 `id = 1` 覆盖更新。
7. **`play_trend_daily_stat`**: 原生无自增 `id` 列，保持 `stat_date` 自然主键。
8. **`play_trend_hourly_stat`**: 原生无自增 `id` 列，保持 `(stat_date, hour)` 复合主键。

### 2. 平滑 Migration 切替 (`internal/model/init.go`)
- 在 `autoMigrateMySQLDevSchema` 初始化流程中引入 `migrateLegacyStatTablesWithoutID`。
- 自动检测并安全 `DropTable` 带有旧 `id` 字段的统计快照表，而后通过 `AutoMigrate` 生成精整无 `id` 的复合主键表，随后启动时通过既有刷新逻辑填充最新排行榜数据。

### 3. 测试与同步对齐 (`internal/model/dashboard_stat_test.go` & `internal/sync/d1_sync.go`)
- 更新 `dashboard_stat_test.go` 中的 sqlmock 数据集列，移除对 `"id"` 的模拟输出。
- 在 `d1_sync.go` 中针对本地读取导出的 `PlaySourceStat` 字段调整为 `(source, count, updated_at)`，移除已删除的 `row.ID` 字段引用。

## 验证结论
- 执行 `go test ./internal/model/...` 0.479s PASS。
- 执行 `go test ./internal/sync/...` 0.419s PASS。
- 执行 `go test ./internal/...` 100% 全部通过。
