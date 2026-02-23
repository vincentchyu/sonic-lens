# Cloudflare D1 数据同步与静态页面展示特性清单

## 背景与目标

为了实现无服务器环境下的音乐数据查询和展示，本项目引入了 Cloudflare D1 作为云端数据存储，并通过 Cloudflare Workers/Pages
提供 API 和静态页面服务。

## 详细变更

### 1. Cloudflare D1 数据同步 (Go)

- **依赖库**: 引入 `github.com/peterheb/cfd1` 作为 D1 驱动。
- **配置**: 在 `config.yaml` 中新增 `cloudflare` 配置段，包含 `account_id`, `api_token`, `d1_database_id`。
- **同步逻辑**:
    - `internal/sync/d1_sync.go`: 实现了 `D1Client`，支持全量和基于 `updated_at` 的增量同步。
    - 使用批量插入策略优化 D1 写入性能。
    - **Scheduler**: `internal/sync/d1_scheduler.go` 实现了定时任务，每日自动触发同步。

### 2. Cloudflare Workers API

- **项目结构**: `cloudflare/worker_project` (独立 Worker) 和 `cloudflare/web_project/functions` (Pages Functions)。
- **API 端点**:
    - `/api/dashboard/stats`: 综合统计
    - `/api/dashboard/trend`: 播放趋势
    - `/api/dashboard/play-counts-by-source`: 来源统计
    - `/api/dashboard/top-artists/[type]`: 热门艺术家
    - `/api/dashboard/top-albums`: 热门专辑
    - `/api/dashboard/top-genres`: 热门流派
    - `/api/recent-plays`: 最近播放
    - `/api/track`: 单曲详情
    - `/api/track-play-counts`: 排行榜
    - `/api/unscrobbled-records`: 未上报记录查看
- **技术细节**: 使用原生 Worker 语法，手动处理路由和 CORS。

### 3. 静态 Web 前端

- **基础**: 基于原有的 `dashboard.html` 模板迁移为纯静态 `index.html`。
- **适配**:
    - 移除所有 Go Template 语法。
    - 使用 `fetch` 调用后端 API 获取数据。
    - 支持配置 `API_BASE_URL`以适应独立 Worker 部署或集成 Pages 部署。
    - 移除/禁用 WebSocket 相关代码 (静态环境不支持)。

## D1 数据库结构细节

数据库 Schema 定义在 `cloudflare/d1_schema.sql`。

- **tracks (曲目统计)**
    - `id`: 主键
    - `artist`, `album`, `track`: 核心元数据（唯一索引）
    - `play_count`: 累计播放次数
    - `source`: 来源 (如 Apple Music)
    - `is_apple_music_fav`, `is_last_fm_fav`: 收藏状态
    - `created_at`, `updated_at`: 时间戳

- **track_play_records (播放历史)**
    - `id`: 主键
    - `artist`, `album`, `track`: 元数据
    - `play_time`: 播放时间（倒序索引）
    - `scrobbled`: 是否已上报 (1/0)
    - `source`: 来源

- **genres (流派统计)**
    - `name`: 流派名称（英文）
    - `name_zh`: 中文名称
    - `play_count`: 该流派总播放次数

- **sync_metadata (同步元数据)**
    - `table_name`: 表名
    - `last_sync_time`: 上次成功同步时间（用于增量同步）

## 部署指引

此特性包含三个主要部分的部署，请按顺序执行：

### 0. 准备 D1 数据库 (Database Setup)

确保可以在本地使用 `wrangler` 命令行工具。

1. **登录**: `npx wrangler login`
2. **创建数据库** (如果尚未创建):
   ```bash
   npx wrangler d1 create lastfm-scrobbler-db
   ```
   *记下返回的 database_id。*
3. **初始化表结构**:
   ```bash
   # 推送到云端 D1 数据库 (生产环境)
   npx wrangler d1 execute lastfm-scrobbler-db --file=cloudflare/d1_schema.sql --remote
   
   # 如果仅在本地测试 (开发环境)
   npx wrangler d1 execute lastfm-scrobbler-db --file=cloudflare/d1_schema.sql
   ```

### 1. 部署 API (Worker)

位于 `cloudflare/worker_project`。

1. **环境准备**: 安装 `wrangler` (`npm install -g wrangler`)。
2. **配置**: 修改 `wrangler.toml` 中的 `database_id` 为你的 D1 数据库 ID。
3. **部署**: 运行 `npx wrangler deploy`。
4. **记录**: 复制部署后生成的 Worker URL (例如 `https://your-api.workers.dev`)。

### 2. 部署前端 (Static Web)

位于 `cloudflare/web_project/public`。

1. **配置**: 修改 `index.html`，设置 `const API_BASE_URL` 为上一步获取的 Worker URL。
2. **部署**:
    - 可部署到 **Cloudflare Pages**, **GitHub Pages**, **Vercel** 或任何静态托管服务。
    - 如果使用 Cloudflare Pages，只需将 `public` 目录作为 Build output directory，无需开启 Functions。

### 3. 配置数据同步 (Local Go Service)

1. **配置**: 在本地 `config.yaml` 中填入 Cloudflare API Token 和 D1 Database ID。
2. **运行**: 重启本地 Go 服务，它将根据定时任务自动同步数据到 D1。

## 影响范围

- 新增 `internal/sync` 模块。
- 新增 `cloudflare/` 目录及其子项目。
- 修改 `main.go` 启动流程。
- 修改 `config/config.go` 配置结构。

## 验证

- 本地全流程同步测试通过。
- Worker API 响应测试通过。
- 静态页面数据加载测试通过。
