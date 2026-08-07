package api

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/ai"
	"github.com/vincentchyu/sonic-lens/core/artwork"
	"github.com/vincentchyu/sonic-lens/core/log"
	coreredis "github.com/vincentchyu/sonic-lens/core/redis"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/core/websocket"
	artistprofilelogic "github.com/vincentchyu/sonic-lens/internal/logic/artistprofile"
	artworklogic "github.com/vincentchyu/sonic-lens/internal/logic/artwork"
	"github.com/vincentchyu/sonic-lens/internal/logic/genre"
	"github.com/vincentchyu/sonic-lens/internal/logic/insight"
	musicbrainzlogic "github.com/vincentchyu/sonic-lens/internal/logic/musicbrainz"
	pendingalbumlogic "github.com/vincentchyu/sonic-lens/internal/logic/pendingalbum"
	"github.com/vincentchyu/sonic-lens/internal/logic/track"
	"github.com/vincentchyu/sonic-lens/internal/model"
	"github.com/vincentchyu/sonic-lens/internal/scrobbler"
)

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

func setupRouter(name string) *gin.Engine {
	r := gin.Default()

	// Add OpenTelemetry middleware
	r.Use(
		otelgin.Middleware(
			name,
			otelgin.WithTracerProvider(telemetry.GetTracerProvider()),
			otelgin.WithMeterProvider(telemetry.GetMeterProvider()),
			otelgin.WithPropagators(otel.GetTextMapPropagator()),
		),
		func(c *gin.Context) {
			traceID := trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()
			c.Header("Trace-Id", traceID)
			c.Next()
		},
	)
	// static files
	r.StaticFile("/static/chartjs-adapter-date-fns.bundle.min.js", "./static/chartjs-adapter-date-fns.bundle.min.js")
	r.StaticFile("/static/full.min.css", "./static/full.min.css")
	r.StaticFile("/static/html2canvas.min.js", "./static/html2canvas.min.js")
	r.StaticFile("/static/lrc-utils.js", "./static/lrc-utils.js")
	r.StaticFile("/static/3.4.17", "./static/3.4.17")
	r.StaticFile("/static/chart.js", "./static/chart.js")
	r.StaticFile("/static/logo.svg", "./static/logo.svg")
	r.StaticFile("/static/logo_black.svg", "./static/logo_black.svg")
	r.StaticFile("/static/logo_all.svg", "./static/logo_all.svg")
	r.StaticFile("/static/logo_all_black.svg", "./static/logo_all_black.svg")
	r.Static("/static/admin", "./static/admin")

	artworkService := artworklogic.NewService()
	artistProfileService := artistprofilelogic.NewService()
	r.GET(
		"/api/artwork/resolve", redisCache(10*time.Minute), func(c *gin.Context) {
			albumID := parseInt64Query(c, "album_id")
			if albumID <= 0 {
				albumID = parseInt64Query(c, "albumID")
			}
			result, err := artworkService.Resolve(
				c.Request.Context(),
				artworklogic.ResolveArtworkInput{
					AlbumID:     albumID,
					AlbumArtist: c.Query("albumArtist"),
					Artist:      c.Query("artist"),
					Album:       c.Query("album"),
					ArtworkKey:  c.Query("artworkKey"),
				},
			)
			if err != nil {
				log.Error(c.Request.Context(), "resolve artwork err", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK,
				gin.H{
					"exists":               result.Exists,
					"cover_art_url":        result.CoverArtURL,
					"cover_art_object_key": result.CoverArtObjectKey,
				},
			)
		},
	)

	r.GET(
		"/api/artwork/:key", func(c *gin.Context) {
			entry, ok := artwork.DefaultStore.Get(c.Param("key"))
			if !ok || len(entry.Data) == 0 {
				c.Status(http.StatusNotFound)
				return
			}

			if entry.MimeType != "" {
				c.Header("Content-Type", entry.MimeType)
			} else {
				c.Header("Content-Type", "application/octet-stream")
			}
			c.Header("Cache-Control", "public, max-age=3600")
			_, _ = c.Writer.Write(entry.Data)
		},
	)
	renderAdminPage := func(c *gin.Context) {
		// 后台总入口由主模板和局部模板组合，避免单文件继续膨胀。
		tmplPath := filepath.Join("templates", "admin.html")
		partialPaths, err := filepath.Glob(filepath.Join("templates", "admin", "*.html"))
		if err != nil {
			log.Error(c.Request.Context(), "解析后台入口模板片段路径失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
			return
		}
		templatePaths := append([]string{tmplPath}, partialPaths...)
		tmpl, err := template.New("admin.html").ParseFiles(templatePaths...)
		if err != nil {
			log.Error(c.Request.Context(), "Failed to parse template", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(c.Writer, nil); err != nil {
			log.Error(c.Request.Context(), "Failed to execute template", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
			return
		}
	}

	// 首页保留历史入口，/admin 作为后台总入口的语义化别名。
	r.GET("/", renderAdminPage)
	r.GET("/admin", renderAdminPage)

	// 全屏歌词简化页
	r.GET(
		"/lyrics-live", func(c *gin.Context) {
			tmplPath := filepath.Join("templates", "pages", "lyrics_live.html")
			tmpl, err := template.New("lyrics_live.html").ParseFiles(tmplPath)
			if err != nil {
				log.Error(c.Request.Context(), "Failed to parse template", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load template"})
				return
			}

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
	musicbrainzService := musicbrainzlogic.NewService()
	pendingAlbumService := pendingalbumlogic.NewService()
	// AI 歌词解析服务
	insightService, insightErr := insight.NewService()
	if insightErr != nil {
		// 记录日志但不阻断整个服务启动，前端调用时再返回错误
		log.Warn(context.Background(), "初始化歌词解析服务失败，将暂时无法使用 AI 歌词解析功能", zap.Error(insightErr))
	}
	registerInsightFeedbackRoutes(r, insightService)
	r.GET(
		"/api/track-play-counts", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			keyword := c.Query("keyword")

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetTrackPlayCounts(c.Request.Context(), limit, offset, keyword)
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
				tmplPath := filepath.Join("templates", "pages", "track_play_counts.html")
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

	registerAIRoutes(r, insightService)

	// 获取某首歌已有的 AI 解析结果 (仅查询)
	r.GET(
		"/api/track-insight", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 歌词解析服务未初始化"})
				return
			}

			artist := c.Query("artist")
			album := c.Query("album")
			track := c.Query("track")
			trackNumber := parseInt8Query(c, "trackNumber")
			discNumber := parseInt8Query(c, "discNumber")

			if artist == "" || track == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "参数不足"})
				return
			}
			insights, err := insightService.GetInsightOnly(
				c.Request.Context(), artist, album, track, trackNumber, discNumber,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK,
				gin.H{
					"insights":               insights,
					"recommended_insight_id": recommendedTrackInsightID(insights),
				},
			)
		},
	)

	// 获取某张专辑已有的 AI 解析结果 (仅查询)
	r.GET(
		"/api/album-insight", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 专辑解析服务未初始化"})
				return
			}

			albumIDStr := c.Query("albumID")
			if albumIDStr == "" {
				albumIDStr = c.Query("album_id")
			}
			albumID, err := strconv.ParseInt(albumIDStr, 10, 64)
			if err != nil || albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的专辑 ID"})
				return
			}

			insights, err := insightService.GetAlbumInsightOnly(c.Request.Context(), albumID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK,
				gin.H{
					"insights": sanitizeAlbumInsightsForClient(
						insights, shouldHideAlbumInsightDebugData(c),
					),
					"recommended_insight_id": recommendedAlbumInsightID(insights),
				},
			)
		},
	)

	// Get play for a specific track
	r.GET(
		"/api/track", redisCache(1*time.Minute), func(c *gin.Context) {
			artist := c.Query("artist")
			album := c.Query("album")
			trackName := c.Query("trackName")
			trackNumber := parseInt8Query(c, "trackNumber")
			discNumber := parseInt8Query(c, "discNumber")

			if artist == "" || album == "" || trackName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "artist, album, and trackName are required"})
				return
			}

			record, err := trackService.GetTrackByIdentity(
				c.Request.Context(), artist, album, trackName, trackNumber, discNumber,
			)
			if err != nil {
				if err.Error() == "record not found" {
					c.JSON(http.StatusOK, gin.H{"play_count": 0})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 通过 TrackAlbum 查询 album_id
			albumID := int64(0)
			if record.ID > 0 {
				if ta, err := trackService.GetTrackAlbumByTrackID(c.Request.Context(), record.ID); err == nil {
					albumID = ta.AlbumID
				}
			}

			c.JSON(
				http.StatusOK, gin.H{
					"id":                 record.ID,
					"artist":             record.Artist,
					"album":              record.Album,
					"track":              record.Track,
					"play_count":         record.PlayCount,
					"album_id":           albumID,
					"duration":           record.Duration,
					"track_number":       record.TrackNumber,
					"disc_number":        record.DiscNumber,
					"genre":              record.Genre,
					"is_apple_music_fav": record.IsAppleMusicFav,
					"is_last_fm_fav":     record.IsLastFmFav,
					"source":             record.Source,
					"updated_at":         record.UpdatedAt,
				},
			)
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
			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery(
					"analysis_target_type", string(common.AnalysisTargetTypeTrack),
				),
			)
			if limit > 100 {
				limit = 100
			}

			insights, total, err := insightService.GetAllInsights(
				c.Request.Context(), limit, offset, keyword, targetType,
			)
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
			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery(
					"analysis_target_type", string(common.AnalysisTargetTypeTrack),
				),
			)

			if err := insightService.ToggleInsightStatus(c.Request.Context(), targetType, insightID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok", "analysis_target_type": targetType})
		},
	)

	// 获取单条解析详情
	r.GET(
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
			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery(
					"analysis_target_type", string(common.AnalysisTargetTypeTrack),
				),
			)

			insight, err := insightService.GetInsightDetail(c.Request.Context(), targetType, insightID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "解析记录不存在"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, insight)
		},
	)

	// 直接删除解析记录及其关联流水
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
			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery(
					"analysis_target_type", string(common.AnalysisTargetTypeTrack),
				),
			)

			if err := insightService.DeleteInsight(c.Request.Context(), targetType, insightID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "解析记录不存在"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if client := coreredis.GetRedisClient(); client != nil {
				cacheKey := buildRedisCacheKey(
					http.MethodGet,
					"/api/insights/"+idStr+"/logs",
					url.Values{"analysis_target_type": []string{string(targetType)}},
					c.GetHeader("Accept"),
				)
				if err := client.Del(c.Request.Context(), cacheKey).Err(); err != nil {
					log.Warn(
						c.Request.Context(), "清理音眸日志缓存失败", zap.Error(err), zap.String("cache_key", cacheKey),
					)
				}
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok", "analysis_target_type": targetType})
		},
	)

	// 获取某次解析关联的 LLM 调用流水
	r.GET(
		"/api/insights/:id/logs", redisCache(72*time.Hour), func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			// 曲目调用日志按 artist + album + track 关联，这里先通过 ID 查到目标曲目信息
			idStr := c.Param("id")
			insightID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || insightID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
				return
			}
			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery(
					"analysis_target_type", string(common.AnalysisTargetTypeTrack),
				),
			)

			logs, err := insightService.GetInsightCallLogs(c.Request.Context(), targetType, insightID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "解析记录不存在"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"logs": logs, "analysis_target_type": targetType})
		},
	)

	// 获取某张专辑对应的 LLM 调用流水
	r.GET(
		"/api/album-insights/:id/logs", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			idStr := c.Param("id")
			albumID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
				return
			}

			logs, err := insightService.GetAlbumCallLogs(c.Request.Context(), albumID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"logs": logs})
		},
	)

	// 获取歌词数据（优先查库，没有则调用 lrcapi 等）
	r.GET(
		"/api/track-lyrics", redisCache(5*time.Minute), func(c *gin.Context) {
			artist := c.Query("artist")
			album := c.Query("album")
			track := c.Query("track")
			trackNumber := parseInt8Query(c, "trackNumber")
			discNumber := parseInt8Query(c, "discNumber")

			if artist == "" || track == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必需参数 artist 和 track"})
				return
			}

			lyricsData, err := insightService.GetLyrics(
				c.Request.Context(), artist, album, track, trackNumber, discNumber,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			lrcContent := lyricsData.LyricsOriginal
			hasLRC := lyricsData.Synced

			c.JSON(
				http.StatusOK, gin.H{
					"lyrics":  lrcContent,
					"has_lrc": hasLRC,
				},
			)
		},
	)

	// --- MusicBrainz 相关接口 ---

	r.GET(
		"/api/pending-albums", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
			if limit > 200 {
				limit = 200
			}
			filter := c.Query("filter")
			groups, err := pendingAlbumService.GetPendingAlbumGroups(c.Request.Context(), limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if filter == "uncreated" {
				var filtered []*model.PendingAlbumGroup
				for _, g := range groups {
					if g.OpenWorkItemID == 0 {
						filtered = append(filtered, g)
					}
				}
				groups = filtered
			}
			c.JSON(http.StatusOK, gin.H{"groups": groups})
		},
	)

	r.POST(
		"/api/pending-albums/work-items", func(c *gin.Context) {
			var req struct {
				IdentityKey string `json:"identity_key"`
			}
			if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.IdentityKey) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "identity_key is required"})
				return
			}
			item, err := pendingAlbumService.CreateOrGetPendingAlbumWorkItem(c.Request.Context(), req.IdentityKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, item)
		},
	)

	r.GET(
		"/api/pending-albums/work-items", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			keyword := c.Query("keyword")
			statusGroup := c.Query("status_group")

			if limit > 100 {
				limit = 100
			}

			items, total, err := pendingAlbumService.ListWorkItems(
				c.Request.Context(), limit, offset, keyword, statusGroup,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK, gin.H{
					"items":  items,
					"total":  total,
					"limit":  limit,
					"offset": offset,
				},
			)
		},
	)

	r.GET(
		"/api/pending-albums/work-items/:id", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			detail, err := pendingAlbumService.GetPendingAlbumWorkItemDetail(c.Request.Context(), workItemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, detail)
		},
	)

	r.POST(
		"/api/pending-albums/work-items/:id/refresh-context", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			if _, err := pendingAlbumService.RefreshPendingAlbumWorkItemContext(
				c.Request.Context(), workItemID,
			); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			detail, err := pendingAlbumService.GetPendingAlbumWorkItemDetail(c.Request.Context(), workItemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, detail)
		},
	)

	r.GET(
		"/api/pending-albums/work-items/:id/musicbrainz/candidates", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			candidates, err := pendingAlbumService.SearchPendingAlbumMBReleases(c.Request.Context(), workItemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, candidates)
		},
	)

	r.POST(
		"/api/pending-albums/work-items/:id/musicbrainz/link", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			var req struct {
				ReleaseMBID int64  `json:"release_mb_id"`
				MBID        string `json:"mbid"`
			}
			if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.MBID) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "mbid is required"})
				return
			}
			if err := pendingAlbumService.LinkPendingAlbumMBRelease(
				c.Request.Context(),
				workItemID,
				req.ReleaseMBID,
				req.MBID,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	r.GET(
		"/api/pending-albums/work-items/:id/musicbrainz/preview", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			releaseMBID, _ := strconv.ParseInt(c.Query("release_mb_id"), 10, 64)
			mbid := c.Query("mbid")
			forceRefresh := c.Query("force_refresh") == "true" || c.Query("force_refresh") == "1"

			preview, err := pendingAlbumService.PreviewPendingAlbumMBMaintenance(
				c.Request.Context(),
				workItemID,
				releaseMBID,
				mbid,
				forceRefresh,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, preview)
		},
	)

	r.POST(
		"/api/pending-albums/work-items/:id/musicbrainz/draft", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			var draft pendingalbumlogic.PendingAlbumMaintenancePreview
			if err := c.ShouldBindJSON(&draft); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := pendingAlbumService.SavePendingAlbumStagingDraft(
				c.Request.Context(), workItemID, &draft,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "草稿成功保存", "status": "staged"})
		},
	)

	r.POST(
		"/api/pending-albums/work-items/:id/deep-maintenance", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}
			preview, err := pendingAlbumService.PreviewPendingAlbumMBMaintenance(
				c.Request.Context(),
				workItemID,
				0,
				"",
				false,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK, gin.H{
					"status":  "staged",
					"message": "已生成草稿快照，请在预审与差异对比弹窗中审核确认后提交应用。",
					"preview": preview,
				},
			)
		},
	)

	r.POST(
		"/api/pending-albums/work-items/:id/manual-maintenance", func(c *gin.Context) {
			workItemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if workItemID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work item id"})
				return
			}

			var req pendingalbumlogic.ManualPendingAlbumInput
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "params error"})
				return
			}

			report, err := pendingAlbumService.ManualMaintainPendingAlbumWorkItem(
				c.Request.Context(),
				workItemID,
				req,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok", "report": report})
		},
	)

	// 1. 搜索补全（初选候选）
	r.GET(
		"/api/musicbrainz/search-releases/:album_id", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("album_id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			if err := musicbrainzService.SearchAndCacheReleases(c.Request.Context(), albumID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 获取已缓存的候选结果
	r.GET(
		"/api/musicbrainz/candidates/:album_id", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("album_id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			candidates, err := musicbrainzService.GetReleasesByAlbumID(c.Request.Context(), albumID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, candidates)
		},
	)

	// 2. 确认关联（用户选定 MBID）
	r.POST(
		"/api/musicbrainz/link-album", func(c *gin.Context) {
			var req struct {
				AlbumID     int64  `json:"album_id"`
				ReleaseMBID int64  `json:"release_mb_id"`
				MBID        string `json:"mbid"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "params error"})
				return
			}
			if err := musicbrainzService.LinkAlbumToMBID(
				c.Request.Context(),
				req.AlbumID,
				req.ReleaseMBID,
				req.MBID,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 3. 深度维护与轨道修正（精选）
	r.POST(
		"/api/musicbrainz/deep-maintenance/:album_id", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("album_id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			if err := musicbrainzService.DeepingMaintenance(c.Request.Context(), albumID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 4. 正式专辑 MB 精选维护预审与对比
	r.GET(
		"/api/albums/:id/musicbrainz/preview", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			releaseMBID, _ := strconv.ParseInt(c.Query("release_mb_id"), 10, 64)
			mbid := c.Query("mbid")
			forceRefresh := c.Query("force_refresh") == "1"

			preview, err := pendingAlbumService.PreviewAlbumMBMaintenance(
				c.Request.Context(), albumID, releaseMBID, mbid, forceRefresh,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, preview)
		},
	)

	// 5. 正式专辑 MB 精选维护提交落库
	r.POST(
		"/api/albums/:id/musicbrainz/apply-maintenance", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			var input pendingalbumlogic.ManualPendingAlbumInput
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input payload"})
				return
			}
			if err := pendingAlbumService.ApplyAlbumMBMaintenance(c.Request.Context(), albumID, &input); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 6. 正式专辑 MB 精选维护保存草稿
	r.POST(
		"/api/albums/:id/musicbrainz/draft", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			var draft pendingalbumlogic.PendingAlbumMaintenancePreview
			if err := c.ShouldBindJSON(&draft); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid draft payload"})
				return
			}
			if err := pendingAlbumService.SaveAlbumStagingDraft(c.Request.Context(), albumID, &draft); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

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

			dateTrendData, hourlyTrendData, err := trackService.GetPlayTrendByDays(ctx, rangeDays)
			if err != nil {
				log.Warn(ctx, "Failed to get trend data from stat table, falling back to realtime scan", zap.Error(err))
				recordMap, fallbackErr := trackService.GetRecentPlayRecordsByDays(ctx, rangeDays)
				if fallbackErr != nil {
					log.Error(ctx, "Failed to get recent play records", zap.Error(fallbackErr))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent play records"})
					return
				}
				dateTrendData = make(map[string]int)
				hourlyTrendData = make(map[string]*model.HourlyPlayTrendData)
				for _, trendCycle := range fillInTrendCycle {
					if records, ok := recordMap[trendCycle]; ok {
						for _, record := range records {
							dateStr := record.PlayTime.Format("2006-01-02")
							hour := record.PlayTime.Hour()
							dateTrendData[dateStr]++
							if _, exists := hourlyTrendData[dateStr]; !exists {
								hourlyTrendData[dateStr] = &model.HourlyPlayTrendData{
									Date:   dateStr,
									Total:  0,
									Hourly: make(map[int]int),
								}
							}
							hourlyTrendData[dateStr].Hourly[hour]++
							hourlyTrendData[dateStr].Total++
						}
					}
				}
			}

			for _, trendCycle := range fillInTrendCycle {
				if _, ok := dateTrendData[trendCycle]; !ok {
					dateTrendData[trendCycle] = 0
				}
				if _, ok := hourlyTrendData[trendCycle]; !ok {
					hourlyTrendData[trendCycle] = &model.HourlyPlayTrendData{
						Date:   trendCycle,
						Total:  0,
						Hourly: map[int]int{},
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

	// 获取艺术家资料列表
	r.GET(
		"/api/artist-profiles", func(c *gin.Context) {
			ctx := c.Request.Context()
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			result, err := artistProfileService.ListProfiles(ctx, limit, offset, c.Query("keyword"))
			if err != nil {
				log.Error(ctx, "获取艺术家资料列表失败", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "获取艺术家资料列表失败"})
				return
			}

			c.JSON(http.StatusOK, result)
		},
	)

	// 获取热门艺术家候选，用作资料源选择
	r.GET(
		"/api/artist-profiles/top-artists", func(c *gin.Context) {
			ctx := c.Request.Context()
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

			items, err := artistProfileService.ListTopArtistSources(ctx, limit)
			if err != nil {
				log.Error(ctx, "获取热门艺术家候选失败", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "获取热门艺术家候选失败"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"items": items})
		},
	)

	// 上传艺术家头像并写入资料表
	r.POST(
		"/api/artist-profiles/avatar", func(c *gin.Context) {
			ctx := c.Request.Context()
			var req struct {
				ArtistName string `json:"artist_name"`
				Data       string `json:"data"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}

			profile, err := artistProfileService.UploadAvatar(ctx, req.ArtistName, req.Data)
			if err != nil {
				status := http.StatusInternalServerError
				if strings.Contains(err.Error(), "required") ||
					strings.Contains(err.Error(), "invalid") ||
					strings.Contains(err.Error(), "unsupported") ||
					strings.Contains(err.Error(), "decode") {
					status = http.StatusBadRequest
				}
				log.Error(
					ctx,
					"上传艺术家头像失败",
					zap.String("artist_name", strings.TrimSpace(req.ArtistName)),
					zap.Error(err),
				)
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(
				http.StatusOK,
				gin.H{
					"profile": artistprofilelogic.ListItem{
						ID:                  profile.ID,
						ArtistName:          profile.ArtistName,
						NormalizedArtistKey: profile.NormalizedArtistKey,
						AvatarURL:           profile.AvatarURL,
						AvatarMime:          profile.AvatarMime,
						AvatarObjectKey:     profile.AvatarObjectKey,
						CreatedAt:           profile.CreatedAt.Format("2006-01-02 15:04:05"),
						UpdatedAt:           profile.UpdatedAt.Format("2006-01-02 15:04:05"),
					},
				},
			)
		},
	)
	// 获取专辑详情（含歌曲列表）
	r.GET(
		"/api/albums/:id", func(c *gin.Context) {
			// "/api/albums/:id", redisCache(2*time.Minute), func(c *gin.Context) {
			idStr := c.Param("id")
			albumID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的专辑 ID"})
				return
			}

			detail, err := trackService.GetAlbumDetail(c.Request.Context(), albumID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, detail)
		},
	)

	// 解除 TrackAlbum 关联（人工修复用）
	r.POST(
		"/api/track-album/unlink", func(c *gin.Context) {
			var req struct {
				TrackID int64 `json:"track_id"`
				AlbumID int64 `json:"album_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}
			if req.TrackID <= 0 || req.AlbumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "track_id and album_id are required"})
				return
			}

			if err := trackService.DeleteTrackAlbumLink(c.Request.Context(), req.TrackID, req.AlbumID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 播放统计页面
	r.GET(
		"/playCounts", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			keyword := c.Query("keyword")

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetTrackPlayCounts(c.Request.Context(), limit, offset, keyword)
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
				tmplPath := filepath.Join("templates", "pages", "track_play_counts.html")
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
			keyword := c.Query("keyword")

			if limit > 100 {
				limit = 100 // Limit max records per page
			}

			records, err := trackService.GetTrackPlayCountsByPeriod(c.Request.Context(), limit, offset, period, keyword)
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

	// 获取热门曲目数据（按播放次数）
	r.GET(
		"/api/dashboard/top-tracks", func(c *gin.Context) {
			ctx := c.Request.Context()

			daysStr := c.DefaultQuery("days", "30")
			days, err := strconv.Atoi(daysStr)
			if err != nil {
				days = 30
			}

			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if limit > 50 {
				limit = 50
			}

			tracks, err := trackService.GetTopTracksByPlayCount(ctx, days, limit)
			if err != nil {
				log.Error(ctx, "获取热门曲目失败", zap.Error(err), zap.Int("days", days), zap.Int("limit", limit))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top tracks"})
				return
			}

			c.JSON(http.StatusOK, tracks)
		},
	)

	// 获取专辑列表（支持分页、搜索、自然排序）
	r.GET(
		"/api/library/sync", func(c *gin.Context) {
			ctx := c.Request.Context()

			sinceVersion, _ := strconv.ParseInt(c.DefaultQuery("since_version", "0"), 10, 64)
			delta, err := trackService.GetLibrarySyncDelta(ctx, sinceVersion)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"sync_version":      delta.Version,
					"generated_at":      time.Now().UTC(),
					"albums":            delta.Albums,
					"tracks":            delta.Tracks,
					"deleted_album_ids": delta.DeletedAlbumIDs,
					"deleted_track_ids": delta.DeletedTrackIDs,
				},
			)
		},
	)

	r.GET(
		"/api/albums", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			keyword := c.Query("keyword")

			if limit > 100 {
				limit = 100
			}

			albums, err := trackService.GetAlbums(c.Request.Context(), limit, offset, keyword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			total, _ := trackService.GetAlbumsCount(c.Request.Context(), keyword)

			c.JSON(
				http.StatusOK, gin.H{
					"albums": albums,
					"total":  total,
					"limit":  limit,
					"offset": offset,
				},
			)
		},
	)

	r.POST(
		"/api/albums/:id/lock", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			album, err := model.GetAlbum(c.Request.Context(), albumID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
				return
			}
			if album.SyncStatus != 3 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "只有进入精选维护完成(status=3)的专辑才可以锁定"})
				return
			}
			if err := model.UpdateAlbumSyncStatus(c.Request.Context(), albumID, 4); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	r.POST(
		"/api/albums/:id/unlock", func(c *gin.Context) {
			albumID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if albumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid album_id"})
				return
			}
			if err := model.UpdateAlbumSyncStatus(c.Request.Context(), albumID, 3); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	// 获取曲目列表（按专辑排序，支持分页、搜索）
	r.GET(
		"/api/tracks", func(c *gin.Context) {
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			keyword := c.Query("keyword")

			if limit > 100 {
				limit = 100
			}

			tracks, err := trackService.GetTracksOrderedByAlbum(c.Request.Context(), limit, offset, keyword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			total, _ := trackService.GetTracksOrderedByAlbumCount(c.Request.Context(), keyword)

			c.JSON(
				http.StatusOK, gin.H{
					"tracks": tracks,
					"total":  total,
					"limit":  limit,
					"offset": offset,
				},
			)
		},
	)

	// 获取热门流派数据（按播放次数和曲目数）
	genreService := genre.NewGenreService()
	r.GET(
		"/api/dashboard/top-genres", redisCache(72*time.Hour), func(c *gin.Context) {
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
				Artist      string `json:"artist"`
				Album       string `json:"album"`
				Track       string `json:"track"`
				TrackNumber int8   `json:"track_number"`
				DiscNumber  int8   `json:"disc_number"`
				Source      string `json:"source"`
				Favorite    bool   `json:"favorite"`
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
			var (
				currentPlaying *websocket.WsInfo
				ok             bool
			)

			if req.TrackNumber <= 0 || req.DiscNumber <= 0 {
				if currentPlaying, ok = scrobbler.GetCurrentPlayingTrack(common.PlayerType(req.Source)); ok {
					if currentPlaying.Data.Artist == req.Artist &&
						currentPlaying.Data.Album == req.Album &&
						currentPlaying.Data.Title == req.Track {
						if req.TrackNumber <= 0 {
							req.TrackNumber = currentPlaying.Data.TrackNumber
						}
						if req.DiscNumber <= 0 {
							req.DiscNumber = currentPlaying.Data.DiscNumber
						}
					}
				}
			}
			var metadata model.TrackMetadata
			metadata.TrackNumber = req.TrackNumber
			metadata.DiscNumber = req.DiscNumber
			if currentPlaying != nil && currentPlaying.Data.PlayerInfoHandler != nil {
				metadata.AlbumArtist = currentPlaying.Data.PlayerInfoHandler.GetAlbumArtist()
				metadata.Duration = currentPlaying.Data.PlayerInfoHandler.GetDuration()
				metadata.Genre = currentPlaying.Data.PlayerInfoHandler.GetGenre()
				metadata.Composer = currentPlaying.Data.PlayerInfoHandler.GetComposer()
				metadata.ReleaseDate = currentPlaying.Data.PlayerInfoHandler.GetReleaseDate()
				metadata.OriginalReleaseDate = currentPlaying.Data.PlayerInfoHandler.GetOriginalReleaseDate()
				metadata.MusicBrainzID = currentPlaying.Data.PlayerInfoHandler.GetMusicBrainzID()
				metadata.Source = currentPlaying.Data.PlayerInfoHandler.GetSource()
				metadata.BundleID = currentPlaying.Data.PlayerInfoHandler.GetBundleID()
				metadata.UniqueID = currentPlaying.Data.PlayerInfoHandler.GetUniqueID()
				metadata.PlayerType = req.Source
				metadata.Confidence = currentPlaying.Data.Confidence
			}
			// 调用logic层方法处理收藏逻辑
			favoriteProjection, err := trackService.SetTrackFavorite(
				ctx,
				req.Artist,
				req.Album,
				req.Track,
				req.Source,
				req.Favorite,
				metadata,
			)

			if err != nil {
				log.Error(ctx, "Failed to set track favorite", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set track favorite"})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"apple_music":       favoriteProjection.AppleMusic,
					"lastfm":            favoriteProjection.LastFM,
					"apple_music_state": favoriteProjection.AppleMusicState,
					"lastfm_state":      favoriteProjection.LastFMState,
					"favorite_state":    favoriteProjection.FavoriteState,
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
			telemetry.GoOnlySafe(
				c.Request.Context(), func(context.Context) {
					websocket.HandleWebSocketMessages(conn)
				},
			)
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

func parseInt8Query(c *gin.Context, key string) int8 {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return int8(v)
}

func parsePathInt64(c *gin.Context, paramName, badRequestMessage string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(paramName))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": badRequestMessage})
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": badRequestMessage})
		return 0, false
	}
	return value, true
}

func parseInt64Query(c *gin.Context, key string) int64 {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseQueryIntWithDefault(c *gin.Context, key string, defaultValue int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

func isAISelectionBadRequest(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "不支持的 AI 平台") ||
		strings.Contains(msg, "不支持的旧模型平台参数") ||
		strings.Contains(msg, "不支持模型") ||
		strings.Contains(msg, "未配置默认模型") ||
		strings.Contains(msg, "未配置:") ||
		strings.Contains(msg, "不能为空")
}

const clientTerminalHeader = "X-SonicLens-Terminal"

func clientTerminal(c *gin.Context) string {
	terminal := strings.TrimSpace(strings.ToLower(c.GetHeader(clientTerminalHeader)))
	if terminal == "" {
		return "web"
	}
	return terminal
}

func shouldHideAlbumInsightDebugData(c *gin.Context) bool {
	switch clientTerminal(c) {
	case "web":
		return false
	default:
		return true
	}
}

func sanitizeAlbumInsightsForClient(insights []*model.AlbumInsight, hideDebugData bool) []*model.AlbumInsight {
	if !hideDebugData {
		return insights
	}

	sanitized := make([]*model.AlbumInsight, len(insights))
	for i, insight := range insights {
		if insight == nil {
			continue
		}
		copyInsight := *insight
		copyInsight.Metadata = ""
		sanitized[i] = &copyInsight
	}
	return sanitized
}

func recommendedTrackInsightID(insights []*insight.InsightWithScore) int64 {
	if len(insights) == 0 || insights[0] == nil || insights[0].TrackInsight == nil {
		return 0
	}
	return insights[0].TrackInsight.ID
}

func recommendedTrackInsightModelID(insights []*model.TrackInsight) int64 {
	if len(insights) == 0 || insights[0] == nil {
		return 0
	}
	return insights[0].ID
}

func recommendedAlbumInsightID(insights []*model.AlbumInsight) int64 {
	if len(insights) == 0 || insights[0] == nil {
		return 0
	}
	return insights[0].ID
}

func recommendedInsightListItemID(items []*model.InsightListItem) int64 {
	if len(items) == 0 || items[0] == nil {
		return 0
	}
	return items[0].ID
}

type aiRouteService interface {
	GetAvailableAIPlatforms() []ai.PlatformOption
	GetPlatformModels(ctx context.Context, platform common.AIModelPlatform) ([]ai.ModelOption, error)
	GetOrCreateTrackInsight(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
		provider, model, legacyModelType string,
	) ([]*model.TrackInsight, bool, error)
	GetOrCreateAlbumInsight(
		ctx context.Context, albumID int64, force bool, provider, model, legacyModelType string,
	) ([]*model.AlbumInsight, bool, error)
	// Deprecated: 流式接口已废弃，后续不再维护与优化 GetOrCreateInsightStream 获取流式解析结果
	GetOrCreateInsightStream(
		ctx context.Context, artist, album, track string, trackNumber, discNumber int8, force bool,
		provider, model, legacyModelType string,
	) (<-chan string, bool, error)
	CreateInsightJob(
		ctx context.Context, req insight.CreateInsightJobRequest,
	) (*model.InsightJob, bool, error)
	GetInsightJob(ctx context.Context, jobID string) (*model.InsightJob, error)
	GetInsightJobCallLogs(ctx context.Context, jobID string) ([]*model.LLMCallLog, error)
	ListInsightJobs(ctx context.Context, query model.InsightJobListQuery) ([]*model.InsightJob, int64, error)
	UpdateInsightJobLiveActivityToken(ctx context.Context, jobID, token string) (*model.InsightJob, error)
	CancelInsightJob(ctx context.Context, jobID string) (*model.InsightJob, error)
	RetryInsightJob(ctx context.Context, jobID string) (*model.InsightJob, bool, error)
	DeleteInsightJob(ctx context.Context, jobID string) error
}

// registerAIRoutes 注册 AI 模型目录与解析相关路由，便于复用和单测注入。
func registerAIRoutes(r gin.IRoutes, insightService aiRouteService) {
	// 获取当前系统支持的 AI 平台列表
	r.GET(
		"/api/ai-models", redisCache(), func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusOK, gin.H{"platforms": []ai.PlatformOption{}})
				return
			}
			platforms := insightService.GetAvailableAIPlatforms()
			c.JSON(http.StatusOK, gin.H{"platforms": platforms})
		},
	)

	// 获取某个平台支持的模型列表
	r.GET(
		"/api/ai-models/:platform/models", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusOK, gin.H{"models": []ai.ModelOption{}})
				return
			}

			platform := common.ParseAIModelPlatform(c.Param("platform"))
			if !platform.IsValid() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 AI 平台"})
				return
			}

			models, err := insightService.GetPlatformModels(c.Request.Context(), platform)
			if err != nil {
				status := http.StatusInternalServerError
				if isAISelectionBadRequest(err) {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"models": models})
		},
	)

	r.POST(
		"/api/insight-jobs", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			var req struct {
				TargetType     string `json:"target_type"`
				Artist         string `json:"artist"`
				Album          string `json:"album"`
				Track          string `json:"track"`
				TrackNumber    int8   `json:"track_number"`
				DiscNumber     int8   `json:"disc_number"`
				AlbumID        int64  `json:"album_id"`
				Provider       string `json:"provider"`
				Model          string `json:"model"`
				ModelType      string `json:"modelType"`
				ClientPlatform string `json:"client_platform"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			job, existing, err := insightService.CreateInsightJob(
				c.Request.Context(),
				insight.CreateInsightJobRequest{
					TargetType:      common.ParseAnalysisTargetType(req.TargetType),
					Artist:          req.Artist,
					Album:           req.Album,
					Track:           req.Track,
					TrackNumber:     req.TrackNumber,
					DiscNumber:      req.DiscNumber,
					AlbumID:         req.AlbumID,
					Provider:        req.Provider,
					Model:           req.Model,
					LegacyModelType: req.ModelType,
					ClientPlatform:  req.ClientPlatform,
				},
			)
			if err != nil {
				status := http.StatusInternalServerError
				if isAISelectionBadRequest(err) {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"job": job, "existing": existing})
		},
	)

	r.GET(
		"/api/insight-jobs", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			statusText := strings.TrimSpace(c.Query("status"))
			targetTypeText := strings.TrimSpace(c.Query("analysis_target_type"))

			query := model.InsightJobListQuery{
				Limit:   limit,
				Offset:  offset,
				Keyword: strings.TrimSpace(c.Query("keyword")),
			}
			if statusText != "" {
				status := common.ParseInsightJobPhase(statusText)
				if string(status) != strings.ToLower(statusText) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务状态"})
					return
				}
				query.Status = status
				query.HasStatus = true
			}
			if targetTypeText != "" {
				targetType := common.ParseAnalysisTargetType(targetTypeText)
				if string(targetType) != strings.ToLower(targetTypeText) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分析对象类型"})
					return
				}
				query.TargetType = targetType
				query.HasTargetType = true
			}

			jobs, total, err := insightService.ListInsightJobs(c.Request.Context(), query)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"jobs": jobs, "total": total})
		},
	)

	r.GET(
		"/api/insight-jobs/:id", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			job, err := insightService.GetInsightJob(c.Request.Context(), c.Param("id"))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			callLogs, err := insightService.GetInsightJobCallLogs(c.Request.Context(), job.ID)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"job": job, "call_logs": callLogs})
		},
	)

	r.POST(
		"/api/insight-jobs/:id/cancel", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			job, err := insightService.CancelInsightJob(c.Request.Context(), c.Param("id"))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				if strings.Contains(err.Error(), "不能为空") || strings.Contains(err.Error(), "不能取消") {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"job": job})
		},
	)

	r.DELETE(
		"/api/insight-jobs/:id", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			jobID := c.Param("id")
			job, err := insightService.GetInsightJob(c.Request.Context(), jobID)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}
			if job.Status != common.InsightJobPhaseFailed && job.Status != common.InsightJobPhaseCanceled {
				c.JSON(http.StatusBadRequest, gin.H{"error": "仅允许删除失败或已取消的任务"})
				return
			}

			if err := insightService.DeleteInsightJob(c.Request.Context(), jobID); err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				if strings.Contains(err.Error(), "仅允许删除失败或已取消的任务") {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			if client := coreredis.GetRedisClient(); client != nil {
				cacheKey := buildRedisCacheKey(
					http.MethodGet,
					"/api/insight-jobs/"+jobID,
					url.Values{},
					c.GetHeader("Accept"),
				)
				if err := client.Del(c.Request.Context(), cacheKey).Err(); err != nil {
					log.Warn(
						c.Request.Context(), "清理音眸任务详情缓存失败", zap.Error(err),
						zap.String("cache_key", cacheKey),
					)
				}
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	r.POST(
		"/api/insight-jobs/:id/retry", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			job, existing, err := insightService.RetryInsightJob(c.Request.Context(), c.Param("id"))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				if strings.Contains(err.Error(), "不能为空") || isAISelectionBadRequest(err) {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"job": job, "existing": existing})
		},
	)

	r.POST(
		"/api/insight-jobs/:id/live-activity-token", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			var req struct {
				Token string `json:"token"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			job, err := insightService.UpdateInsightJobLiveActivityToken(c.Request.Context(), c.Param("id"), req.Token)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				if isAISelectionBadRequest(err) || strings.Contains(err.Error(), "不能为空") {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"job": job})
		},
	)

	// 获取 / 生成某首歌的歌词解析结果 old
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
				Artist      string `json:"artist"`
				Album       string `json:"album"`
				Track       string `json:"track"`
				TrackNumber int8   `json:"track_number"`
				DiscNumber  int8   `json:"disc_number"`
				Provider    string `json:"provider"`
				Model       string `json:"model"`
				ModelType   string `json:"modelType"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			ctx := c.Request.Context()
			insights, cached, err := insightService.GetOrCreateTrackInsight(
				ctx, req.Artist, req.Album, req.Track, req.TrackNumber, req.DiscNumber, true, req.Provider, req.Model,
				req.ModelType,
			)
			if err != nil {
				log.Error(
					ctx, "获取或生成歌词解析失败",
					zap.String("artist", req.Artist),
					zap.String("album", req.Album),
					zap.String("track", req.Track),
					zap.Error(err),
				)
				status := http.StatusInternalServerError
				if isAISelectionBadRequest(err) {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"insights":               insights,
					"cached":                 cached,
					"recommended_insight_id": recommendedTrackInsightModelID(insights),
				},
			)
		},
	)

	// 获取 / 生成某张专辑的聚合解析结果  old
	r.POST(
		"/api/album-insight", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(
					http.StatusServiceUnavailable, gin.H{
						"error": "AI 专辑解析服务未正确初始化，请检查模型配置",
					},
				)
				return
			}

			var req struct {
				AlbumID   int64  `json:"album_id"`
				Provider  string `json:"provider"`
				Model     string `json:"model"`
				ModelType string `json:"modelType"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}
			if req.AlbumID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的专辑 ID"})
				return
			}

			ctx := c.Request.Context()
			insights, cached, err := insightService.GetOrCreateAlbumInsight(
				ctx, req.AlbumID, true, req.Provider, req.Model, req.ModelType,
			)
			if err != nil {
				log.Error(
					ctx, "获取或生成专辑解析失败",
					zap.Int64("album_id", req.AlbumID),
					zap.Error(err),
				)
				status := http.StatusInternalServerError
				if isAISelectionBadRequest(err) {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(
				http.StatusOK, gin.H{
					"insights": sanitizeAlbumInsightsForClient(
						insights, shouldHideAlbumInsightDebugData(c),
					),
					"cached":                 cached,
					"recommended_insight_id": recommendedAlbumInsightID(insights),
				},
			)
		},
	)

	// 流式获取歌词解析结果 (SSE)  old
	r.GET(
		"/api/track-insight-stream", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			artist := c.Query("artist")
			album := c.Query("album")
			track := c.Query("track")
			trackNumber := parseInt8Query(c, "trackNumber")
			discNumber := parseInt8Query(c, "discNumber")
			force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))
			provider := c.Query("provider")
			modelName := c.Query("model")
			modelType := c.Query("modelType")

			if artist == "" || track == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "参数不足"})
				return
			}

			ch, _, err := insightService.GetOrCreateInsightStream(
				c.Request.Context(), artist, album, track, trackNumber, discNumber, force, provider, modelName,
				modelType,
			)
			if err != nil {
				status := http.StatusInternalServerError
				if isAISelectionBadRequest(err) {
					status = http.StatusBadRequest
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("Transfer-Encoding", "chunked")
			c.Header("X-Accel-Buffering", "no")

			c.Stream(
				func(w io.Writer) bool {
					if chunk, ok := <-ch; ok {
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
}

type insightFeedbackService interface {
	RecordFeedback(ctx context.Context, insightID int64, req insight.FeedbackRecordRequest) error
	RecordAlbumFeedback(ctx context.Context, insightID int64, req insight.FeedbackRecordRequest) error
	GetTrackInsightFeedbacks(ctx context.Context, insightID int64) ([]*model.TrackInsightFeedback, error)
	GetAlbumInsightFeedbacks(ctx context.Context, insightID int64) ([]*model.AlbumInsightFeedback, error)
	GetInsightFeedbackSummary(
		ctx context.Context, targetType common.AnalysisTargetType, insightID int64,
	) (*insight.InsightFeedbackSummary, error)
	GetInsightFeedbackHistory(
		ctx context.Context, targetType common.AnalysisTargetType, insightID int64, limit int,
	) ([]*insight.InsightFeedbackHistoryItem, error)
	GetInsightHistory(
		ctx context.Context, targetType common.AnalysisTargetType, insightID int64, limit int,
	) ([]*model.InsightListItem, error)
}

type insightFeedbackRequest struct {
	Score          int      `json:"score"`
	Comment        string   `json:"comment"`
	ReasonCodes    []string `json:"reason_codes"`
	SectionKey     string   `json:"section_key"`
	SourcePlatform string   `json:"source_platform"`
}

func registerInsightFeedbackRoutes(r gin.IRoutes, insightService insightFeedbackService) {
	r.POST(
		"/api/track-insight/:id/feedback", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 歌词解析服务未正确初始化，请检查配置"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 insight ID")
			if !ok {
				return
			}

			var req insightFeedbackRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			ctx := c.Request.Context()
			if err := insightService.RecordFeedback(
				ctx,
				insightID,
				insight.FeedbackRecordRequest{
					Score:          req.Score,
					Comment:        req.Comment,
					ReasonCodes:    req.ReasonCodes,
					SectionKey:     req.SectionKey,
					SourcePlatform: req.SourcePlatform,
				},
			); err != nil {
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

	r.POST(
		"/api/album-insight/:id/feedback", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 歌词解析服务未正确初始化，请检查配置"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 insight ID")
			if !ok {
				return
			}

			var req insightFeedbackRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
				return
			}

			ctx := c.Request.Context()
			if err := insightService.RecordAlbumFeedback(
				ctx,
				insightID,
				insight.FeedbackRecordRequest{
					Score:          req.Score,
					Comment:        req.Comment,
					ReasonCodes:    req.ReasonCodes,
					SectionKey:     req.SectionKey,
					SourcePlatform: req.SourcePlatform,
				},
			); err != nil {
				log.Error(
					ctx, "记录专辑解析反馈失败",
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

	r.GET(
		"/api/insights/:id/feedback-summary", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 ID")
			if !ok {
				return
			}

			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery("analysis_target_type", string(common.AnalysisTargetTypeTrack)),
			)
			summary, err := insightService.GetInsightFeedbackSummary(c.Request.Context(), targetType, insightID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, summary)
		},
	)

	r.GET(
		"/api/insights/:id/feedback-history", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 ID")
			if !ok {
				return
			}

			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery("analysis_target_type", string(common.AnalysisTargetTypeTrack)),
			)
			limit := parseQueryIntWithDefault(c, "limit", 10)
			if limit <= 0 {
				limit = 10
			}
			history, err := insightService.GetInsightFeedbackHistory(c.Request.Context(), targetType, insightID, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"feedbacks": history, "analysis_target_type": targetType})
		},
	)

	r.GET(
		"/api/insights/:id/history", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 ID")
			if !ok {
				return
			}

			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery("analysis_target_type", string(common.AnalysisTargetTypeTrack)),
			)
			limit := parseQueryIntWithDefault(c, "limit", 20)
			if limit <= 0 {
				limit = 20
			}

			history, err := insightService.GetInsightHistory(c.Request.Context(), targetType, insightID, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(
				http.StatusOK,
				gin.H{
					"insights":               history,
					"total":                  len(history),
					"limit":                  limit,
					"offset":                 0,
					"analysis_target_type":   targetType,
					"recommended_insight_id": recommendedInsightListItemID(history),
				},
			)
		},
	)

	r.GET(
		"/api/insights/:id/feedbacks", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 ID")
			if !ok {
				return
			}

			targetType := common.ParseAnalysisTargetType(
				c.DefaultQuery("analysis_target_type", string(common.AnalysisTargetTypeTrack)),
			)

			switch targetType {
			case common.AnalysisTargetTypeAlbum:
				feedbacks, err := insightService.GetAlbumInsightFeedbacks(c.Request.Context(), insightID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"feedbacks": feedbacks, "analysis_target_type": targetType})
			default:
				feedbacks, err := insightService.GetTrackInsightFeedbacks(c.Request.Context(), insightID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"feedbacks": feedbacks, "analysis_target_type": targetType})
			}
		},
	)

	r.GET(
		"/api/album-insights/:id/feedbacks", func(c *gin.Context) {
			if insightService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 服务未初始化"})
				return
			}

			insightID, ok := parsePathInt64(c, "id", "无效的 ID")
			if !ok {
				return
			}

			feedbacks, err := insightService.GetAlbumInsightFeedbacks(c.Request.Context(), insightID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"feedbacks": feedbacks, "analysis_target_type": common.AnalysisTargetTypeAlbum})
		},
	)
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
