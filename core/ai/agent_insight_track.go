package ai

import (
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/vincentchyu/sonic-lens/common"
)

/*
trackInsight
*/
var (
	trackInsightSystemPromptFeedbackSectionFmt1 = `
═══════════════════════════════════════════
【重要：历史用户反馈】
═══════════════════════════════════════════
`
	trackInsightSystemPromptFeedbackSectionFmt2 = `
⚠️ 请在分析时特别注意避免重复上述问题，确保本次分析质量更高。
`

	trackInsightSystemPromptFmt1 = `你是一位多维度音乐分析专家，精通文学翻译、乐评分析、文化史研究。请深度分析这首歌曲。`

	trackInsightSystemPromptFmt2 = `
═══════════════════════════════════════════
【核心约束与格式规范】
═══════════════════════════════════════════
1. 只能输出 JSON，不要包含 Markdown 代码块标记（如 \u0060\u0060\u0060json）。
2. 所有字符串使用 UTF-8 编码。
3. 如信息不足，在相关字段填入"背景信息有限"。
4. 优先保证 lyrics_translation、analysis_summary 和 appreciate_analysis 的完整性。
5. 使用 \n 表示换行，不要在 JSON 中使用实际换行符。
6. 不要输出任何思考过程，只输出最终 JSON。
7. 【非常重要】所有歌词标签必须直接输出标准闭合标签文本：<original>...</original> <translation>...</translation> <explain>...</explain>，严禁使用 Unicode 转义（如 \u003c \u003e），严禁将包含标签的内容作为 JSON 字符串再嵌套，不允许 JSON inside JSON。
8. lyrics_translation 和 analysis_by_section.appreciate_analysis 必须为纯文本字段。

【标签输出示例】
原文是【非中文】（必须严格执行原文+翻译逐行对照）：
<original>Hello darkness, my old friend</original>
<translation>你好黑暗，我的老友</translation>

原文是【纯中文】（无需翻译，直接输出原文，但要保留<translation>标签）：
<original>你好世界</original>
<translation></translation>

【分句/分段赏析示例】
分句赏析必须包含 <explain> 标签：
<original>就在一瞬间</original>
<translation></translation>
<explain>表用户的的惆怅</explain>

═══════════════════════════════════════════
【任务指令】请按以下四个角色依次分析：
═══════════════════════════════════════════
【角色一：文学家、翻译家】
1. 双语翻译：解析原文并提供信雅达的双语翻译或原文（纯中文）。
2. 文学分析：解析歌词中的核心意象、隐喻和修辞手法；分析叙事结构和情感递进；和外语原文进行详细的解读，不少于300字，重点关注歌词含义、隐喻、立意等。
3. 分段解读(appreciate_analysis)：必须包含完整的歌词内容。对于重复度高、多次循环出现的段落/副歌，仅在首次出现时进行详细拆解，后续重复时请进行合并说明（例如：[重复副歌，略过并进行融合赏析]），以节省篇幅，防止因内容过多导致输出被强行截断，但严禁遗漏任何非重复的核心歌词。

【角色二：乐评人】
1. 音乐风格：判断歌曲的音乐流派和风格特征，分析编曲的层次感和乐器运用。
2. 演唱表现：分析歌手的演唱技巧和情感表达，说明歌曲的记忆点。

【角色三：文化史学家】
1. 创作背景：说明这首歌的大致创作背景，分析创作动机。
2. 时代语境：说明歌曲所处时代的大致文化/社会语境。

【角色四：综合分析师】
1. 整体评价：总结这首歌的核心价值和艺术成就，提炼 2-3 个最突出的亮点。
`

	trackInsightSystemPromptFmt3 = `
═══════════════════════════════════════════
【JSON Schema】
═══════════════════════════════════════════
{"type":"object","properties":{"lyrics_translation":{"type":"string","description":"必须为纯文本，包含 <original> <translation> 标签，禁止 JSON 字符串包裹，禁止 \\u003c 转义"},"analysis_summary":{"type":"string"},"analysis_by_section":{"type":"object","properties":{"appreciate_analysis":{"type":"string","description":"分段赏析，必须包含完整歌词原文标签，使用 <original> <translation> <explain>，不得转义"}},"required":["appreciate_analysis"],"additionalProperties":{"type":"string"}},"background_info":{"type":"string"},"era_context":{"type":"string"},"metadata":{"type":"object","additionalProperties":true}},"required":["lyrics_translation","analysis_summary","analysis_by_section"],"additionalProperties":false}

【JSON 注解含义】
{"lyrics_translation":"逐行双语对照结果（非中文歌曲）或原文（中文歌曲）","analysis_summary":"综合分析师的整体评价（200-300字）","analysis_by_section":{"literary_analysis":"文学翻译家的深度解读（意象、修辞、叙事）","appreciate_analysis":"分段、句进行赏析 and 解读，必须包含完整歌词原文标签","musical_analysis":"乐评人的专业评价（风格、编曲、演唱）","cultural_context":"文化史学家的背景与时代分析","translation_notes":"翻译难点说明或语言特色分析"},"background_info":"创作背景信息","era_context":"时代文化语境","metadata":{"analysis_depth":"深度分析","model_size":"模型id"}}
`
	trackInsightSystemPromptFmt4 = `
请根据歌曲信息进行深度分析：`
)

