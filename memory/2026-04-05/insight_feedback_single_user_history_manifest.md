# 单用户音眸反馈历史与再生成闭环

## 背景

- SonicLens 当前是单用户自部署形态，音眸反馈不再使用“社区/投票”语义，而是服务于“我的反馈”“历史反馈”“待修正意见”与后续再生成纠偏。

## 本次变更

- 后端反馈写入继续允许同一条曲目/专辑 insight 多次追加点赞与点踩，保留历史事件流。
- `track_insight_feedbacks` 与 `album_insight_feedbacks` 补齐 `reason_codes`、`section_key`、`source_platform`，支持结构化问题标签与终端来源。
- 新增统一读取接口：
  - `GET /api/insights/:id/feedback-summary`
  - `GET /api/insights/:id/feedback-history`
- 音眸列表摘要补齐 `like_count`、`dislike_count`、`latest_feedback_score`，Bridge 列表页可直接渲染 `未评价 / 已认可 / 待修正`。
- Bridge 曲目详情、专辑详情、音眸列表详情统一新增“我的反馈 / 历史反馈”模块；点踩改为结构化问题面板，点踩历史默认聚焦最近负反馈。
- 再生成时不再只拼接最后一条 comment，而是把多条历史负反馈整理为高频问题标签、重点分区与最近备注后再注入大模型 `FeedbackContext`。

## 长期约束

- 单用户反馈语义禁止回退到“社区反馈”“认可率”“其他用户评论”。
- 列表页只看反馈摘要，不拉完整反馈历史；详情页负责展示最近历史。
- 历史负反馈是后续再生成的核心纠偏输入，不能被覆盖成单条当前状态。
