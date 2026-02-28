package api

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/lyrics"
	"github.com/vincentchyu/sonic-lens/core/websocket"
	"github.com/vincentchyu/sonic-lens/internal/logic/analysis"
	"github.com/vincentchyu/sonic-lens/internal/logic/genre"
	"github.com/vincentchyu/sonic-lens/internal/logic/insight"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func setupRouter(name string) *gin.Engine {
	r := gin.Default()

	// Add OpenTelemetry middleware
	r.Use(
		otelgin.Middleware(name),
		func(c *gin.Context) {
			traceID := trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()
			c.Header("Trace-Id", traceID)
			c.Next()
		},
	)

	// V2前端静态文件服务
	/*r.Static("/v2/assets", "./frontend-v2/dist/assets")
	r.StaticFile("/v2/favicon.ico", "./frontend-v2/dist/favicon.ico")
	// SPA回退路由，所有/v2/*路由都返回index.html
	r.GET("/v2/*filepath", func(c *gin.Context) {
		c.File("./frontend-v2/dist/index.html")
	})*/

	r.StaticFile("/static/chartjs-adapter-date-fns.bundle.min.js", "./static/chartjs-adapter-date-fns.bundle.min.js")
	r.StaticFile("/static/html2canvas.min.js", "./static/html2canvas.min.js")
	r.StaticFile("/static/chart.js", "./static/chart.js")
	r.StaticFile("/static/logo.svg", "./static/logo.svg")
	r.StaticFile("/static/logo_black.svg", "./static/logo_black.svg")
	r.StaticFile("/static/logo_all.svg", "./static/logo_all.svg")
	r.StaticFile("/static/logo_all_black.svg", "./static/logo_all_black.svg")

	// 首页
	r.GET(
		"/", func(c *gin.Context) {
			// Load HTML template
			tmplPath := filepath.Join("templates", "dashboard.html")
			tmpl, err := template.New("dashboard.html").ParseFiles(tmplPath)
			if err != nil {
				log.Error(c.Request.Context(), "Failed to parse template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
				return
			}

			// Set content type and write HTML response
			c.Header("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(c.Writer, nil); err != nil {
				log.Error(c.Request.Context(), "Failed to execute template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
				return
			}
		},
	)

	// Get track play counts with pagination
	trackService := track.NewTrackService()
	// AI 歌词解析服务
	insightService, insightErr := insight.NewService()
	if insightErr != nil {
		// 记录日志但不阻断整个服务启动，前端调用时再返回错误
		log.Warn(context.Background(), "初始化歌词解析服务失败，将暂时无法使用 AI 歌词解析功能", zap.Error(insightErr))
	}
	r.GET(
		"/api/track-play-counts", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetTrackPlayCounts(c.Request.Context(), limit, offset)
			log.Info(
				c.Request.Context(), "Fetched track play counts", zap.Int("count", len(records)),
				zap.Int("limit", limit), zap.Int("offset", offset),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Check if client expects HTML response
			acceptHeader := c.GetHeader("Accept")
			if strings.Contains(acceptHeader, "text/html") || c.Query("format") == "html" {
				// Load HTML template
				tmplPath := filepath.Join("templates", "track_play_counts.html")
				tmpl, err := template.New("track_play_counts.html").Funcs(
					template.FuncMap{
						"addOne": func(i int) int {
							return i + 1
						},
						"add": func(a, b int) int {
							return a + b
						},
						"subtract": func(a, b int) int {
							return a - b
						},
					},
				).ParseFiles(tmplPath)
				if err != nil {
					log.Error(c.Request.Context(), "Failed to parse template", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
					return
				}

				// Execute template with records data
				data := struct {
					Records     []*model.Track
					Limit       int
					Offset      int
					RecordCount int
				}{
					Records:     records,
					Limit:       limit,
					Offset:      offset,
					RecordCount: len(records),
				}

				// Set content type and write HTML response
				c.Header("Content-Type", "text/html; charset=utf-8")
				if err := tmpl.Execute(c.Writer, data); err != nil {
					log.Error(c.Request.Context(), "Failed to execute template", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
					return
				}
			} else {
				// Return JSON response for API clients
				c.JSON(http.StatusOK, records)
			}
		},
	)

	// 获取当前系统支持的 AI 分析模型
	r.GET(
		"/api/ai-models", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusOK, gin.H{"models": []string{}})
				return
			}
			models := insightService.GetAvailableAIProviders()
			c.JSON(http.StatusOK, gin.H{"models": models})
		},
	)

	// 获取某首歌已有的 AI 解析结果 (仅查询)
	r.GET(
		"/api/track-insight", func(c *gin.Context) {
			artist := c.Query("artist")
			album := c.Query("album")
			track := c.Query("track")

			if artist == "" || track == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "参数不足"})
				return
			}
			insights, err := insightService.GetInsightOnly(c.Request.Context(), artist, album, track)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"insights": insights})
		},
	)

	// Get play for a specific track
	r.GET(
		"/api/track", func(c *gin.Context) {
			artist := c.Query("artist")
			album := c.Query("album")
			trackName := c.Query("trackName")

			if artist == "" || album == "" || trackName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "artist, album, and trackName are required"})
				return
			}

			record, err := trackService.GetTrack(c.Request.Context(), artist, album, trackName)
			if err != nil {
				if err.Error() == "record not found" {
					c.JSON(http.StatusOK, gin.H{"play_count": 0})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, record)
		},
	)

	// 获取 / 生成某首歌的歌词解析结果
	r.POST(
		"/api/track-insight", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(
					http.StatusServiceUnavailable, gin.H{
						"error": "AI 歌词解析服务未正确初始化，请检查 OPENAI_API_KEY 等配置",
					},
				)
				return
			}

			var req struct {
				Artist    string `json:"artist"`
				Album     string `json:"album"`
				Track     string `json:"track"`
				ModelType string `json:"modelType"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			ctx := c.Request.Context()
			// POST 接口语义为强制分析
			insights, cached, err := insightService.GetOrCreateInsight(
				ctx, req.Artist, req.Album, req.Track, true, req.ModelType,
			)
			if err != nil {
				log.Error(
					ctx, "获取或生成歌词解析失败",
					zap.String("artist", req.Artist),
					zap.String("album", req.Album),
					zap.String("track", req.Track),
					zap.Error(err),
				)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "生成歌词解析失败"})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"insights": insights,
					"cached":   cached,
				},
			)
		},
	)

	// 流式获取歌词解析结果 (SSE)
	r.GET(
		"/api/track-insight-stream", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			artist := c.Query("artist")
			album := c.Query("album")
			track := c.Query("track")
			force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))
			modelType := c.Query("modelType")

			if artist == "" || track == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "参数不足"})
				return
			}

			ch, _, err := insightService.GetOrCreateInsightStream(
				c.Request.Context(), artist, album, track, force, modelType,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 设置 SSE 必要的 Response Headers
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("Transfer-Encoding", "chunked")
			// 针对 Nginx 等代理，禁用响应缓冲，实现真正的实时流
			c.Header("X-Accel-Buffering", "no")

			c.Stream(
				func(w io.Writer) bool {
					if chunk, ok := <-ch; ok {
						// 写入 SSE 格式消息：data: <content>\n\n
						c.Render(
							-1, sse.Event{
								Event: "message",
								Data:  chunk,
							},
						)
						return true
					}
					return false
				},
			)
		},
	)

	// 对某次歌词解析结果进行点赞 / 点踩反馈
	r.POST(
		"/api/track-insight/:id/feedback", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(
					http.StatusServiceUnavailable, gin.H{
						"error": "AI 歌词解析服务未正确初始化，请检查配置",
					},
				)
				return
			}

			idStr := c.Param("id")
			insightID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || insightID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 insight ID"})
				return
			}

			var req struct {
				Score   int    `json:"score"`   // 1 点赞，-1 点踩
				Comment string `json:"comment"` // 可选备注
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			ctx := c.Request.Context()
			if err := insightService.RecordFeedback(ctx, insightID, req.Score, req.Comment); err != nil {
				log.Error(
					ctx, "记录歌词解析反馈失败",
					zap.Int64("insight_id", insightID),
					zap.Int("score", req.Score),
					zap.Error(err),
				)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "记录反馈失败"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 获取所有 AI 解析记录 (分页管理)
	r.GET(
		"/api/insights/all", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			keyword := c.Query("keyword")
			if limit > 100 {
				limit = 100
			}

			insights, total, err := insightService.GetAllInsights(c.Request.Context(), limit, offset, keyword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK, gin.H{
					"insights": insights,
					"total":    total,
					"limit":    limit,
					"offset":   offset,
				},
			)
		},
	)

	// 切换解析记录状态 (禁用 / 启用)
	r.POST(
		"/api/insights/:id/toggle-status", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}
			idStr := c.Param("id")
			insightID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || insightID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
				return
			}

			if err := insightService.ToggleInsightStatus(c.Request.Context(), insightID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 直接删除解析记录
	r.DELETE(
		"/api/insights/:id", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}
			idStr := c.Param("id")
			insightID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || insightID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
				return
			}

			if err := insightService.DeleteInsight(c.Request.Context(), insightID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 获取某次解析关联的 LLM 调用流水
	r.GET(
		"/api/insights/:id/logs", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			// 由于目前模型定义中日志是按 artist + track 关联的，我们先通过 ID 查到曲目信息
			idStr := c.Param("id")
			insightID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || insightID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
				return
			}

			// 分页查询所有以找到目标（简单实现，建议在 Logic 层增加按 ID 特化查询）
			insights, _, err := insightService.GetAllInsights(c.Request.Context(), 1000, 0, "")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			var target *model.TrackInsight
			for _, ins := range insights {
				if ins.ID == insightID {
					target = ins
					break
				}
			}

			if target == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "解析记录不存在"})
				return
			}

			logs, err := insightService.GetTrackCallLogs(c.Request.Context(), target.Artist, target.Track)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"logs": logs})
		},
	)

	// 获取关联的用户反馈记录
	r.GET(
		"/api/insights/:id/feedbacks", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}
			idStr := c.Param("id")
			insightID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || insightID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
				return
			}

			feedbacks, err := insightService.GetInsightFeedbacks(c.Request.Context(), insightID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"feedbacks": feedbacks})
		},
	)

	// 获取歌词数据（优先查库，没有则调用 lrcapi 等）
	r.GET(
		"/api/track-lyrics", func(c *gin.Context) {
			artist := c.Query("artist")
			album := c.Query("album")
			track := c.Query("track")

			if artist == "" || track == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必需参数 artist 和 track"})
				return
			}

			// 尝试从服务层 helper 获取（重用逻辑）
			// 由于 service 层没有暴露 getOrFetchLyrics，如果需要复用逻辑最好暴露出来，
			// 或者在这里重新实现一遍查询 + 保存逻辑。
			// 鉴于 server.go 不应包含太多业务逻辑，理想做法是在 insightService 暴露一个 GetLyrics 方法。
			// 但既然现在在修改 server.go，我们可以直接查 model。

			// 1. 查询数据库 TrackLyrics
			ctx := c.Request.Context()
			lyricsData, err := model.GetTrackLyrics(ctx, artist, album, track)

			lrcContent := ""
			if err == nil {
				lrcContent = lyricsData.LyricsOriginal
			} else {
				// 2. 如果数据库没有，尝试调用 provider (这里简单的复用 insightService 可能没有直接暴露获取单独歌词的方法
				// 如果 server.go 这里不能方便调用 lrcapi，那最好是在 insightService 加一个 GetLyrics 方法。
				// 这里的代码我们暂时复用 logic/insight 里的逻辑，或者直接调用 provider。
				// 为了保持整洁，我们在 handler 里临时实例化 provider 是不推荐的。
				// 最好的方案：修改 api/server.go 为调用 insightService.GetLyrics(...)。
				// 但现在为了快速修复编译错误并跑通功能，我们假设如果找不到就返回空，或者前端触发 /api/track-insight 时会自动补全。
				// 用户需求里说“前端页面点击歌词查看歌词的时候可以直接调用lrcapi在没有歌词数据的时候获取歌词数据”。
				// 所以这里必须实现“没有则获取”的逻辑。

				// 简化起见，我们在 server.go 不直接依赖具体的 lrcapi implementation，
				// 而是应该让 insight service 提供这个能力。
				// 但由于 insight package 的 NewService 返回的是 interface，我们需要在 interface 加方法。
				// 让我们先暂时在这里只读库。如果用户点击“分析”，分析过程会补全歌词。
				// 如果用户只是点“查看歌词”，我们希望也能触发获取。
				// 所以最好是修改 Insight Service 接口。

				// **修正方案**：我们假设先只读库。如果为空，前端显示无歌词。
				// 实际上用户期望的是“在没有歌词数据的时候获取歌词数据”。
				// 我们可以在这里简单调用 lrcapi，因为 server.go 已经 import 了 "github.com/vincentchyu/sonic-lens/core/lyrics"

				// 实例化一个临时的 provider 列表 (不太优雅但有效)
				lrcProvider := lyrics.NewLrcAPIProvider()
				fetched, lErr := lrcProvider.GetLyrics(ctx, artist, album, track)
				if lErr == nil && fetched != "" {
					lrcContent = fetched
					// 异步入库
					go func() {
						bgCtx := context.Background()
						newLyrics := &model.TrackLyrics{
							Artist:         artist,
							Album:          album,
							Track:          track,
							LyricsOriginal: fetched,
							LyricsSource:   "lrcapi",
							Synced:         strings.Contains(fetched, "[") && strings.Contains(fetched, "]"),
						}
						_, _ = model.GetOrCreateTrackLyrics(bgCtx, newLyrics)
					}()
				}
			}

			// 判断歌词是否包含 LRC 时间戳格式
			hasLRC := false
			if lrcContent != "" {
				lrcPattern := `\[\d{1,2}:\d{2}[\.\d]*\]`
				matched, _ := regexp.MatchString(lrcPattern, lrcContent)
				hasLRC = matched
			}

			c.JSON(
				http.StatusOK, gin.H{
					"lyrics":  lrcContent,
					"has_lrc": hasLRC,
				},
			)
		},
	)

	// Generate music preference report
	musicAnalysisService := analysis.NewMusicAnalysisService()

	// Generate music recommendations

	// 获取仪表板统计数据
	r.GET(
		"/api/dashboard/stats", func(c *gin.Context) {
			ctx := c.Request.Context()

			// 获取总播放次数
			totalPlays, err := trackService.GetTotalPlayCount(ctx)
			if err != nil {
				log.Error(ctx, "Failed to get total play count", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total play count"})
				return
			}

			// 获取曲目总数
			totalTracks, err := trackService.GetTrackCounts(ctx)
			if err != nil {
				log.Error(ctx, "Failed to get track counts", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get track counts"})
				return
			}

			// 获取艺术家总数
			totalArtists, err := trackService.GetArtistCounts(ctx)
			if err != nil {
				log.Error(ctx, "Failed to get artist counts", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get artist counts"})
				return
			}

			// 获取专辑总数
			totalAlbums, err := trackService.GetAlbumCounts(ctx)
			if err != nil {
				log.Error(ctx, "Failed to get album counts", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get album counts"})
				return
			}

			// 返回统计数据
			stats := gin.H{
				"totalPlays":   totalPlays,
				"totalTracks":  totalTracks,
				"totalArtists": totalArtists,
				"totalAlbums":  totalAlbums,
			}

			c.JSON(http.StatusOK, stats)
		},
	)

	// 获取趋势图数据
	r.GET(
		"/api/dashboard/trend", func(c *gin.Context) {
			ctx := c.Request.Context()

			// 获取时间范围参数，默认7天
			rangeStr := c.DefaultQuery("range", "7")
			rangeDays := 7
			switch rangeStr {
			case "30":
				rangeDays = 30
			case "90":
				rangeDays = 90
			default:
				rangeDays = 7
			}
			fillInTrendCycle := FillInTrendCycle(rangeDays)

			// 获取指定天数的播放记录
			recordMap, err := trackService.GetRecentPlayRecordsByDays(ctx, rangeDays)
			if err != nil {
				log.Error(ctx, "Failed to get recent play records", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent play records"})
				return
			}
			// 处理数据以适应趋势图
			// 按日期统计播放次数
			dateTrendData := make(map[string]int)
			// 按日期和小时统计播放次数
			hourlyTrendData := make(map[string]*model.HourlyPlayTrendData)
			for _, trendCycle := range fillInTrendCycle {
				if records, ok := recordMap[trendCycle]; ok {
					for _, record := range records {
						dateStr := record.PlayTime.Format("2006-01-02")
						hour := record.PlayTime.Hour()
						// 统计每日总播放次数
						dateTrendData[dateStr]++
						// 初始化该日期的小时统计数据
						if _, exists := hourlyTrendData[dateStr]; !exists {
							hourlyTrendData[dateStr] = &model.HourlyPlayTrendData{
								Date:   dateStr,
								Total:  0,
								Hourly: make(map[int]int),
							}
						}

						// 统计该小时的播放次数
						hourlyTrendData[dateStr].Hourly[hour]++
						hourlyTrendData[dateStr].Total++
					}
				} else {
					dateTrendData[trendCycle] = 0
					hourlyTrendData[trendCycle] = &model.HourlyPlayTrendData{
						Date:  trendCycle,
						Total: 0,
						Hourly: map[int]int{
							0: 0,
							1: 0,
							2: 0,
						},
					}
				}
			}

			// 构造返回数据
			result := gin.H{
				"daily":  dateTrendData,
				"hourly": hourlyTrendData,
			}

			c.JSON(http.StatusOK, result)
		},
	)

	// 获取热门艺术家数据（按播放次数）
	r.GET(
		"/api/dashboard/top-artists/plays", func(c *gin.Context) {
			ctx := c.Request.Context()

			// 获取限制参数，默认10个
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if limit > 50 {
				limit = 50 // 限制最大数量
			}

			// 获取按播放次数统计的热门艺术家
			artists, err := trackService.GetTopArtistsByPlayCount(ctx, limit)
			if err != nil {
				log.Error(ctx, "Failed to get top artists by play count", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top artists by play count"})
				return
			}

			c.JSON(http.StatusOK, artists)
		},
	)

	// 获取热门艺术家数据（按曲目数）
	r.GET(
		"/api/dashboard/top-artists/tracks", func(c *gin.Context) {
			ctx := c.Request.Context()

			// 获取限制参数，默认10个
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if limit > 50 {
				limit = 50 // 限制最大数量
			}

			// 获取按曲目数统计的热门艺术家
			artists, err := trackService.GetTopArtistsByTrackCount(ctx, limit)
			if err != nil {
				log.Error(ctx, "Failed to get top artists by track count", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top artists by track count"})
				return
			}

			c.JSON(http.StatusOK, artists)
		},
	)

	// 分析报告页面
	r.GET(
		"/report", func(c *gin.Context) {
			// Create a background context for the report generation
			ctx := c.Request.Context()

			// Generate the report data
			reportData, err := musicAnalysisService.GenerateMusicPreferenceReport(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Load HTML template
			tmplPath := filepath.Join("templates", "report.html")
			tmpl, err := template.New("report.html").Funcs(
				template.FuncMap{
					"addOne": func(i int) int {
						return i + 1
					},
				},
			).ParseFiles(tmplPath)
			if err != nil {
				log.Error(ctx, "Failed to parse template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
				return
			}

			// Set content type and write HTML response
			c.Header("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(c.Writer, reportData); err != nil {
				log.Error(ctx, "Failed to execute template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
				return
			}
		},
	)

	// 播放统计页面
	r.GET(
		"/playCounts", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetTrackPlayCounts(c.Request.Context(), limit, offset)
			log.Info(
				c.Request.Context(), "Fetched track play counts", zap.Int("count", len(records)),
				zap.Int("limit", limit), zap.Int("offset", offset),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Check if client expects HTML response
			acceptHeader := c.GetHeader("Accept")
			if strings.Contains(acceptHeader, "text/html") || c.Query("format") == "html" {
				// Load HTML template
				tmplPath := filepath.Join("templates", "track_play_counts.html")
				tmpl, err := template.New("track_play_counts.html").Funcs(
					template.FuncMap{
						"addOne": func(i int) int {
							return i + 1
						},
						"add": func(a, b int) int {
							return a + b
						},
						"subtract": func(a, b int) int {
							return a - b
						},
					},
				).ParseFiles(tmplPath)
				if err != nil {
					log.Error(c.Request.Context(), "Failed to parse template", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
					return
				}

				// Execute template with records data
				data := struct {
					Records     []*model.Track
					Limit       int
					Offset      int
					RecordCount int
				}{
					Records:     records,
					Limit:       limit,
					Offset:      offset,
					RecordCount: len(records),
				}

				// Set content type and write HTML response
				c.Header("Content-Type", "text/html; charset=utf-8")
				if err := tmpl.Execute(c.Writer, data); err != nil {
					log.Error(c.Request.Context(), "Failed to execute template", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
					return
				}
			} else {
				// Return JSON response for API clients
				c.JSON(http.StatusOK, records)
			}
		},
	)

	// 音乐推荐页面
	r.GET(
		"/recommendations", func(c *gin.Context) {
			// Create a background context for the recommendation generation
			ctx := c.Request.Context()

			// Generate recommendations
			recommendations, err := musicAnalysisService.GenerateRecommendations(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Load HTML template
			tmplPath := filepath.Join("templates", "recommendations.html")
			tmpl, err := template.New("recommendations.html").Funcs(
				template.FuncMap{
					"addOne": func(i int) int {
						return i + 1
					},
				},
			).ParseFiles(tmplPath)
			if err != nil {
				log.Error(ctx, "Failed to parse template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
				return
			}

			// Execute template with recommendations data
			data := struct {
				Recommendations []analysis.MusicRecommendation
			}{
				Recommendations: recommendations,
			}

			// Set content type and write HTML response
			c.Header("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(c.Writer, data); err != nil {
				log.Error(ctx, "Failed to execute template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
				return
			}
		},
	)

	// 最近播放接口
	r.GET(
		"/api/recent-plays", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

			if limit > 100 {
				limit = 100 // Limit max records
			}

			records, err := trackService.GetRecentPlayRecords(c.Request.Context(), limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, records)
		},
	)

	// 按时间段获取播放排行榜接口
	r.GET(
		"/api/track-play-counts/period", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			period := c.Query("period") // 支持 week, month

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetTrackPlayCountsByPeriod(c.Request.Context(), limit, offset, period)
			log.Info(
				c.Request.Context(), "Fetched track play counts by period", zap.String("period", period),
				zap.Int("count", len(records)), zap.Int("limit", limit), zap.Int("offset", offset),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, records)
		},
	)

	// 获取按来源统计的播放次数
	r.GET(
		"/api/dashboard/play-counts-by-source", func(c *gin.Context) {
			ctx := c.Request.Context()

			sourceCounts, err := trackService.GetPlayCountsBySource(ctx)
			if err != nil {
				log.Error(ctx, "Failed to get play counts by source", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get play counts by source"})
				return
			}

			c.JSON(http.StatusOK, sourceCounts)
		},
	)

	// 获取热门专辑数据（按播放次数）
	r.GET(
		"/api/dashboard/top-albums", func(c *gin.Context) {
			ctx := c.Request.Context()

			// 获取时间范围参数，默认30天
			daysStr := c.DefaultQuery("days", "30")
			days, err := strconv.Atoi(daysStr)
			if err != nil {
				// 如果无法解析天数，默认使用30天
				days = 30
			}

			// 获取限制参数，默认10个
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if limit > 50 {
				limit = 50 // 限制最大数量
			}

			// 获取按播放次数统计的热门专辑
			albums, err := trackService.GetTopAlbumsByPlayCount(ctx, days, limit)
			if err != nil {
				log.Error(ctx, "Failed to get top albums by play count", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top albums by play count"})
				return
			}

			c.JSON(http.StatusOK, albums)
		},
	)

	// 获取热门流派数据（按播放次数和曲目数）
	genreService := genre.NewGenreService()
	r.GET(
		"/api/dashboard/top-genres", func(c *gin.Context) {
			ctx := c.Request.Context()

			// 获取限制参数，默认10个
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if limit > 50 {
				limit = 50 // 限制最大数量
			}

			// 获取热门流派的详细信息
			genres, err := genreService.GetTopGenresWithDetails(ctx, limit)
			if err != nil {
				log.Error(ctx, "Failed to get top genres with details", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top genres with details"})
				return
			}

			c.JSON(http.StatusOK, genres)
		},
	)

	// 获取未同步到Last.fm的播放记录（分页）
	r.GET(
		"/api/unscrobbled-records", func(c *gin.Context) {
			ctx := c.Request.Context()

			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetUnscrobbledRecordsWithPagination(ctx, limit, offset)
			if err != nil {
				log.Error(ctx, "Failed to get unscrobbled records", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unscrobbled records"})
				return
			}

			c.JSON(http.StatusOK, records)
		},
	)

	// 获取未同步到Last.fm的播放记录总数
	r.GET(
		"/api/unscrobbled-records/count", func(c *gin.Context) {
			ctx := c.Request.Context()

			count, err := trackService.GetUnscrobbledRecordsCount(ctx)
			if err != nil {
				log.Error(ctx, "Failed to get unscrobbled records count", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unscrobbled records count"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"count": count})
		},
	)

	// 同步选中的未同步记录到Last.fm
	r.POST(
		"/api/unscrobbled-records/sync", func(c *gin.Context) {
			ctx := c.Request.Context()

			var req struct {
				IDs []uint `json:"ids"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
				return
			}

			if len(req.IDs) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No record IDs provided"})
				return
			}

			// 将 req.IDs 从 []uint 转换为 []int64
			ids := make([]int64, len(req.IDs))
			for i, id := range req.IDs {
				ids[i] = int64(id)
			}

			// 调用logic层方法同步选中的记录
			successCount, failedRecords, err := trackService.SyncSelectedUnscrobbledRecords(ctx, ids)
			if err != nil {
				log.Error(ctx, "Failed to sync selected unscrobbled records", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync records"})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"success_count":  successCount,
					"failed_count":   len(failedRecords),
					"failed_records": failedRecords,
				},
			)
		},
	)

	// 处理收藏请求
	r.POST(
		"/api/favorite", func(c *gin.Context) {
			ctx := c.Request.Context()

			var req struct {
				Artist   string `json:"artist"`
				Album    string `json:"album"`
				Track    string `json:"track"`
				Source   string `json:"source"`
				Favorite bool   `json:"favorite"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
				return
			}

			// 验证必要参数
			if req.Artist == "" || req.Album == "" || req.Track == "" || req.Source == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "artist, album, track, and source are required"})
				return
			}

			// 调用logic层方法处理收藏逻辑
			appleMusicFav, lastFmFav, err := trackService.SetTrackFavorite(
				ctx, req.Artist, req.Album, req.Track, req.Source, req.Favorite, model.TrackMetadata{},
			)
			if err != nil {
				log.Error(ctx, "Failed to set track favorite", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set track favorite"})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"apple_music": appleMusicFav,
					"lastfm":      lastFmFav,
				},
			)
		},
	)

	// WebSocket endpoint
	r.GET(
		"/ws", func(c *gin.Context) {
			// 升级HTTP连接到WebSocket连接
			conn, err := websocket.UpgradeConnection(c.Writer, c.Request)
			if err != nil {
				log.Error(c.Request.Context(), "Failed to upgrade to WebSocket", zap.Error(err))
				return
			}

			// 添加连接到连接池
			websocket.AddClient(conn)

			// 启动goroutine处理WebSocket消息
			go websocket.HandleWebSocketMessages(conn)
		},
	)

	// Health check endpoint
	r.GET(
		"/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	return r
}

func StartHTTPServer(ctx context.Context, name string) {
	r := setupRouter(name)
	port := config.ConfigObj.HTTP.Port
	if port == "" {
		port = "8080" // Default port
	}
	log.Info(ctx, "Starting HTTP server on port", zap.String("port", port))
	err := r.Run(":" + port)
	if err != nil {
		panic(err)
	}
}

// FillInTrendCycle FillInTrendCycle
func FillInTrendCycle(rangeDays int) []string {
	now := time.Now()
	rangeDayList := make([]string, 0, rangeDays)
	rangeDayList = append(rangeDayList, now.AddDate(0, 0, -rangeDays).Format("2006-01-02"))
	start := now.AddDate(0, 0, -rangeDays)
	for start.Before(now) {
		start = start.AddDate(0, 0, 1)
		rangeDayList = append(rangeDayList, start.Format("2006-01-02"))
	}
	rangeDayList = append(rangeDayList, now.AddDate(0, 0, 1).Format("2006-01-02"))
	return rangeDayList
}
