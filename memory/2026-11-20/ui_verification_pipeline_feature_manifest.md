# 特性清单：三层 UI 自检闭环、聆听趋势热力图对齐与侧边栏性能优化

- **日期**：2026-11-20
- **关联目标**：`api` / `internal/model` / `soniclens-bridge` / `Makefile` / `.agents/skills/soniclens-ui-verify` / `~/.gemini/config/mcp_config.json`

---

## 1. 背景与解决的核心问题

1. **热力图视觉对齐、时间窗口与可读性修复**：
   - 修复聆听趋势热力图 Y 轴时间标签与格点垂直对齐偏移、X 轴倾斜日期锚点与网格列漂移；
   - 修复后端趋势统计多返回次日（`now + 1`，例如 20 日出现 `11/21`）的时间窗口越界问题；
   - 为热力图增加桌面端鼠标悬停十字准星、实时数据悬浮提示（Tooltip）与边缘安全翻转，并补全窄宽换行、空态卡片居中展示与图例 `"暂无数据"` 单行防换行保护。
2. **侧边栏收折动画 UI 卡顿治理**：
   - 解决侧边栏折叠动画期间容器阴影高频离屏重算、`GeometryReader` 全量同步重排与 macOS `ViewThatFits` 布局抖动导致的掉帧问题。
3. **建立三层 AI UI 自动化自检与可视化验收闭环**：
   - **Layer 1（可测可定位 UI）**：统一无障碍标识常量与 `#if DEBUG` 确定性 Mock 启动路由；
   - **Layer 2（离屏快照 + 模拟器截图 + XCUITest + XcodeBuildMCP）**：macOS 几秒级离屏 PNG 渲染及几何断言、iOS 模拟器自动启动 Mock 截图、双端 XCUITest 骨架及 `XcodeBuildMCP`（`mobilebuildmcp@latest`）接入；
   - **Layer 3（工程技能与门禁契约）**：沉淀项目级技能 `.agents/skills/soniclens-ui-verify/SKILL.md` 与 `Makefile` 快捷入口，强制执行“`read_file` 亲眼审图”验收准则。

---

## 2. 变更详情

### 2.1 后端时间窗口收口
- [`api/server.go`](../../api/server.go)：收紧 `FillInTrendCycle`、`fillInTrendCycleFrom`、`buildTrendWindowResponse` 日期范围，严格按 `[today - (rangeDays - 1), today]` 补齐趋势桶。
- [`internal/model/dashboard_stat.go`](../../internal/model/dashboard_stat.go)：更新 `GetPlayTrendFromStatByDays` 为半开区间窗口查询。
- [`api/server_test.go`](../../api/server_test.go)、[`internal/model/dashboard_stat_test.go`](../../internal/model/dashboard_stat_test.go)：新增 `TestFillInTrendCycleFrom`、`TestBuildTrendWindowResponse`、`TestGetPlayTrendFromStatByDaysUsesHalfOpenWindow`。

### 2.2 热力图修复与桌面端性能治理
- [`soniclens-bridge/SoniclensBridge/Views/HomeView.swift`](../../soniclens-bridge/SoniclensBridge/Views/HomeView.swift)：
  - 统一 `resolvedCellHeight` / `preferredHeight`，重构 Y 轴 `ZStack` 坐标系、X 轴 `-rotationDegrees` `.bottomLeading` 锚点及刻度线对齐；
  - 将滚动提示收口至网格底部独立状态胶囊 `FixedScrollHintState`；
  - 新增 `TrendHeatmapInteractiveGrid`（`.onContinuousHover` 坐标换算 + 十字准星高亮 + 自动边界翻转 Tooltip），并支持 `previewHoveredCell` 用于快照定格验证；
  - 新增 `TrendHeatmapView.emptyStateView`（`chart.bar.xaxis` + `"暂无播放趋势数据"`），修复 `DashboardTrendSectionBody.headerText` 副标题窄宽 `.fixedSize(horizontal: false, vertical: true)` 换行及 `TrendLegendView` `"暂无数据"` 宽度约束。
- [`soniclens-bridge/SoniclensBridge/Views/Theme.swift`](../../soniclens-bridge/SoniclensBridge/Views/Theme.swift)、[`GlassStyles.swift`](../../soniclens-bridge/SoniclensBridge/Views/GlassStyles.swift)、[`PlaybackBarWindowOverlayController.swift`](../../soniclens-bridge/SoniclensBridge/Views/PlaybackBarWindowOverlayController.swift)、[`HomeHotModulesView.swift`](../../soniclens-bridge/SoniclensBridge/Views/HomeHotModulesView.swift)、[`AlbumGridView.swift`](../../soniclens-bridge/SoniclensBridge/Views/AlbumGridView.swift)、[`LibraryView.swift`](../../soniclens-bridge/SoniclensBridge/Views/LibraryView.swift)：
  - `GlassPanel` 与 `GlassCardStyle` 阴影绑定至底层填充圆角形状；
  - `AmbientBackgroundView` 固定子视图标识（`id: \.offset`）并在简化模式下采用 `RadialGradient`；
  - 新增 `WidthDebounceController` / `measureDebouncedWidth` 取代易引发布局抖动的外部 `GeometryReader`；
  - macOS `HotModuleSectionHeader` 移除 `ViewThatFits`；`PlaybackBarWindowOverlayController` 补充 `layout()` 与 `updateShadowPath()`。

