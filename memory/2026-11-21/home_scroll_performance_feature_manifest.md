# 特性清单：macOS 主页滚动高 CPU / 掉帧治理

- **日期**：2026-11-21
- **现象**：主页上下滚动时 `Soniclens Bridge` CPU 约 54%、`WindowServer` 约 34.5%、GPU 约 31%，肉眼可见掉帧。

## 根因与对应改动

1. **底部播放条实时毛玻璃**：[`PlaybackBarWindowOverlayController.swift`](../../soniclens-bridge/SoniclensBridge/Views/PlaybackBarWindowOverlayController.swift) 的 `NSVisualEffectView` 由 `.withinWindow` 改为 `.behindWindow`，并叠加半透明底色保持层级感（滚动不再迫使 WindowServer 逐帧重模糊）。
2. **`GlassPanel` 整卡圆角离屏裁剪**：[`Theme.swift`](../../soniclens-bridge/SoniclensBridge/Views/Theme.swift) 新增 `clipsContent`（默认 `false`），阴影半径收敛；`PlayerView.swift` 两处 `padding: 0` 且内含滚动内容的卡片显式传 `clipsContent: true`；`PhoneHomeHero` / `PadHomeHero` 的渐变背景自行 `clipShape`，避免圆角溢出。
3. **主页结构**：[`HomeView.swift`](../../soniclens-bridge/SoniclensBridge/Views/HomeView.swift) 拆平 `LazyVStack` 内的巨型 `VStack`；[`HomeHotModulesView.swift`](../../soniclens-bridge/SoniclensBridge/Views/HomeHotModulesView.swift) 将卡片内 `LazyVGrid` 换成 `StaticTwoColumnGrid`，`ProfileDonutChart` 圆环加 `.drawingGroup()`；`RecentPlaysSection` 实现 `Equatable`。
4. **热力图**：`TrendHeatmapCanvas` 按 5 个色阶合并 `Path` 绘制（原先逐格 `fill`，约 2160 次）；macOS 下新增 `HeatmapScrollHoverGate`，滚轮滚动期间暂停 Hover 命中并清空 tooltip；`PreferenceKey` 回调加 0.5pt 去噪。
5. **顶层重算**：[`AppLayoutView.swift`](../../soniclens-bridge/SoniclensBridge/Views/AppLayoutView.swift) 移除对 `PlaybackStore` 的直接读取，播放条/沉浸播放浮层桥接与 `Cmd+J` 下沉到 `MacPlaybackOverlayHostView`。
6. **封面解码**：[`PlaybackBarView.swift`](../../soniclens-bridge/SoniclensBridge/Views/PlaybackBarView.swift) `ArtworkSquareView` 与 `ArtistAvatarView`（改用 `LazyImage`）接入 `ImageProcessors.Resize` 按显示尺寸下采样；[`project.yml`](../../soniclens-bridge/project.yml) 三个 App Target 增加 `Nuke` 产品依赖。

## 验证（实测）

- `xcodegen generate --spec soniclens-bridge/project.yml`：退出码 0
- `make ui-snapshots`：退出码 0；已 `read_file` 审阅 `trend_macos_dark/light`、`trend_macos_hover_light`、`trend_empty_dark`，无回归
- `make ios`、`make ipad`：退出码 0（含既有 `LiveActivityArtworkStore.swift:147` 隔离警告）
- `verify_ui.sh ios-sim --tab home --appearance dark`：退出码 0；首轮审图发现 Hero 渐变溢出圆角，修复后重跑并再次审图确认已贴合

## 未验证项

- 未实测修复后的真实滚动 CPU / WindowServer 占用，需安装新版本后用 iStat 对比。
- 未运行 `make test-bridge`；`verify_ui.sh phone-tests` 因历史单测编译错误本就失败（见 2026-11-20 清单）。
