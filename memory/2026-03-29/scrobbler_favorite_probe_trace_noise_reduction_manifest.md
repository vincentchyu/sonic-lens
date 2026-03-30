# Scrobbler 收藏探测与 Trace 降噪闭环

- **日期**: 2026-03-29
- **范围**: `internal/scrobbler/`、`internal/logic/track/`

## 背景

播放器轮询链路接入 OTel 后，Jaeger 已经能按“单首歌一个根 span”观察播放过程，但稳态播放期间仍然存在大量重复噪音：

1. `Apple Music` 的 `player.is_favorite` 会随每轮 `now_playing` 轮询重复触发。
2. `ProbeAndSyncTrackFavorite` 在同歌稳态且收藏态未变化时，仍会反复重建 favorite projection，导致 `select track`、`select track_favorite_event` 持续出现。
3. `Last.fm` 虽然已有 Redis 缓存，但 scrobbler 仍会每轮调用 `lastfmIsFavorite`，导致大量 Redis `GET cache:isFavorite:lastfm:*` span。
4. Trace 结构虽然正确，但单首歌链路仍然偏胖，不利于排查真正的业务转折。

## 本次收敛

### 1. 单首歌根 span 收口为长生命周期会话

1. `BasePlayerChecker` 改为持有当前曲目的活动 `context/span`。
2. 换歌时：
   - 结束上一首歌的根 span
   - 为新歌创建新的 `TrackPlayback` 根 span
3. 停止播放或 goroutine 退出时统一收尾，避免 span 泄漏或上下文串链。
4. 根 span 追加曲目属性：
   - `player.source`
   - `track.title`
   - `track.artist`
   - `track.album`
   - `track.album_artist`
   - `track.track_number`
   - `track.disc_number`
   - `track.duration_sec`
   - `track.metadata_confidence`

### 2. 高频子 span 按业务变化触发

1. 新增三个阶段子 span：
   - `player.resolve_now_playing`
   - `player.sync_favorite_state`
   - `player.handle_playback_threshold`
2. `player.resolve_now_playing` 只在换歌时出现，不再每轮都打。
3. `player.sync_favorite_state` 只在以下场景出现：
   - 换歌
   - 收藏投影真的发生变化
4. 关键状态转折收口为根 span event：
   - `track_started`
   - `favorite_state_changed`
   - `scrobble_threshold_reached`
   - `player_stopped`

### 3. Apple Music 控制器收藏态降频

1. `BasePlayerChecker` 为当前曲目缓存 `controllerFavorite`。
2. 仅在以下场景重新调用 `controller.IsFavorite(ctx)`：
   - 换歌
   - 曲目标识变化
   - 上次探测超过 `15s`
3. 稳态播放期间不再每个轮询周期都重复打 `player.is_favorite`。

### 4. Favorite projection 会话缓存

1. `TrackServiceImpl` 新增进程内 favorite projection 缓存。
2. `ProbeAndSyncTrackFavorite` 会先比较当前 probe 状态：
   - Apple Music 探测结果
   - Last.fm 探测结果
   - 当前曲目标识
3. 如果 probe 未变化，直接复用上次 projection，不再重复查询：
   - `track`
   - `track_favorite_event`
4. 本地收藏写入 `SetTrackFavorite` 成功后会递增版本号，主动使 projection 缓存失效，确保 `/api/favorite` 不会被旧缓存污染。

### 5. Last.fm 收藏探测热缓存与退避

1. `TrackServiceImpl` 在调用 `lastfmIsFavorite` 之前新增进程内热缓存。
2. 缓存键为标准化后的 `artist|track`。
3. 正结果探测间隔固定为 `15s`。
4. 负结果采用退避：
   - 第一次 `15s`
   - 第二次 `30s`
   - 第三次 `60s`
   - 后续上限 `120s`
5. 在热缓存有效期间，scrobbler 不再触发 `lastfmIsFavorite`，因此也不会再打 Redis `GET cache:isFavorite:lastfm:*` span。
6. 收藏写入版本号变化后，Last.fm 热缓存也会同步失效，保证本地收藏操作可尽快反映到当前播放态。

## 约束

1. 单首歌 trace 的事实标准是“一个长生命周期根 span + 少量按变化触发的阶段 span/event”，不要回退到“每次轮询一组 span”。
2. `player.is_favorite` 这类播放器附加状态探测不能与播放主轮询同频，默认应走 TTL 或显式失效。
3. 收藏投影读取优先复用逻辑层缓存，只有在探测结果变化或本地收藏写入后才重建 projection。
4. Last.fm 收藏态探测不能只依赖 Redis 缓存；在轮询链路里还需要进程内热缓存，避免每轮重复打 Redis `GET`。
5. 缓存失效必须优先由 `SetTrackFavorite` 驱动，避免 `/api/favorite` 写入后当前播放态滞留旧值。

## 验证

1. `go test ./internal/logic/track/... ./internal/scrobbler/...` 通过。
2. 新增单测覆盖：
   - 单首歌根 span 生命周期
   - 阶段子 span / event 生成
   - `controller.IsFavorite` 15 秒 TTL
   - favorite projection 会话缓存命中与版本失效
   - Last.fm 热缓存命中与负结果退避

## 结果

在不牺牲收藏同步完整性与当前播放 WS 语义的前提下：

1. Jaeger 中单首歌 trace 更接近“业务阶段轨迹”而不是“轮询流水账”。
2. `player.is_favorite`、`select track`、`select track_favorite_event`、`GET cache:isFavorite:lastfm:*` 的重复次数显著下降。
3. 当前播放收藏态仍可在换歌、阈值、副作用写入与本地收藏操作后正确收敛。
