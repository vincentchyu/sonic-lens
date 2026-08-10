# ShareKit macOS 大屏 Bento Grid 双栏分页海报与专辑分享全覆盖特性清单

## 1. 概述
为提升 SonicLens 在 macOS 大屏下的分享与导出体验，将 ShareKit 引擎全面升级扩展至 macOS，实现了 16:9 (1920x1080) 画幅下的 **Bento Grid 双栏流式分页 (Dual-Column)**，修复了测算尺寸偏窄、单页溢出不渲染、单页截断以及关闭模态遮罩交互等一系列问题。同时清理了旧有的 `SnapshotExport.swift` 界面快照代码，完成了曲目与专辑分享的统一。

---

## 2. 核心架构与功能变更

### 2.1 macOS 16:9 双栏 (Bento Grid) 分页计算引擎
- **全屏切片测算 (`SharePayloadPaginator.swift`)**：
  - 通过 `#if os(iOS)` / `#if os(macOS)` 区分平台测算尺寸。
  - macOS 端右侧内容画布独立测算宽度设为 `704px`（双栏单列宽度），单页双栏总纵向高度预算设为 `(1080 - 156 - 28) * 2 = 1848pt`。
  - `MacSharePaginatedPosterView` 自动根据单页节点高度动态分流为 `Left -> Right` 两列呈现，避免大屏横向阅读文字单行过长问题。
  - 增加空节点保底机制，确保在节点为空时依然生成 1 页 Slice，防止预览框空死。

### 2.2 多页批量导出与原生剪贴板动作 (`MacShareActionHelper` & `SharePreviewView`)
- **粘贴板多图支持 (`MacShareActionHelper.copyImagesToPasteboard`)**：
  - 一键点击“复制图片”时，支持将所有分页生成的 `NSImage` 一起写入 macOS 系统剪贴板（可以直接批量粘贴到聊天软件或备忘录中）。
- **文件批量导出**：
  - 点击“保存”时，若导出的海报超过 1 页，自动按照 `filename-1.png`, `filename-2.png` 批量输出至用户指定目录。
- **画廊式水平滚动预览 (Horizontal Gallery Preview)**：
  - 将 macOS 预览框中的单帧静态渲染重构为 `ScrollView(.horizontal)`。
  - 在预览窗口中 100% 真实展示所有即将导出的 `1920x1080` 海报切片。
  - 在背景遮罩 `Color.black.opacity(0.85)` 上挂载 `.onTapGesture { dismiss() }`，支持点击任意空白处顺滑退出模态框。

### 2.3 专辑分享全覆盖与旧快照清理 (`AlbumDetailView` & `TrackDetailView`)
- **专辑页接入 ShareKit**：
  - 在 `AlbumDetailView` 中彻底替换了旧的 `exportSnapshotPNG` 按钮逻辑，统一使用 `openSharePreview(scene:)` 打开 ShareKit 预览框。
  - 菜单文案统一规范为“导出海报：基础信息”和“导出海报：音眸专辑”。
- **未生成音眸降级优雅提示 (`InsightShareParser.swift`)**：
  - 当专辑/曲目尚未生成 AI 音眸时，自动解析并返回“当前专辑/曲目尚未生成 AI 音眸解析”的友好的引导卡片，解决此前空音眸导致预览无反应的 BUG。
- **废弃旧快照引擎**：
  - 彻底清理移除了 `AlbumSnapshotView`、`AlbumInsightSnapshotView` 以及底层的 `SnapshotExport.swift`，并使用 `xcodegen generate` 重新构建工程。

---

## 3. 架构约束与端侧隔离 (Preserved Constraints)

- **iPhone 共享逻辑 100% 独立隔离**：
  - iPhone 端的 `SharePreviewView` 依然保持在 `#if os(iOS)` 中，固定保留 390pt 画布与 310pt 单栏切片计算。
  - iPhone 端点击“系统分享”依然强制输出 `.singleLongImage` 单张长图，点击“保存图片”依然通过 Action Sheet 允许用户在“长图”与“分页”间自由切换。

---

## 4. 验证结论

- **macOS Build**: `xcodebuild -scheme SoniclensBridgeMac` 验证成功 (**BUILD SUCCEEDED**)。
- **iOS Build**: `xcodebuild -scheme SoniclensBridgePhone` 验证成功 (**BUILD SUCCEEDED**)。
