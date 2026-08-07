# 专辑音眸后端闭环特性清单

## 日期

- 2026-03-26

## 背景

- `track_insight` 已经具备单曲音眸分析能力，但仓库缺少“按专辑聚合现有曲目 insight 并生成专辑级分析”的后端闭环。
- `core/ai/agent.go` 原先只有专辑分析 TODO，`album` 与 `track_album` 实际已存在，因此本次能力的关键是补齐专辑级 schema、模型、服务和接口，而不是重新设计专辑主表。

## 本次改动

### 1. 新增专辑级 AI 契约

- 在 `core/ai/provider.go` 新增：
  - `AlbumTrackContext`
  - `AlbumAnalysisRequest`
  - `AlbumAnalysisResult`
- 将 `LLMProvider` 扩展为同时支持 `AnalyzeTrack` 和 `AnalyzeAlbum`。
- 在 `core/ai/agent.go` 新增 `GetAlbumInsightSchema()` 与专辑级 prompt 生成函数，聚焦：
  - 时代意义
  - 文学解读
  - 作者动机
  - 哲学反思
  - 曲序与听感结构

### 2. 各模型 Provider 支持专辑分析

- `OpenAIProvider`
- `GeminiProvider`
- `DoubaoProvider`
- `OllamaProvider`
- `CustomProvider`

以上 provider 均已实现 `AnalyzeAlbum(ctx, req)`，并沿用现有 LLM 调用流水记录能力。

### 3. 新增 `album_insight` 数据模型

- 新增 `internal/model/album_insight.go`
- 表结构与 `track_insight` 风格保持一致，包含：
  - `album_id`
  - `artist`
  - `album`
  - `analysis_summary`
  - `analysis_by_section`
  - `background_info`
  - `era_context`
  - `llm_provider`
  - `metadata`
  - `last_used_at`
  - `is_disabled`
- `internal/model/init.go` 已将 `AlbumInsight` 纳入 AutoMigrate。

### 4. 专辑聚合规则收口到 `internal/logic/insight/service.go`

- 新增：
  - `GetAlbumInsightOnly(ctx, albumID)`
  - `GetOrCreateAlbumInsight(ctx, albumID, force, modelType)`
- 专辑分析输入构造规则：
  - 使用 `GetAlbumWithTracks` 获取专辑与 `track_album` 顺序
  - 按 `disc_number ASC, track_number ASC` 遍历曲目
  - 每首歌只选一条 `track_insight`
  - 选择策略为“总分高优先；同分时最新优先”
  - 专辑 prompt 默认跳过 `appreciate_analysis`，只汇总摘要、背景、时代语境和其他结构化 section，避免上下文过长
- 若整张专辑没有任何可用的曲目 insight，则直接返回错误，提示先生成曲目分析。

### 5. 新增 API

- `GET /api/album-insight?albumID=...`
  - 只读取已有专辑 insight
- `POST /api/album-insight`
  - 请求体：`{ "album_id": 123, "modelType": "gemini" }`
  - 强制生成最新专辑 insight

## 关键约束

- 专辑 insight 不是重新分析整张专辑歌词，而是基于现有曲目 insight 做二次归纳。
- 曲目选择顺序以 `track_album` 物理位置为准，不允许前端或上层逻辑自行重排。
- 每首歌必须先做“高分优先、同分最新优先”的去重，再进入专辑聚合。
- `metadata` 中应保留 `album_id`、`total_tracks`、`analyzed_tracks`、`selected_track_insight_ids`，便于后续追踪来源。

## 验证

- `go test ./core/ai ./internal/logic/insight`
- `go test ./internal/model -run '^$'`
- `go test ./api`

## 后续建议

- 前端专辑详情页可直接增加“音眸专辑分析”按钮，调用 `POST /api/album-insight`。
- 若后续需要后台管理或用户反馈，可继续为 `album_insight` 增加列表接口和反馈表，但本次先不扩散范围。

## 追加收口

- `llm_call_logs` 已补齐专辑/曲目二分：
  - `analysis_target_type` 区分 `track` 与 `album`
  - `target_key` 作为统一查询键
  - `target_metadata` 独立保存对象元数据，`track_info` 只保留兼容展示文本
- `/api/insights/all` 已按 `analysis_target_type` 拆成曲目/专辑两类列表，前端与 Bridge 都要保留双 tab 或等价分流，不允许再把两类解析混成一个展示面。
- 调用流水的 `request_json` 现在保存 provider 最终出站请求体：
  - OpenAI / 自定义 OpenAI 兼容接口保存完整 chat/completions payload
  - 豆包与 Ollama 保存实际 SDK 请求体
  - Gemini 保存等价的模型请求描述，包含 system prompt、user prompt 与 schema
- MySQL 老库已增加运行时补丁：
  - `internal/model/schema_llm_call_log.go`
  - `internal/model/init.go` 在初始化阶段自动执行补列与补索引
- 前端日志弹窗已同步展示：
  - 目标类型
  - 对象键
  - 对象元数据 JSON
  - 请求与响应原文
