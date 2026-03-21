# AI 模型平台与模型目录闭环特性清单

- **日期**: 2026-03-27
- **范围**: `common/`、`core/ai/`、`internal/logic/insight/`、`api/`、`templates/dashboard.html`、`soniclens-bridge/`

## 背景

原有 AI 模型选择链路把 `modelType` 同时当作“平台”和“模型”使用：

- `/api/ai-models` 只返回 provider 字符串数组；
- Web 和 Bridge 客户端把选中的字符串直接回传给后端；
- 实际模型仍依赖 `config` 里的默认值，切换模型需要改配置并重启；
- `llm_call_logs` 虽然记录了 `provider/model`，但前后端没有形成“平台 -> 模型 -> 实际调用 -> 日志存证”的闭环。

## 本次闭环

### 1. 后端平台/模型抽象拆分

- 新增 `common.AIModelPlatform`，统一 `openai/gemini/ollama/doubao/custom` 平台枚举。
- `core/ai` 新增平台工厂与模型目录抽象：
  - `GetConfiguredPlatforms()` 返回当前可用平台；
  - `GetModelsByPlatform(ctx, platform)` 返回平台模型目录；
  - `ResolveProviderSelection(...)` 统一处理 `provider + model` 与旧 `modelType` 兼容。
- `internal/logic/insight` 的 provider cache key 升级为 `platform::model`，避免同平台不同模型复用同一个 provider 实例。

### 2. 模型目录查询与 Redis 缓存

- 保留 `GET /api/ai-models`，语义改为“返回平台列表”。
- 新增 `GET /api/ai-models/:platform/models`，返回模型目录。
- 模型目录走 Redis 读穿缓存，key 带平台与配置指纹；Redis 不可用时自动降级为实时查询。
- 各平台目录来源：
  - `openai/custom`: OpenAI 兼容 `/v1/models`
  - `ollama`: SDK `List()`
  - `gemini`: SDK `Models.All`
  - `doubao`: ARK 管理侧 `ListEndpoints`
- Doubao 新增可选管理侧配置：`managementAccessKey`、`managementSecretKey`、`managementRegion`、`projectName`；缺失时回退只返回当前配置的运行时模型。

### 3. 调用入口与数据库存证

- `POST /api/track-insight`
- `POST /api/album-insight`
- `GET /api/track-insight-stream`

以上入口优先接收 `provider + model`，同时兼容旧 `modelType`。

- 若显式传入 `model` 且不在当前平台目录中，接口直接返回 `400`。
- `llm_call_logs.provider/model` 继续保存实际调用的平台和模型。
- `llm_call_logs.target_metadata` 新增：
  - `requested_provider`
  - `requested_model`
  - `effective_provider`
  - `effective_model`
- `track_insight.llm_provider`、`album_insight.llm_provider` 继续以 `provider:model` 形式落库。

### 4. 客户端闭环

- 老 `dashboard.html` 的 AI 选择弹窗升级为“平台 + 模型”两级选择。
- Bridge Track/Album 详情页同步升级为两级选择，并新增平台/模型本地偏好记忆。
- Bridge 兼容旧偏好值：若旧值命中平台枚举，则迁移为平台偏好并使用该平台默认模型。

## 风险与约束

- 本次未改 `web/dashboard-v2`。
- 未新增 MySQL 模型目录表，模型目录只放 Redis 与客户端本地缓存。
- Doubao 管理侧目录依赖新增配置；缺失配置时不会阻塞运行时调用，但目录能力会降级为单模型回退。
