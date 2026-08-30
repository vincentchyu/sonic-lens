# 🧭 macOS 全局多级双栈历史导航与原生快捷键体系特性清单

## 1. 架构目标与背景

为了在 macOS 客户端（`SoniclensBridgeMac`）提供媲美 Safari / Finder / Xcode 的沉浸式原生操作体验，本次重构落地了：

1. **全站统一多级双栈导航引擎 (`NavigationCoordinator`)**：
    - 彻底废弃页面局部状态（如 `@State selectedAlbum`），将侧边栏 Tab 切换与页面内卡片点击统一为全局时间线快照（
      `NavigationSnapshot`）。
    - 彻底避免侵入式埋点，采用 SwiftUI 原生“声明式值路由 (`NavigationLink(value:)`) + 根栈自动捕获与 Binding 代理拦截”。
2. **原生快捷键矩阵**：
    - **`Command + [`**：全局后退（跨页面/跨 Tab 原路回退）；
    - **`Command + ]`**：全局前进（原路重现被回退的历史视图）；
    - **`Command + J`**：正在播放（Now Playing）全屏大画布开关 / 关闭（支持 `Esc` 快速退出）；
    - **`Ctrl + 1...6`**：侧边栏 6 大核心模块直达（主页、专辑、曲目、未上报、音眸、未来功能）。
3. **交互精细化增强**：
    - 全屏正在播放页（歌词、曲目名、音眸乐评）与详情页头部全面支持 **触控板三指拖移划选与复制**；
    - 曲目列表行保留灵敏点击并支持 **macOS 原生右键上下文菜单（Context Menu）一键拷贝曲目与歌手**。

---

## 2. 核心数据结构与工作原理

### 2.1 双栈快照模型 (Two-Stack Snapshot Model)

```swift
struct NavigationSnapshot: Equatable {
    var tab: SidebarDestination
    var path: [AppRoute]
}

@MainActor
final class NavigationCoordinator: ObservableObject {
    @Published var selectedTab: SidebarDestination = .home
    @Published var path: [AppRoute] = []
    
    private var backHistory: [NavigationSnapshot] = []
    private var forwardHistory: [NavigationSnapshot] = []
}
```

### 2.2 0 侵入声明式监听拦截

- **子视图**：仅声明 `NavigationLink(value: AppRoute.trackDetail(track: track))`，与导航器完全解耦；
- **根容器**：`NavigationStack(path: $navigationCoordinator.path)` 自动监听视图树抛出的路由值，并在 `didSet` 中自动计算快照与更新栈；
- **侧边栏**：通过 `sidebarSelectionBinding` 代理拦截 `List(selection:)`，自动记录 Tab 切换节点。

---

## 3. 涉及的核心代码清单

- `soniclens-bridge/SoniclensBridge/Navigation/NavigationCoordinator.swift`：双栈快照协调器总控；
- `soniclens-bridge/SoniclensBridge/Views/AppLayoutView.swift`：根级 NavigationStack 绑定、路由分发与全局快捷键；
- `soniclens-bridge/SoniclensBridge/Views/NowPlayingView.swift`：正在播放 `Cmd + J` / `Esc` 关闭与文本选择；
- `soniclens-bridge/SoniclensBridge/Views/AlbumDetailView.swift`：曲目行右键菜单与声明式值路由；
- `soniclens-bridge/SoniclensBridge/Views/HomeView.swift`：首页热门卡片与最近播放历史路由下钻；
- `soniclens-bridge/SoniclensBridge/Views/SonicLensInsightsView.swift`：音眸卡片声明式路由；
- `soniclens-bridge/Docs/ARCHITECTURE.md` & `IA.md`：设计与架构规范同步；
- `GEMINI.md`：核心契约落地。

---

## 4. 验证报告

- **macOS Client Target (`SoniclensBridgeMac`)**：`xcodebuild` 构建 100% 成功（Exit code: 0）；
- **全站后端单元测试**：`go test -count=1 ./...` 100% PASS。
