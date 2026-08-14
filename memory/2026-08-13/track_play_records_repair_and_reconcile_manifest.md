# 听歌流水历史残缺数据自动修补与全表播放量纠偏特性清单

## 1. 特性背景与概述
在过去的听歌流水数据积累中，`multimedia.track_play_records` 表包含 9002 条历史记录。由于早期客户端上报格式未做完整规范剥离，部分听歌流水带有 Apple Music 发行类型后缀（如 ` - Single`、` - EP`、` - LP`），且缺少 `resolved_track_id`、`album_id` 或 `genre` 等关键归因字段。

本特性建立了多层级智能元数据自动修补引擎，并提供了 CLI 命令（`go run . replay-track-play-records --repair --reconcile`），实现了**历史听歌流水残缺字段的自动对齐与修补**，并基于修复后的流水数据作为唯一事实源（SST）**全量纠偏校对 `track.play_count`、`album.play_count` 及流派播放统计**。

---

## 2. 核心变更细节

### 2.1 听歌流水修补引擎 (`internal/model/track_play_record.go`)
- **新增结构与接口**：
  - `RepairAndReconcileTrackPlayRecordsReport`：输出修补统计。
  - `RepairAndReconcileTrackPlayRecordsTx` / `RepairAndReconcileTrackPlayRecords`：在数据库事务中扫描元数据残缺的听歌流水记录。
- **4 层修补归因策略**：
  1. **Pass 1 (同名完备流水继承)**：利用已成功归因的同 `(artist, clean_album, track)` 流水继承 `resolved_track_id`, `album_id`, `genre`, `album_subtitle`。
  2. **Pass 2 (Track 主表匹配)**：自动剥离 `Album` 中的连字符后缀（`common.ParseAlbumTitleAndReleaseType`），并经由 `normalizeTrackStorageText` 规范化后与 `Track` 主表实体匹配，补全 `resolved_track_id` 并定位 `album_id`。
  3. **Pass 3 (Album 主表匹配)**：以 `artist` + `clean_album` 在 `Album` 主表查找，补全 `album_id` 与 `genre`。
  4. **Pass 4 (封面与关联修复)**：补齐 `cover_art_path` 封面路径并更新 `resolution_status = 'resolved'`、`library_applied = true`。

### 2.2 CLI 命令扩展 (`cmd/replay_track_play_records.go`)
- 为 `replay-track-play-records` 添加 `--repair` 和 `--reconcile` 选项：
  - `--repair`：触发 `RepairAndReconcileTrackPlayRecords` 流水修复引擎。
  - `--reconcile`：依次触发 `ReconcileTrackPlayCounts`、`ReconcileAlbumPlayCounts` 和 `ReconcileGenrePlayCounts` 对账校对。

---

## 3. 验证与修补结果

- **单元测试**：运行 `go test -v ./internal/model/... -run "TestRepairAndReconcileTrackPlayRecords"` 全部 PASS。
- **MySQL 真实数据修复表现**：
  - **处理残缺记录**：`7,203` 条
  - **修补 ResolvedTrackID**：成功补全 `4,909` 条（未归因流水从 6891 条骤降至 1982 条）
  - **修补 AlbumID**：成功补全 `284` 条
  - **修补 Genre**：成功补全 `3,681` 条
  - **播放量全表对账**：《冀西南林路行》(140次)、《万能青年旅店》(128次)、《OK Computer》(124次) 等热门专辑物理 `play_count` 100% 精确复原并支持倒序排列。
