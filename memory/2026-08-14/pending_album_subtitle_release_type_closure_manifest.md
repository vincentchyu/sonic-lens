# 待归因专辑与全链路 album_subtitle & release_type 闭环修复特性清单

- **特性分支/提交**: `feat(pendingalbum): full-link album_subtitle and release_type closure`
- **交付日期**: 2026-08-14
- **相关模块**: `internal/logic/pendingalbum`, `templates/admin/modals.html`, `static/admin/pending-albums.js`, `api/api.md`, `Docker MySQL (multimedia)`

---

## 1. 业务背景与问题根因

### 1.1 缺陷现象
在待归因工作项执行维护（如 Apple Music 采集的 `Communiqué (Remastered)`）后，发现目标 `album` 表的 `name_subtitle`、关联 `track` 的 `album_subtitle` 以及流水 `track_play_records` 的 `album_subtitle` 全部丢失并被更新为 `""`（空字符串），导致所有下游展示和统计均失去版本说明。

### 1.2 链路断裂根因
1. **DTO 结构体缺失**：`ManualPendingAlbumAlbumInput` 与 `PendingAlbumAlbumPreview` 均缺失 `album_subtitle` 与 `release_type`。
2. **物料解析丢失**：`resolveManualMaterial` 仅取了基础字段，未对 `AlbumCandidate.NameSubtitle` 赋值，且未从 `PendingAlbumWorkItem` 兜底继承。
3. **前端表单与 Payload 断流**：模态框与 JS 脚本在收集 `manual_album` 时遗漏了副标题与发行格式。
4. **事务落库多米诺覆盖**：`applyPendingAlbumStructureTx` 在更新 `track` 与回填 `track_play_records` 时，以 `resolvedAlbum.NameSubtitle`（空值）作为事实源执行全覆盖，导致原始采集数据被抹除。
5. **发行类型落库遗漏**：`applyPendingAlbumStructureTx` 在 `UpdateAlbumFieldsTx` 时遗漏了 `release_type` 的持久化。

---

## 2. 核心架构与修改实现

1. **DTO 与数据契约增强** (`internal/logic/pendingalbum/service.go`)：
   - `ManualPendingAlbumAlbumInput` 与 `PendingAlbumAlbumPreview` 均增加 `AlbumSubtitle` (`json:"album_subtitle"`) 与 `ReleaseType` (`json:"release_type"`)。
2. **预审与草稿生成完整性** (`service.go`)：
   - `PreviewPendingAlbumMBMaintenance` 与 `PreviewAlbumMBMaintenance` 均完整输出 `AlbumSubtitle` 与 `ReleaseType`。
3. **物料解析与三级保障继承** (`service.go`)：
   - `resolveManualMaterial` 自动调用 `common.ParseAlbumTitleAndReleaseType` 剥离连字符后缀保存干净主标题。
   - `AlbumCandidate.NameSubtitle` 支持“输入优先 -> 工作项上下文继承 -> 标题解析推导”三级保障。
   - `AlbumCandidate.ReleaseType` 同样支持显式输入优先与标题解析推导。
4. **事务持久化闭环** (`service.go`)：
   - `applyPendingAlbumStructureTx` 与 `ApplyAlbumMBMaintenance` 将 `name_subtitle` 与 `release_type` 纳入更新字典。
5. **Web 前端模态框与脚本** (`templates/admin/modals.html`, `static/admin/pending-albums.js`)：
   - 手动维护模态框与差异预审模态框增加“副标题 / 版本说明”及“发行类型”输入框，支持回显与提交。
6. **播放量递增与双重对账闭环**：
   - 在 `internal/model/track_play_record.go` 的 `ApplyTrackPlayRecordToResolvedTrackTx` 中，补齐在应用未归因流水时对 `IncrementAlbumPlayCountTx(tx, albumID)` 的原子递增调用。
   - 在 `service.go` 的待归因维护完成（`maintainPendingAlbumWorkItem`）与精选维护正式专辑（`ApplyAlbumMBMaintenance`）落库后，触发 `model.ReconcileAlbumPlayCounts` 兜底对账，确保目标专辑的 `play_count` 绝对与真实听歌流水和曲目播放数一致。
7. **单元测试与数据修复**：
   - 在 `service_test.go` 中新增副标题、发行格式及 `album.PlayCount` 计数全链路持久化单元测试。
   - 修正 Docker MySQL 库中受影响的专辑（ID 4764, ID 4765）及关联曲目与流水记录。

---

## 3. 验证与对账结果

- **单元测试**：
  ```bash
  go test -v ./internal/logic/pendingalbum/...
  go test ./internal/model/... ./internal/logic/track/... ./api/...
  ```
  结果全部 PASS。
- **数据库对账**：
  - `album` 4764（`name_subtitle = 'Remastered'`, `release_type = 'album'`）、`track`（9 首，`album_subtitle = 'Remastered'`）以及 `track_play_records`（12 条，`album_subtitle = 'Remastered'`）100% 结构化对齐。
  - `album` 4765（`name = 'Genesis'`, `name_subtitle = 'Remastered'`, `play_count = 9`，`sync_status = 3`）与关联的 9 条听歌流水和 9 首曲目 100% 精准对账一致。
