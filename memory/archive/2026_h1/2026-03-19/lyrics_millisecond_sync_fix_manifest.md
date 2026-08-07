# 歌词毫秒级同步链路修复清单

## 背景

歌词高亮与滚动失效的根因不是单点 bug，而是整条链路同时存在三个问题：

1. WebSocket `now_playing.data.position` 只有整秒，播放器原始小数秒被截断。
2. Web 与 Bridge 分别维护各自的 LRC 解析逻辑，时间标签规则长期漂移。
3. `track_lyrics.synced` / `/api/track-lyrics.has_lrc` 过去按是否包含 `[]` 粗略判断，容易把段落标签或元数据标签误判成同步歌词。

## 本次修复

### 1. WebSocket 对外协议补齐毫秒位置

- `core/websocket.WsTrackData` 新增 `position_ms` 字段，单位为毫秒。
- `position` 保留为兼容字段，仍表示整秒。
- `internal/scrobbler.BasePlayerChecker` 广播当前播放时，直接使用播放器原始 `float64 position` 生成 `position_ms`。
- `PlayerInfoHandler` 继续保留在服务端内存缓存中供收藏补元数据使用，但通过 `json:"-"` 从对外 WS payload 隐藏，避免前端依赖内部结构。

### 2. 统一 LRC 判定语义

- `core/lyrics` 新增统一的时间标签解析与 `IsSyncedLRC` 判定。
- 仅当文本中存在至少一个合法时间标签 `[mm:ss] / [mm:ss.x] / [mm:ss.xx] / [mm:ss.xxx]` 时，才认为是同步歌词。
- 小数位统一按右补零到 3 位后解释为毫秒。
- `[Verse]` 之类段落标签允许前端展示，但 `[ar:...]`、`[ti:...]` 这类元数据标签不再触发同步歌词判定。

### 3. 前端与 Bridge 统一切到毫秒锚点时钟

- Dashboard 歌词浮窗与 `lyrics_live.html` 不再使用 `currentPosition += 1` 的整秒累加模型。
- 页面收到 WS 后会记录最近一次 `position_ms` 与本地单调时钟，播放中通过“锚点毫秒 + 本地经过时间”推导当前位置。
- `static/lrc-utils.js` 统一前端 LRC 解析、同步判定与激活行定位逻辑，避免 `dashboard.html` 与 `lyrics_live.html` 再次分叉。
- Bridge `NowPlaying` 模型新增 `positionMs`，`PlayerViewModel` 改为毫秒锚点进度推进，`PlayerView` / `NowPlayingView` / `PadNowPlayingView` 统一接入 `positionMs`，并保留对旧 `position` 的回退同步。
- Bridge 歌词解析抽到 `SoniclensCore/Models/PlayerModels.swift` 中的共享 `LRCParser`，由 `PlayerViewModel` 与 `TrackDetailViewModel` 共用。

## 验证

- `go test ./core/lyrics ./internal/logic/insight`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -sdk macosx -derivedDataPath /tmp/SoniclensBridgeDerivedData build`

## 后续约束

- 前端不得再直接消费 `PlayerInfoHandler.Position`。
- 新的歌词同步功能优先使用 `position_ms`，只有旧服务端兼容场景才允许回退到 `position * 1000`。
- 后续新增歌词页面或 Bridge 视图时，必须复用共享 LRC 解析器，不允许再复制第三套时间标签解析逻辑。
