package ai

import (
	"encoding/json/v2"
	"fmt"
)

/*
trackInsight
*/
var (
	albumInsightSystemPromptFeedbackSectionFmt1 = `
═══════════════════════════════════════════
【重要：专辑历史用户反馈】
═══════════════════════════════════════════
`
	albumInsightSystemPromptFeedbackSectionFmt2 = `
⚠️ 请在专辑分析时特别注意避免重复上述问题，确保本次分析质量更高。
`
	albumInsightSystemPromptFmt1 = `你是一位资深专辑研究者、文学评论家与音乐史分析师。请基于输入的专辑信息和按曲序整理的曲目音眸结果，完成一次专辑级深度分析。`
	albumInsightSystemPromptFmt2 = `
═══════════════════════════════════════════
【分析目标】
═══════════════════════════════════════════
1. 从整张专辑出发，而不是逐首歌机械拼接，识别其核心主题、叙事推进与结构设计。
2. 聚焦以下重点：
   - 时代意义：专辑在艺术家创作生涯、流派发展或时代文化中的位置
   - 文学解读：贯穿全专的意象、母题、叙事方法与修辞风格
   - 作者动机：创作者可能的表达意图、创作处境与情绪驱动力
   - 哲学反思：专辑隐含的价值观、存在主题、社会观察或精神困境
   - 听感结构：曲序安排、开场/中段/收束的组织方式，哪些曲目承担转折或定锚作用
3. 当输入里某些曲目没有音眸分析时，要明确说明结论是“基于已分析曲目归纳”，不要假装覆盖了整张专辑全部细节。
4. 如果信息不足，请在相关字段写“背景信息有限”。

═══════════════════════════════════════════
【输出要求】
═══════════════════════════════════════════
1. 只能输出 JSON，不要 Markdown 代码块，不要额外说明。
2. analysis_summary 必须是对整张专辑的整体判断，不少于 220 字。
3. analysis_by_section 建议至少覆盖以下键：
   - album_positioning：专辑在艺术家生涯和作品谱系中的位置
   - theme_and_narrative：主题母题、叙事线或情绪线，不少于 220 字
   - literary_analysis：意象、修辞、象征与文本组织
   - musical_analysis：曲风、编排、听感推进与曲序设计
   - author_motivation：创作者动机与创作处境
   - philosophical_reflection：哲学层面的反思与精神命题,不少于500字
   - key_tracks：点名关键曲目及其在专辑中的作用，不少于 200 字
   - listening_guide：建议的欣赏路径与切入角度
4. metadata 中可以补充 analyzed_tracks、total_tracks、method 等信息。
5. 不要输出思考过程，只输出最终 JSON。`
)

/*
	AlbumInsight
	TODOD：
		专辑表、专辑歌曲关联表（可以在track中得到，使用trackId进行关联）、专辑分析表（专辑id关联）
		根据专辑中歌曲的顺序，
		搜索当前专辑下全部的音眸分析，每首歌只保留最高分或者最新的分析数据，

		聚焦（时代意义、文学解读、作者动机、哲学反思，还有什么关于专辑可以分析的要点？结合曲目汇总结果的要点？）
		前端专辑详情增加 音眸专辑分析按钮 增加album_insight表？
		原音眸分析为 音眸曲目分析 库表track_insight

		GetAlbumInsightSchema、AlbumInsight prompt、定义
		专辑分析的深度比较深可以提前做prompt角色抽象，代码规划
		汇总大模型上下文数据，进行专辑分析
*/

// GetAlbumInsightSchema 返回用于专辑分析的结构化 JSON Schema 对象。
func GetAlbumInsightSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"analysis_summary": map[string]any{
				"type":        "string",
				"description": "对整张专辑的整体评价、主题提炼和艺术判断",
			},
			"analysis_by_section": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"album_positioning": map[string]any{
						"type":        "string",
						"description": "专辑在艺术家生涯和作品谱系中的位置",
					},
					"theme_and_narrative": map[string]any{
						"type":        "string",
						"description": "整张专辑的主题母题、叙事线与情绪推进",
					},
					"literary_analysis": map[string]any{
						"type":        "string",
						"description": "意象、修辞、象征与文本组织层面的文学解读",
					},
					"musical_analysis": map[string]any{
						"type":        "string",
						"description": "曲风、编排、听感推进与曲序设计",
					},
					"author_motivation": map[string]any{
						"type":        "string",
						"description": "创作者动机、创作处境与表达意图",
					},
					"philosophical_reflection": map[string]any{
						"type":        "string",
						"description": "专辑折射出的价值观、存在主题与哲学反思",
					},
					"key_tracks": map[string]any{
						"type":        "string",
						"description": "关键曲目及其在整张专辑中的作用",
					},
					"listening_guide": map[string]any{
						"type":        "string",
						"description": "欣赏整张专辑的切入角度和收听建议",
					},
				},
				"additionalProperties": map[string]any{"type": "string"},
			},
			"background_info": map[string]any{
				"type":        "string",
				"description": "专辑创作背景信息",
			},
			"era_context": map[string]any{
				"type":        "string",
				"description": "专辑所处时代的文化与社会语境",
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

func buildAlbumInsightSystemPrompt() string {
	return "系统提示：\n" + albumInsightSystemPromptFmt1 + albumInsightSystemPromptFmt2 + "\n"
}

func buildAlbumInsightSystemPromptAll() string {
	schemaBytes, _ := json.Marshal(GetAlbumInsightSchema())
	return "系统提示：\n" + albumInsightSystemPromptFmt1 + albumInsightSystemPromptFmt2 + "\n【JSON Schema】\n" + string(schemaBytes) + "\n"
}

func buildAlbumInsightUserPrompt(req AlbumAnalysisRequest) string {
	userPromptData := map[string]interface{}{
		"album_id":        req.AlbumID,
		"artist":          req.Artist,
		"album":           req.Album,
		"release_date":    req.ReleaseDate,
		"genre":           req.Genre,
		"track_count":     req.TrackCount,
		"analyzed_tracks": req.AnalyzedTracks,
		"track_contexts":  req.TrackContexts,
	}
	userPromptBytes, _ := json.Marshal(userPromptData)
	str := fmt.Sprintf("输入数据（JSON）：\n%s\n请严格按照【输出要求】输出解析结果\n", userPromptBytes)

	if req.FeedbackContext != "" {
		feedbackSection := albumInsightSystemPromptFeedbackSectionFmt1 + req.FeedbackContext + albumInsightSystemPromptFeedbackSectionFmt2
		str += feedbackSection
	}

	return str
}

func buildAlbumInsightMergedPrompt(req AlbumAnalysisRequest) string {
	return buildAlbumInsightSystemPromptAll() + buildAlbumInsightUserPrompt(req)
}

/*
	xxInsight
	规划：
*/
