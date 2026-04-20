# Bridge macOS Mini 播放条窗口级玻璃宿主闭环

## 特性摘要

Bridge macOS 端底部 mini 播放条这次不再把 `NSVisualEffectView` 当成 SwiftUI `background` 或 overlay 修饰链里的补丁层使用，而是正式改成窗口内容区域内长期驻留的 AppKit 玻璃容器。播放器玻璃底由独立 `PlaybackBarWindowOverlayController` 托管，内容层只保留 SwiftUI `PlaybackBarContentView`，从根上修复前后台切换时“先实色、再延迟变玻璃”的退化问题。

## 变更范围

- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarView.swift` 将 mini 播放条拆分为共享的 `PlaybackBarContentModel`、`PlaybackBarContentView` 与 `PlaybackBarSurface`，移除 macOS 上把 `NSVisualEffectView` 作为 SwiftUI 背景补丁层的路径。
- `soniclens-bridge/SoniclensBridge/Views/PlaybackBarWindowOverlayController.swift` 新增 macOS 专用窗口级宿主：`NSVisualEffectView` 作为稳定玻璃背景，`NSHostingView<PlaybackBarContentView>` 承载内容层，性能模式下回退到稳定实色表面。
- `soniclens-bridge/SoniclensBridge/Views/AppLayoutView.swift` 不再直接把 mini 播放条渲染在 SwiftUI 页面树底部，而是通过 `PlaybackBarWindowOverlayBridge` 把播放态、展示状态和点击回调交给窗口级宿主；`NowPlaying` 展开时同步隐藏 mini 播放条，避免窗口级 overlay 压在全屏播放页之上。
- `soniclens-bridge/project.yml` 明确将 `PlaybackBarWindowOverlayController.swift` 只交给 macOS target，Pad/Phone 继续沿用现有 SwiftUI 安全区挂载方案，并通过 `xcodegen generate` 重生工程。

## 关键约束

- macOS 端的真实玻璃层必须由 AppKit 宿主长期持有，禁止再回到“SwiftUI `background`/`clipShape`/按钮样式里嵌 `NSVisualEffectView`”的实现；这条路径在前后台切换时会丢失稳定 backdrop。
- `PlaybackBarContentView` 只负责内容与交互，播放进度与展开逻辑统一经 `PlaybackBarContentModel` 收口；不要再把播放态推进、背景材质和窗口事件监听揉在一个视图里。
- `SoniclensBridge.xcodeproj` 由 `soniclens-bridge/project.yml` 生成。涉及 mini 播放条 target 接入时，必须先改 `project.yml` 再执行 `xcodegen generate`，不能手改工程文件。
- iPad/iPhone 不应复用 macOS 的 AppKit 宿主。三端共享的只有内容模型与内容视图，窗口语义必须留在 macOS 容器层。

## 验证

- 手动确认 macOS 端滚动内容能从 mini 播放条下方经过，玻璃层持续显示 `withinWindow` 模糊而不是延迟退回实色。
- `NowPlaying` 展开/关闭后 mini 播放条不会遮住全屏播放页。
- 构建验证尝试使用 `xcodebuild -scheme SoniclensBridgeMac build`，当前在沙箱环境下仍会被 SwiftPM / `sandbox-exec` 权限限制阻断，需在本机正常 Xcode 环境补做最终编译确认。

## 说明

- 这次修复的重点不是继续给 `NSVisualEffectView` 补通知监听，而是把玻璃层放回 Apple 推荐的宿主位置。后续若再出现视觉退化，应优先检查窗口级 overlay 宿主、`project.yml` target 配置和 `NSVisualEffectView` 的承载位置，而不是新增更多前后台刷新补丁。
