# 🏷️ Web 后台流派全生命周期管理 (CRUD) 与缓存归因调试器特性清单

**特性名称**: Web 后台流派全生命周期管理 (CRUD) 与缓存归因调试器 (Resolver Sandbox)  
**变更日期**: 2026-08-14  
**影响模块**: `internal/model/genre.go`、`internal/logic/genre/`、`api/server.go`、`templates/admin/`、`static/admin/`

---

## 1. 特性背景与设计动机

SonicLens 核心原则以 `genre` 物理表 + `GenreCache` 为全局唯一权威源，彻底废弃静态字典，严禁自动插入伪流派。过去后台仅有 Dashboard 的热门流派 Top 50（只读榜单），缺乏全量流派库的管理视图，导致：
1. **中文译名维护困难**：新流派抓取后仅有英文，无法在后台维护其规范中文译名；
2. **缺乏全生命周期 CRUD**：无法主动录入标准流派字典、检索全部流派库或清理无效流派；
3. **缓存归因黑盒排查难**：当歌曲流派未命中时，无法直观验证 `GenreCache` 繁简转换、分段剥离与权威归因链路。

本特性在 Web Admin【我的资料库】下正式落地【流派】模块，补齐了全量 CRUD、中英文双向过滤、全量流水对账与轻量级【流派缓存归因调试器 (Resolver Sandbox)】。

---

## 2. 核心架构与技术实现

### 2.1 数据模型与服务层扩展
- **Model 层 (`internal/model/genre.go`)**：
  - 新增 `GetAllGenresWithFilter(ctx, keyword, sortBy, limit, offset)` 与 `GetGenreCountWithFilter`，支持按中英文名称模糊检索及排序；
  - 增强 `GetGenreByID` 与 `DeleteGenre` 支持 `int64` 主键。
- **Logic 层 (`internal/logic/genre/service.go`)**：
  - `GenreService` 接口扩展 `GetAllGenresWithFilter` 与 `ResolveGenreTest`；
  - `ResolveGenreTest` 封装首 Segment 提取、权威流派匹配校验与整体归一化结果返回。

### 2.2 RESTful API 接口群
在 `api/server.go` 新增：
- `GET /api/genres`：支持 `keyword`, `sort`, `limit`, `offset` 分页检索；
- `POST /api/genres`：创建流派（英文名强制防中文与 cn-slug 校验，自动同步刷新 `GenreCache`）；
- `PUT /api/genres/:id`：修改流派（支持更新中文译名与英文标准名，自动刷新 `GenreCache`）；
- `DELETE /api/genres/:id`：物理删除流派并刷新 `GenreCache`；
- `POST /api/genres/reconcile`：触发全量流水清洗与播放量对账；
- `POST /api/genres/resolve-test`：归因沙盒测试接口。

### 2.3 Web 前端交互与沙盒调试组件
- **侧边栏导航 (`templates/admin/navbar.html`)**：在【我的资料库】下新增【流派】Tab；
- **流派管理视图 (`templates/admin/main_sections.html`)**：
  - **流派缓存归因调试器**：支持输入任意原始流派文本（如 `中國流行樂; Pop`），即时渲染首分段提取、权威英文名、规范中文名、最终归一化结果及 `已认证匹配` / `未认证` 状态徽标；
  - **权威流派数据表**：支持中英文即时搜索防抖、排序切换、新增流派弹窗、行内快速编辑、一键对账与分页；
- **前端交互脚本 (`static/admin/genres.js`)**：完整封装模块交互逻辑，并通过 `navigation.js` 统一调度路由。

---

## 3. 验证与质量保证

1. **单元测试通过**：
   - `internal/logic/genre/service_test.go` 100% 覆盖新增的过滤器与沙盒解析方法；
   - 执行 `go test -count=1 ./internal/logic/... ./internal/cache/...` 全部独立秒级通过。
2. **完整工程编译通过**：
   - 执行 `go build -o /dev/null .` 成功无告警。
3. **接口文档同步更新**：
   - 已同步维护 `api/api.md`。
