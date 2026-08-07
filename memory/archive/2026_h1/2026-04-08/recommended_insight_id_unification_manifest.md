# Recommended Insight ID Unification（2026-04-08）

## 背景

- 音眸曲目、专辑和历史版本在客户端、Web Dashboard 与后端之间长期依赖“第一条就是推荐”的隐式约定。
- 曲目与专辑的返回顺序并不总一致，且详情页、工作台与列表页分别有各自的默认选中逻辑，容易出现“展示顺序”和“推荐语义”漂移。

## 本次改动

1. 后端统一推荐语义：
   - `GET /api/track-insight`、`POST /api/track-insight`、`GET /api/album-insight`、`POST /api/album-insight` 与 `GET /api/insights/:id/history` 现在都显式返回 `recommended_insight_id`。
   - 曲目与专辑解析结果在服务层统一按“总分降序，分数相同按创建时间降序”排序，再把第一条作为推荐版本。
   - 历史版本列表在返回前也按同一规则重排，避免不同入口看到不同的默认版本。

2. 数据层补齐：
   - 新增 `internal/model/album_insight_feedback.go` 中的 `GetAlbumInsightsTotalScores`，用于专辑解析推荐排序。
   - 曲目侧继续复用 `GetInsightsTotalScores`，专辑侧则用独立的反馈汇总查询。

3. Web 与 Bridge 对齐：
   - `templates/dashboard.html` 不再默认把第一条当成推荐，而是优先使用 `recommended_insight_id`。
   - Bridge 的曲目/专辑详情 ViewModel 也改成按 `recommended_insight_id` 定位默认选中版本。
   - 详情页分享与当前展示版本改为跟随当前选中项，而不是强制读取数组首项。

4. 文档同步：
   - `api/api.md` 补齐了新增响应字段。
   - 本清单同步记录推荐版本闭环的统一语义，供后续实现者参考。

## 验证

- `go test ./api ./internal/logic/insight`
- 由于仓库里 `internal/model` 存在与本次改动无关的既有失败，未把全量 model 测试作为通过标准。

## 说明

- 这次改动的核心不是“再排一次序”，而是把“推荐版本”从隐式顺序改成显式字段，后续所有端都应优先消费 `recommended_insight_id`。
