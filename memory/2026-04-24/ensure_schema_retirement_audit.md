# ensure*Schema 退场审计（2026-04-24）

本清单基于以下事实源交叉整理：

- `internal/model/schema_*.go`
- `internal/model/init.go`
- `internal/model/*.go` struct 定义
- `internal/model/sql/ddl/*.sql`
- `internal/sync/d1_sync.go`

目标不是继续接受“启动时自动补库”，而是为稳定期迁移到“显式 migration / DDL + 启动校验”提供删改顺序。

补充现状（2026-04-24）：

- MySQL 主链路中的 `ensure*Schema()` / `Ensure*Schema()` 调用已从 `internal/model/init.go`、`dashboard_stat.go` 和 `internal/sync/d1_sync.go` 退场。
- 本文保留为“为什么能退、退掉了哪些、剩余 D1 为什么暂不硬删”的审计记录，不再表示这些 MySQL ensure 仍在线上主路径执行。

---

## 结论总览

### A. 可在补齐正式 migration 后删除的 MySQL ensure

这些 ensure 对应的字段/索引/表结构已经能在代码和 `sql/ddl` 中找到稳定事实源；稳定期应改为：

1. 先执行显式 migration / SQL
2. 启动时只做只读校验
3. 删除运行时 `ensure*`

清单：

- `ensureTrackIdentitySchema`
- `ensureTrackPlayRecordTraceSchema`
- `ensureTrackPlayRecordCoverArtSchema`
- `ensureTrackPlayRecordIdentitySchema`
- `ensureAlbumCoverSchema`
- `ensureAlbumOriginalReleaseDateSchema`
- `ensureAlbumTitleSubtitleSchema`
- `ensureAlbumTitleMetadataSchema`
- `ensureAlbumIdentitySchema`
- `ensureTrackFavoriteEventIdentitySchema`
- `ensureLLMCallLogSchema`
- `ensureInsightFeedbackSchema`
- `ensureInsightJobSchema`
- `EnsureArtistProfileSchema`
- `EnsureDashboardStatSchema`

说明：

- 这批逻辑目前仍在 `internal/model/init.go` 的 MySQL 启动链路里执行。
- 其中大部分已经有对应 DDL；本轮已补齐 `insight_job.sql`、`album.title_metadata`、`track_play_records.cover_art_path`、`top_album_stat.album_subtitle`。
- `ensureInsightJobSchema` 当前只是 `AutoMigrate(&InsightJob{})` 包装，不应再作为生产结构演进手段。
- `EnsureDashboardStatSchema` / `EnsureArtistProfileSchema` 本质上也是“缺表建表、缺列补列”，稳定期不应继续在主启动链路做写库修复。

### B. 需要优先补 migration 再删的高风险项

这些项虽然已能在代码/DDL 中找到事实源，但因为它们直接影响核心身份、归因与客户端展示，删除 ensure 前必须先确认生产库结构百分百齐平。

清单：

- `ensureTrackIdentitySchema`
  原因：涉及 `track` 唯一索引、`track_play_records.disc_number`、`track_rank_stat` 字段和 `track_album` 索引。
- `ensureAlbumTitleSubtitleSchema`
  原因：一次牵涉 `album` / `track` / `track_play_records` / `track_favorite_event` / `pending_album_work_item` / `top_album_stat` 多表。
- `ensureTrackPlayRecordIdentitySchema`
  原因：影响播放流水归因、补写和最近播放 fallback join。
- `ensureInsightJobSchema`
  原因：之前没有独立 DDL，且表通过 `AutoMigrate` 演进；本轮虽已补档，但生产库仍需显式迁移。
- `EnsureDashboardStatSchema`
  原因：包含多张 dashboard 表的建表/补列逻辑，影响管理端统计链路。

### C. 暂时保留，但应改造成“专用迁移”而不是“同步时自修复”

这类 ensure 不在 MySQL 主库启动链路，而在 D1 镜像同步链路中；它们当前承担“远端镜像首次建表 + 旧结构迁移”的职责。

清单：

- `D1Client.ensureSyncMetadataSchema`
- `D1Client.ensureTracksSchema`
- `D1Client.ensureTrackPlayRecordsSchema`
- `D1Client.ensureTopAlbumStatSchema`

建议目标状态：

- 保留 D1 独立 migration 入口
- 去掉 `SyncAll`/构造链路中的隐式 repair
- 在同步开始前做 schema version 校验，不匹配则报错并引导执行迁移

原因：

- D1 是远端镜像库，不像本地 SQLite 可以默认重建
- 当前 `migrateTracksTable` / `migrateTrackPlayRecordsTable` 直接在同步链路里做 rename-copy-drop，稳定期风险过高

---

## 按函数逐项判断

### 1. `ensureTrackIdentitySchema`

状态：`补 migration 后删除`

依据：

- `track.sql` 已有 `album_subtitle`、`disc_number`、`uidx_t_aaastdntn`
- `track_rank_stat.sql` 已有 `track_number`、`disc_number`、`track_id`、封面字段、唯一索引
- `track_album.sql` 已有 `idx_ta_album_disc_track`

