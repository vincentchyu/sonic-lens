# DAO 边界与 MusicBrainz 事务收口特性清单

## 日期

2026-03-15

## 背景

SonicLens 既有代码虽然约定 `internal/model/` 是数据库访问层，但 `internal/logic/musicbrainz/service.go` 仍然存在多处 `model.GetDB().WithContext(ctx)...` 直接查询、更新和事务调用，导致：

- 表级 SQL 语义散落在 logic 层
- 多表事务直接暴露给业务编排层
- DAO 能力不可发现、不可复用
- 后续治理其他 logic/api 越层 DB 调用时缺少统一模式

同时，`internal/model/model_test.go` 只有简单参数校验，缺少真正可复用的 model 测试基座，无法稳定支撑 DAO 边界重构。后续扫描还发现 `api/server.go` 虽然去掉了裸 DB 调用，但仍直接调用多个 model DAO 与包级 MusicBrainz 逻辑，handler 中还混入了歌词回源与按 1000 条回扫 Insight 的业务流程。

## 本次改动

### 1. 新增 model 层统一事务入口

- 新增 `internal/model/tx.go`
- 提供 `InTx(ctx, func(tx *gorm.DB) error) error`
- 约束事务能力由 model 暴露，logic 只负责组合多个 DAO

### 2. 为 MusicBrainz 链路补齐 tx-aware DAO

补充了以下 DAO 能力，使 `musicbrainz` logic 不再直接拼接 GORM 语句：

- `DeleteAlbumReleaseMBByAlbumIDTx`
- `ClearTrackAlbumMBRecordingIDByAlbumIDTx`
- `UpdateAlbumSyncStatusTx`
- `UpdateAlbumFieldsTx`
- `UpdateReleaseMBJSONDataTx`
- `GetTrackAlbumsByAlbumTx`
- `GetTrackByIDTx`
- `UpdateTrackMusicBrainzPositionTx`
- `SaveTrackAlbumTx`
- `CountTrackAlbumsByAlbumAndRecordingIDTx`
- `CreateTrackAlbumTx`

### 3. 收口 MusicBrainz logic 的直接 DB/事务调用

- `SearchAndCacheReleases` 在状态重置时改为调用 `model.InTx(...) + DAO`
- `SearchAndCacheReleases` 更新专辑同步状态改为 `model.UpdateAlbumSyncStatus`
- `DeepingMaintenance` 的事务入口改为 `model.InTx(...)`
- 事务闭包内不再直接 `tx.Where/First/Save/Create/Updates`
- 事务闭包仅负责流程编排，具体表操作全部委派给 model DAO

### 4. 建立 model 测试基座

- `internal/model/model_test.go` 新增 `newModelTestDB`
- 测试基座接管 `config.ConfigObj.Database.Type` 与全局 DB 指针
- 基座改为 `GORM MySQL dialector + sqlmock`，直接校验 MySQL 方言 SQL 和事务顺序
- 新增 `internal/model/tx_test.go`
  - 验证 `InTx` 出错回滚
  - 验证 MusicBrainz 相关 DAO 在事务内生成的 MySQL SQL 与事务顺序

### 5. 收口 API 到 logic 层

- `api/server.go` 中原先直接调用 `model.GetTrackAlbumByTrackID`、`model.GetTrackLyricsByLookup`、`model.GetOrCreateTrackLyrics`、`model.GetReleasesByAlbumID`、`model.LinkAlbumToMBID`、`model.GetAlbumWithTracks`、`model.DeleteTrackAlbumLink`
- 新增 `internal/logic/musicbrainz/api_service.go`，为 API 层提供 `SearchAndCacheReleases`、`GetReleasesByAlbumID`、`LinkAlbumToMBID`、`DeepingMaintenance` 的统一入口
- `internal/logic/insight/service.go` 新增 `GetInsightByID` 与 `GetLyrics`
- `internal/logic/track/service.go` 新增 `GetTrackAlbumByTrackID`、`GetAlbumDetail`、`DeleteTrackAlbumLink`
- `/api/insights/:id/logs` 改为按 ID 直取 Insight，不再通过 `GetAllInsights(..., 1000, 0, "")` 回扫
- `/api/track-lyrics` 的歌词查库、回源和缓存写入全部回收到 `insightService.GetLyrics`
- `/api/musicbrainz/*` 与专辑详情、解绑接口全部通过 logic service 调用

### 6. 清理默认测试面的环境耦合

- 将以下依赖真实机器环境或外部服务的测试标记为 `integration`
  - `internal/logic/track/service_test.go`
  - `core/exec/exec_test.go`
  - `core/lastfm/lastfm_test.go`
  - `core/musicbrainz/musicbrainz_test.go`
  - `core/musixmatch/musixmatch_test.go`
- 默认 `go test ./...` 不再要求本地音乐文件路径、私有 `config_bak.yaml`、Last.fm/MusicBrainz/Musixmatch 凭据
- 如需执行这些手工验证或联调测试，应显式使用 `go test -tags=integration ./...`

## 影响范围

- `internal/logic/musicbrainz/service.go`
- `internal/logic/musicbrainz/api_service.go`
- `internal/logic/insight/service.go`
- `internal/logic/track/service.go`
- `internal/model/tx.go`
- `internal/model/album_release_mb.go`
- `internal/model/album.go`
- `internal/model/release_mb.go`
- `internal/model/track.go`
- `internal/model/track_album.go`
- `internal/model/track_insight.go`
- `internal/model/model_test.go`
- `internal/model/tx_test.go`
- `api/server.go`

## 新约束

- 后续在 `logic/api/cmd` 中禁止新增 `model.GetDB().WithContext(ctx)...`
- 新的单表 SQL 一律先补 DAO，再由上层调用
- 多表事务统一走 `model.InTx(...)`
- 需要参与事务的 DAO 应优先提供 `Tx` 版本
- `api/` 不应直接调用 model DAO；HTTP handler 只负责参数绑定、响应编码和错误映射，业务流程统一经 `internal/logic/`
- 默认测试面必须保持可自动化、可重复；本地脚本型或外部服务型测试不能直接留在无 tag 的 `_test.go` 中

## 验证

执行：

```bash
go test ./internal/model/... ./internal/logic/musicbrainz/... ./api/...
go test ./...
```

结果：

- `internal/model` 通过
- `internal/logic/musicbrainz` 通过
- `api` 通过
- 默认 `go test ./...` 通过

## 后续计划

- 继续治理 `internal/logic/insight/service.go` 的直连 `TrackInsight` 查询
- 继续治理 `api/server.go` 中直连 `TrackAlbum` 的查询和删除
- 逐步形成“ctx 包装 + tx 内核”的 DAO 统一模式
