# 🔬 SonicLens AI 音眸分析深度审计（P0 聚焦版）

> **审计范围**：[`agent_insight_track.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/agent_insight_track.go) + [`agent_insight_album.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/agent_insight_album.go) 的 Prompt 工程与模型参数
>
> **核心约束**：🔒 所有调整必须保证 `TrackAnalysisResult` / `AlbumAnalysisResult` 的 JSON Schema 与数据库格式完全一致，不可变更字段名、类型与必填约束。

---

## 1. 🔴 P0-1：System Prompt 质量是否为最佳实践

### 1.1 曲目分析 System Prompt 逐层审计

当前曲目 prompt 由 4 个 `Fmt` 片段拼接而成（[L21-L175](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/agent_insight_track.go#L21-L175)）：

| 片段 | 内容 | 行业最佳实践对标 |
|------|------|-----------------|
| `Fmt1` | 角色声明："多维度音乐分析专家" | ✅ 符合"角色扮演+任务分解"范式 |
| `Fmt2` | 四角色任务指令 + 格式规范 + 约束清单 | ⚠️ 存在多处可优化点（见下） |
| `Fmt3` | JSON Schema 文本 + 注解含义 | ⚠️ Schema 描述与 `GetTrackInsightSchema()` 存在微妙差异 |
| `Fmt4` | 简短过渡句 | ✅ 无问题 |

#### 1.1.1 ✅ 做得好的部分

1. **四角色设计**非常精妙——文学家→乐评人→文化史学家→综合分析师，形成清晰的认知链条
2. **`<original>/<translation>/<explain>` 标签体系**有充足的 few-shot 示例（中文分句、中文分段、外语分段）
3. **反馈机制** `FeedbackContext` 实现了"人在回路"迭代

#### 1.1.2 🔴 存在的严重问题

**问题 A：prompt 长度膨胀但信息密度低**

当前 `Fmt2` 约 165 行、约 4000 字符。行业最佳实践认为：
- System Prompt 超过 2000 token 后，模型对中后段指令的遵从度显著下降
- 当前 prompt 中"格式示例"重复了 4 遍（中文分句、中文分段、外语分段、格式要求），核心规则被稀释

> [!CAUTION]
> **"禁止 `\u003c` 转义"这条规则在 Fmt2 中出现了 3 次**（L30、L107、L159-L161），说明历史上反复出现此问题。但重复写规则不如换一种策略：在 Schema description 中直接约束，或在后处理中自动修复。反复在 prompt 中强调同一规则实际上是在浪费 token 预算。

**问题 B：Gemini 使用 `buildTrackInsightSystemPrompt()` 丢失了 Fmt3 的字段语义指导**

```go
// L237-238：Gemini 走这个——不含 Fmt3（Schema 文本+注解）
func buildTrackInsightSystemPrompt() string {
    return "系统提示：\n" + Fmt1 + Fmt2 + Fmt4 + "\n"
}
```

Gemini 虽然有原生 `ResponseJsonSchema` 强约束输出格式，但 Schema 的 `description` 字段只有一句话（如 `"综合分析师的整体评价"`），远不如 Fmt3 中的注解详细（如 `"综合分析师的整体评价（200-300字）"`）。**这导致 Gemini 缺少字数引导和字段用途的详细说明**。

**问题 C：约束清单位置不佳**

约束清单（L147-L164）放在了四角色任务指令之后、Schema 之前。行业最佳实践是：
- **关键约束放在 System Prompt 的最前面或最后面**（recency bias + primacy bias）
- 当前夹在中间，容易被模型"跳过"

**问题 D：`FeedbackContext` 放在 User Prompt 末尾而非 System Prompt**

```go
// L257-260：反馈被拼在 user prompt 尾部
if req.FeedbackContext != "" {
    feedbackSection := Fmt1 + req.FeedbackContext + Fmt2
    str = str + feedbackSection
}
```

对于"必须避免的错误"这类**强约束**，放在 user prompt 末尾的权重低于 system prompt。尤其是当输入数据（歌词）很长时，FeedbackContext 会被推到上下文窗口的更远处。

### 1.2 专辑分析 System Prompt 审计

专辑 prompt（[`agent_insight_album.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/agent_insight_album.go) L20-L51）相比曲目 prompt 要精炼得多：

| 维度 | 评价 | 说明 |
|------|------|------|
| 结构清晰度 | ⭐⭐⭐⭐⭐ | 分析目标 + 输出要求，两段式结构，无冗余 |
| 字数引导 | ⭐⭐⭐⭐ | 明确要求 `analysis_summary ≥ 220字`、`philosophical_reflection ≥ 500字` |
| 角色定义 | ⭐⭐⭐⭐ | "资深专辑研究者、文学评论家与音乐史分析师"一句到位 |
| 缺失处理 | ⭐⭐⭐⭐ | 明确要求"基于已分析曲目归纳" |
| Schema 传递 | ⚠️ | `buildAlbumInsightSystemPromptAll()` 直接序列化 `GetAlbumInsightSchema()` 拼入 prompt，但 Gemini 用的 `buildAlbumInsightSystemPrompt()` 同样缺少 Schema 文本 |

**专辑 prompt 是比曲目 prompt 更接近最佳实践的设计**，可以作为曲目 prompt 重构的参考模板。

### 1.3 System Prompt 最佳实践改进建议

> [!IMPORTANT]
> 以下所有改进**不涉及 Schema 变更**，仅优化 prompt 文本，输出 JSON 结构与数据库字段完全保持一致。

#### 建议 1：重构 prompt 为"三段式"结构

```
[角色定义 + 核心约束] → [任务指令（精简）] → [格式规范 + Schema]
```

将约束清单从中间提到最前面，紧跟角色定义。将重复的格式示例合并为一份精简版。

#### 建议 2：为 Gemini 补充 Schema 语义注解

在 `GetTrackInsightSchema()` 的 `description` 字段中补充字数要求和详细用途说明。这样即使 Gemini 不走 Fmt3，原生 Schema 的 description 也能提供足够的语义指导。**这不会改变 Schema 的 type/required/properties 结构**。

#### 建议 3：FeedbackContext 同时注入 System Prompt

将 FeedbackContext 作为 System Prompt 的"约束追加段"而非 User Prompt 的尾部追加：

```go
// 改进：feedback 注入 system prompt 约束段
systemPrompt := buildTrackInsightSystemPrompt()
if req.FeedbackContext != "" {
    systemPrompt += feedbackSection(req.FeedbackContext)
}
```

#### 建议 4：去重"禁止转义"规则

将 3 处重复的 `\u003c` 禁止规则合并为约束清单中的一条，并在后处理层增加自动修复兜底。

---

*（第二部分：模型参数分析 + 优先级整改 见 part2）*
