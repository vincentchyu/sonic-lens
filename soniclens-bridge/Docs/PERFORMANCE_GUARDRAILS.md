# SonicLens Bridge 性能治理与约束清单

## 1. 治理目标

- 三端共享层优先消除重复工作，再处理单页体感问题。
- 资料库必须坚持 local-first，不把排序、筛选、搜索重新推回远端分页或大数组内存过滤。
- 高密度列表、播放页和首页热区必须优先为“可持续运行”设计，而不是只追求首屏视觉效果。

## 2. 已落地约束

### 2.1 状态层

- `AppStore` 只承载低频全局状态：
  - `currentServer`
  - `connectionStatus`
  - `recentServers`
  - `activeConnectionTargetKey`
  - `InsightAnalysisCoordinator`
- 高频 UI 状态必须落在细粒度切片：
  - `PlaybackStore`
  - `FavoriteStore`
- 密集列表、详情页、播放条禁止直接依赖 `AppStore` 的高频字段。

### 2.2 资料库

- `LibraryViewModel` 的后台同步必须 single-flight；页面出现、回前台、`librarySyncDidUpdate` 只能合并请求。
- 专辑/曲目列表必须执行“第一页先返回，总数异步补”。
- 专辑/曲目重载必须带请求 token，旧条件结果不得回写覆盖新条件。
- 收藏变更优先 patch 当前可见行；只有收藏筛选页才允许回到整页重载。
- `LibraryIndexStore` 当前以 SQLite + FTS5 为主，schema `7` 的关键约束为：
  - `track_index.is_favorited_effective`
  - favorites / unreported / recent 对应复合索引

### 2.3 连接

- 连接状态必须显式区分阶段：
  - `resolving`
  - `healthCheck`
  - `establishingRealtime`
- 连接中必须同时具备：
  - 顶部状态反馈
  - 行内阶段反馈
  - 取消能力
- Bonjour 候选如果已经解析出 socket 地址，连接时必须优先走解析地址，避免 `.local` 二次解析拖慢健康检查。
- 三端主界面必须提供显式断开入口，允许用户重新走连接流程。

### 2.4 当前播放与首页

- `APIClient` 默认复用共享 `URLSession`。
- `PlayerViewModel.load` 需要并行请求可并行资源，并丢弃过期任务结果。
- iPhone 默认使用更轻量的动态背景策略；iPad / macOS 允许更丰富表现，但必须保留性能模式降级路径。
- 大面积 blur / shadow / perpetual animation 不能叠加到长列表和常驻页面。

## 3. 回归清单

### 3.1 连接

- 进入连接页后，点击自动发现候选应立即看到阶段反馈。
- 同一目标重复点击应取消当前连接，而不是并发多条链路。
- 连接成功后，工具栏 `power` 按钮应能断开并返回连接流程。

### 3.2 资料库

- 首次进入专辑页后立刻点排序。
- 首次进入曲目页后立刻点筛选。
- 收藏筛选、未上报筛选首次进入不应出现明显卡顿。
- 快速切换排序 / 筛选 / 搜索时，旧结果不能回跳。

### 3.3 播放与首页

- 连续切歌时歌词 / insight 不应互相阻塞。
- 反复打开 / 关闭正在播放页不应出现明显掉帧。
- 首页常驻 1 到 2 分钟后切页再返回，动态背景不应明显升温或抖动。

## 4. 建议验证工具

- Xcode Hangs
- Instruments Time Profiler
- Instruments SwiftUI
- Instruments Core Animation
- Instruments Network

## 5. 当前未闭环项

- iPhone 中文九宫格搜索输入组合态卡顿仍是已知问题，当前没有纳入已闭环基线；后续若重新处理，必须优先保证搜索框尺寸和布局不回归。