### 2.3 三层 UI 验证基建与工程配置
- **确定性 Mock 与无障碍标识（Layer 1）**：
  - [`soniclens-bridge/SoniclensBridge/Testing/UIAccessibilityID.swift`](../../soniclens-bridge/SoniclensBridge/Testing/UIAccessibilityID.swift)
  - [`soniclens-bridge/SoniclensBridge/Testing/UIVerificationSupport.swift`](../../soniclens-bridge/SoniclensBridge/Testing/UIVerificationSupport.swift)
  - [`soniclens-bridge/SoniclensCore/Store/AppStore.swift`](../../soniclens-bridge/SoniclensCore/Store/AppStore.swift)、`HomeViewModel.swift`、`LibraryViewModel.swift`、`SoniclensBridgeApp.swift`、`SoniclensBridgePadApp.swift`、`SoniclensBridgePhoneApp.swift`、`AppLayoutView.swift`、`PadAppLayoutView.swift`、`PhoneAppLayoutView.swift`
- **测试 Target、快照与脚本（Layer 2）**：
  - [`soniclens-bridge/project.yml`](../../soniclens-bridge/project.yml) & `SoniclensBridge.xcodeproj`：新增 `SoniclensBridgeMacTests`、`SoniclensBridgeMacUITests`、`SoniclensBridgePhoneTests`、`SoniclensBridgePhoneUITests`。
  - [`soniclens-bridge/SoniclensBridgeMacTests/SnapshotRenderer.swift`](../../soniclens-bridge/SoniclensBridgeMacTests/SnapshotRenderer.swift) & `TrendHeatmapSnapshotTests.swift`：离屏 `NSWindow` + `NSHostingView` PNG 渲染器与 6 组场景快照（浅色/深色、90/30 天、空态、Hover 及边缘翻转）。
  - [`soniclens-bridge/SoniclensBridgeMacUITests/DashboardMacUITests.swift`](../../soniclens-bridge/SoniclensBridgeMacUITests/DashboardMacUITests.swift) & [`soniclens-bridge/SoniclensBridgePhoneUITests/PhoneDashboardUITests.swift`](../../soniclens-bridge/SoniclensBridgePhoneUITests/PhoneDashboardUITests.swift)。
  - [`soniclens-bridge/scripts/verify_ui.sh`](../../soniclens-bridge/scripts/verify_ui.sh) 与根目录 [`Makefile`](../../Makefile)（新增 `ui-snapshots`、`ui-verify-ios`、`test-bridge`）。
  - 全局 `/Users/vincent/.gemini/config/mcp_config.json` 配置 `XcodeBuildMCP`（`npx -y mobilebuildmcp@latest mcp`，v2.7.1）。
- **技能与工程规范（Layer 3）**：
  - [`.agents/skills/soniclens-ui-verify/SKILL.md`](../../.agents/skills/soniclens-ui-verify/SKILL.md)
  - [`GEMINI.md`](../../GEMINI.md)

---

## 3. 实测验证结果

| 验证项 | 命令 | 结果 | 观察说明 |
| :--- | :--- | :--- | :--- |
| 后端 Go 全量单测 | `/Users/vincent/.asdf/installs/golang/1.27.1/go/bin/go test -count=1 ./...` | 通过（退出码 0） | 包含 `TestFillInTrendCycleFrom`、`TestBuildTrendWindowResponse`、`TestGetPlayTrendFromStatByDaysUsesHalfOpenWindow` |
| macOS 离屏快照生成与审图 | `make ui-snapshots` | 通过（退出码 0，耗时 4.343s） | 产出 `.derivedData/ui_snapshots/macos/` 下 6 张 PNG；经 `read_file` 审阅 `trend_empty_dark.png` 与 `trend_phone_light.png` 确认空态卡片、`"暂无数据"` 单行与窄屏换行全部正常 |
| iOS / iPadOS App 编译 | `make ios` / `make ipad` | 通过（退出码 0） | 无新增编译错误 |
| macOS Test & UITest 构建 | `xcodebuild build-for-testing -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgeMac -configuration Debug -destination 'platform=macOS' -derivedDataPath .derivedData/macOS_test CODE_SIGN_IDENTITY="-" -quiet` | 通过（退出码 0） | `SoniclensBridgeMacTests` 与 `SoniclensBridgeMacUITests` 均构建成功 |
| iPhone 模拟器 Mock 启动截图 | `bash ./soniclens-bridge/scripts/verify_ui.sh ios-sim --tab home --appearance dark` | 通过（退出码 0） | 于 `iPhone 17 Pro`（UDID: `BD1B1DE7-0178-4D8F-B371-E786295153DF`）成功启动并生成 `ios_home_dark_dense.png`，经 `read_file` 确认热力图与安全区布局正常 |
| XcodeGen 与 MCP CLI 校验 | `xcodegen generate --spec soniclens-bridge/project.yml` / `npx -y mobilebuildmcp@latest --version` | 通过（退出码 0） | 工程生成成功；`mobilebuildmcp` 版本 `2.7.1` |
| iOS 历史单元测试（共享目录） | `bash ./soniclens-bridge/scripts/verify_ui.sh phone-tests` | 失败（退出码 65） | 预存在未修改文件 `SoniclensBridgeTests/AppStoreFavoriteActionTests.swift:15,50`（隐式 `self` 闭包捕获）及 `HomeHotModulePresentationTests.swift:122,127`（旧版 `RecentPlayRecord.init` 参数与 `Date` 类型）编译失败 |
