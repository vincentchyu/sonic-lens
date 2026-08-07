# Bridge 暂停静默播放态守卫修复清单

## 背景

当服务端在音乐暂停后没有继续广播 `stop` 或新的 `now_playing` 事件时，Bridge 端会继续持有旧的播放快照。此前 mini 播放条和“正在播放”页面在重新进入时会把这份旧快照当成“刚更新”的数据重新启动本地计时，导致暂停后仍显示活动进度，且还能点击进入播放中页。

## 本次修复

- `soniclens-bridge/SoniclensCore/Models/NowPlaying.swift`
  - 为 `NowPlaying` 增加 `receivedAt`，把最近一次 WS 快照到达时间收口为客户端播放态新鲜度的事实来源。
  - 增加 `playbackActivityState(at:)`，统一按 `5s pausedStale / 10s inactive` 判定播放活跃度。
- `soniclens-bridge/SoniclensCore/Networking/NowPlayingService.swift`
  - 每次收到新的 `now_playing` 时写入新的 `receivedAt`，避免后续页面重新打开时误把旧快照当成新消息。
- `soniclens-bridge/SoniclensBridge/ViewModels/PlayerViewModel.swift`
  - `startProgress` / `syncProgress` 改为显式接收 `receivedAt`，重新进入页面时会立即按快照真实年龄判断是活跃、暂停静默还是失活。
- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarView.swift`
  - mini 播放条仅在 `PlaybackActivityState.active` 时展示活动播放内容并允许点击展开。
  - 对于静默暂停态，不再继续表现成活动播放。
- `soniclens-bridge/SoniclensBridge/Views/AppLayoutView.swift`
  - macOS 窗口级 mini 播放条点击展开前会先校验共享播放态是否仍为活跃。
- `soniclens-bridge/SoniclensBridge/Views/PadAppLayoutView.swift`
  - iPad 端只允许从活跃播放态进入 `NowPlaying` 全屏页。
- `soniclens-bridge/SoniclensBridge/Views/PhoneAppLayoutView.swift`
  - iPhone 端“打开正在播放”入口同样只允许活跃播放态进入。

## 结果

- 暂停后即便后端没有继续发送 WS 事件，Bridge 重新打开 `NowPlaying` 时也不会重新空跑进度。
- mini 播放条和“打开正在播放”入口不会再把静默超时的旧快照误当成正在播放中的歌曲。
