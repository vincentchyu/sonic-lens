# 收藏投影视图与待归因收藏态闭环特性清单

## 日期

- 2026-03-28

## 背景

- 低信任来源的收藏操作会先写入 `track_favorite_event`，待后续归因完成后才回填到 `track`。
- 旧协议下，`/api/favorite` 与正在播放 `WS` 只暴露 `apple_music` / `lastfm` 两个布尔值，客户端无法区分“未收藏”和“已记录收藏但待归因”。
- 该问题会导致用户在当前播放页重复看到未点亮的收藏按钮，误以为前一次收藏失败。

## 本次收敛

- 在 `common` 中新增统一收藏枚举：`not_favorited`、`favorited`、`favorite_pending`、`unfavorite_pending`。
- 在 `internal/model/track_favorite_event.go` 新增按曲目身份读取最新未归因收藏意图的只读快照能力。
- 在 `internal/logic/track/` 新增 favorite projection 逻辑，统一合成：
  - `track` 表中的稳定收藏事实
  - `track_favorite_event` 表中的 pending 收藏意图
- `POST /api/favorite` 改为返回：
  - `apple_music`
  - `lastfm`
  - `apple_music_state`
  - `lastfm_state`
  - `favorite_state`
- `WS now_playing` payload 同步增加上述三个 state 字段，确保当前播放页与收藏接口语义一致。

## 规则

- `track` 表只表示已归因后的稳定事实，不能承载待归因临时态。
- `track_favorite_event` 只表示未完成归因的收藏意图，不能被客户端直接当作稳定事实使用。
- API 与 WS 必须统一通过 logic 层的 favorite projection 输出收藏态，禁止在 handler、scrobbler 或客户端重复拼接一套收藏判断。
- 对兼容客户端，`apple_music` / `lastfm` 继续保留，但其语义升级为“有效布尔态”：
  - `favorited` => `true`
  - `favorite_pending` => `true`
  - `unfavorite_pending` => `false`
  - `not_favorited` => `false`

## 聚合优先级

- source 级状态：
  - 若 pending 意图与稳定事实相反，则以 pending 枚举表示最新意图。
  - 否则回落到稳定事实。
- 曲目级 `favorite_state` 聚合优先级：
  - `unfavorite_pending`
  - `favorite_pending`
  - `favorited`
  - `not_favorited`

## 验证

- `go test ./internal/logic/track ./internal/scrobbler -run 'Test(BuildFavoriteProjection|ProbeAndSyncTrackFavorite|ProcessPlayingTrack)'`
- `go test ./internal/model -run TestGetPendingTrackFavoriteSnapshotReturnsLatestBySource`
- `go test ./api ./core/websocket -run TestDoesNotExist`

## 影响面

- 后端接口文档：`api/api.md`
- 长期记忆：`GEMINI.md`
- 当前播放实时协议：`core/websocket/websocket.go`
- 收藏读模型编排：`internal/logic/track/favorite_projection.go`
