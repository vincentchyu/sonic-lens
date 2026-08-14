# 冗余 CMD 工具、无效补数据定时器、遗留配置与历史迁移代码清理特性清单

## 1. 特性概述
本次重构完成了对项目中历史遗留的一次性脚本、废弃子命令、多余定时器、废弃配置字段及启动迁移嗅探代码的地毯式清理。
通过彻底消除无用代码与失效配置，项目消除了长期存在的认知幻觉与后台空跑噪音，同时严格保留了 `internal/model/` 底层的纯粹 DAO 与对账算法能力。

---

## 2. 变更清单与清理范围

### 2.1 命令行工具层 (`cmd/`) 彻底解耦与精简
- **删除独立脚手架目录**：
  - `cmd/reconcile_genres_only/`（历史单次流派对账脚本）
  - `cmd/restore_genres/`（历史流派补齐脚本与未使用的 6000+ 流派矩阵代码）
- **删除已废弃子命令实现**：
  - `cmd/album_cleanup.go`（`cleanup-duplicate-albums`, `cleanup-release-type-suffixes`）
  - `cmd/sync_records.go`（`sync-records`）
  - `cmd/replay_track_play_records.go`（`replay-track-play-records`）
  - `cmd/memory_tool.go`（`memory-tool`）
- **主程序 `main.go` 精简**：
  - 移除全部子命令挂载，移除对 `cmd` 包的 import，恢复极简纯粹的 API 服务入口。

### 2.2 定时器与同步层 (`internal/sync/`)
- **删除补归因重放定时器**：
  - 删除 `internal/sync/play_replay_scheduler.go`
  - 从 `main.go` 中移除 `d1sync.StartTrackPlayReplayScheduler` 启动调用
  - 从 `config/config.go` 及 yaml 配置文件中清理 `PlayReplayConfig` 字段与配置段。

### 2.3 配置系统与数据库初始化 (`config/` & `internal/model/`)
- **数据库配置与签名净化**：
  - 从 `config.DatabaseConfig` 移除 SQLite 历史残留字段 `Path` 和 `Type`。
  - 简化 `model.InitDB(l *zap.Logger)`，移除无用入参 `dataSourceName`。
  - 引入模型层通用数据库方言判定 `isMySQL(db)`，动态依据 GORM Dialector 判断 MySQL/SQLite，彻底解耦对全局配置 `Database.Type` 的硬编码依赖。
- **启动旧表 Drop 嗅探清理**：
  - 移除 `internal/model/init.go` 中的 `migrateLegacyStatTablesWithoutID` 方法，消除每次启动时的冗余 DDL 探测。

### 2.4 API 接口与管理后台前端 (`api/` & `static/` & `templates/`)
- **接口清理**：
  - 从 `api/server.go` 移除 `POST /api/admin/genres/backfill` 与 `POST /api/admin/genres/restore` 历史回补端点。
- **管理后台前端清理**：
  - 从 `templates/admin/main_sections.html` 移除 `btnRunBackfillGenres` 按钮，保留核心全量对账按钮。
  - 从 `static/admin/stats-reconcile.js` 移除 `triggerBackfillGenres()` 函数。

### 2.5 DAO 能力保留（严格遵守保留原则）
- `internal/model/album_cleanup.go`（`CleanupDuplicateAlbums`, `CleanupReleaseTypeSuffixes`）
- `internal/model/genre_restore.go`（`BackfillTrackPlayRecordGenresTx`）
- `internal/model/track_sync.go`（`GetTracksUpdatedSince` 等）
- `internal/model/track_play_record.go`（`RepairAndReconcileTrackPlayRecords`, `ReconcileTrackPlayCounts` 等）
全部保留作为纯粹的底层领域访问层方法，供日常测试、后台统计或管理后台调用。

---

## 3. 验证结果
- `go test -count=1 ./...` 全量单元测试 **100% 独立无依赖秒级通过**。
- `go build -v -o /dev/null .` 主程序二进制构建成功。
- `go run . --help` 命令行帮助信息干净整洁。
