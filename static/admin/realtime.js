function connectWebSocket() {
        // 创建WebSocket连接
        ws = new WebSocket("ws://" + window.location.host + "/ws");

        // 连接打开时的处理
        ws.onopen = function (event) {
            console.log("WebSocket连接已建立");
            insightJobWSConnected = true;
            updateInsightJobRealtimeIndicator();
        };

        // 收到消息时的处理
        ws.onmessage = function (event) {
            const data = JSON.parse(event.data);
            if (data.type === "now_playing") {
                // 更新正在播放的信息
                updateNowPlaying(data.source, data.data);
            } else if (data.type === "stop") {
                // 隐藏正在播放的悬浮窗
                document.getElementById("nowPlaying").style.display = "none";
                const coverArtElement = document.getElementById("trackCoverArt");
                if (coverArtElement) {
                    applyArtworkPlaceholder(coverArtElement, "", { compact: true });
                }
                currentTrackInfo = null;
                currentSource = null;
                lastTrackKey = "";
                // 清除进度更新定时器
                if (progressInterval) {
                    clearInterval(progressInterval);
                    progressInterval = null;
                }
                currentPosition = 0;
                currentPositionMs = 0;
                positionAnchorMs = 0;
                positionAnchorPerf = performance.now();
                trackDuration = 0;
                trackDurationMs = 0;
            } else if (data.type === "insight_job_updated") {
                handleInsightJobRealtimeUpdate(data.data);
                if (typeof window.handleActiveInsightJobUpdate === "function") {
                    window.handleActiveInsightJobUpdate(data.data);
                }
            }
        };

        // 连接关闭时的处理
        ws.onclose = function (event) {
            console.log("WebSocket连接已关闭");
            insightJobWSConnected = false;
            updateInsightJobRealtimeIndicator();
            // 清除进度更新定时器
            if (progressInterval) {
                clearInterval(progressInterval);
                progressInterval = null;
            }
            // 5秒后尝试重新连接
            setTimeout(connectWebSocket, 5000);
        };

        // 连接错误时的处理
        ws.onerror = function (error) {
            console.log("WebSocket连接出错:", error);
            insightJobWSConnected = false;
            updateInsightJobRealtimeIndicator();
            // 清除进度更新定时器
            if (progressInterval) {
                clearInterval(progressInterval);
                progressInterval = null;
            }
        };
    }

    // 当前播放的曲目信息
    let currentTrackInfo = null;
    let currentSource = null;
    let currentPosition = 0; // 当前播放位置（秒）
    let currentPositionMs = 0; // 当前播放位置（毫秒）
    let positionAnchorMs = 0;
    let positionAnchorPerf = 0;
    let trackDuration = 0; // 歌曲总时长
    let trackDurationMs = 0;
    let progressInterval = null; // 进度更新定时器
    let currentTrackInsight = null; // 当前曲目的 AI 解析结果 (兼容旧代码)
    
    // AI Insight 模块化状态管理
    const insightStates = {
        nowPlaying: { 
            insight: null, 
            allInsights: [], 
            eventSource: null,
            trackInfo: null,
            targetType: 'track'
        },
        list: { 
            insight: null, 
            allInsights: [], 
            eventSource: null,
            trackInfo: null,
            targetType: 'track'
        },
        details: { 
            insight: null, 
            allInsights: [], 
            eventSource: null,
            trackInfo: null,
            targetType: 'track'
        }
    };

    const albumInsightState = {
        albumID: 0,
        albumMeta: null,
        insights: [],
        insight: null,
        focusInsightID: 0,
        view: 'summary',
        loading: false,
        generating: false,
        lastError: '',
    };

    let currentAlbumDetailTab = 'info';
    let pendingAlbumInsightFocusID = 0;

    let lastTrackKey = ""; // 用于检测切歌
    let currentRankingType = "all"; // 当前排行榜类型
    let rankingSearchKeyword = ""; // 当前排行榜搜索关键字
    let rankingSearchTimeout = null; // 排行榜搜索防抖计算器
    const artworkResolveCache = new Map();
    const artworkResolveInflight = new Map();
    let artworkResolveSequence = 0;

    // 格式化时间（秒转为mm:ss）
    function formatTime(seconds) {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins.toString().padStart(2, "0")}:${secs
            .toString()
            .padStart(2, "0")}`;
    }

    function resolveIncomingPositionMs(data) {
        if (data && typeof data.position_ms === "number" && isFinite(data.position_ms)) {
            return Math.max(0, Math.floor(data.position_ms));
        }
        if (data && typeof data.position === "number" && isFinite(data.position)) {
            return Math.max(0, Math.round(data.position * 1000));
        }
        return 0;
    }

    function setPlaybackAnchor(positionMs) {
        currentPositionMs = Math.max(0, Math.floor(positionMs || 0));
        currentPosition = currentPositionMs / 1000;
        positionAnchorMs = currentPositionMs;
        positionAnchorPerf = performance.now();
    }

    function getCurrentPositionMs() {
        let nextMs = positionAnchorMs;
        if (trackDurationMs > 0) {
            nextMs += Math.max(0, performance.now() - positionAnchorPerf);
            nextMs = Math.min(nextMs, trackDurationMs);
        }
        return Math.max(0, Math.floor(nextMs));
    }

    function syncPlaybackClock() {
        currentPositionMs = getCurrentPositionMs();
        currentPosition = currentPositionMs / 1000;
        return currentPositionMs;
    }

    function startProgressTicker() {
        if (progressInterval) {
            clearInterval(progressInterval);
            progressInterval = null;
        }
        updateProgressDisplay();
        if (trackDurationMs <= 0) return;
        const intervalMs = isLowEndMode() ? 500 : 200;
        progressInterval = setInterval(() => {
            syncPlaybackClock();
            updateProgressDisplay();
            updateLyricsHighlight();
            if (currentPositionMs >= trackDurationMs) {
                clearInterval(progressInterval);
                progressInterval = null;
            }
        }, intervalMs);
    }

    // 更新播放进度显示
    function updateProgressDisplay() {
        if (trackDuration <= 0) return;

        syncPlaybackClock();

        // 更新进度条
        const progressPercent = (currentPosition / trackDuration) * 100;
        document.getElementById("progressFill").style.width = `${Math.min(
            progressPercent,
            100
        )}%`;

        // 更新时间显示
        document.getElementById("currentTime").textContent =
            formatTime(currentPosition);
        document.getElementById("totalTime").textContent =
            formatTime(trackDuration);
    }

    function resolveCoverArtURL(rawURL) {
        if (!rawURL) return "";
        const trimmed = String(rawURL).trim();
        if (!trimmed) return "";
        if (/^https?:\/\//i.test(trimmed)) return trimmed;

        const normalizedPath = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
        const protocol = window.location.protocol === "https:" ? "https:" : "http:";
        return `${protocol}//${window.location.hostname}:9000${normalizedPath}`;
    }

    function getAlbumInitial(name) {
        const trimmed = String(name || "").trim();
        if (!trimmed) return "?";
        const chars = Array.from(trimmed);
        return (chars[0] || "?").toUpperCase();
    }

    function buildArtworkResolveCacheKey(entity) {
        const albumID = Number(entity.albumID || entity.album_id || entity.resolved_album_id || 0);
        const albumArtist = String(entity.albumArtist || entity.album_artist || "").trim();
        const artist = String(entity.artist || "").trim();
        const album = String(entity.album || "").trim();
        const albumSubtitle = String(entity.albumSubtitle || entity.album_subtitle || "").trim();
        const artworkKey = String(entity.artworkKey || entity.artwork_key || "").trim();

        if (albumID > 0) return `album:${albumID}`;
        if (albumArtist || artist || album) return `seed:${albumArtist}|${artist}|${album}|${albumSubtitle}`;
        if (artworkKey) return `artwork:${artworkKey}`;
        return "";
    }

    function renderArtworkPlaceholder(albumName, compact = false, extraStyle = "") {
        return `<div class="artwork-fallback ${compact ? "artwork-fallback--small" : ""}" style="${extraStyle}">${esc(getAlbumInitial(albumName))}</div>`;
    }

    function applyArtworkPlaceholder(container, albumName, options = {}) {
        if (!container) return;
        container.dataset.albumName = albumName || "";
        container.dataset.artworkCompact = options.compact ? "1" : "0";
        container.dataset.placeholderStyle = options.placeholderStyle || "";
        container.innerHTML = renderArtworkPlaceholder(
            albumName,
            options.compact,
            options.placeholderStyle || "",
        );
    }

    function applyArtworkImage(container, rawURL, altText) {
        if (!container) return false;
        const resolvedURL = resolveCoverArtURL(rawURL);
        if (!resolvedURL) return false;
        container.innerHTML =
            `<img src="${esc(resolvedURL)}" alt="${esc(altText || "专辑封面")}" loading="lazy" onerror="window.handleArtworkImageError(this)">`;
        return true;
    }

    window.handleArtworkImageError = function (img) {
        if (!img || !img.parentElement) return;
        const container = img.parentElement;
        applyArtworkPlaceholder(
            container,
            container.dataset.albumName || "",
            {
                compact: container.dataset.artworkCompact === "1",
                placeholderStyle: container.dataset.placeholderStyle || "",
            },
        );
    };

    async function resolveArtworkForEntity(entity = {}) {
        const normalized = {
            albumID: Number(entity.albumID || entity.album_id || entity.resolved_album_id || 0),
            albumArtist: String(entity.albumArtist || entity.album_artist || "").trim(),
            artist: String(entity.artist || "").trim(),
            album: String(entity.album || "").trim(),
            albumSubtitle: String(entity.albumSubtitle || entity.album_subtitle || "").trim(),
            artworkKey: String(entity.artworkKey || entity.artwork_key || "").trim(),
            coverArtURL: String(entity.coverArtURL || entity.cover_art_url || "").trim(),
        };

        if (normalized.coverArtURL) {
            return {
                exists: true,
                cover_art_url: normalized.coverArtURL,
                cover_art_object_key: normalized.artworkKey,
            };
        }

        const cacheKey = buildArtworkResolveCacheKey(normalized);
        if (!cacheKey) {
            return { exists: false, cover_art_url: "", cover_art_object_key: "" };
        }
        if (artworkResolveCache.has(cacheKey)) {
            return artworkResolveCache.get(cacheKey);
        }
        if (artworkResolveInflight.has(cacheKey)) {
            return artworkResolveInflight.get(cacheKey);
        }

        const params = new URLSearchParams();
        if (normalized.albumID > 0) params.set("album_id", String(normalized.albumID));
        if (normalized.albumArtist) params.set("albumArtist", normalized.albumArtist);
        if (normalized.artist) params.set("artist", normalized.artist);
        if (normalized.album) params.set("album", normalized.album);
        if (normalized.albumSubtitle) params.set("albumSubtitle", normalized.albumSubtitle);
        if (normalized.artworkKey) params.set("artworkKey", normalized.artworkKey);

        const request = fetch(`/api/artwork/resolve?${params.toString()}`)
            .then(resp => resp.ok ? resp.json() : Promise.reject(new Error(`resolve artwork failed: ${resp.status}`)))
            .then(data => ({
                exists: Boolean(data && data.exists && data.cover_art_url),
                cover_art_url: String((data && data.cover_art_url) || "").trim(),
                cover_art_object_key: String((data && data.cover_art_object_key) || "").trim(),
            }))
            .catch(err => {
                console.warn("resolve artwork failed:", err);
                return { exists: false, cover_art_url: "", cover_art_object_key: "" };
            })
            .then(result => {
                artworkResolveCache.set(cacheKey, result);
                artworkResolveInflight.delete(cacheKey);
                return result;
            });

        artworkResolveInflight.set(cacheKey, request);
        return request;
    }

    function hydrateArtworkSlot(container, entity = {}, options = {}) {
        if (!container) return;

        const albumName = entity.album || entity.name || "";
        const compact =
            options.compact !== undefined ? options.compact : container.dataset.artworkCompact === "1";
        const placeholderStyle = options.placeholderStyle || container.dataset.placeholderStyle || "";
        const altText = options.altText || albumName || "专辑封面";

        const requestToken = String(++artworkResolveSequence);
        container.dataset.artworkRequestToken = requestToken;
        applyArtworkPlaceholder(container, albumName, { compact, placeholderStyle });

        if (applyArtworkImage(container, entity.coverArtURL || entity.cover_art_url, altText)) {
            return;
        }

        resolveArtworkForEntity(entity).then(result => {
            if (container.dataset.artworkRequestToken !== requestToken) return;
            if (result && result.exists && result.cover_art_url) {
                applyArtworkImage(container, result.cover_art_url, altText);
                return;
            }
            applyArtworkPlaceholder(container, albumName, { compact, placeholderStyle });
        });
    }

    function hydrateArtworkSlots(root) {
        if (!root) return;
        root.querySelectorAll("[data-artwork-slot='1']").forEach(container => {
            hydrateArtworkSlot(
                container,
                {
                    albumID: Number(container.dataset.albumId || container.dataset.resolvedAlbumId || 0),
                    albumArtist: container.dataset.albumArtist || "",
                    artist: container.dataset.artist || "",
                    album: container.dataset.album || "",
                    albumSubtitle: container.dataset.albumSubtitle || "",
                    artworkKey: container.dataset.artworkKey || "",
                    coverArtURL: container.dataset.coverArtUrl || "",
                },
                {
                    compact: container.dataset.artworkCompact === "1",
                    altText: container.dataset.altText || "专辑封面",
                    placeholderStyle: container.dataset.placeholderStyle || "",
                },
            );
        });
    }

    function isNowPlayingExpanded() {
        const nowPlaying = document.getElementById("nowPlaying");
        return Boolean(
            nowPlaying &&
            nowPlaying.style.display !== "none" &&
            !nowPlaying.classList.contains("minimized"),
        );
    }

    function syncNowPlayingArtwork() {
        const coverArtElement = document.getElementById("trackCoverArt");
        if (!coverArtElement || !currentTrackInfo) return;

        const albumName = currentTrackInfo.album || currentTrackInfo.title || "";
        const compact = true;
        const placeholderStyle = "";
        const altText = currentTrackInfo.album || currentTrackInfo.title || "专辑封面";

        if (!isNowPlayingExpanded()) {
            applyArtworkPlaceholder(coverArtElement, albumName, { compact, placeholderStyle });
            coverArtElement.dataset.artworkRequestToken = "";
            return;
        }

        hydrateArtworkSlot(
            coverArtElement,
            {
                albumArtist: currentTrackInfo.album_artist || "",
                artist: currentTrackInfo.artist || "",
                album: currentTrackInfo.album || "",
                artworkKey: currentTrackInfo.artwork_key || "",
                coverArtURL: currentTrackInfo.cover_art_url || "",
            },
            {
                compact,
                altText,
                placeholderStyle,
            },
        );
    }

    // 更新正在播放的信息
    function updateNowPlaying(source, data) {
        // 检测是否切歌
        const newTrackKey = `${data.artist}-${data.title}`;
        const trackChanged = newTrackKey !== lastTrackKey;
        lastTrackKey = newTrackKey;

        // 保存当前播放信息
        currentTrackInfo = data;
        currentSource = source;

        // 重置播放进度
        setPlaybackAnchor(resolveIncomingPositionMs(data));
        trackDuration = data.duration || 0;
        trackDurationMs = trackDuration * 1000;

        // 清除之前的进度更新定时器
        startProgressTicker();

        // 显示悬浮窗，默认保持收起，避免在未展开时提前拉取封面
        const nowPlaying = document.getElementById("nowPlaying");
        const wasHidden = Boolean(
            nowPlaying &&
            window.getComputedStyle(nowPlaying).display === "none",
        );
        if (nowPlaying) {
            nowPlaying.style.display = "block";
            if (wasHidden) {
                nowPlaying.classList.add("minimized");
            }
        }

        // 如果歌曲改变了，且歌词窗口是打开的，则更新歌词
        if (trackChanged) {
            const lyricsFloating = document.getElementById("lyricsFloating");
            if (lyricsFloating && lyricsFloating.style.display === "block") {
                showLyrics();
            }
        }

        // 根据来源更新信息
        let sourceClass = "";
        let sourceText = source || "";
        syncNowPlayingArtwork();

        document.getElementById("trackTitle").textContent = " " + data.title;
        document.getElementById("trackAlbum").textContent =
            "《 " + data.album + " 》";
        document.getElementById("trackArtist").textContent =
            " - " + data.artist;

        if (typeof getSourceBadgeInfo === "function") {
            const sourceBadge = getSourceBadgeInfo(source);
            sourceClass = sourceBadge.sourceClass;
            sourceText = sourceBadge.sourceText;
        }

        // 更新播放渠道显示样式
        const trackSourceElement = document.getElementById("trackSource");
        trackSourceElement.textContent = sourceText;
        trackSourceElement.className = "play-source " + sourceClass;

        // 更新喜欢状态指示器
        updateFavoriteIndicators(data.apple_music, data.lastfm);

        // 更新点赞按钮状态
        updateFavoriteButtonState(data.apple_music);
    }

    // 更新喜欢状态指示器
    function updateFavoriteIndicators(appleMusicFav, lastFmFav) {
        const indicatorsContainer =
            document.getElementById("favoriteIndicators");
        indicatorsContainer.innerHTML = "";

        if (appleMusicFav && lastFmFav) {
            // 两个都喜欢，显示文本提示
            const indicator = document.createElement("span");
            indicator.className = "favorite-indicator";
            indicator.innerHTML = "Apple Music 和 Last.fm 都喜欢";
            indicator.title = "Apple Music 和 Last.fm 都喜欢";
            indicatorsContainer.appendChild(indicator);
        } else if (appleMusicFav) {
            // 只有Apple Music喜欢，显示文本提示
            const indicator = document.createElement("span");
            indicator.className = "favorite-indicator";
            indicator.innerHTML = "Apple Music 喜欢";
            indicator.title = "Apple Music 喜欢";
            indicatorsContainer.appendChild(indicator);
        } else if (lastFmFav) {
            // 只有Last.fm喜欢，显示文本提示
            const indicator = document.createElement("span");
            indicator.className = "favorite-indicator";
            indicator.innerHTML = "Last.fm 喜欢";
            indicator.title = "Last.fm 喜欢";
            indicatorsContainer.appendChild(indicator);
        }
        // 如果两个都不喜欢，则不显示任何指示器
    }

    // 更新点赞按钮状态
    function updateFavoriteButtonState(isLiked) {
        const favoriteButton = document.getElementById("favoriteButton");
        if (isLiked) {
            favoriteButton.innerHTML = "★";
            favoriteButton.className = "favorite-button liked";
            favoriteButton.title = "已添加到喜欢";
        } else {
            favoriteButton.innerHTML = "⭐︎";
            favoriteButton.className = "favorite-button unliked";
            favoriteButton.title = "添加到喜欢";
        }
    }

    let pendingAnalysisTrack = null; // 暂存待分析的曲目信息（用于处理详情页分析）
    let pendingAnalysisMode = 'track';
    let pendingAnalysisAlbumID = 0;

    // --- AI 解析交互逻辑重构 ---

    // 1. 处理正在播放面板的“AI 分析”点击
