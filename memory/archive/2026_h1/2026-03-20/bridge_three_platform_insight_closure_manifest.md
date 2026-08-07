# Bridge 三端产品线与音眸闭环记忆清单

## 日期

- 2026-03-20

## 背景

- `soniclens-bridge` 已从早期共享壳演进为 macOS、iPadOS、iPhone 三条客户端产品线。
- Bridge 端的音眸展示此前长期停留在“平铺伪代码摘要”阶段，`analysis_by_section`、`<original>/<translation>/<explain>` 标签语义与 Web 端不一致。
- 本轮迭代完成了 Bridge 音眸解析与富渲染闭环，并在三端全屏播放页与歌曲详情页落到同一套共享语义。

## 本次沉淀的事实标准

### 1. 三端产品线边界

- 三端入口已固定为：
  - `SoniclensBridgeMac`
  - `SoniclensBridgePad`
  - `SoniclensBridgePhone`
- `SoniclensCore` 与 `ViewModels` 继续作为共享层。
- `NowPlayingView.swift`、`PadNowPlayingView.swift`、`PhoneNowPlayingView.swift` 只负责端私有容器、交互密度和滚动高度，不再各自维护一套音眸解析规则。

### 2. 音眸数据契约与语义来源

- Bridge 端 `Insight` 结构必须以 `core/ai/agent.go` 的 `GetTrackInsightSchema()` 和 `templates/lyrics_live.html` 为事实标准。
- `analysis_by_section` 是核心字段，不能再被扁平化忽略。
- `analysis_by_section` 需兼容：
  - 标准对象
  - 历史 JSON 字符串
  - 非法脏数据回退
- 标签语义固定为：
  - `<original>`：原文
  - `<translation>`：翻译
  - `<explain>`：解释

### 3. Bridge 共享音眸渲染闭环

- 共享数据解析和展示模型已下沉到：
  - `soniclens-bridge/SoniclensCore/Models/LibraryModels.swift`
  - `soniclens-bridge/SoniclensBridge/Views/InsightDetailView.swift`
- 三端全屏播放页和歌曲详情页必须复用 `InsightPrimaryContentView` 这一套共享富渲染树。
- 三端允许分叉的维度仅限：
  - 外层容器与 padding
  - 滚动容器高度
  - 窄屏排版（例如 iPhone 原文/翻译上下堆叠）

### 4. 当前视觉规则

- 沉浸播放页的音眸内容不再使用明显的白色玻璃卡片，而是嵌入液态背景。
- 解释是视觉第一层：
  - Mac / iPad：解释字体更大、更亮、更重。
  - iPhone：解释仍为第一层，但字号按窄屏收敛。
- 原文与翻译是辅助层：
  - 原文弱于解释
  - 翻译进一步弱化
  - iPhone 端翻译使用斜体

### 5. 数据消费规则

- API 返回多条 insight 时，Bridge 默认只消费 `insights.first` 作为主展示对象。
- 该排序依赖后端 `total_score desc, created_at desc` 的既有行为。

## 已完成的页面闭环

- Mac 全屏播放页 `NowPlayingView.swift`
- iPad 全屏播放页 `PadNowPlayingView.swift`
- iPhone 全屏播放页 `PhoneNowPlayingView.swift`
- 歌曲详情页 `TrackDetailView.swift`
- 独立音眸详情页 `InsightDetailView.swift`

## 当前仍需记住的残留债务

- `soniclens-bridge/SoniclensBridge/Views/PlayerView.swift` 里旧的 `InsightLiveSection` 仍保留纯摘要卡片实现，不代表当前三端正式播放页/详情页方案。
- 后续若继续维护旧 `PlayerView.swift` 路径，需显式迁移到 `InsightPrimaryContentView`，否则不要把它误记为三端音眸主链路的一部分。

## 后续迭代约束

- 修改 Bridge 音眸展示时，优先改共享解析层或共享渲染层，不要回到端内字符串切割。
- 若后端 `GetTrackInsightSchema()` 发生结构变动，Bridge 端需同步更新 `Insight` 解码与标签解析规则。
- 新增客户端音眸页面时，应优先复用共享渲染树，只在外层容器做端特化。