// GetTrackInsightSchema 返回用于歌曲分析的结构化 JSON Schema 对象
func GetTrackInsightSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"lyrics_translation": map[string]any{
				"type":        "string",
				"description": "逐行双语对照结果（非中文歌曲）或原文（中文歌曲）。必须为纯文本，包含 <original> <translation> 标签，禁止 JSON 字符串包裹，禁止 \\u003c 转义",
			},
			"analysis_summary": map[string]any{
				"type":        "string",
				"description": "综合分析师的整体评价（200-300字）",
			},
			"analysis_by_section": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"appreciate_analysis": map[string]any{
						"type":        "string",
						"description": "分段赏析，必须包含完整歌词原文标签，使用 <original> <translation> <explain>，不得转义",
					},
					"literary_analysis": map[string]any{
						"type":        "string",
						"description": "文学翻译家的深度解读（意象、修辞、叙事），不少于300字",
					},
					"musical_analysis": map[string]any{
						"type":        "string",
						"description": "乐评人的专业评价（风格、编曲、演唱）",
					},
					"cultural_context": map[string]any{
						"type":        "string",
						"description": "文化史学家的背景与时代分析",
					},
					"translation_notes": map[string]any{
						"type":        "string",
						"description": "翻译难点说明或语言特色分析",
					},
				},
				"required":             []string{"appreciate_analysis"},
				"additionalProperties": map[string]any{"type": "string"},
			},
			"background_info": map[string]any{
				"type":        "string",
				"description": "创作背景信息",
			},
			"era_context": map[string]any{
				"type":        "string",
				"description": "时代文化语境",
			},
			"metadata": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
		"required":             []string{"lyrics_translation", "analysis_summary", "analysis_by_section"},
		"additionalProperties": false,
	}
}

// buildTrackInsightSystemPromptWithoutSchema 提供与 Ollama 一致的系统提示词（不含 Schema 文本）
func buildTrackInsightSystemPromptWithoutSchema(req TrackAnalysisRequest) string {
	prompt := "系统提示：\n" + trackInsightSystemPromptFmt1 + trackInsightSystemPromptFmt2
	if req.FeedbackContext != "" {
		prompt += trackInsightSystemPromptFeedbackSectionFmt1 + req.FeedbackContext + trackInsightSystemPromptFeedbackSectionFmt2
	}
	prompt += trackInsightSystemPromptFmt4 + "\n"
	return prompt
}

// buildTrackInsightSystemPromptWithSchema 包含完整的 Schema 文本
func buildTrackInsightSystemPromptWithSchema(req TrackAnalysisRequest) string {
	prompt := "系统提示：\n" + trackInsightSystemPromptFmt1 + trackInsightSystemPromptFmt2
	if req.FeedbackContext != "" {
		prompt += trackInsightSystemPromptFeedbackSectionFmt1 + req.FeedbackContext + trackInsightSystemPromptFeedbackSectionFmt2
	}
	prompt += trackInsightSystemPromptFmt3 + trackInsightSystemPromptFmt4 + "\n"
	return prompt
}

