# iPhone ShareKit 公共壳层收口特性清单

## 日期

2026-03-24

## 摘要

Bridge 的 iPhone ShareKit 进一步收口公共层，将分享预览的顶部公共标题、背景、玻璃拟态容器、封面式 hero 头部与底部居中品牌信息统一抽象到 `SharePosterShell`。基础信息、歌词、音眸三种分享场景不再重复拼接外壳，只保留各自正文内容。

## 范围

1. **公共壳层**
   - 新增 `SharePosterShell` 作为 iPhone 分享卡片的统一外壳。
   - 外壳内部统一组合 `SharePosterBackground`、`SharePosterHeader`、`SharePosterCard`、`SharePosterFooter`。
   - 顶部公共标题由 `SharePosterShell` 统一渲染，位于刘海下方并与头图保持固定间距。
   - 头部 hero 区仍由公共 `SharePosterHeader` 渲染，保留封面、收藏态、位置标签与指标标签。
2. **场景收敛**
   - `TrackInfoPosterView`、`LyricsLongPosterView`、`InsightLongPosterView` 只负责各自正文内容。
   - 基础信息、歌词、音眸三种分享场景共享同一封面结构与底部闭环。
3. **工程接入**
   - 新增公共壳层文件并挂入 `SoniclensBridgePhone` / `Pad` / `Mac` 三个 target。
   - 渲染与分页逻辑保持不变，仍复用现有 `ShareRenderer` 与 `LongPosterPaginator`。

## 已稳定约束

1. **公共层职责**
   - 分享页面的顶部标题、背景、头图、正文卡片和底部品牌信息属于公共层，不允许在三个场景里重复拼接。
   - `SharePosterHeader` 的主头图必须保持“左侧封面、右侧歌名/艺人专辑、左下位置/指标、右下收藏态”结构，不再把位置标签放在卡片顶部。
   - `SharePosterFooter` 需要整体居中展示，品牌文案与 slogan、署名、时间信息按纵向分行排布。
2. **场景职责**
   - 场景视图只允许定义正文内容，不允许再持有外壳结构。
3. **渲染一致性**
   - 单图导出与分页导出都必须通过同一套公共外壳测量和渲染。

## 验证

- `xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj -scheme SoniclensBridgePhone -destination 'generic/platform=iOS' -derivedDataPath .derivedData/ShareShellPhone CODE_SIGNING_ALLOWED=NO build`
  - 结果：`BUILD SUCCEEDED`

## 后续注意

- 如果后续新增分享场景，应优先复用 `SharePosterShell`，只补充正文内容组件。
- 若公共头图需要变化，应先修改 `SharePosterHeader`，不要在具体场景视图中复制一份新头图。
