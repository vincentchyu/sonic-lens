# Insight Feedback Reason Enum Alignment（2026-04-07）

## 背景

- Bridge 端已经定义 `InsightFeedbackReason`，并通过 `reason_codes` 上报结构化负反馈标签。
- 后端此前虽然支持 `reason_codes` 落库、汇总 `top_reason_codes` 并把高频问题注入 prompt，但仍以裸字符串处理，缺少统一枚举和白名单约束。

## 本次改动

1. 在 `common` 新增 `InsightFeedbackReason`：
   - 枚举值与 Bridge 端保持一致：`不准确 / 太空泛 / 不贴合歌曲/专辑 / 缺少关键信息 / 结构混乱 / 其他`。
   - 提供 `ParseInsightFeedbackReason`、`NormalizeInsightFeedbackReasons`、`AllInsightFeedbackReasons`。

2. 反馈归一规则收口：
   - `internal/logic/insight/service.go` 的 `normalizeReasonCodes` 改为复用 `common.NormalizeInsightFeedbackReasons`。
   - 非法 reason code 会被过滤；重复值会去重；合法值会按统一枚举顺序输出。

3. 闭环影响：
   - 反馈写入、历史返回、摘要 `top_reason_codes` 以及 track/album prompt 的 `FeedbackContext` 现在都基于同一套后端枚举。
   - 后端不再依赖前端任意传入字符串，避免“展示可见、聚合失真、prompt 污染”的漂移。

## 验证

- `go test ./common ./internal/logic/insight/... ./api/...`

## 约束

- 后续若 Bridge 端新增或调整反馈原因，必须先同步更新 `common.InsightFeedbackReason`，再允许客户端发布。
