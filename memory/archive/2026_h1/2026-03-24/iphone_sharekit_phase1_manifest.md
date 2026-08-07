# iPhone ShareKit 一期闭环特性清单

## 日期

2026-03-24

## 摘要

Bridge 新增 `ShareKit` 目录，首期在 iPhone `TrackDetailView` 落地三类分享能力：曲目信息海报、歌词长图、音眸全文长图。能力闭环包含预览、单图/分页 PNG 渲染、保存到 Photos、系统分享与本地分享事件日志。

## 范围

1. **ShareKit 分层**
   - `Domain`：分享场景、payload、音眸 segment、导出结果类型。
   - `Builder`：由 `Track`、歌词、主 insight、收藏状态装配分享数据。
   - `Template/iPhone`：iPhone 专属海报版式与预览页。
   - `Render`：固定宽度渲染、长图高度测量、分页切片、临时 PNG 落盘。
   - `Action`：Photos `.addOnly` 保存与 `UIActivityViewController` 分享。
   - `Analytics`：本地分享事件日志抽象。
2. **iPhone 首期入口**
   - `TrackDetailView` 将原“导出快照”替换为 iPhone “分享”菜单。
   - 非 iPhone 端保留原有快照导出路径，不复用新海报布局。
3. **音眸全文分享**
   - 复用 `InsightTaggedContentParser` 语义，将 `<original>/<translation>/<explain>` 解析为稳定 segment。
   - 非 tagged 文本（如背景信息、时代语境）进入“补充解读”区，不丢失内容。
4. **工程与权限**
   - `Info.plist` / `project.yml` 增加 `NSPhotoLibraryAddUsageDescription`。
   - 新增 `SoniclensBridgePhoneTests`，覆盖 parser / builder 基本行为。

## 已稳定约束

1. **分享入口**
   - iPhone 分享首期只从 `TrackDetailView` 进入，不再在详情页里拼装临时快照逻辑。
   - `ShareKit` 复用的是数据装配、渲染、保存和系统分享动作，不复用 macOS 快照布局。
2. **保存与分享模式**
   - 系统分享只输出单张长图。
   - 保存图片时允许用户显式选择“长图”或“分页”。
   - 长图导出如果 PNG 写入失败，会回退到高质量 JPEG，避免整单失败。
3. **音眸富渲染**
   - `lyrics_translation` 必须作为单卡完整渲染，原文弱化、翻译强调，不再切成多个小卡。
   - `analysis_summary`、`background_info`、`era_context` 保持独立卡片。
   - `analysis_by_section` 里的 `appreciate_analysis` 以 `<original>/<translation>/<explain>` 为一组，保留标签完整性，不显示“分句或段 1-n”这类人工编号。
   - `analysis_by_section` 的纯文本段落只显示真实数据，缺失内容直接省略，不补空标题。

## 验证

- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePhone -destination 'generic/platform=iOS' -derivedDataPath .derivedData/iOSPhone CODE_SIGNING_ALLOWED=NO build`
  - 结果：`BUILD SUCCEEDED`
- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePhoneTests -destination 'generic/platform=iOS Simulator' -derivedDataPath .derivedData/iOSPhoneTests CODE_SIGNING_ALLOWED=NO build-for-testing`
  - 结果：`TEST BUILD SUCCEEDED`

## 后续注意

- 目前测试 target 已编译通过，但未在具体 simulator 上执行 `xcodebuild test`。
- `PhoneNowPlayingView` 仍未接入分享入口，后续若扩展入口，应继续复用 `ShareKit` builder/render/action，不要复制一套模板或导出逻辑。
