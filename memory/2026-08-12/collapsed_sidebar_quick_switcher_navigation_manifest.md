# macOS 侧边栏折叠状态 Quick Switcher 导航与 Toolbar 统一 UI/UX 视觉重构特性清单

## 1. 特性背景与概述

在 macOS 桌面客户端应用中，用户关闭/收起侧边栏（Sidebar）主要为了追求最大化主体空间或沉浸式视觉。但经典交互模式下，侧边栏关闭后切换模块需要“展开侧边栏 -> 点击目标项 -> (可选)重新收起侧边栏”三步操作，导航阻尼感较重。同时原 Toolbar 存在下拉符号重复（`∨ 专辑 ∨`）、右侧操作按钮混乱堆叠、信息层级倒置等痛点。

本特性在 SonicLens Bridge 中实现了 **居中 Quick Switcher Menu**、**系统 Option 快捷键 (`⌥1` ~ `⌥6`)** 以及 **Toolbar 三段式逻辑层级重构**，在完全保留大屏沉浸感的前提下实现了极致优雅的控件呈现。

---

## 2. 关键设计与代码落地点

### 2.1 居中 Quick Switcher 彻底消除重复箭头 (`ToolbarTitleMenu`)
- **改动文件**：[AppLayoutView.swift](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensBridge/Views/AppLayoutView.swift)
- **技术细节**：
  - 增加 `.menuIndicator(.hidden)` 修饰符，强行屏蔽 macOS 系统默认追加的多余 Indicator。
  - 居中标题干净呈现为 `专辑 ▾`，且带微透明 hover 悬停高亮背景（Hover Glass Pill）。
  - 下拉菜单按“浏览”、“深度内容”、“规划”分组，支持 SF Symbols 图标回显与当前激活状态的打勾。

### 2.2 右侧操作区三段式分层重排 (Toolbar Hierarchy)
- **改动文件**：[AppLayoutView.swift](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensBridge/Views/AppLayoutView.swift)
- **排列规则**：
  `[ 搜索框 (ToolbarSearchField) ] -> [ 排序 / 筛选 Menu ]  |分隔线|  [ 性能模式 (PerformanceModeToolbarButton) ] -> [ 断开电源 ]`
- **效果**：高频视图操作（搜索、排序、筛选）贴近屏幕中央与左侧，低频系统级配置（性能模式、电源）被优雅隔离在最右侧。

### 2.3 性能模式按钮重构 (PerformanceModeToolbarButton)
- 将原生宽大的 Toggle Switch 开关替换为精致的图标胶囊按钮 `PerformanceModeToolbarButton`（`gauge.with.dots`），大幅收缩空间开销。

### 2.4 快捷键避让与双重响应 (`⌥1` ~ `⌥6`)
- 为全站 6 个模块分配 `⌥1`~`⌥6` 快捷键，在下拉菜单项与 `AppLayoutView` 视图根部挂载双重键盘响应机制。

---

## 3. 验证结果

- **构建验证**：使用 `xcodebuild -scheme SoniclensBridgeMac` 构建项目成功，`** BUILD SUCCEEDED **`。
