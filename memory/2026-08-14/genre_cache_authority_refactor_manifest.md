# 🎵 流派权威源收口至 GenreCache 与防污染重构特性清单

## 1. 概述与背景
在历史实现中，流派映射存在硬编码静态字典 `common.GenreMap` 与内存动态权威缓存 `GenreCache` 双轨并存的问题。这导致了以下缺陷：
1. **伪流派污染**：当流水遇到未知中文流派标签（如“当代歌手”、“歌曲作者”）时，系统自动生成类似 `cn-slug-4edff828` 的伪英文流派。
2. **对账无门槛自建记录**：服务启动执行 `RefreshDashboardStatsHeavy` -> `ReconcileGenrePlayCountsTx` 时，直接在 `genre` 物理表中 `tx.Create` 插入伪流派与中文流派脏数据。
3. **并发安全与脱节**：`/api/genres/map` 动态修改全局非线程安全 `GenreMap`，与 `GenreCache` 物理表缓存脱节。

本次重构确立以 `internal/cache/genre_cache.go`（从数据库物理表动态加载）为全局唯一权威源，彻底清理无用代码 `common.GenreMap` 与 `cn-slug` 生成机制，并关闭对账自动建表红线。

---

## 2. 核心架构变更与关键模块

### 2.1 彻底移除 `common.GenreMap`
- 清理了 `common/enum.go` 中的静态 `GenreMap` 字典。
- 清理了 `api/server.go` 中对 `common.GenreMap` 的动态并发写入代码。

### 2.2 强化 `GenreCache` 权威解析能力
- **`ResolveCanonicalGenreDetail`**: 在 `internal/cache/genre_cache.go` 中提供标准英文规范名、中文名及严格匹配状态解析。
- **动态 Resolver 注册**: 通过 `model.SetGenreCacheResolver` 将权威缓存注入底层 `model` 层，避免模块反向循环依赖。

### 2.3 `internal/model/genre.go` 规整与防脏收口
- **`ResolveGenreIdentity` & `ResolveStrictGenreIdentity`**:
  - 优先查权威 `GenreCache` 与 DB 已认证记录；
  - 彻底移除 `cn-slug-xxx` 伪流派生成逻辑；
  - 未认证标签严格标记为 `matched = false`，不捏造英文 Name。
- **`NormalizeGenre`**:
  - 规范中文流派统一进行 `ConversionSimplifiedFx` 繁转简入库。
- **`ReconcileGenrePlayCountsTx`**:
  - **历史脏数据清洗**：自动将物理表中包含 `cn-slug-`、中文 `name` 或多段拼接的脏记录 `play_count` 置 0；
  - **关闭自动建表**：未匹配流水项仅收集到 `GetUnmatchedGenres()` 供后台人工干预映射，**绝对不再自动执行 `tx.Create` 污染物理表**；
  - **仅更新权威记录**：仅对认证成功的合法标准流派计算并更新 `play_count`。

### 2.4 人工映射热刷新闭环
- 在 `/api/genres/map` 接口中，用户在后台完成流派映射落库后，即时调用 `cache.GetGenreCache().RefreshFromDB(ctx)`，实现内存权威缓存的热更新与即时生效。

---

## 3. 验证与测试
- 针对 `GenreCache`、`NormalizeGenre`、`ResolveStrictGenreIdentity` 与 `ReconcileGenrePlayCountsTx` 编写了完整测试用例。
- 验证未知中文流派（如“完全陌生的未归因流派”）在对账后：
  1. `genre` 物理表未增加任何新记录；
  2. 历史 `cn-slug-` 与中文 Name 脏流派 `play_count` 被置 0；
  3. 未匹配项正确出现在 `GetUnmatchedGenres()` 供人工维护。
- 全量单元测试 `go test -count=1 ./...` 100% 通过。
