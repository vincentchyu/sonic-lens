# 🔬 SonicLens AI 音眸分析深度审计（P0 聚焦版）— 第二部分

> 🔒 **核心约束**：所有调整保证 `TrackAnalysisResult` / `AlbumAnalysisResult` 的 JSON Schema 与数据库格式完全一致。`GetTrackInsightSchema()` 和 `GetAlbumInsightSchema()` 的 `type`、`required`、`properties`、`additionalProperties` 结构不可变更。

---

## 2. 🔴 P0-2：模型参数合理性深度分析

### 2.1 当前各 Provider 参数全景

| 参数 | OpenAI | Gemini | Doubao | Ollama | Custom |
|------|--------|--------|--------|--------|--------|
| **temperature** | ❌ 未设置（默认1.0） | ❌ 未设置 | ✅ `0.1` | ❌ 未设置 | ❌ 未设置 |
| **topP** | ❌ 未设置 | ❌ 未设置 | ✅ `1.0` | ❌ 未设置 | ❌ 未设置 |
| **topK** | — | ❌ 未设置 | ✅ `0`（关闭） | ❌ 未设置 | — |
| **repetition_penalty** | — | — | ✅ `1.0` | ❌ 未设置 | — |
| **thinking** | — | ✅ `ThinkingLevelMedium` | ✅ `ThinkingTypeEnabled` | ✅ `Think: "medium"` | — |
| **response_format** | ❌ 未使用 | ✅ **原生 JSON Schema** | ❌ 注释掉了 | ❌ 未使用 | ❌ 未使用 |
| **流式 thinking** | — | ❌ 无 config | ❌ `ThinkingTypeDisabled` | ✅ `"medium"` | — |

### 2.2 逐 Provider 参数审计

#### 2.2.1 OpenAI — 参数缺失最严重

