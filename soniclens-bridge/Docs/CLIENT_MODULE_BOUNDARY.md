# SonicLens Bridge 客户端模块边界清单（Mac / iPad / iPhone）

## 1. App 入口与路由链路

| 入口 App | Root 入口 | 主容器 |
| --- | --- | --- |
| `SoniclensBridgeApp.swift`（macOS） | `RootView` -> `MacRootView` | `AppLayoutView` |
| `SoniclensBridgePadApp.swift`（iOS / iPad） | `PadRootView` | `PadAppLayoutView` |
| `SoniclensBridgePhoneApp.swift`（iOS / iPhone） | `PhoneRootView` | `PhoneAppLayoutView` |

说明：
- 三个 `@main App` 都只负责创建 `AppStore` 并注入 `environmentObject`。
- iPad / iPhone 不经过 `RootView`，分别走各自 Root 容器。

## 2. 共享模块（改动会跨端扩散）

### 2.1 共享状态与业务层（SoniclensCore）

- `SoniclensCore/Store/AppStore.swift`
- `SoniclensCore/Store/LibraryIndexStore.swift`
- `SoniclensCore/Store/LibrarySyncService.swift`
- `SoniclensCore/Networking/*`（`APIClient`、`NowPlayingService`、`WebSocketClient` 等）
- `SoniclensCore/Models/*`
- `SoniclensCore/Discovery/BonjourDiscovery.swift`

影响面：
- 连接态、播放态、收藏、WebSocket 增量同步等核心能力均由此层提供，默认影响 Mac / iPad / iPhone 三端。
- `Insight` 结构、`analysis_by_section` 兼容解码、标签解析模型同样属于共享层语义，改动默认影响三端音眸展示。

### 2.2 共享 ViewModel 层

- `ViewModels/HomeViewModel.swift`
- `ViewModels/LibraryViewModel.swift`
- `ViewModels/PlayerViewModel.swift`
- `ViewModels/AlbumDetailViewModel.swift`
- `ViewModels/TrackDetailViewModel.swift`

影响面：
- 数据加载、分页、筛选、详情拉取策略变更会影响所有引用这些 ViewModel 的页面。

### 2.3 共享 UI 组件与页面

- 连接与通用视觉：`ConnectionView.swift`、`Theme.swift`、`GlassStyles.swift`、`PlatformUI.swift`
- 通用播放条/组件：`PlaybackBarView.swift`、`PlayerView.swift`（如被复用）
- 资料库与详情：`LibraryView.swift`、`AlbumDetailView.swift`、`TrackDetailView.swift`
- 音眸共享渲染：`InsightDetailView.swift`（包含共享 `InsightPrimaryContentView` / `InsightRichContentView` 富渲染树）
- 共用业务页组件：`AlbumGridView`、`TrackListView`、`SonicLensInsightsView`、`UnreportedListView`（定义在 `Views` 共用文件中）

影响面：
- 样式 token、通用控件、列表行为修改通常会波及多端（至少两个端）。
- 音眸区块顺序、标签文本样式、`explain` 视觉优先级等若在共享渲染树中调整，需按 Mac / iPad / iPhone 联动回归。

## 3. 平台私有模块（改动主要局部生效）

### 3.1 Mac 私有

- 入口链：`SoniclensBridgeApp.swift`、`RootView.swift`、`MacRootView.swift`
- 容器与导航：`AppLayoutView.swift`（`NavigationSplitView` + macOS sidebar + 窗口 chrome 控制）
- 沉浸播放页：`NowPlayingView.swift`（包含 `NSWindow` 全屏状态观察）
- 仅 macOS 能力：`SnapshotExport.swift`

主要影响：
- 仅影响 `SoniclensBridgeMac` 目标（除非改动到共享组件/Store）。

### 3.2 iPad 私有

- 入口链：`SoniclensBridgePadApp.swift`、`PadRootView.swift`
- 容器：`PadAppLayoutView.swift`（iPad split 导航与 fullScreenCover 播放中）
- 页面：`PadHomeView.swift`、`PadNowPlayingView.swift`

主要影响：
- 仅影响 iPad 产品线（除非改动到共享层）。

### 3.3 iPhone 私有

- 入口链：`SoniclensBridgePhoneApp.swift`、`PhoneRootView.swift`
- 容器：`PhoneAppLayoutView.swift`（Tab 容器 + 紧凑播放条）
- 页面：`PhoneHomeView.swift`、`PhoneNowPlayingView.swift`

主要影响：
- 仅影响 iPhone 产品线（除非改动到共享层）。

## 4. 迭代时的影响面判定规则

1. 改 `SoniclensCore/*` 或 `ViewModels/*`：默认按三端联动评估，必须做跨端回归。
2. 改 `Theme.swift` / `GlassStyles.swift` / 通用组件：至少按 Mac + iPad + iPhone 做视觉/交互冒烟。
3. 改 `AppLayoutView.swift` / `NowPlayingView.swift`：优先判定为 Mac 私有变更。
4. 改 `Pad*` 文件：优先判定为 iPad 私有变更。
5. 改 `Phone*` 文件：优先判定为 iPhone 私有变更。
6. 涉及 `#if os(macOS)` 或 `#if os(iOS)` 条件块时，需检查对应 target 是否都可编译。
7. 改 `InsightDetailView.swift` 或 `LibraryModels.swift` 中的音眸解析/渲染逻辑：必须按三端共享变更处理，默认不允许在端内再复制一套字符串解析。

## 5. 本清单维护要求

- 新增 Bridge 页面时，优先判断是“共享组件”还是“端私有容器”。
- 若新增 `Pad*` / `Phone*` / `Mac*` 或跨端共享模块，需同步更新本清单与 `GEMINI.md` 的 Bridge 约束条目。
