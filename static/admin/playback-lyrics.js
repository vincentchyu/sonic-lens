function handleFavoriteButtonClick() {
        if (!currentTrackInfo || !currentSource) {
            console.log("没有正在播放的曲目");
            return;
        }

        const favoriteButton = document.getElementById("favoriteButton");
        const isCurrentlyLiked = favoriteButton.classList.contains("liked");

        // 如果已经收藏，则不执行任何操作
        if (isCurrentlyLiked) {
            return;
        }

        // 立即更新按钮状态，提供即时反馈
        updateFavoriteButtonState(!isCurrentlyLiked);

        // 发送请求到后端处理收藏逻辑
        fetch("/api/favorite", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                artist: currentTrackInfo.artist,
                album: currentTrackInfo.album,
                track: currentTrackInfo.title,
                source: currentSource,
                favorite: !isCurrentlyLiked,
            }),
        })
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                // 更新喜欢状态指示器
                updateFavoriteIndicators(data.apple_music, data.lastfm);
                // 更新按钮状态（以防后端返回的状态与前端不一致）
                updateFavoriteButtonState(data.apple_music);
            })
            .catch((error) => {
                console.error("收藏操作失败:", error);
                // 恢复按钮原始状态
                updateFavoriteButtonState(isCurrentlyLiked);
                alert("收藏操作失败: " + error.message);
            });
    }

    // 初始化正在播放悬浮窗的拖动和缩小功能
    function initNowPlayingDrag() {
        const nowPlaying = document.getElementById("nowPlaying");
        const header = document.querySelector(".now-playing-header");
        const minimizeHandle = document.querySelector(".minimize-handle");

        if (!nowPlaying || !header || !minimizeHandle) return;

        let isDragging = false;
        let offsetX, offsetY;
        let originalX, originalY;
        let dragRafId = null;
        let pendingPosition = null;

        function flushDragPosition() {
            dragRafId = null;
            if (!pendingPosition) return;
            updateFloatingDragPosition(nowPlaying, pendingPosition);
        }

        // 鼠标按下事件（拖动）- 在整个标题栏都可以触发
        header.addEventListener("mousedown", function (e) {
            // 如果点击的是缩小按钮，则不触发拖动
            if (e.target === minimizeHandle) return;

            isDragging = true;
            nowPlaying.classList.add("dragging");
            document.body.classList.add("floating-dragging");

            // 获取悬浮窗当前位置
            const rect = nowPlaying.getBoundingClientRect();
            originalX = rect.left;
            originalY = rect.top;

            // 计算鼠标相对于悬浮窗的位置
            offsetX = e.clientX - originalX;
            offsetY = e.clientY - originalY;

            // 防止文本选择
            e.preventDefault();
        });

        // 鼠标移动事件（拖动）
        document.addEventListener("mousemove", function (e) {
            if (!isDragging) return;

            pendingPosition = {
                x: e.clientX - offsetX,
                y: e.clientY - offsetY,
            };
            if (!dragRafId) {
                dragRafId = requestAnimationFrame(flushDragPosition);
            }
        });

        // 鼠标释放事件（拖动）
        document.addEventListener("mouseup", function () {
            isDragging = false;
            nowPlaying.classList.remove("dragging");
            document.body.classList.remove("floating-dragging");
            if (dragRafId) {
                cancelAnimationFrame(dragRafId);
                dragRafId = null;
            }
            if (pendingPosition) {
                updateFloatingDragPosition(nowPlaying, pendingPosition);
                pendingPosition = null;
            }
        });

        // 点击缩小/展开按钮
        minimizeHandle.addEventListener("click", function () {
            nowPlaying.classList.toggle("minimized");
            if (!nowPlaying.classList.contains("minimized")) {
                syncNowPlayingArtwork();
            }
        });
    }



    // ========== 歌词功能 ==========
    let currentLyricsData = null;
    let lyricsLineElements = [];
    let activeLyricIndex = -1;
    const lyricsCache = {};
    const LYRICS_CACHE_TTL_MS = 5 * 60 * 1000;

    function buildLyricsCacheKey(artist, album, track, trackNumber, discNumber) {
        return `${artist || ""}::${album || ""}::${track || ""}::${trackNumber || 0}::${discNumber || 0}`;
    }

    function parseLRC(lrcText) {
        return window.SonicLRC.parseLRC(lrcText);
    }

    function parseTaggedLyricLine(text) {
        const clean = (text || '').replace(/\\n/g, '\n').trim();
        const originalMatch = clean.match(/<original>([\s\S]*?)<(?:\/)?original>/i);
        const translationMatch = clean.match(/<translation>([\s\S]*?)<(?:\/)?translation>/i);
        if (!originalMatch && !translationMatch) {
            return { original: clean, translation: '', hasTag: false };
        }
        return {
            original: originalMatch ? originalMatch[1].trim() : '',
            translation: translationMatch ? translationMatch[1].trim() : '',
            hasTag: true
        };
    }

    function parseTaggedLyricRows(text) {
        const clean = (text || '').replace(/\\n/g, '\n');
        const regex = /<(original|translation)>([\s\S]*?)<(?:\/)?\1>/gi;
        const rows = [];
        let matched = false;
        let match;
        while ((match = regex.exec(clean)) !== null) {
            matched = true;
            const tag = (match[1] || '').toLowerCase();
            const content = (match[2] || '').trim();
            if (tag === 'original') {
                rows.push({ original: content, translation: '' });
            } else if (tag === 'translation') {
                if (!rows.length) {
                    rows.push({ original: '', translation: content });
                } else {
                    rows[rows.length - 1].translation = content;
                }
            }
        }
        if (!matched) return null;
        return rows;
    }

    function renderPlainLyrics(container, text) {
        container.innerHTML = '';
        const taggedRows = parseTaggedLyricRows(text);
        if (taggedRows && taggedRows.length) {
            const fragment = document.createDocumentFragment();
            taggedRows.forEach(row => {
                const lineDiv = document.createElement('div');
                lineDiv.className = 'lyrics-line';
                const originalDiv = document.createElement('div');
                originalDiv.className = 'lyrics-line-original';
                originalDiv.textContent = row.original || ' ';
                lineDiv.appendChild(originalDiv);
                if (row.translation) {
                    const translationDiv = document.createElement('div');
                    translationDiv.className = 'lyrics-line-translation';
                    translationDiv.textContent = row.translation;
                    lineDiv.appendChild(translationDiv);
                }
                fragment.appendChild(lineDiv);
            });
            container.appendChild(fragment);
            return;
        }

        const plainLyrics = document.createElement('div');
        plainLyrics.className = 'lyrics-text';
        plainLyrics.textContent = text;
        container.appendChild(plainLyrics);
    }

    async function fetchLyrics(artist, album, track, trackNumber, discNumber) {
        try {
            const cacheKey = buildLyricsCacheKey(artist, album, track, trackNumber, discNumber);
            const cached = lyricsCache[cacheKey];
            if (cached && Date.now() - cached.ts < LYRICS_CACHE_TTL_MS) {
                return cached.data;
            }

            const params = new URLSearchParams({
                artist: artist,
                album: album || '',
                track: track,
                trackNumber: String(trackNumber || 0),
                discNumber: String(discNumber || 0)
            });
            const response = await fetch(`/api/track-lyrics?${params}`);
            if (!response.ok) throw new Error('获取歌词失败');
            const data = await response.json();
            lyricsCache[cacheKey] = {
                ts: Date.now(),
                data: data
            };
            return data;
        } catch (error) {
            console.error('获取歌词失败:', error);
            return {lyrics: '', has_lrc: false};
        }
    }

    async function showLyrics() {
        if (!currentTrackInfo) {
            alert('当前没有正在播放的曲目');
            return;
        }
        const lyricsFloating = document.getElementById('lyricsFloating');
        const lyricsContainer = document.getElementById('lyricsContainer');
        lyricsContainer.innerHTML = '<div class="lyrics-text">正在加载歌词...</div>';
        lyricsFloating.style.display = 'block';

        try {
            const lyricsData = await fetchLyrics(
                currentTrackInfo.artist,
                currentTrackInfo.album,
                currentTrackInfo.title,
                currentTrackInfo.track_number,
                currentTrackInfo.disc_number
            );

            if (!lyricsData.lyrics) {
                lyricsContainer.innerHTML = '<div class="lyrics-text">暂无歌词数据</div>';
                currentLyricsData = null;
                return;
            }

            if (window.SonicLRC.isSyncedLRC(lyricsData.lyrics)) {
                currentLyricsData = parseLRC(lyricsData.lyrics);
                renderLRCLyrics(currentLyricsData);
                startLyricsSync();
            } else {
                renderPlainLyrics(lyricsContainer, lyricsData.lyrics);
                currentLyricsData = null;
                stopLyricsSync();
            }
        } catch (error) {
            console.error('歌词渲染失败:', error);
            lyricsContainer.innerHTML = '<div class="lyrics-text">歌词加载失败，请稍后重试</div>';
            currentLyricsData = null;
            stopLyricsSync();
        }
    }

    function renderLRCLyrics(lrcData) {
        const lyricsContainer = document.getElementById('lyricsContainer');
        lyricsContainer.innerHTML = '';
        activeLyricIndex = -1;
        lyricsLineElements = [];
        const fragment = document.createDocumentFragment();
        lrcData.forEach((item, index) => {
            const lineDiv = document.createElement('div');
            lineDiv.className = 'lyrics-line';
            if (item.isSectionLabel) {
                lineDiv.classList.add('lyrics-line-section');
            }
            const pair = parseTaggedLyricLine(item.text);
            if (item.isSectionLabel) {
                lineDiv.textContent = pair.original || item.text;
            } else if (pair.hasTag) {
                const originalDiv = document.createElement('div');
                originalDiv.className = 'lyrics-line-original';
                originalDiv.textContent = pair.original || ' ';
                lineDiv.appendChild(originalDiv);

                if (pair.translation) {
                    const translationDiv = document.createElement('div');
                    translationDiv.className = 'lyrics-line-translation';
                    translationDiv.textContent = pair.translation;
                    lineDiv.appendChild(translationDiv);
                }
            } else {
                lineDiv.textContent = pair.original || item.text;
            }
            lineDiv.dataset.timeMs = item.timeMs == null ? '' : String(item.timeMs);
            lineDiv.dataset.index = index;
            lyricsLineElements.push(lineDiv);
            fragment.appendChild(lineDiv);
        });
        lyricsContainer.appendChild(fragment);
    }

    function updateLyricsHighlight() {
        if (!currentLyricsData || !currentTrackInfo) return;
        const nextActiveIndex = window.SonicLRC.findActiveIndex(
            currentLyricsData,
            getCurrentPositionMs(),
            activeLyricIndex
        );

        if (nextActiveIndex === activeLyricIndex) return;

        if (activeLyricIndex >= 0 && lyricsLineElements[activeLyricIndex]) {
            lyricsLineElements[activeLyricIndex].classList.remove('active');
        }
        activeLyricIndex = nextActiveIndex;

        if (activeLyricIndex >= 0 && lyricsLineElements[activeLyricIndex]) {
            const line = lyricsLineElements[activeLyricIndex];
            line.classList.add('active');
            keepLyricLineVisible(line);
        }
    }

    function startLyricsSync() {
        stopLyricsSync();
        updateLyricsHighlight();
    }

    function stopLyricsSync() {
        if (activeLyricIndex >= 0 && lyricsLineElements[activeLyricIndex]) {
            lyricsLineElements[activeLyricIndex].classList.remove('active');
        }
        activeLyricIndex = -1;
    }

    function closeLyrics() {
        document.getElementById('lyricsFloating').style.display = 'none';
        stopLyricsSync();
        currentLyricsData = null;
    }

    function keepLyricLineVisible(line) {
        const lyricsContainer = document.getElementById("lyricsContainer");
        if (!lyricsContainer || !line) return;

        const containerTop = lyricsContainer.scrollTop;
        const containerBottom = containerTop + lyricsContainer.clientHeight;
        const lineTop = line.offsetTop;
        const lineBottom = lineTop + line.offsetHeight;
        const padding = Math.max(24, Math.floor(lyricsContainer.clientHeight * 0.2));

        if (
            lineTop >= containerTop + padding &&
            lineBottom <= containerBottom - padding
        ) {
            return;
        }

        const targetTop = Math.max(
            lineTop - Math.floor((lyricsContainer.clientHeight - line.offsetHeight) / 2),
            0
        );

        try {
            lyricsContainer.scrollTo({
                top: targetTop,
                behavior: isLowEndMode() ? "auto" : "smooth",
            });
        } catch (e) {
            lyricsContainer.scrollTop = targetTop;
        }
    }

    function initLyricsDrag() {
        const lyricsFloating = document.getElementById('lyricsFloating');
        const header = document.querySelector('.lyrics-header');
        const closeBtn = document.getElementById('lyricsCloseBtn');
        const fullscreenBtn = document.getElementById('lyricsFullscreenBtn');
        if (!lyricsFloating || !header || !closeBtn) return;

        let isDragging = false;
        let offsetX, offsetY;
        let dragRafId = null;
        let pendingPosition = null;

        function flushDragPosition() {
            dragRafId = null;
            if (!pendingPosition) return;
            updateFloatingDragPosition(lyricsFloating, pendingPosition);
        }

        header.addEventListener('mousedown', function (e) {
            if (e.target === closeBtn || e.target === fullscreenBtn) return;
            isDragging = true;
            lyricsFloating.classList.add('dragging');
            document.body.classList.add("floating-dragging");
            const rect = lyricsFloating.getBoundingClientRect();
            offsetX = e.clientX - rect.left;
            offsetY = e.clientY - rect.top;
            e.preventDefault();
        });

        document.addEventListener('mousemove', function (e) {
            if (!isDragging) return;
            pendingPosition = {
                x: e.clientX - offsetX,
                y: e.clientY - offsetY,
            };
            if (!dragRafId) {
                dragRafId = requestAnimationFrame(flushDragPosition);
            }
        });

        document.addEventListener('mouseup', function () {
            isDragging = false;
            lyricsFloating.classList.remove('dragging');
            document.body.classList.remove("floating-dragging");
            if (dragRafId) {
                cancelAnimationFrame(dragRafId);
                dragRafId = null;
            }
            if (pendingPosition) {
                updateFloatingDragPosition(lyricsFloating, pendingPosition);
                pendingPosition = null;
            }
        });

        closeBtn.addEventListener('click', closeLyrics);
    }

    function initLyricsButton() {
        const lyricsButton = document.getElementById('lyricsButton');
        if (lyricsButton) {
            lyricsButton.addEventListener('click', function () {
                // 低端设备直接进入简化全屏歌词页
                if (isLowEndMode()) {
                    openLyricsLivePage();
                    return;
                }
                showLyrics();
            });
        }
    }

    function openLyricsLivePage() {
        const params = new URLSearchParams();
        if (isLowEndMode()) {
            params.set("lowEnd", "1");
            try {
                localStorage.setItem("lyricsLiveLowEnd", "1");
            } catch (e) {
            }
        } else {
            try {
                localStorage.removeItem("lyricsLiveLowEnd");
            } catch (e) {
            }
        }
        if (currentTrackInfo) {
            params.set("artist", currentTrackInfo.artist || "");
            params.set("album", currentTrackInfo.album || "");
            params.set("track", currentTrackInfo.title || currentTrackInfo.track || "");
            params.set("position", String(currentPosition || 0));
            params.set("position_ms", String(currentPositionMs || 0));
            params.set("duration", String(trackDuration || 0));
            params.set("source", currentSource || "");
        }
        window.location.href = "/lyrics-live?" + params.toString();
    }

    function initLyricsFullscreenButton() {
        const btn = document.getElementById("lyricsFullscreenBtn");
        if (!btn) return;
        btn.addEventListener("click", openLyricsLivePage);
    }

    // ========== 歌词功能结束 ==========

    // 初始化 AI 歌词解析按钮
    function initAiInsightButton() {
        const reanalyzeButton = document.getElementById("reanalyzeInsightButton");
        // 移除冗余的 aiInsightButton 监听，HTML 中已声明 onclick="handleAiInsightClick()"
        
        if (reanalyzeButton) {
            reanalyzeButton.addEventListener("click", function () {
                // 点击“重新分析”调用模型选择器，上下文为 'nowPlaying'
                showModelPicker(null, 'nowPlaying');
            });
        }
    }

    // 初始化统计卡片
