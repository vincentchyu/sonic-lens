# SigNoz OTLP 可观测性闭环

- **日期**: 2026-03-28
- **范围**: `core/telemetry`、`api/server.go`、`core/db/`、`core/redis/`、`internal/model/init.go`、`internal/sync/d1_sync.go`、`core/objectstorage/`、`core/musicbrainz/`、`core/ai/`、`core/lyrics/`、`config/`

## 背景

项目原先虽然已经接入了部分 OpenTelemetry API：

1. HTTP 入站有 `otelgin`
2. Redis 已引入 `redisotel`
3. GORM logger / Redis hook 里存在手写 span

但核心问题是：

1. `core/telemetry/initTelemetry` 仍然只把 trace 打到 stdout，没有接到 SigNoz Collector。
2. 全局没有 `MeterProvider`，导致 Redis 指标、HTTP 指标、数据库连接池指标无法稳定上报。
3. GORM / Redis 混用了手写 span 与标准 instrumentation，容易出现重复埋点与语义漂移。
4. 自建 `http.Client` 的外呼链路缺少统一 trace 传播。

## 本次收敛

### 1. Telemetry 底座

1. `core/telemetry/telemetry.go` 改为优先使用 OTLP gRPC exporter：
   - trace exporter: `otlptracegrpc`
   - metric exporter: `otlpmetricgrpc`
2. 当配置或环境变量未提供 OTLP endpoint 时，回退到 stdout trace exporter，避免单测和本地无 collector 环境直接失败。
3. 新增全局 `MeterProvider`，并暴露：
   - `GetTracerProvider()`
   - `GetMeterProvider()`
   - `HasMeterProvider()`
4. `resource` 统一补齐：
   - `service.name`
   - `service.version`
   - `deployment.environment.name`
   - `process / host / os / container / telemetry.sdk`
5. 默认采样策略改为 `ParentBased(TraceIDRatioBased(...))`。
6. 启用 Go runtime metrics（可配置开关）。
7. `Shutdown` 统一释放 metric callback registration、meter provider、tracer provider。
8. Telemetry 初始化成功后会打印一条启动自检日志，明确当前 `trace_exporter`、`metric_exporter`、OTLP endpoint、sampler、runtime/db stats metrics 开关，便于快速排查 SigNoz 未收数场景。

### 2. HTTP 入站与外呼

1. `api/server.go` 的 `otelgin.Middleware` 显式绑定当前 tracer provider / meter provider / propagator。
2. 新增 `core/telemetry/http.go`，统一包装出站 `http.Client` 的 transport。
3. 已接入外呼 trace 的调用方包括：
   - `core/ai/openai.go`
   - `core/ai/custom.go`
   - `core/ai/gemini.go`
   - `core/ai/ollama.go`
   - `core/ai/openai_compatible.go`
   - `core/lyrics/lrcapi.go`
   - `core/lyrics/musixmatch.go`

### 3. Redis

1. 保留 `redisotel` 作为标准 tracing + metrics instrumentation。
2. `core/redis/redis.go` 显式把全局 tracer/meter provider 传给 `redisotel`。
3. `core/redis/hook.go` 去掉手写 span，只保留日志，避免和 `redisotel` 产生重复 span。

### 4. Database

1. GORM 查询 trace 迁移到 `gorm.io/plugin/opentelemetry/tracing`。
2. `core/db/db.go` 的 custom logger 去掉手写 SQL span，只保留慢 SQL / 错误 SQL 日志。
3. `internal/model/init.go` 在 SQLite / MySQL 初始化后统一启用 GORM tracing plugin。
4. `core/db/instrumentation.go` 使用 `otelsql.RegisterDBStatsMetrics` 注册 `database/sql` 连接池指标：
   - open / idle / inuse
   - wait count / wait duration
   - max lifetime / max idle close 等
5. `internal/sync/d1_sync.go` 改为使用 `otelsql.Open(...)` 打开 Cloudflare D1 driver，并复用同一套 tracer/meter provider 与连接池指标注册能力，避免 D1 同步链路成为 SQL trace 盲区。

### 5. 对象存储

1. `core/objectstorage/s3_provider.go` 通过 AWS SDK v2 / smithy 原生 OTel adapter 接入 tracing + metrics：
   - `smithyoteltracing.Adapt(...)`
   - `smithyotelmetrics.Adapt(...)`
2. `HeadBucket`、`CreateBucket`、`HeadObject`、`PutObject`、`DeleteObject(s)` 等对象存储调用现在会自然出现在 SigNoz 链路中。

### 6. MusicBrainz

1. `core/musicbrainz/musicbrainz.go` 为当前实际使用的 `SearchReleases`、`LookupRelease` 增加统一包装层，补齐客户端 span。
2. 由于 `musicbrainzws2` 当前未暴露自定义 `http.Client` / transport 注入点，因此没有采用反射或 fork 依赖的方式强行接底层 HTTP instrumentation，而是在项目封装层显式创建客户端 span，保证 trace 连续性且不引入额外维护成本。

## 约束

1. 后续不要再在 GORM logger、Redis hook 里手写 client span。
2. Redis 标准埋点统一走 `redisotel`。
3. GORM 标准埋点统一走 `gorm opentelemetry tracing plugin`。
4. D1 这类直接使用 `database/sql` 的链路，优先走 `otelsql.Open(...)`，并补齐 `RegisterDBStatsMetrics(...)`。
5. AWS S3 / MinIO / R2 这类基于 AWS SDK v2 的对象存储链路，优先走 smithy 原生 OTel adapter，不要再额外叠加手写 span。
6. 需要自建 `http.Client` 时，优先走 `core/telemetry.WrapHTTPClient(...)`。
7. 对于未暴露底层 transport 注入点的第三方 SDK，优先在 `core/*` 封装层补稳定的客户端 span，不要用反射去修改私有字段。
8. 需要 Collector 上报时，优先配置 OTLP gRPC endpoint（SigNoz 默认 `4317`）。

## 配置

`config/config.yaml.example` 新增 telemetry 示例：

- `endpoint`
- `insecure`
- `sampler`
- `metricIntervalSeconds`
- `runtimeMetricsEnabled`
- `dbStatsMetricsEnabled`
- `environment`

## 验证

1. `go test ./core/telemetry/... ./core/log ./core/ai ./core/lyrics ./core/db ./core/redis ./api ./internal/model -run '^$|TestGo|TestTraceLogging|TestCustom|TestOpenAI|TestProvider'` 通过。
2. `go test ./...` 仍然失败，但失败仍集中在既有 `internal/model` 用例：
   - `TestCleanupDuplicateAlbumsMergesRelationsIntoCanonicalAlbum`
   - `TestCleanupDuplicateAlbumsContinueOnErrorSkipsConflictedGroup`
   - `TestLinkAlbumToMBIDTx`
   - `TestLinkAlbumToMBIDUsesContextDB`

## 结果

SonicLens 现在已经具备面向 SigNoz 的基础可观测性闭环：

1. HTTP 入站 trace
2. HTTP 外呼 trace
3. Redis trace + metrics
4. GORM/SQL trace
5. database/sql 连接池 metrics
6. D1 同步 SQL trace + metrics
7. S3/兼容对象存储 trace + metrics
8. MusicBrainz 客户端 span
9. Go runtime metrics

剩余未统一覆盖的外呼链路，可继续按 `WrapHTTPClient` 模式逐步收敛。
