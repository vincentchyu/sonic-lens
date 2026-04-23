# Bridge WS 优先播放存在性与 mini 播放条修复清单

## 日期

- 2026-04-23

## 背景

- 后端 `scrobbler` 在活跃播放期间会稳定通过 `WS now_playing` 推送当前歌曲、进度和时长，客户端也已经为每条快照记录 `receivedAt`。
- 之前 Bridge 共享层把“是否仍算活动播放”与“是否还存在播放对象”混成了一件事：`5s` 后进入 `pausedStale`，`10s` 后进入 `inactive`，而 mini 播放条与三端入口又把 `inactive` 直接渲染成“无活动播放”。
- 这会放大偶发链路抖动。只要客户端某一小段时间没把新的快照喂到共享模型，就会把仍在播放的歌曲误判成“无活动播放”，与“WebSocket 已经正常推到播放事件”这一事实相冲突。
- macOS 端还额外存在一层窗口级 overlay render-state 去重缺陷：同一首歌连续播放时，render token 没有纳入 `position`、`positionMs`、`receivedAt`，导致新的 WS 进度更新被错误短路。

## 本次修复

- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarView.swift`
  - `PlaybackBarContentModel` 转发内层 `PlaybackBarProgressModel.objectWillChange`，保证 mini 播放条会随着本地计时器持续刷新时间与进度。
  - 播放条内容显隐从“`playbackState.active` 才展示歌曲”改为“`nowPlaying != nil` 就展示歌曲”，`pausedStale` 仅展示 `已暂停/已停止更新` banner，不再退回空卡片。
  - 点击展开规则同步改为只校验 `nowPlaying != nil`，不再因为客户端静默分段状态阻断进入正在播放页。
- `soniclens-bridge/SoniclensCore/Store/AppStore.swift`
  - `PlaybackStore.hasActiveNowPlaying` 改为以 `nowPlaying != nil` 为准，收口三端播放入口的共享判定。
- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarWindowOverlayController.swift`
  - macOS mini 播放条 overlay token 增加 `position`、`positionMs`、`receivedAt`，避免同曲目连续推送时被错误去重。
  - `NowPlaying` 窗口 overlay token 同步纳入这些字段，避免窗口级共享渲染层再次吞掉时间推进类更新。

## 新规则

- `WS now_playing` 是 Bridge 前台“存在播放对象”的最高优先级事实源；只要客户端当前仍持有这条播放对象，就必须保留歌曲卡片与进入播放页的能力。
- 客户端的 `pausedStale` / `inactive` 只负责本地进度冻结、状态 banner 和降噪，不得反向把一条仍存在的播放对象渲染成“无活动播放”。
- 真正清空播放对象只能来自明确的 `WS stop`、断开连接后的显式 reset，或用户主动断连，而不是客户端自己对活跃度的二次猜测。

## 验证

- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -configuration Debug -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePad -configuration Debug -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePhone -configuration Debug -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`

## 影响面

- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarView.swift`
- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarWindowOverlayController.swift`
- `soniclens-bridge/SoniclensCore/Store/AppStore.swift`