剩余动作：

- 把这组变更整理成一次正式 migration
- 生产执行后删除启动时 ensure

### 2. `ensureTrackPlayRecordTraceSchema`

状态：`补 migration 后删除`

依据：

- `track_play_records.sql` 已包含 `trace_id`、`root_span_id`、`trace_sampled`

### 3. `ensureTrackPlayRecordCoverArtSchema`

状态：`补 migration 后删除`

依据：

- 本轮已将 `cover_art_path` 补入 `track_play_records.sql`

### 4. `ensureTrackPlayRecordIdentitySchema`

状态：`补 migration 后删除`

依据：

- `track_play_records.sql` 已包含 `idx_track_play_records_identity_subtitle`

### 5. `ensureAlbumCoverSchema`

状态：`补 migration 后删除`

依据：

- `album.sql` 已包含 `cover_art_url`、`cover_art_mime`、`cover_art_object_key`、`idx_album_cover_art_object_key`

### 6. `ensureAlbumOriginalReleaseDateSchema`

状态：`补 migration 后删除`

依据：

- `album.sql` 已包含 `original_release_date`

### 7. `ensureAlbumTitleSubtitleSchema`

状态：`补 migration 后删除`

依据：

- `album.sql` 已包含 `name_subtitle`
- `track.sql`、`track_play_records.sql`、`track_favorite_event.sql`、`pending_album_work_item.sql` 已包含 `album_subtitle`
- `stat.sql` 的 `top_album_stat` 本轮已补齐 `album_subtitle`

### 8. `ensureAlbumTitleMetadataSchema`

状态：`补 migration 后删除`

依据：

- 本轮已将 `title_metadata` 补入 `album.sql`

### 9. `ensureAlbumIdentitySchema`

状态：`补 migration 后删除`

依据：

- `album.sql` 已包含 `uidx_album_artist_name_subtitle_release_date`

### 10. `ensureTrackFavoriteEventIdentitySchema`

状态：`补 migration 后删除`

依据：

- `track_favorite_event.sql` 已包含 `idx_tfe_identity_subtitle`

### 11. `ensureLLMCallLogSchema`

状态：`补 migration 后删除`

依据：

- `llm_call_log.sql` 已包含 `analysis_target_type`、`target_key`、`target_metadata` 以及对应索引

### 12. `ensureInsightFeedbackSchema`

状态：`补 migration 后删除`

依据：

- `track_insight_feedbacks.sql`、`album_insight_feedbacks.sql` 已包含 `reason_codes`、`section_key`、`source_platform`

### 13. `ensureInsightJobSchema`

状态：`优先补 migration 后删除`

依据：

- 原先通过 `AutoMigrate(&InsightJob{})` 维护
- 本轮已新增 `internal/model/sql/ddl/insight_job.sql`

风险：

- 之前没有独立 DDL，生产库很可能靠历史启动自动补齐

### 14. `EnsureArtistProfileSchema`

状态：`补 migration 后删除`

依据：

- `artist_profile.go` struct 稳定
- `artist_profile.sql` 已独立存在
- `stat.sql` 中旧的重复 `artist_profile` 定义已在本轮移除

### 15. `EnsureDashboardStatSchema`

状态：`优先补 migration 后删除`

依据：

- `stat.sql` 覆盖了 dashboard 统计表
- 本轮已修正 `top_album_stat.album_subtitle`
- 代码中仍用 `ensureTableAndColumns` 在运行时补表补列

风险：

- 涉及多张表
- 当前仍被 `RefreshDashboardStats*` 调用链依赖

---

## D1 侧单独结论

`internal/sync/d1_sync.go` 里的 ensure 不建议直接硬删，应分两步走：

1. 先抽出为“D1 schema migration”命令或独立入口
2. 再把同步链路里的隐式修复逻辑删除

原因：

- 这几段逻辑包含 rename-copy-drop 的在线迁移
- 直接删除会导致旧 D1 镜像实例无法自举
- 但继续留在同步主链路，会把“同步失败”和“远端 schema 变更”耦合到一起

---

## 推荐落地顺序

1. 为 MySQL 主库补一批正式 migration，覆盖本文件 A/B 组全部项。
2. 在预发/生产执行 migration，并加启动时只读 schema 校验。
3. 删除 `internal/model/init.go` 中对这些 `ensure*Schema()` / `Ensure*Schema()` 的调用。
4. 保留开发环境 `AutoMigrate`，但将生产路径与其彻底隔离。
5. 为 D1 增加独立 migration 入口，随后删除 `D1Client` 同步主链路中的 ensure/migrate 逻辑。

---

## 本轮已完成的对齐

- 新增 `internal/model/sql/ddl/insight_job.sql`
- 补齐 `album.sql.title_metadata`
- 补齐 `track_play_records.sql.cover_art_path`
- 补齐 `stat.sql.top_album_stat.album_subtitle`
- 移除 `stat.sql` 中与独立 `artist_profile.sql` 冲突的旧定义
- 修正 `EnsureDashboardStatSchema()`，把 `TopAlbumStat.AlbumSubtitle` 纳入补列清单