// buildTrackInsightUserPrompt 格式化用户输入数据
func buildTrackInsightUserPrompt(req TrackAnalysisRequest) string {
	userPromptData := map[string]interface{}{
		"title":       req.Title,
		"artist":      req.Artist,
		"album":       req.Album,
		"lyrics":      req.Lyrics,
		"lang_source": req.LangSource,
		"lang_target": req.LangTarget,
	}
	userPromptBytes, _ := json.Marshal(userPromptData)
	return fmt.Sprintf("输入数据（JSON）：\n%s\n请严格按照【核心约束与格式规范】输出解析结果\n", userPromptBytes)
}

func buildTrackInsightMergedPrompt(req TrackAnalysisRequest) string {
	return buildTrackInsightSystemPromptWithSchema(req) + buildTrackInsightUserPrompt(req)
}

// GetTrackInsightStep1Schema 返回 Step 1 的 JSON Schema
func GetTrackInsightStep1Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"lyrics_translation": map[string]any{
				"type":        "string",
				"description": "逐行双语对照结果（非中文歌曲）或原文（中文歌曲）。必须为纯文本，包含 <original> <translation> 标签，禁止 JSON 字符串包裹，禁止 \\u003c 转义",
			},
		},
		"required":             []string{"lyrics_translation"},
		"additionalProperties": false,
	}
}

var (
	trackInsightStep1SystemPromptConstraints = `
═══════════════════════════════════════════
【核心约束与格式规范】
═══════════════════════════════════════════
1. 只能输出 JSON，不要包含 Markdown 代码块标记（如 \u0060\u0060\u0060json）。
2. 所有字符串使用 UTF-8 编码。
3. 如信息不足，在相关字段填入"背景信息有限"。
4. 使用 \n 表示换行，不要在 JSON 中使用实际换行符。
5. 不要输出任何思考过程，只输出最终 JSON。
6. 【非常重要】所有歌词标签必须直接输出标准闭合标签文本：<original>...</original> <translation>...</translation>，严禁使用 Unicode 转义（如 \u003c \u003e），严禁将包含标签的内容作为 JSON 字符串再嵌套，不允许 JSON inside JSON。
7. lyrics_translation 必须为纯文本字段。

【标签输出示例】
原文是【非中文】（必须严格执行原文+翻译逐行对照）：
<original>Hello darkness, my old friend</original>
<translation>你好黑暗，我的老友</translation>

原文是【纯中文】（无需翻译，直接输出原文，但要保留<translation>标签）：
<original>你好世界</original>
<translation></translation>

═══════════════════════════════════════════
【任务指令】请作为文学家与翻译家，完成歌词翻译任务：
═══════════════════════════════════════════
双语翻译：解析输入数据中的 lyrics，并提供信雅达的双语对照翻译（非中文歌曲）或原文对照（中文歌曲），输出在 lyrics_translation 字段中。
`

	trackInsightStep2SystemPromptConstraints = `
═══════════════════════════════════════════
【核心约束与格式规范】
═══════════════════════════════════════════
1. 只能输出 JSON，不要包含 Markdown 代码块标记（如 \u0060\u0060\u0060json）。
2. 所有字符串使用 UTF-8 编码。
3. 如信息不足，在相关字段填入"背景信息有限"。
4. 使用 \n 表示换行，不要在 JSON 中使用实际换行符。
5. 不要输出任何思考过程，只输出最终 JSON。
6. 【非常重要】所有歌词标签必须直接输出标准闭合标签文本：<original>...</original> <translation>...</translation> <explain>...</explain>，严禁使用 Unicode 转义（如 \u003c \u003e），严禁将包含标签的内容作为 JSON 字符串再嵌套，不允许 JSON inside JSON。
7. appreciate_analysis 必须为纯文本字段。

【分句/分段赏析示例】
分句赏析必须包含 <explain> 标签：
<original>就在一瞬间</original>
<translation></translation>
<explain>表用户的的惆怅</explain>

═══════════════════════════════════════════
【任务指令】请作为文学家，对歌词进行分段/分句赏析：
═══════════════════════════════════════════
分段解读(appreciate_analysis)：必须包含完整的歌词内容。对于重复度高、多次循环出现的段落/副歌，仅在首次出现时进行详细拆解，后续重复时请进行合并说明（例如：[重复副歌，略过并进行融合赏析]），以节省篇幅，防止因内容过多导致输出被强行截断，但严禁遗漏任何非重复的核心歌词。也可以自动分段（将多句内容综合分析），但前提是这些分段歌词应该是一个段落。
`

	trackInsightStep3SystemPromptConstraints = `
═══════════════════════════════════════════
【核心约束与格式规范】
═══════════════════════════════════════════
1. 只能输出 JSON，不要包含 Markdown 代码块标记（如 \u0060\u0060\u0060json）。
2. 所有字符串使用 UTF-8 编码。
3. 如信息不足，在相关字段填入"背景信息有限"。
4. 使用 \n 表示换行，不要在 JSON 中使用实际换行符。
5. 不要输出任何思考过程，只输出最终 JSON。

═══════════════════════════════════════════
【任务指令】请作为乐评人、文化史学家及综合分析师，对歌曲进行整体深度分析：
═══════════════════════════════════════════
请根据【参考翻译】和【参考分段/分句赏析】（如有），产出以下维度的深度长文分析并填入 JSON 对应字段：
1. 整体评价 (analysis_summary)：总结这首歌的核心价值和艺术成就，提炼 2-3 个最突出的亮点，不少于200字。
2. 文学分析 (analysis_by_section.literary_analysis)：解析歌词中的核心意象、隐喻和修辞手法，叙事结构和情感递进，不少于300字。
3. 音乐流派与风格 (analysis_by_section.musical_analysis)：判断歌曲的流派、风格特征、编曲层次感、乐器运用及演唱技巧与情感。
4. 文化时代背景 (analysis_by_section.cultural_context)：说明这首歌的创作背景与所处时代的大致文化/社会语境。
5. 翻译难点说明 (analysis_by_section.translation_notes)：说明翻译难点或语言特色。
6. 创作背景信息 (background_info)
7. 时代文化语境 (era_context)
`
)

