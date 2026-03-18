# Bridge 双端产品线特性清单

## 日期
2026-03-18

## 特性摘要
- `soniclens-bridge` 从单一 macOS 客户端扩展为 `macOS + iPadOS` 双产品线。
- 工程新增独立 iPad target 与 scheme，允许 iPad 端使用原生 iPadOS 导航、全屏播放与触控交互，不再复刻 macOS 窗口语义。
- `SoniclensCore` 作为共享业务层继续承载连接、Bonjour 发现、WebSocket、收藏、资料库本地索引与增量同步能力。
- `RootView` 改为按平台路由到 `MacRootView` / `PadRootView`，共享数据流但允许平台 UI 容器分叉。
- iPad 端新增 `PadAppLayoutView`、`PadHomeView`、`PadNowPlayingView`，形成独立的分栏导航与沉浸式播放画布。
- macOS 与 iPadOS 的播放页、布局壳和平台生命周期适配允许分别演进，不再要求一套桌面壳跨平台硬复用。

## 工程改动
- `soniclens-bridge/project.yml` 新增：
  - `SoniclensBridgeMac`
  - `SoniclensBridgePad`
  - 对应独立 scheme、bundle id、Info.plist 与资源排除规则。
- `SoniclensBridge/SoniclensBridgePadApp.swift` 新增 iPad 应用入口。
- `SoniclensBridge/Views/RootView.swift` 改为平台分流入口。
- `SoniclensBridge/Views/MacRootView.swift`、`SoniclensBridge/Views/PadRootView.swift` 分别承载 macOS / iPadOS 根容器。

## 共享层约束
- `SoniclensCore` 继续作为 Bridge 双端共享层，集中维护：
  - `AppStore`
  - API / WebSocket / Bonjour
  - 资料库本地 SQLite 索引与增量同步
  - 播放状态、收藏状态与相关 ViewModel
- 资料库页仍必须遵守“本地索引优先”约束，不能因新增 iPad 客户端回退为远端分页加内存筛选。

## 平台分叉约束
- macOS 保留桌面窗口生命周期与全屏状态监听逻辑。
- iPadOS 不使用 `NSWindow` / `NSApplication` / `NSViewRepresentable` 语义，改用原生 iPadOS 布局与 safe area 处理。
- 主题、颜色 token 和平台适配层应保持可共享；窗口控制、截图导出等桌面能力必须收敛在 macOS 编译边界内。
- 沉浸式播放页允许按平台分别设计，但应共享播放数据流、歌词跟随与音眸洞察数据源。

## 已验证
- `xcodegen generate`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -destination 'generic/platform=macOS' build`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePad -destination 'platform=iOS Simulator,name=iPad Air 5,OS=26.1' build`

## 后续建议
- 后续 Bridge 新功能先判断是共享业务能力还是平台壳能力，避免把 `AppKit` / `UIKit` 细节重新污染到共享层。
- 若 iPadOS 与 macOS 的播放页继续分化，优先共享子组件与状态模型，不要回退到单个 View 中堆条件编译。
