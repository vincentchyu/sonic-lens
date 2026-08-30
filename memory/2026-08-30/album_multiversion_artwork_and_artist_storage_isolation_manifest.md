# 专辑封面多版本隔离与艺术家存储空间解耦特性清单

- **特性分支/提交**: `feat(artwork): multi-version album artwork isolation and artist domain decoupling`
- **交付日期**: 2026-08-30
- **相关模块**: `core/artwork`, `core/objectstorage`, `internal/logic/artwork`, `internal/logic/artistprofile`,
  `internal/model`, `api/server.go`, `api/api.md`, `static/admin/`

---

## 1. 业务背景与问题根因

### 1.1 缺陷现象

1. **多版本同名专辑封面串图与覆盖**： 同歌手同名不同版本（如 `Brothers In Arms` 原版、`Remastered 1996` 重制版、
   `40th Anniversary` 40周年版）封面在对象存储、内存缓存与数据库回退层被强制折叠为同一物理 Key，导致无法展示各自版本的独特唱片封面。
2. **艺术家头像空间与专辑封面空间边界混杂**： 早期缺乏存储域物理路径隔离，部分查询回退到专辑封面，造成视觉与存储混淆。

### 1.2 根因分析

1. `core/artwork/seed.go` 的 `BuildAlbumArtworkSeed` 仅依赖 `(Artist, Album)` 二元组，未包含 `albumSubtitle`。
2. `/api/artwork/resolve` 解析接口与前端 `artwork-slot` 节点缺少 `albumSubtitle` 透传，无法进行精准版本封面分流。

---

## 2. 核心架构与修改实现

1. **三元组专辑封面 Seed 升级与向后兼容** (`core/artwork/seed.go`)：
    - 升级函数签名：`BuildAlbumArtworkSeed(albumArtist, artist, album, albumSubtitle string) string`。
    - 当 `albumSubtitle == ""` 时，生成的 Seed 为 `artist|album`（与历史数据 100% 保持一致）；
    - 当 `albumSubtitle != ""` 时，生成的 Seed 为 `artist|album|subtitle`，生成物理隔离的专属封面 Key。
2. **精准版本专辑查询与回退** (`internal/model/album.go`, `internal/logic/artwork/service.go`)：
    - `model.GetAlbumByArtistNameAndSubtitle` 支持多版本副标题精准查询；
    - `artworklogic.Service.Resolve` 与 `EnsureAlbumCover` 全链路支持 `AlbumSubtitle`。
3. **API 契约增强** (`api/server.go`, `api/api.md`)：
    - `GET /api/artwork/resolve` 支持 `albumSubtitle` / `album_subtitle` 参数并加入 Redis 缓存 Key。
4. **前端渲染透传与版本 Tag 闭环** (`static/admin/realtime.js`, `static/admin/pending-albums.js`,
   `static/admin/library.js`)：
    - `hydrateArtworkSlots` 与 `buildArtworkResolveCacheKey` 均透传 `albumSubtitle`；
    - 待归因卡片与工作项卡片展示独立版本 Tag 徽章。
5. **单元测试与验证** (`core/artwork/seed_test.go`, 全量 `go test ./...`)：
    - 覆盖三元组生成、空副标题兼容与全站集成单测，全部通过。

---

## 3. 验证结果

- 全量自动化测试秒级通过（100% PASS）。
- 数据库与前端实测同名多版本专辑封面与待归因数据严格隔离，彻底消除串图与覆盖。