// buildTrackInsightStep1SystemPrompt 构建 Step 1 的系统提示词
func buildTrackInsightStep1SystemPrompt(req TrackAnalysisRequest) string {
	prompt := "系统提示：\n你是一位翻译专家。请解析原文并提供信雅达的双语翻译或原文（纯中文）。\n" + trackInsightStep1SystemPromptConstraints
	if req.FeedbackContext != "" {
		prompt += trackInsightSystemPromptFeedbackSectionFmt1 + req.FeedbackContext + trackInsightSystemPromptFeedbackSectionFmt2
	}
	schemaBytes, _ := json.Marshal(GetTrackInsightStep1Schema())
	prompt += "\n【JSON Schema】\n" + string(schemaBytes) + "\n"
	prompt += trackInsightSystemPromptFmt4 + "\n"
	return prompt
}

// GetTrackInsightStep2Schema 返回 Step 2 的 JSON Schema
func GetTrackInsightStep2Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"appreciate_analysis": map[string]any{
				"type":        "string",
				"description": "分段赏析，必须包含完整歌词原文标签，使用 <original> <translation> <explain>，不得转义",
			},
		},
		"required":             []string{"appreciate_analysis"},
		"additionalProperties": false,
	}
}

// buildTrackInsightStep2SystemPrompt 构建 Step 2 的系统提示词
func buildTrackInsightStep2SystemPrompt(req TrackAnalysisRequest, translated string) string {
	prompt := "系统提示：\n你是一位文学家。请对以下翻译过的歌词进行分段解读。如果作品不是好作品你也要大胆的指出\n" + trackInsightStep2SystemPromptConstraints
	if req.FeedbackContext != "" {
		prompt += trackInsightSystemPromptFeedbackSectionFmt1 + req.FeedbackContext + trackInsightSystemPromptFeedbackSectionFmt2
	}
	schemaBytes, _ := json.Marshal(GetTrackInsightStep2Schema())
	prompt += "\n【JSON Schema】\n" + string(schemaBytes) + "\n"
	prompt += "\n【参考翻译】\n" + translated + "\n"
	prompt += trackInsightSystemPromptFmt4 + "\n"
	return prompt
}

