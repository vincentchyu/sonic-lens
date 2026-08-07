# macOS 正在播放主副卡槽栈式交互重构与阴影裁剪修复特性清单

- **日期**: 2026-08-09
- **模块**: Bridge macOS 客户端 (`soniclens-bridge/SoniclensBridge/Views/NowPlayingView.swift`)

---

## 1. 特性背景与动机

针对 macOS 正在播放沉浸式视图 (`NowPlayingView`) 进行交互体验重构。原本的左侧固定专辑信息、右侧单 Tab 切换模式在观看音眸（Insight）赏析时无法获得歌词实时定位。
重构后支持类似“卡牌/栈”的主副卡槽对称切换，并在音眸主屏模式下于左侧 380px 副屏保留 LRC 实时歌词，极大地改善了播放外语歌曲时对照音眸阅读翻译与歌词赏析的体验。

---

## 2. 核心变更细节

1. **主副卡槽栈式转场 (`Secondary & Primary Stack Layout`)**：
   - 默认模式：左侧副展示区显示 `NowPlayingLeftPanel`（封面与曲目信息），右侧主展示区显示 `NowPlayingLyricsPanel`（主歌词）。
   - 音眸模式：左侧副展示区显示 `NowPlayingSideLyricsPanel`（副屏实时 LRC 歌词），右侧主展示区显示 `MacNowPlayingInsightPanel`（音眸赏析）。
   - 切回逻辑：点击左侧副屏歌词或顶栏“歌词”Tab，触发 `withAnimation(.spring(response: 0.38, dampingFraction: 0.82))` 平滑切回默认模式。
2. **边缘阴影裁切割痕修复 (`Clipped Border Fix`)**：
   - 修复了左侧容器上因外部硬套 `.clipped()` 导致的阴影硬裁剪问题。
   - **根本原因**：`NowPlayingArtwork` 带有 `radius: 32` 扩散模糊阴影，超出 380px 的区域被 `.clipped()` 直接垂直割断，形成黑线；移除 `.clipped()` 后阴影自然渐变羽化。
3. **架构与平台隔离验证**：
   - 确认变动 100% 局限在 `NowPlayingView.swift` 内部。
   - `SoniclensBridgeMac` 与 `SoniclensBridgePhone` 均为 `** BUILD SUCCEEDED **`，完全不影响 iOS 与 iPadOS 沉浸界面。
