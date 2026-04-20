# Album Insight Feedback Closure（2026-04-05）

## 背景

- 现有音眸体系已将曲目与专辑解析拆成两张表，但反馈链路仍然只覆盖 `track_insight_feedbacks`。
- 客户端需要一条完整的“专辑解析 -> 提交反馈 -> 下次专辑分析反哺上下文 -> 管理端查看反馈”的闭环。

## 本次改动

1. 数据库与 DAO：
   - 新增 `album_insight_feedbacks` 独立表，字段与曲目反馈表保持同构：`insight_id`、`score`、`comment`、`created_at`。
   - 新增 `internal/model/album_insight_feedback.go`，提供 `CreateAlbumInsightFeedback`、`GetAlbumInsightFeedbacks` 和 `GetNegativeAlbumFeedbacksByLookup`。
   - `internal/model/init.go` 的 SQLite / MySQL AutoMigrate 同步纳入 `AlbumInsightFeedback`。
   - 新增 `internal/model/sql/ddl/album_insight_feedbacks.sql`，并同步 `internal/model/sql/dml/init.sql` 注释。

2. 逻辑层闭环：
   - `insight.Service` 新增专辑反馈写入与读取方法，和曲目反馈分开处理。
   - `RecordAlbumFeedback` 会像曲目反馈一样累加 `like_count` / `dislike_count`。
   - `GetOrCreateAlbumInsight` 会读取历史差评并注入 `FeedbackContext`，让下次专辑分析真正受反馈驱动。

3. API 闭环：
   - 新增 `POST /api/album-insight/:id/feedback`。
   - `GET /api/insights/:id/feedbacks` 现在支持 `analysis_target_type=track|album`。
   - 新增 `GET /api/album-insights/:id/feedbacks` 作为专辑反馈的显式入口。
   - `api/api.md` 已补齐接口表和调用链说明。

4. 测试：
   - 补了 `album_insight_feedbacks` 的 model 层写入和查询测试。
   - 补了 API 层专辑反馈写入与反馈列表路由测试。

## 验证

- `go test ./internal/model -run 'TestCreateAlbumInsightFeedback|TestGetAlbumInsightFeedbacks'`
- `go test ./internal/logic/insight/...`
- `go test ./api -run 'TestRegisterInsightFeedbackRoutesAlbumFeedbackPassesThrough|TestRegisterInsightFeedbackRoutesAlbumFeedbackListUsesTargetType'`

## 说明

- `go test ./internal/model/...` 仍然存在若干与本次改动无关的既有失败，主要集中在 `album_cleanup`、`album_release_mb` 等旧测试用例。
