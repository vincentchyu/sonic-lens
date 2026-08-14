# 播放统计闭环统一重构与 Web 可视化对账特性清单

## 1. 特性背景与概述
在过去，SonicLens 的 Top 热门专辑、专辑基础列表、流派专辑及流派统计分别采用了独立计算与不同存储介质（如内存计算 vs 独立 Stat 表 vs 实时 Incr）。这不仅导致统计口径撕裂，也使 `album` 列表在按播放量排序时必须由 Golang 在内存中进行全量排序，同时给 Bridge 端本地 SQLite 的热度排序造成全 0 的缓存失效现象。

本特性完成了播放计数的**事实源归一 (SST)**、**Album 物理列 `play_count` 落地**、**Hot/Cold 增量与对账双轨驱动**，并**全量拒绝命令行黑盒维护**，在 Web 管理后台 (`/admin`) 中实现了可视化的统计对账与一致性纠偏控制面板。

---

## 2. 核心变更细节

### 2.1 数据库与 DAO 层变更 (`internal/model/`)
1. **`album.go`**：
   - 将 `Album.PlayCount` 改为持久化索引列 `gorm:"column:play_count;type:bigint;default:0;index:idx_album_play_count" json:"play_count"`。
   - 新增 `IncrementAlbumPlayCountTx(tx, albumID)` 原子递增函数。
   - 新增 `ReconcileAlbumPlayCountsTx` / `ReconcileAlbumPlayCounts` 对账校对函数。
   - 重构 `GetAlbumsByGenre` 等列表查询，将排序直接下推至 SQL：`ORDER BY play_count DESC, id DESC`。
2. **`track_play_record.go`**：
   - 在 `ProcessTrackPlayRecord` 归因到 `albumID` 事务逻辑中，原子调用 `IncrementAlbumPlayCountTx` 增加播放计数。
3. **`genre.go`**：
   - 新增 `ReconcileGenrePlayCountsTx` / `ReconcileGenrePlayCounts` 对账更新函数。
4. **`album_cleanup.go`**：
   - 在重复专辑合并与迁移完成后，自动触发 `ReconcileAlbumPlayCountsTx` 保证主专辑完整继承播放历史。
5. **`dashboard_stat.go`**：
   - 重构 `refreshDashboardStatsHeavyWithOptions`，在刷新重统计前自动全量对账。

### 2.2 API 与 Web 管理后台 (`api/`, `templates/admin/`, `static/admin/`)
1. **HTTP API**：
   - `POST /api/admin/stats/reconcile`：触发全量播放量一致性对账与修复。
   - `GET /api/admin/stats/reconcile/status`：获取当前对账与差异指标状态。
2. **Web 后台交互**：
   - 在 `templates/admin/main_sections.html` 插入“播放统计一致性与对账”卡片。
   - 编写 `static/admin/stats-reconcile.js` 处理一键对账与透明运行状态回显。
   - 在 `templates/admin/scripts.html` 引入前端入口。

---

## 3. 验证与结果
- **单元测试**：运行 `go test -v ./internal/model/... -run "TestIncrement|TestProcessTrackPlayRecord|TestAlbum"` 全部 PASS。
- **构建测试**：运行 `go build ./...` 100% 通过无错误。
- **验证结论**：成功实现对账逻辑，全面消除了黑盒命令行维护，解封了客户端 Bridge 本地 SQLite 的热度排序。
