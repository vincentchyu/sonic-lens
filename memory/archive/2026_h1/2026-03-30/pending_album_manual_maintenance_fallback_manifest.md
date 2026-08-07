# 待归因专辑手动维护兜底闭环

## 日期

- 2026-03-30

## 背景

- 待归因专辑工作台原先强依赖 MusicBrainz 候选与 `selected_mbid` 才能执行深度维护。
- 中文、小语种或冷门专辑在 MusicBrainz 缺失时，工单无法继续，只能依赖零散手工补库。

## 本次闭环

- 新增 `POST /api/pending-albums/work-items/:id/manual-maintenance`。
- 待归因专辑维护现在统一收口为四段骨架：
  - 解析来源数据
  - 落库专辑结构
  - replay 播放/收藏
  - 收尾工单状态
- MusicBrainz 模式与手动模式共享“专辑结构写入 + replay + 完成态”逻辑，唯一差异是来源解析方式不同，且只有 MusicBrainz 模式会写 `release_mb` / `album_release_mb`。
- 手动模式提交整专曲目表后，会直接把目标专辑写成稳定专辑：
  - `album.sync_status=3`
  - `pending_album_work_item.status=completed`
- 手动模式不会伪造 MusicBrainz 发行版关系；缺失 MB 数据时只维护 `album/track/track_album`。

## 前端交互

- 仅更新 v1 [dashboard.html](../../../templates/dashboard.html)。
- 在待归因专辑维护弹窗右侧新增“手动创建专辑”面板。
- 表单默认预填冻结上下文中的：
  - 专辑名
  - 专辑艺术家
  - 显示艺人
  - 曲名
  - 已有碟号/曲序
  - 时长与弱线索（若存在）
- 对每个默认值明确提示其来源，便于用户直接复用或覆盖修正。
- 无物理位置的冻结曲目会生成“待补位置”行，必须补齐 `disc_number/track_number` 后才能提交。
- 手动曲目行现在使用实时输入同步；用户修改曲名后即使不失焦直接提交，也必须以当前输入值为准。
- 当用户修改 `disc_number/track_number`、新增或删除曲目时，前端会立即按“同碟号、同曲序优先”规则重排显示；曲名允许重复，唯一约束只落在 `(disc_number, track_number)`。

## 后端落库规则

- 专辑仍通过 `ResolveCanonicalAlbumForPendingContextTx` 复用既有 curated album，避免重复建专辑。
- 曲目仍通过 `GetOrCreateTrackByIdentityTx` / `UpdateTrackCuratedMetadataTx` / `UpsertTrackAlbumTx` 维护。
- 手动维护提交的 `manual_tracks[].title` 是 curated 曲目标题的唯一真值；`evidence_titles` 只用于冻结上下文 replay 时的兜底归因，不得反向覆盖手填曲名。
- 若工作项或历史 replay 已绑定到旧 `resolved_track_id`，手动维护复用该曲目时也必须把 `track.artist/album/track` 身份字段修正到本次提交值，避免数据库继续保留旧默认标题。
- 当冻结播放流水/收藏事件里的旧标题与手填曲名不一致时，不能只依赖后续 replay 再按标题匹配；维护流程需在落库阶段直接把该工单冻结的 `track_play_records` / `track_favorite_event` 显式绑定到目标 `track.id`，避免出现 `track_album.track` 已修正但 `track.track`、播放回填、收藏回填仍停留在旧标题的分裂状态。
- 冻结播放流水与收藏事件继续复用：
  - `ReplayTrackPlayRecords`
  - `ApplyTrackFavoriteEventsByIDs`

## 文件

- `internal/logic/pendingalbum/service.go`
- `internal/logic/pendingalbum/service_test.go`
- `internal/model/track.go`
- `api/server.go`
- `api/api.md`
- `templates/dashboard.html`

## 验证

- `go test ./internal/logic/pendingalbum -run 'TestManualMaintainPendingAlbumWorkItem'`
- `go test ./internal/model -run 'TestUpdateTrackMusicBrainzMetadataTx|TestGetPendingAlbumGroupsAndCreateWorkItem'`
- `go test ./api/...`

## 注意事项

- 全量 `go test ./internal/model` 当前仍有既存失败，本次只验证了改动直接命中的用例。
- 手动维护是“提交即执行”，不持久化前端草稿。