// GetTrackInsightStep3Schema 返回 Step 3 的 JSON Schema
func GetTrackInsightStep3Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"analysis_summary": map[string]any{
				"type":        "string",
				"description": "综合分析师的整体评价（200-300字）",
			},
			"analysis_by_section": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"literary_analysis": map[string]any{
						"type":        "string",
						"description": "文学翻译家的深度解读（意象、修辞、叙事），不少于300字",
					},
					"musical_analysis": map[string]any{
						"type":        "string",
						"description": "乐评人的专业评价（风格、编曲、演唱）",
					},
					"cultural_context": map[string]any{
						"type":        "string",
						"description": "文化史学家的背景与时代分析",
					},
					"translation_notes": map[string]any{
						"type":        "string",
						"description": "翻译难点说明或语言特色分析",
					},
				},
				"additionalProperties": map[string]any{"type": "string"},
			},
			"background_info": map[string]any{
				"type":        "string",
				"description": "创作背景信息",
			},
			"era_context": map[string]any{
				"type":        "string",
				"description": "时代文化语境",
			},
			"metadata": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
		"required":             []string{"analysis_summary", "analysis_by_section"},
		"additionalProperties": false,
	}
}

// buildTrackInsightStep3SystemPrompt 构建 Step 3 的系统提示词
func buildTrackInsightStep3SystemPrompt(req TrackAnalysisRequest, translated string, appreciateAnalysis string) string {
	prompt := "系统提示：\n你是一位多维度音乐分析专家，精通文学翻译、乐评分析、文化史研究。请深度分析这首歌曲。无需再输出逐句赏析，只需输出整体评价和各维度的深度长文分析。如果作品不是好作品你也要大胆的指出。\n" + trackInsightStep3SystemPromptConstraints
	if req.FeedbackContext != "" {
		prompt += trackInsightSystemPromptFeedbackSectionFmt1 + req.FeedbackContext + trackInsightSystemPromptFeedbackSectionFmt2
	}
	schemaBytes, _ := json.Marshal(GetTrackInsightStep3Schema())
	prompt += "\n【JSON Schema】\n" + string(schemaBytes) + "\n"
	prompt += "\n【参考翻译】\n" + translated + "\n"
	if appreciateAnalysis != "" {
		prompt += "\n【参考分段/分句赏析】\n" + appreciateAnalysis + "\n"
	}
	prompt += trackInsightSystemPromptFmt4 + "\n"
	return prompt
}

// IsInstrumentalOrEmptyLyrics 判断当前曲目请求是否为纯音乐或无歌词曲目
func IsInstrumentalOrEmptyLyrics(req TrackAnalysisRequest) bool {
	lyrics := strings.TrimSpace(req.Lyrics)
	if lyrics == "" {
		return true
	}
	lower := strings.ToLower(lyrics)
	keywords := []string{
		"[instrumental]", "(instrumental)", "纯音乐", "纯音乐，请欣赏",
		"纯音乐请欣赏", "音乐暂无歌词", "无歌词", "纯音乐 - 无歌词",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// IsChineseLyrics 判断歌词是否为纯中文歌词（支持带英文衬词、数字、标点和LRC标记）
func IsChineseLyrics(lyrics string) bool {
	return common.IsChineseLyrics(lyrics)
}

// FormatChineseLyricsTranslation 将纯中文歌词格式化为包含 <original> 与 <translation> 标签的 Step 1 标准格式
func FormatChineseLyricsTranslation(lyrics string) string {
	return common.FormatChineseLyricsTranslation(lyrics)
}
