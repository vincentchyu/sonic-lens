# 热门流派关联专辑 0 匹配与 URL 编码穿透/SQLite 字段缺失修复特性清单

- **日期**: 2026-08-14
- **作者**: Antigravity AI Agent
- **关联需求**: 修复点击热门流派（如 `#14 成人另类 ADULT ALTERNATIVE`）关联收录专辑为 0、显示“资料库中暂未检测到标记为‘成人另类’的物理专辑”的缺陷。

---

## 1. 缺陷根本原因复盘 (Root Cause Analysis)

1. **客户端网络层路径二次编码 (Double-Escaping Bug)**:
   - `soniclens-bridge/SoniclensCore/Networking/APIClient.swift` 中的 `buildURL` 与 `post` 原实现使用 `baseURL.appendingPathComponent(path)`。由于 `path` 是完整路径字符串且包含已转义的 `%20`（如 `/api/genres/Adult%20Alternative/albums`），`appendingPathComponent` 将整个路径当成单一片段，把 `%` 再次转义为 `%2520`。
   - 最终发出的请求为 `GET /api/genres/Adult%2520Alternative/albums`。
2. **服务端未防御二次转义直接渲染入 SQL (Server-Side Decoding & SQL Bug)**:
   - `api/server.go` 的 `/api/genres/:name/albums` 路由只进行一次标准解码，取到的参数为字面量 `"Adult%20Alternative"`。
   - 由于未做 URL 反转义清洗，后端将字面量 `'Adult%20Alternative'` 直接传入 DAO 作为参数拼入 SQL：`WHERE (genre = 'Adult%20Alternative' ...)`。
   - 而数据库中真实存储的是带真实空格的 `'Adult Alternative'`，导致匹配结果始终为 0 张。
3. **客户端本地 SQLite 索引遗漏持久化 `genre` 字段 (Fallback Data Missing)**:
   - `LibraryIndexStore.swift` 中 `upsertAlbums` 的 `INSERT INTO album_index` 语句只有 12 个字段，遗漏了 `genre` 字段的写入与绑定，导致本地 SQLite 中 `album_index.genre` 始终为 NULL。
   - 当 API 请求失败回退到本地 SQLite 查询时，同样无法匹配到任何物理专辑。
4. **单测覆盖盲区 (Unit Test Blindspot)**:
   - 之前的测试直接调用 DAO 并传入干净的 `"Rock"` / `"Rock Musical"` 字符串，未在 API 接口层模拟 HTTP 真实路径参数（含空格、`%20`、`%2520`）与客户端网络请求。

---

## 2. 核心改动明细 (Changelog)

### 2.1 客户端网络层修复
- **[`APIClient.swift`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensCore/Networking/APIClient.swift)**:
  - 弃用 `baseURL.appendingPathComponent(path)`，改用 `URL(string: normalizedPath, relativeTo: baseURL)?.absoluteURL` 安全构建 URL，彻底杜绝路径二次编码与斜杠转义问题。

### 2.2 客户端本地 SQLite 索引持久化修复
- **[`LibraryIndexStore.swift`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensCore/Store/LibraryIndexStore.swift)**:
  - 在 `upsertAlbums` 中补齐 `genre` 字段的 SQL 列声明、`ON CONFLICT DO UPDATE SET genre = excluded.genre` 以及 statement 参数绑定。

### 2.3 后端路由与模型层防御性解码
- **[`api/server.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/api/server.go)**:
  - 在 `/api/genres/:name/albums` 与 `/api/albums?genre=...` 中增加对流派入参的 `url.PathUnescape` / `url.QueryUnescape` 以及二次转义防御清洗。
- **[`internal/model/genre.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/model/genre.go)**:
  - 导出 `NormalizeGenreQueryToken` 通用清洗函数，在 `GetGenreByName` 中防范非法编码。
- **[`internal/model/album.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/model/album.go)**:
  - 在 `GetAlbumsByGenre` 与 `GetAlbumsByGenreCount` 中使用 `NormalizeGenreQueryToken` 清洗入参，确保 SQL 检索永远接收真实字符。

### 2.4 补齐严密单元测试
- **[`api/server_test.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/api/server_test.go)**:
  - 新增 `TestGenreAlbumsRouteUnescaping`，覆盖标准 URL 编码（`Adult%20Alternative`）和二次编码（`Adult%2520Alternative`）的 HTTP 路由测试。
- **[`internal/model/play_reconciliation_test.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/model/play_reconciliation_test.go)**:
  - 在 `TestGetAlbumsByGenreExactMatching` 中补齐单测，断言 `Adult%20Alternative` 与 `Adult%2520Alternative` 在 DAO 层的正确命中。

---

## 3. 验证结果 (Verification)

- **Go 单元测试**: `go test -count=1 ./...` 全量独立秒级 PASS（100% 通过）。
- **客户端构建**: `SoniclensBridgeMac` 与 `SoniclensBridgePhone` 编译 100% 成功。
