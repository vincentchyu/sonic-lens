# 热门流派 Top 50 扩容与端到端闭环特性清单

## 1. 背景与痛点

用户反馈：“热门流派现在需要 top50 以前不是只有 20 吗”。

经全面排查，系统在热门流派聚合、存储、查询及客户端呈现链路上存在以下截断瓶颈：
1. **快照容量受限截断**：`internal/model/dashboard_stat.go` 中的 `refreshTopGenreStats(tx, topN)` 依赖全局 `TopN`（默认为 10，历史部分配置或场景为 20），导致预聚合表 `top_genre_stat` 中仅写入了 10~20 条数据。
2. **DAO 查询层强行返回截断快照**：`internal/model/genre.go` 的 `GetTopGenresWithDetails` 原先只要 `len(statRows) > 0` 就直接返回快照。当调用方请求 `limit=50` 时，因快照仅有 10~20 条而被硬截断，且未触发回退。
3. **API 默认 Limit 参数受限**：`api/server.go` 中 `/api/dashboard/top-genres` 之前默认 `limit=10`，最大限制仅为 50。
4. **客户端请求未显式指定 Limit**：`HomeViewModel.swift` 中的 `fetchTopGenres` 未携带 query 参数，请求继承了服务端的默认值。
5. **客户端圆环图在大数量下的视觉挤压**：50 个切片直接送入圆环图会因 `gap` 间隙叠加（50 * 0.012 = 0.6）导致圆环变形。

---

## 2. 核心重构与闭环改动

### 2.1 Go 后端与 DAO 改造
- **快照容量提升 (`internal/model/dashboard_stat.go`)**：
  在 `refreshTopGenreStats(tx *gorm.DB, topN int)` 中引入 `genreLimit := max(topN, 50)` 保底机制，确保预聚合快照表 `top_genre_stat` 中至少写入 Top 50 热门流派。
- **DAO 自适应降级与补齐 (`internal/model/genre.go`)**：
  重构 `GetTopGenresWithDetails(ctx context.Context, limit int)`：
  - 若 `statRows` 数量 `>= limit`：直接返回预聚合快照数据；
  - 若 `statRows` 数量 `< limit`（例如快照表尚未刷新或请求更大 limit）：自动回退查询物理 `genre` 表实时拉取 `limit` 条记录并补齐 `Rank`（`index + 1`）；
  - 若物理表查询异常：透明降级返回已有 `statRows`，彻底消除截断并保证高可用。
- **API 接口升级 (`api/server.go`)**：
  将 `/api/dashboard/top-genres` 的默认 `limit` 升级为 `50`，最大允许 `limit` 扩大至 `100`。
- **Edge Worker 同步 (`cloudflare/worker_project/src/index.js`)**：
  同步 Worker `/api/dashboard/top-genres` 默认 `limit=50` 及 `results.length < limit` 时的降级查询逻辑。
- **API 文档更新 (`api/api.md`)**：
  更新热门流派接口参数说明为 `limit` (默认 50，最大 100)。

### 2.2 SwiftUI 客户端 (iOS/macOS/iPad) 改造
- **请求显式传参 (`HomeViewModel.swift`)**：
  在 `fetchTopGenres(server:)` 中增加 `queryItems: [URLQueryItem(name: "limit", value: "50")]`，确保稳定拉取 Top 50。
- **圆环图与长列表分层渲染 (`HomeHotModulesView.swift`)**：
  - `ProfileDonutChart`：重构 `chartSegments` 算法，圆环图聚焦渲染 Top 8 鲜明切片，并将超出部分自适应归并或平滑衔接，消除了 50 个切片间隙叠加导致的圆环变形。
  - `ListeningProfileDetailSheet`：下方的“口味偏好完整数据”列表完整渲染 50 个流派项，支持流畅滚动浏览与点按精准跳转关联专辑。

---

## 3. 验证结果

- **Go 单元测试**：全量执行 `go test ./internal/... ./api/...` 全部 PASS。新增 `internal/logic/genre/service_test.go` 中针对 Top 50 的服务接口用例。
- **Swift 客户端构建**：针对 `SoniclensBridgeMac` 和 `SoniclensBridgePhone` 编译构建成功（`BUILD SUCCEEDED`），无警告。
