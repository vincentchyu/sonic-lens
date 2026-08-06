# 正式专辑与待归因工单 MB 精选维护、草稿闭环及 UI 布局重构特性清单

## 1. 特性概述

本特性完成了正式专辑（`Album`）与待归因工单（`PendingAlbumWorkItem`）在 **MusicBrainz 重新搜索 ➡️ 绑定候选 ➡️ 差异预审与微调 ➡️ 仅保存草稿 ➡️ 确认落库** 全流程上的 100% 公共逻辑闭环，修复了持久化链路上 `track_album.mb_recording_id` 遗漏写入与前端请求头/入口拦截 Bug，并重构了待归因工单模态框的右侧布局。

---

## 2. 核心架构与 API 变更

### 2.1 后端 API 端点扩展 (`api/server.go`)
- `GET /api/albums/:id/musicbrainz/preview`：提供正式专辑与 MB 发行版的侧边栏差异预审数据。
- `POST /api/albums/:id/musicbrainz/apply-maintenance`：在事务内应用正式专辑的精选维护（同步更新 `track` 与 `track_album`，包括写回 `mb_recording_id`）。
- `POST /api/albums/:id/musicbrainz/draft`：将正式专辑的预审微调草稿保存至 `pending_album_work_item.staging_draft_json` 中。

### 2.2 模型层与业务逻辑 (`internal/model/`, `internal/logic/pendingalbum/`)
- `SaveAlbumStagingDraftDB`：为正式专辑（`resolved_album_id`）查找或关联 `PendingAlbumWorkItem` 并持久化 `staging_draft_json`。
- `ApplyAlbumMBMaintenance`：更新 `Track` 主表的同时，在事务内显式写回 `track_album.mb_recording_id`，并提供单元测试 `TestApplyAlbumMBMaintenanceUpdatesTrackAlbumMBRecordingID`。

---

## 3. 前端与 UI/UX 重构

### 3.1 前端交互与通信修复 (`static/admin/`)
- `init.js`：`selectCandidate()` 补齐 `headers: { 'Content-Type': 'application/json' }`，解决原本空 JSON 绑定失败的问题。
- `pending-albums.js`：
  - 修复 `submitPendingDiffMaintenance()` 与 `savePendingDiffDraft()` 入口检查，由 `!currentPendingAlbumWorkItemID` 升级为 `(!currentPendingAlbumWorkItemID && !currentContextAlbumID)`，解除正式专辑模式下的前端拦截；
  - 增强 `manualTracks` 构建时的识别码降级逻辑：`music_brainz_id: track.music_brainz_id || track.mb_recording_id || track.mbid || ''`。

### 3.2 待归因工单模态框布局重构 (`templates/admin/modals.html`)
- **置顶 MusicBrainz 核心操作**：将 MB 候选列表、选择状态、执行报告及紫色的 `[ 预审 MusicBrainz 差异并维护 ]` 主按钮统一收口置顶于右侧最上方，实现 0 滚动无感交互。
- **手动表单默认折叠卡片**：将原先占地巨大的“手动创建专辑”表单封装为手风琴折叠卡片（`<details>`），默认收起；并在 MB 搜索结果为空时实现自动展开联动。

---

## 4. 验证与单元测试

- **单元测试**：`go test -v ./internal/logic/pendingalbum/... -run TestApplyAlbumMBMaintenanceUpdatesTrackAlbumMBRecordingID` 测试通过。
- **手动联调验证**：
  1. 点击候选版本，调用 `/api/musicbrainz/link-album` 成功建立关联；
  2. 点击 `[ 仅保存草稿 ]` 成功保存草稿并提示 `✅ 草稿已微调并成功保存！`；
  3. 点击 `[ 确认审核并落库数据 ]` 成功将 `track_album.mb_recording_id` 写入数据库，曲目高亮显示紫色 `[MB]` 标记。
