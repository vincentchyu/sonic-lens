# Bridge 正在播放收藏投影接入闭环特性清单

## 日期

- 2026-03-28

## 背景

- 后端已经为 `/api/favorite` 与 `WS now_playing` 补齐 `apple_music_state`、`lastfm_state`、`favorite_state`。
- Bridge 三端此前仍只消费 `apple_music` / `lastfm` 两个布尔位，并在 `AppStore` 中用 `favoriteKeys` 做本地覆盖，导致 `favorite_pending` / `unfavorite_pending` 在客户端被抹平。
- 用户在正在播放页点击收藏后，即使后端已经记录待归因收藏事件，三端界面也可能仍显示“未收藏”，形成误导。

## 本次收敛

- 在 `soniclens-bridge/SoniclensCore/Models/FavoriteModels.swift` 新增统一收藏枚举 `TrackFavoriteState` 与共享投影 `TrackFavoriteProjection`。
- `FavoriteResponse` 与 `NowPlaying` 同时接入 `apple_music_state`、`lastfm_state`、`favorite_state`，并统一回落到 projection 计算逻辑。
- `AppStore` 新增 `favoriteProjections`，不再只依赖布尔集合；`isFavorite`、`toggleFavorite` 与当前播放态 patch 全部改走 projection。
- `NowPlayingService` 保留 WS 新收藏态字段，避免在标准化封面 URL 时丢失收藏投影。
- macOS / iPad / iPhone 三端 `NowPlaying` 统一改为消费 projection：
  - `favorite_pending` 显示“收藏已记录，等待归因”
  - `unfavorite_pending` 显示“取消收藏处理中”
  - `partial/full` 继续区分单平台收藏与双平台收藏
  - pending 态下禁用重复收藏动作，减少误触

## 规则

- Bridge 端的正在播放收藏态必须以 `favorite_state` 为第一事实来源，source 级展示由 `apple_music_state` / `lastfm_state` 补充，不得再用两个布尔位自行猜测 pending。
- `favoriteKeys` 只允许作为“有效收藏态”的兼容覆盖层，不能继续承载完整收藏语义。
- 三端 `NowPlaying` 的收藏按钮、提示文案和禁用逻辑必须共享同一套 `NowPlayingFavoriteStatus` 推导，禁止各端各自拼接收藏规则。

## 验证

- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -configuration Debug -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePhone -configuration Debug -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePad -configuration Debug -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`

## 影响面

- `soniclens-bridge/SoniclensCore/Models/FavoriteModels.swift`
- `soniclens-bridge/SoniclensCore/Models/NowPlaying.swift`
- `soniclens-bridge/SoniclensCore/Networking/NowPlayingService.swift`
- `soniclens-bridge/SoniclensCore/Store/AppStore.swift`
- `soniclens-bridge/SoniclensBridge/Views/NowPlayingView.swift`
- `soniclens-bridge/SoniclensBridge/Views/PadNowPlayingView.swift`
- `soniclens-bridge/SoniclensBridge/Views/PhoneNowPlayingView.swift`
