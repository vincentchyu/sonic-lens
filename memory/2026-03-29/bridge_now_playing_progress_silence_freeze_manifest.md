# Bridge 正在播放进度静默冻结修复清单

## 日期

- 2026-03-29

## 背景

- Bridge 的正在播放页和底部播放条都依赖本地定时器，把最近一次 `WS now_playing` 的 `position_ms` 持续外推成进度。
- 当播放器暂停后，后端如果没有继续推送 `now_playing` 或 `stop`，客户端仍会拿着最后一次锚点继续自增，造成“旧歌曲还在播、进度一直前进”的错觉。
- 这个问题在 macOS/iPad/iPhone 的沉浸式正在播放页，以及全局播放条上都会出现，因为它们共用同一套进度推进思路。

## 本次修复

- `PlayerViewModel` 新增分段静默检测：最近一次同步超过 5 秒后先进入“已暂停/已停止更新”，超过 10 秒后切到“无活动播放”。
- `PlaybackBarProgressModel` 同步增加同样的分段静默逻辑，避免全局播放条在暂停后继续自增，并在 10 秒后降级为无活动卡片。
- 三端全屏正在播放页的提示 tag 改为内联在“正在播放”标题后，不再单独占用一行，避免歌词与主内容区发生下沉。
- `inactive` 态不再展示重复 tag 或“等待新的 now_playing”这类补充文案，只保留标题“无活动播放”。
- 三端全屏正在播放页的统一状态提示只保留“已暂停/已停止更新”这一类真正有区分度的 tag。
- 两套进度模型都保留对后续 `now_playing` 更新的自动恢复能力，新的 WS 包一到就会从最新位置继续推进。
- 进度冻结逻辑仍然只依赖现有 `now_playing` 协议，不要求后端额外新增暂停字段，兼容当前服务端实现。

## 验证

- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -configuration Debug -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePhone -configuration Debug -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePad -configuration Debug -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`

## 影响面

- `soniclens-bridge/SoniclensBridge/ViewModels/PlayerViewModel.swift`
- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarView.swift`
