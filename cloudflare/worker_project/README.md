# Cloudflare Worker API Project

这是一个标准的 Cloudflare Worker 项目，提供 SonicLens 数据的 API 接口。

## 目录结构

- `src/index.js`: 包含所有 API 逻辑和简易路由。
- `wrangler.toml`: 配置文件。

## 开发

1. 修改 `wrangler.toml` 中的 `database_id` 为你真实的 D1 数据库 ID。
2. 运行本地开发服务器:
   ```bash
   npx wrangler dev --d1 DB=lastfm-scrobbler-db
   ```
   (如果数据库名为 `lastfm-scrobbler-db`)

## 部署

```bash
npx wrangler deploy
```

部署成功后，你将获得一个 Worker URL (例如 `https://lastfm-scrobbler-api.your-name.workers.dev`)。
你需要将这个 URL 配置到你的静态前端项目中。