**源码位置**：[`openai.go:269-281`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/openai.go#L269-L281)

```go
payload := openAIChatRequest{
    Model:    p.model,
    Messages: []openAIChatMessage{...},
    // ← 没有 temperature、没有 response_format
}
```

**问题分析**：
- OpenAI GPT 系列默认 `temperature=1.0`，这在 JSON 结构化输出场景下**极不稳健**
- 未使用 `response_format: {"type": "json_object"}` 或 Structured Outputs
- 这是 5 个 Provider 中**唯一没有任何采样约束**的实现

**推荐参数**（JSON 结构化输出场景）：

| 参数 | 推荐值 | 理由 |
|------|--------|------|
| `temperature` | `0.3` | JSON 稳定性与分析创意的平衡点；纯 JSON 可用 `0`，但音乐分析需要适度创意 |
| `response_format` | `{"type": "json_object"}` | GPT-4o 原生支持，强约束输出为合法 JSON |
| `top_p` | `0.95` | 配合低 temperature 使用，避免过度截断 |

> [!IMPORTANT]
> OpenAI 的 `response_format: json_object` **不会改变输出的字段结构**，只保证输出是合法 JSON。字段结构仍由 System Prompt 中的 Schema 控制。因此使用它**不影响与数据库的 Schema 握手**。

#### 2.2.2 Gemini — 架构最先进但参数可补充

**源码位置**：[`gemini.go:220-234`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/gemini.go#L220-L234)

```go
&genai.GenerateContentConfig{
    ThinkingConfig: &genai.ThinkingConfig{
        IncludeThoughts: true,
        ThinkingBudget:  nil,          // ← 未限制思考 token 预算
        ThinkingLevel:   genai.ThinkingLevelMedium,
    },
    SystemInstruction:  genai.NewContentFromText(insightSystemPrompt, genai.RoleUser),
    ResponseMIMEType:   "application/json",
    ResponseJsonSchema: GetTrackInsightSchema(),   // ← ✅ 唯一使用原生 Schema 的
}
```

**评价**：
- ✅ **唯一使用原生 Structured Output 的 Provider**——输出 JSON 100% 符合 Schema，这是最佳实践
- ✅ Thinking 配置合理（Medium 级别平衡质量和速度）
- ⚠️ 未设置 `Temperature`，Gemini 默认值依模型而异（通常 1.0）
- ⚠️ `ThinkingBudget: nil` 意味着不限制思考 token，可能导致成本失控

**推荐补充**：

| 参数 | 推荐值 | 理由 |
|------|--------|------|
| `Temperature` | `0.3` | 同理：JSON 稳定 + 分析创意平衡 |
| `ThinkingBudget` | `8192` | 限制思考 token 上限，避免成本失控 |

#### 2.2.3 Doubao — 当前参数设置最佳的 Provider

**源码位置**：[`doubao.go:112-167`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/doubao.go#L112-L167)

```go
temperature:       0.1,    // JSON 稳定
topP:              1.0,    // 不截断候选
topK:              0,      // 关闭 topK 避免 JSON 符号被挤掉
repetitionPenalty: 1.0,    // 不惩罚重复，避免 JSON key 缺失
```

**评价**：
- ✅ **每个参数都有详细的中文注释说明选择理由**——这是工程最佳实践
- ✅ 参数组合针对 JSON 输出场景做了专门优化
- ⚠️ `temperature: 0.1` 略低，对于音乐分析的深度解读可能导致输出偏保守/模式化
- ⚠️ `ResponseFormat`（Structured Output）被注释掉了（L415-L423），理由不明

**一个微调建议**：

| 参数 | 当前值 | 建议值 | 理由 |
|------|--------|--------|------|
| `temperature` | `0.1` | `0.2~0.3` | 当前值过于保守，音乐评论需要适度的表达多样性 |
| `ResponseFormat` | 注释掉 | 取消注释启用 | 如果豆包 SDK 支持，应该启用以获得与 Gemini 同等的 Schema 强约束 |

#### 2.2.4 Ollama — 本地模型的合理妥协

**源码位置**：[`ollama.go:191-197`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/ollama.go#L191-L197)

```go
ollamaReq := &api.GenerateRequest{
    Model:  p.model,
    System: prompt,             // ← 用 merged prompt（system+user 合体）
    Stream: new(bool),
    Think:  &api.ThinkValue{Value: "medium"},
    Prompt: "不做 appreciate_analysis",  // ← 显式降级
}
```

**评价**：
- ✅ `Think: "medium"` 合理利用本地模型的推理能力
- ✅ 显式放弃 `appreciate_analysis` 是对本地模型能力的务实判断
- ⚠️ 完全没设置 temperature/topP 等采样参数
- ⚠️ 使用 `System` 字段传递 merged prompt 而非分开的 system/user，不符合 Ollama API 最佳实践

**推荐补充**：

| 参数 | 推荐值 | 理由 |
|------|--------|------|
| `Options.Temperature` | `0.1~0.2` | 本地模型 JSON 稳定性更差，需要更低的 temperature |
| `Format` | `"json"` | Ollama 支持 `format: "json"` 强约束输出 |

#### 2.2.5 Custom — 完全无参数

**源码位置**：[`custom.go:234-241`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/core/ai/custom.go#L234-L241)

```go
payload := customChatRequest{
    Model:    p.model,
    Messages: []openAIChatMessage{...},
    Stream:   stream,
    // ← 无 temperature、无 top_p、无 response_format
}
```

**评价**：Custom Provider 定位是通用兼容层，不设参数有合理性（用户可能连接各种后端）。但至少应提供 `temperature` 的可配置入口。

### 2.3 模型参数最佳实践总结

对于**"音乐分析 + JSON 结构化输出"**这个特定场景，行业最佳实践参数组合为：

```
┌─────────────────────────────────────────────────────┐
│  JSON 结构化音乐分析 — 推荐参数基线                    │
├─────────────────┬───────────┬────────────────────────┤
│ temperature     │ 0.2~0.3   │ 稳定 JSON + 适度创意    │
│ top_p           │ 0.95~1.0  │ 不截断候选              │
│ top_k           │ 0 或不设   │ 避免 JSON 符号被挤掉    │
│ rep_penalty     │ 1.0       │ JSON 不能惩罚重复 key   │
│ response_format │ 启用       │ 每个平台用原生方式      │
│ thinking        │ medium    │ 分析质量与成本平衡       │
└─────────────────┴───────────┴────────────────────────┘
```

---

## 3. 整改优先级（重新排序）

> [!IMPORTANT]
> 🔒 **Schema 兼容红线**：以下所有整改项均不涉及 `TrackAnalysisResult` / `AlbumAnalysisResult` 的字段名、字段类型、required 约束的变更。`GetTrackInsightSchema()` 和 `GetAlbumInsightSchema()` 的结构保持不变。允许的变更范围仅限于：Schema `description` 字段的文本优化、Prompt 文本内容、模型采样参数、Provider 内部实现。

### P0（必须立即修复）

| # | 整改项 | 影响 | Schema 影响 |
|---|--------|------|------------|
| **P0-1** | **重构曲目 System Prompt 为"三段式"结构**：约束前置、任务精简、示例去重 | 所有 Provider 的曲目分析质量 | 🔒 无。仅改 prompt 文本 |
| **P0-2** | **为 Gemini/OpenAI 补充 temperature + response_format** | JSON 输出稳定性 | 🔒 无。仅改采样参数 |
| **P0-3** | **OpenAI AnalyzeTrack 迁移到共享 prompt 体系** | OpenAI 曲目分析质量 | 🔒 无。统一使用已有 Schema |
| **P0-4** | **Gemini Schema description 补充字数要求和详细用途说明** | Gemini 分析深度和字数达标率 | 🔒 仅改 description 文本，不改结构 |
| **P0-5** | **Gemini 流式补充 GenerateContentConfig（system prompt + thinking + json schema）** | Gemini 流式分析质量 | 🔒 无。与同步使用相同 Schema |

### P1（下一迭代修复）

| # | 整改项 | 影响 | Schema 影响 |
|---|--------|------|------------|
| **P1-1** | **FeedbackContext 从 User Prompt 末尾移到 System Prompt 约束段** | 反馈遵从度 | 🔒 无 |
| **P1-2** | **Doubao 取消注释 ResponseFormat，启用 Structured Output** | 豆包 JSON 稳定性 | 🔒 无。使用已有 `GetTrackInsightSchema()` |
| **P1-3** | **Doubao temperature 从 0.1 调至 0.2~0.3** | 分析文本多样性 | 🔒 无 |
| **P1-4** | **Doubao 流式 Thinking 从 Disabled 改为 Enabled** | 流式分析质量 | 🔒 无 |
| **P1-5** | **Ollama 补充 `Format: "json"` 和 `Options.Temperature`** | 本地模型 JSON 稳定性 | 🔒 无 |
| **P1-6** | **提取公共 `parseTrackResult()` / `parseAlbumResult()`** | 消除 5 个 Provider 的重复代码 | 🔒 无。纯重构 |

### P2（优化项）

| # | 整改项 | Schema 影响 |
|---|--------|------------|
| **P2-1** | `repairPrematureTopLevelObjectClosure` 提升到公共层 | 🔒 无 |
| **P2-2** | `\\n` → `\n` 修复提取到公共后处理函数 | 🔒 无 |
| **P2-3** | Stream 接口返回 `<-chan StreamChunk`（含 Err） | 🔒 无。内部传输类型，不影响最终 Result |
| **P2-4** | Doubao `arkruntime.Client` OTel 包装 | 🔒 无 |
| **P2-5** | `buildTrackInsightSystemPrompt` / `All` 重命名为 `WithSchema` / `WithoutSchema` | 🔒 无 |
| **P2-6** | Gemini/Custom 流式补充 CallLog 记录 | 🔒 无 |

---

## 4. Schema 兼容保证清单

以下是当前与数据库强握手的 Schema 结构，**任何整改都不得变更**：

### TrackAnalysisResult（曲目）
```json
{
  "lyrics_translation": "string (required)",
  "analysis_summary": "string (required)",
  "analysis_by_section": {
    "appreciate_analysis": "string (required)",
    "literary_analysis": "string",
    "musical_analysis": "string",
    "cultural_context": "string",
    "translation_notes": "string",
    "additionalProperties": "string"
  },
  "background_info": "string",
  "era_context": "string",
  "metadata": "object (any)"
}
```

### AlbumAnalysisResult（专辑）
```json
{
  "analysis_summary": "string (required)",
  "analysis_by_section": {
    "album_positioning": "string",
    "theme_and_narrative": "string",
    "literary_analysis": "string",
    "musical_analysis": "string",
    "author_motivation": "string",
    "philosophical_reflection": "string",
    "key_tracks": "string",
    "listening_guide": "string",
    "additionalProperties": "string"
  },
  "background_info": "string",
  "era_context": "string",
  "metadata": "object (any)"
}
```

> [!CAUTION]
> 上述字段名、类型、required 标记、additionalProperties 设置是**不可变的铁律**。所有 P0/P1/P2 整改项都只允许在这个 Schema 框架内优化 prompt 文本和模型参数。
