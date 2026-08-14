# Top热门流派与流派专辑播放统计闭环对账重构特性清单

## 1. 特性背景与重构动机
在之前的播放统计统一重构中，虽然确定了 `track_play_records` 为全局唯一播放事实源，但系统在**热门流派统计（`top_genre_stat` / `genre`）与流派专辑检索（`GetAlbumsByGenre`）**之间存在统计口径撕裂：
- `top_genre_stat` 基于听歌流水按精准字符串 `GROUP BY`，导致流派 `"Rock"` 的播放数为 132；
- `GetAlbumsByGenre` 采用 `LIKE '%Rock%'` 通配子串泛化匹配，把 `Rock Musical`、`Indie Rock`、`Alternative Rock` 等派生流派全盘拉入，造成关联专辑播放量求和与热门流派展示数值背离。

经过方案选择，实施**【方案 A：精准规范化对齐模式 (Strict Normalized Exact Matching)】**，彻底打通流派统计与流派专辑列表的闭环对账。

---

## 2. 核心代码变更与架构调整

### 2.1 听歌流水流派拆分对账 (`internal/model/genre.go`)
- **`splitGenreTags`**：新增流派标签清洗与拆分工具，支持对逗号 `,`、斜杠 `/`、分号 `;`、竖线 `|` 分隔的复合流派字符串进行规范化解包。
- **`extractPrimaryGenreTag`**：多段组合流派（如 `Alternative Rock,Electronic,Electronica...`）改为只取首个主体 Segment (`Alternative Rock`) 进行匹配与呈现，避免多处重复拆分计数。
- **`ResolveGenreIdentity` & `GenreMap`**：智能寻址流派身份。英文标签（如 Post-Rock, Shoegaze 等）全量原汁原味无损保留；中文标签（如 摇滚, 另类）通过 `common.GenreMap` 字典 / `genre.name_zh` 反查对应的标准英文 `Name`，彻底封堵中文字符写入 `genre.name` 字段的隐患。
- **未匹配流派曝光与人工干预 (`/api/genres/unmatched` & `/api/genres/map`)**：在 Web 管理后台（`templates/admin/main_sections.html` & `static/admin/charts.js`）提供“待人工干预未归因流派”卡片与映射交互，允许用户手工将未关联流派绑定到已知标准英文流派，并自动触发无损对账。

### 2.2 流派专辑精准 Token 匹配 (`internal/model/album.go`)
- **`buildExactGenreMatchClause`**：构造针对 `album.genre` 和 `track.genre` 的跨多标签精准词界匹配 SQL 条件（精确匹配独立 Token，区分逗号/斜杠/分号分隔符）。
- **`GetAlbumsByGenre` / `GetAlbumsByGenreCount`**：替换原有的 `LIKE '%<genre>%'` 模糊子串模式，改为精准词界 Token 匹配，确保检索流派 `"Rock"` 时只精确匹配主体或多标签中包含 `"Rock"` 的专辑，彻底剔除 `Rock Musical`、`Indie Rock` 等无关派生流派。

### 2.3 Bridge SwiftUI 客户端交互贯通 (`soniclens-bridge/`)
- **`HomeHotModulesView.swift`**：
  - 为 `ListeningProfileCard`、`ListeningProfileGenrePanel`、`ListeningProfileSourcePanel` 及 `ListeningProfilePanelShell` 新增 `onOpenDetail` 回调闭包，在头部右上角加入 `查看全部 >` 入口按钮。
  - 为 `ListeningProfileDetailSheet` 扩展 `onSelectGenre` 跳转闭包，当用户在全量流派列表点击任意流派（包括第 4 名的 Rock）时，可优雅切至原生 `GenreAlbumsSheet` 展示关联专辑。
- **`HomeView.swift` / `PadHomeView.swift`**：
  - 透传全量 Top 10 流派和渠道数据，挂载 `ListeningProfileDetailSheet` Sheet 路由，实现“查看全部 -> 调起全量面板 -> 点按流派直达流派专辑视图”全链路流畅交互。

---

## 3. 测试与验证结果

### 3.1 单元测试与 Xcode 验证
- **Go 单元测试**：`TestExtractPrimaryGenreTag`、`TestResolveGenreIdentity`、`TestReconcileGenrePlayCounts` 与 `TestGetAlbumsByGenreExactMatching` 全部 PASS。
- **macOS Bridge 构建**：运行 `xcodegen generate` 并使用 `xcodebuild -scheme SoniclensBridgeMac` 编译，**BUILD SUCCEEDED** 无报错。

### 3.2 真实数据库与全项目构建
- 在 Docker MySQL 真实数据下验证 `GetAlbumsByGenre("Rock")` SQL 查询，成功将包含多标签 `classic rock,rock,space rock` 和纯 `Rock` 的专辑精准选出，排除无关衍生流派。
- 运行 `go build ./...` 100% 编译通过。
