# SonicLens Bridge 客户端模块边界清单（Mac / iPad / iPhone）

## 1. App 入口与路由链路

| 入口 App | Root 入口 | 主容器 |
| --- | --- | --- |
| `SoniclensBridgeApp.swift`（macOS） | `RootView` -> `MacRootView` | `AppLayoutView` |
| `SoniclensBridgePadApp.swift`（iOS / iPad） | `PadRootView` | `PadAppLayoutView` |
| `SoniclensBridgePhoneApp.swift`（iOS / iPhone） | `PhoneRootView` | `PhoneAppLayoutView` |

说明：
- 三个 `@main App` 都负责创建 `AppStore` 并注入 `environmentObject`，同时把 `PlaybackStore`、`FavoriteStore` 作为 typed environment 下发。
- iPad / iPhone 不经过 `RootView`，分别走各自 Root 容器。

## 2. 共享模块（改动会跨端扩散）

### 2.1 共享状态与业务层（SoniclensCore）

- `SoniclensCore/Store/AppStore.swift`
- `SoniclensCore/Store/AppStore.swift` 持有的 `PlaybackStore` / `FavoriteStore`
- `SoniclensCore/Store/LibraryIndexStore.swift`
- `SoniclensCore/Store/LibrarySyncService.swift`
- `SoniclensCore/Networking/*`（`APIClient`、`NowPlayingService`、`WebSocketClient` 等）
- `SoniclensCore/Models/*`
- `SoniclensCore/Discovery/BonjourDiscovery.swift`

影响面：
- 连接态、播放态、收藏、WebSocket 增量同步等核心能力均由此层提供，默认影响 Mac / iPad / iPhone 三端。
- `AppStore` 现在主要影响连接与容器级逻辑；`PlaybackStore` / `FavoriteStore` 的结构调整会直接改变三端高频 UI 的重绘范围。
- `Insight` 结构、`analysis_by_section` 兼容解码、标签解析模型同样属于共享层语义，改动默认影响三端音眸展示。

### 2.2 共享 ViewModel 层

- `ViewModels/HomeViewModel.swift`
- `ViewModels/LibraryViewModel.swift`
- `ViewModels/PlayerViewModel.swift`
- `ViewModels/AlbumDetailViewModel.swift`
- `ViewModels/TrackDetailViewModel.swift`

影响面：
- 数据加载、分页、筛选、详情拉取策略变更会影响所有引用这些 ViewModel 的页面。
- `LibraryViewModel` 当前承担资料库 single-flight、页优先加载、统计后补和请求 token 丢弃过期结果；改动这里默认要做三端资料库回归。
- `PlayerViewModel` 当前承担歌词 / insight 并行拉取与过期任务丢弃；改动这里默认要做播放页联动回归。

### 2.3 共享 UI 组件与页面

- 连接与通用视觉：`ConnectionView.swift`、`Theme.swift`、`GlassStyles.swift`、`PlatformUI.swift`
- 通用播放条/组件：`PlaybackBarView.swift`、`PlayerView.swift`（如被复用）
- 资料库与详情：`LibraryView.swift`、`AlbumDetailView.swift`、`TrackDetailView.swift`
- 音眸共享渲染：`InsightDetailView.swift`（包含共享 `InsightPrimaryContentView` / `InsightRichContentView` 富渲染树）
- 分享基础设施：`ShareKit/Domain/*`、`ShareKit/Builder/*`、`ShareKit/Render/*`、`ShareKit/Action/*`、`ShareKit/Analytics/*`
- 共用业务页组件：`AlbumGridView`、`TrackListView`、`SonicLensInsightsView`、`UnreportedListView`（定义在 `Views` 共用文件中）

影响面：
- 样式 token、通用控件、列表行为修改通常会波及多端（至少两个端）。
- `ConnectionView.swift` 的阶段反馈、取消按钮、Bonjour 候选呈现和断开入口文案属于共享连接体验，一处改动会影响三端连接流程。
- 音眸区块顺序、标签文本样式、`explain` 视觉优先级等若在共享渲染树中调整，需按 Mac / iPad / iPhone 联动回归。
- ShareKit 的 payload 结构、音眸 segment 解析、分页导出和保存/分享动作属于共享基础设施；改动时至少按 iPhone 首期入口和后续跨端复用性一起评估。
- 资料库页的排序/筛选摘要属于页面级反馈，放在标题或控制区附近即可；专辑网格和曲目长列表只接收容器传入的派生值，不要直接订阅高频全局状态。

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
- 分享模板与预览：`ShareKit/Template/iPhone/*`
- 首期入口：`TrackDetailView` 的 iPhone 分享菜单与 `SharePreviewView` 全屏预览

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
8. 改 `TrackDetailView` 的 iPhone 分享入口、`SharePreviewView` 或 `ShareKit/Template/iPhone/*`：优先按 iPhone 私有体验改动处理，但必须确认没有破坏共享 payload / render / action 层接口。
9. 改 `AppStore` / `PlaybackStore` / `FavoriteStore` 的字段边界时，必须重点回归资料库长列表、播放条、正在播放页和收藏交互，防止重新放大高频广播范围。
10. 改 `LibraryIndexStore` 或 `LibraryViewModel` 时，除功能回归外，还要确认 favorites / unreported / recent 的本地查询结构没有退回全表扫或整页重载。
11. 改 `BonjourDiscovery`、`ConnectionView` 或 `AppStore.connect` 时，必须验证“自动发现 -> 点击连接 -> 取消 / 成功 / 断开”整条路径，并确认阶段反馈仍然即时。

## 5. 本清单维护要求

- 新增 Bridge 页面时，优先判断是“共享组件”还是“端私有容器”。
- 若新增 `Pad*` / `Phone*` / `Mac*` 或跨端共享模块，需同步更新本清单与 `GEMINI.md` 的 Bridge 约束条目。
- 资料库、连接链路或正在播放页的性能策略变动时，需同步更新 `Docs/PERFORMANCE_GUARDRAILS.md`。
