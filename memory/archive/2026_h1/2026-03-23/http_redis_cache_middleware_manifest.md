# HTTP Redis Cache Middleware

## 日期
2026-03-23

## 特性摘要
- 在 `api/` 层新增了面向读接口的 Redis 响应缓存 middleware，支持按路由注册。
- middleware 默认 TTL 为 5 分钟，允许单路由自定义 TTL。
- 空结果会走 3 秒短 TTL 负缓存，避免把缺失数据长期占住 Redis。
- 缓存命中时会回写 `ETag` 和 `Cache-Control`，并支持 `If-None-Match` 的 304 语义。
- Redis 不可用时会自动降级为直通模式，不阻断接口可用性。

## 后端改动
- `api/cache_middleware.go`
  - 新增 HTTP 缓存 store 抽象和 Redis 实现。
  - 新增基于 Gin 的响应捕获器，用于在写回前计算 `ETag` 并落库。
  - 缓存 key 由方法、路径、查询串和 `Accept` 头组合生成，避免跨表示混用。
  - 仅对 `GET` / `HEAD` 生效，非缓存请求头 `no-cache` / `no-store` 会绕过缓存。
  - `200` 空响应、`404`、`204` 等空结果会使用短 TTL，防止缓存穿透场景长期占位。
- `api/server.go`
  - `GET /api/ai-models` 挂载默认 TTL 的 Redis 缓存 middleware。
  - `GET /api/artwork/resolve` 挂载 10 分钟 TTL 的 Redis 缓存 middleware。
- `api/cache_middleware_test.go`
  - 覆盖默认 TTL 写入、缓存命中、`ETag` 回放和 `304 Not Modified` 行为。

## 验证
- `go test ./api/...`
