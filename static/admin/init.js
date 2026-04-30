function createAutoUpdater(fetchFunction, interval = 60000) {
        let running = false;
        return setInterval(function () {
            if (running) return;
            if (document.hidden) return;

            running = true;
            let result;
            try {
                result = fetchFunction();
            } catch (e) {
                console.error("自动更新失败:", e);
                running = false;
                return;
            }
            Promise.resolve(result).then(
                function () {
                    running = false;
                },
                function (e) {
                    console.error("自动更新失败:", e);
                    running = false;
                }
            );
        }, interval);
    }

    function getActiveFilterValue(selector, attrName, defaultValue) {
        const el = document.querySelector(selector);
        if (!el) return defaultValue;
        const value = el.getAttribute(attrName);
        return value || defaultValue;
    }

    function isMainDashboardVisible() {
        const statsContainer = document.querySelector(".stats-container");
        return !!statsContainer && statsContainer.style.display !== "none";
    }

    // 公共的定时更新函数
    function createDashboardAutoUpdater() {
        // 存储所有定时器的ID
        const timers = [];

        // 清除所有定时器
        function clearAllTimers() {
            timers.forEach((timerId) => clearInterval(timerId));
            timers.length = 0; // 清空数组
        }

        // 添加定时器到数组
        function addTimer(timerId) {
            timers.push(timerId);
        }

        // 返回公共接口
        return {
            clearAllTimers,
            addTimer,
        };
    }

    // 创建全局的定时更新器实例
    const dashboardUpdater = createDashboardAutoUpdater();

    // 初始化拖动功能
    function initDraggableCards() {
        // 最近播放和播放排行榜不再支持拖动
    }

    // 页面加载完成后初始化
    document.addEventListener("DOMContentLoaded", function () {
        enablePerformanceModeIfNeeded();

        // 自动切换日夜模式
        toggleDarkModeBasedOnSystem();

        // 监听系统主题变化
        watchSystemThemeChange();

        // 初始化各个组件
        initStats();
        initSourceChart(); // 初始化播放来源扇形图
        initTrendChart(30); // 默认加载30天数据
        initArtistChart("plays"); // 默认按播放次数排序
        initAlbumChart(30); // 默认加载30天数据
        initGenreChart(); // 初始化热门流派图表
        initRanking("all"); // 默认总排行
        initRecentPlays(); // 初始化最近播放列表
        hasDashboardBootstrapped = true;

        // 连接到WebSocket服务器
        connectWebSocket();

        // 清除之前的定时器
        dashboardUpdater.clearAllTimers();

        // 设置定时器，每5秒刷新一次各个组件
        dashboardUpdater.addTimer(createAutoUpdater(() => {
            if (!isMainDashboardVisible()) return;
            initStats();
        }, getPollingInterval(60000))); // 每60秒更新统计卡片
        dashboardUpdater.addTimer(createAutoUpdater(() => {
            if (!isMainDashboardVisible()) return;
            initSourceChart();
        }, getPollingInterval(60000))); // 每60秒更新播放来源扇形图
        dashboardUpdater.addTimer(
            createAutoUpdater(
                () => {
                    if (!isMainDashboardVisible()) return;
                    initTrendChart(
                        getActiveFilterValue(
                            ".time-filter.active[data-range]",
                            "data-range",
                            30
                        )
                    );
                },
                getPollingInterval(60000)
            )
        ); // 每60秒更新趋势图
        dashboardUpdater.addTimer(
            createAutoUpdater(
                () => {
                    if (!isMainDashboardVisible()) return;
                    initArtistChart(
                        getActiveFilterValue(
                            ".time-filter.active[data-type]",
                            "data-type",
                            "plays"
                        )
                    );
                },
                getPollingInterval(60000)
            )
        ); // 每60秒更新艺术家图表
        dashboardUpdater.addTimer(
            createAutoUpdater(
                () => {
                    if (!isMainDashboardVisible()) return;
                    initAlbumChart(
                        getActiveFilterValue(
                            ".time-filter.active[data-days]",
                            "data-days",
                            30
                        )
                    );
                },
                getPollingInterval(60000)
            )
        ); // 每60秒更新专辑图表
        dashboardUpdater.addTimer(createAutoUpdater(() => {
            if (!isMainDashboardVisible()) return;
            initGenreChart();
        }, getPollingInterval(60000))); // 每60秒更新流派图表
        dashboardUpdater.addTimer(
            createAutoUpdater(
                () => {
                    if (!isMainDashboardVisible()) return;
                    initRanking(
                        getActiveFilterValue(
                            ".time-filter.active[data-ranking]",
                            "data-ranking",
                            "all"
                        )
                    );
                },
                getPollingInterval(60000)
            )
        ); // 每60秒更新排行榜
        dashboardUpdater.addTimer(createAutoUpdater(() => {
            if (!isMainDashboardVisible()) return;
            refreshRecentPlays();
        }, getPollingInterval(60000))); // 每60秒更新最近播放列表
        dashboardUpdater.addTimer(createAutoUpdater(() => {
            if (currentSectionID !== "insightJobList") return;
            loadInsightJobList(currentInsightJobPage);
        }, getPollingInterval(15000))); // 任务列表额外保留 15 秒轮询兜底

        if (insightJobDurationTicker) {
            clearInterval(insightJobDurationTicker);
        }
        insightJobDurationTicker = setInterval(() => {
            if (document.hidden) return;
            refreshInsightJobDurationsLocally();
        }, 1000);

        window.addEventListener("resize", () => {
            if (currentSectionID === "insightJobList") {
                syncInsightJobWorkspaceHeight();
            }
        });

        // 添加时间过滤器点击事件
        document.querySelectorAll(".time-filter").forEach((filter) => {
            // 跳过全屏按钮
            if (
                filter.id === "trendChartFullscreenBtn" ||
                filter.id === "genreChartFullscreenBtn"
            )
                return;

            filter.addEventListener("click", function () {
                // 移除同组其他过滤器的active类
                const parent = this.parentElement;
                parent.querySelectorAll(".time-filter").forEach((f) => {
                    // 跳过全屏按钮
                    if (
                        f.id === "trendChartFullscreenBtn" ||
                        f.id === "genreChartFullscreenBtn"
                    )
                        return;
                    f.classList.remove("active");
                });

                // 为当前过滤器添加active类
                this.classList.add("active");

                // 根据过滤器更新数据
                let filterType = "";
                const cardHeader = parent.classList.contains("card-header") ? parent : null;
                const chartCard = parent.closest(".chart-card");
                const rankingCard = parent.closest(".ranking-card");

                if (cardHeader) {
                    filterType = cardHeader.parentElement.querySelector(".card-title").textContent;
                } else if (chartCard) {
                    filterType = chartCard.querySelector(".card-title").textContent;
                } else if (rankingCard) {
                    filterType = rankingCard.querySelector(".card-title").textContent;
                }

                if (!filterType) return;

                if (filterType.includes("播放趋势")) {
                    // 更新趋势图
                    const range = this.getAttribute("data-range");
                    initTrendChart(range);
                } else if (filterType.includes("热门艺术家")) {
                    // 更新艺术家图表
                    const type = this.getAttribute("data-type");
                    initArtistChart(type);
                } else if (filterType.includes("热门专辑")) {
                    // 更新专辑图表
                    const days = this.getAttribute("data-days");
                    initAlbumChart(days);
                } else if (filterType.includes("播放排行榜")) {
                    // 更新排行榜
                    const ranking = this.getAttribute("data-ranking");
                    initRanking(ranking, rankingSearchKeyword);
                }
            });
        });

        // 全屏按钮点击事件
        document
            .getElementById("trendChartFullscreenBtn")
            .addEventListener("click", function () {
                toggleTrendChartFullscreen();
            });

        // 口味流派图表全屏按钮点击事件
        document
            .getElementById("genreChartFullscreenBtn")
            .addEventListener("click", function () {
                toggleGenreChartFullscreen();
            });

        // 页面加载完成后调整排行榜列表高度
        setTimeout(adjustRankingListsHeight, 1000);

        // 初始化拖动功能
        initDraggableCards();

        // 初始化正在播放悬浮窗的拖动功能
        initNowPlayingDrag();
        // 初始化 AI 歌词解析按钮
        initAiInsightButton();
        // 初始化歌词按钮和拖动功能
        initLyricsButton();
        initLyricsFullscreenButton();
        initLyricsDrag();

        // 列表点击事件委托：避免每次刷新都重复绑定每行点击监听器
        const recentPlaysList = document.getElementById("recentPlaysList");
        if (recentPlaysList && !recentPlaysList.dataset.clickBound) {
            recentPlaysList.addEventListener("click", function (e) {
                const item = e.target.closest(".ranking-item");
                if (!item || !recentPlaysList.contains(item)) return;
                const artist = item.dataset.artist || "";
                const album = item.dataset.album || "";
                const track = item.dataset.track || "";
                const trackNumber = Number(item.dataset.trackNumber || 0);
                const discNumber = Number(item.dataset.discNumber || 0);
                if (artist && track) {
                    showTrackDetails(artist, album, track, trackNumber, discNumber, {
                        playTime: item.dataset.playTime || "",
                        source: item.dataset.source || "",
                        scrobbled: item.dataset.scrobbled === "1",
                        traceId: item.dataset.traceId || "",
                        rootSpanId: item.dataset.rootSpanId || "",
                        traceSampled: item.dataset.traceSampled === "1",
                        resolutionStatus: item.dataset.resolutionStatus || "",
                    });
                }
            });
            recentPlaysList.dataset.clickBound = "1";
        }

        const rankingList = document.getElementById("rankingList");
        if (rankingList && !rankingList.dataset.clickBound) {
            rankingList.addEventListener("click", function (e) {
                const item = e.target.closest(".ranking-item");
                if (!item || !rankingList.contains(item)) return;
                const artist = item.dataset.artist || "";
                const album = item.dataset.album || "";
                const track = item.dataset.track || "";
                const trackNumber = Number(item.dataset.trackNumber || 0);
                const discNumber = Number(item.dataset.discNumber || 0);
                if (artist && track) {
                    showTrackDetails(artist, album, track, trackNumber, discNumber);
                }
            });
            rankingList.dataset.clickBound = "1";
        }

        // 未上报标签页点击事件
        document
            .getElementById("unscrobbledTab")
            .addEventListener("click", function (e) {
                e.preventDefault();

                // 隐藏其他内容
                document.querySelector(".stats-container").style.display = "none";
                document.querySelector(".charts-container").style.display = "none";
                document.querySelector(".rankings-container").style.display =
                    "none";

                // 显示未上报容器
                document.getElementById("unscrobbledContainer").style.display =
                    "block";

                // 更新导航栏活动状态
                document.querySelectorAll(".nav-links a").forEach((link) => {
                    link.classList.remove("active");
                });
                this.classList.add("active");

                // 加载未上报记录
                currentUnscrobbledPage = 1;
                loadUnscrobbledRecords();
            });

        // 刷新按钮点击事件
        document
            .getElementById("refreshUnscrobbledBtn")
            .addEventListener("click", function () {
                loadUnscrobbledRecords();
            });

        // 同步选中按钮点击事件
        document
            .getElementById("syncSelectedBtn")
            .addEventListener("click", function () {
                syncSelectedRecords();
            });

        // 分页按钮事件
        document
            .getElementById("prevPage")
            .addEventListener("click", function () {
                if (currentUnscrobbledPage > 1) {
                    currentUnscrobbledPage--;
                    loadUnscrobbledRecords();
                }
            });

        document
            .getElementById("nextPage")
            .addEventListener("click", function () {
                const totalPages = Math.ceil(
                    totalUnscrobbledRecords / unscrobbledPageSize
                );
                if (currentUnscrobbledPage < totalPages) {
                    currentUnscrobbledPage++;
                    loadUnscrobbledRecords();
                }
            });

        // 点赞按钮点击事件
        document
            .getElementById("favoriteButton")
            .addEventListener("click", handleFavoriteButtonClick);

        // 音眸列表标签页点击事件
        document.getElementById("insightListTab").addEventListener("click", function (e) {
            e.preventDefault();
            document.querySelector(".stats-container").style.display = "none";
            document.querySelector(".charts-container").style.display = "none";
            document.querySelector(".rankings-container").style.display = "none";
            document.getElementById("unscrobbledContainer").style.display = "none";
            document.getElementById("insightListContainer").style.display = "block";

            document.querySelectorAll(".nav-links a").forEach((link) => link.classList.remove("active"));
            this.classList.add("active");

            currentInsightPage = 1;
            loadInsightList();
        });

        // 音眸列表相关事件
        const refreshBtn = document.getElementById("refreshInsightListBtn");
        if (refreshBtn) {
            refreshBtn.addEventListener("click", () => {
                currentInsightPage = 1;
                loadInsightList();
            });
        }

        const searchInput = document.getElementById("insightSearchInput");
        if (searchInput) {
            searchInput.addEventListener("input", (e) => {
                const val = e.target.value.trim();
                if (insightSearchTimeout) clearTimeout(insightSearchTimeout);
                
                insightSearchTimeout = setTimeout(() => {
                    currentInsightKeyword = val;
                    currentInsightPage = 1;
                    loadInsightList();
                }, 500);
            });
        }

        const insightJobRefreshBtn = document.getElementById("refreshInsightJobListBtn");
        if (insightJobRefreshBtn) {
            insightJobRefreshBtn.addEventListener("click", () => {
                currentInsightJobPage = 1;
                loadInsightJobList();
            });
        }

        const insightJobSearchInput = document.getElementById("insightJobSearchInput");
        if (insightJobSearchInput) {
            insightJobSearchInput.addEventListener("input", (e) => {
                const val = e.target.value.trim();
                if (insightJobSearchTimeout) clearTimeout(insightJobSearchTimeout);

                insightJobSearchTimeout = setTimeout(() => {
                    currentInsightJobKeyword = val;
                    currentInsightJobPage = 1;
                    loadInsightJobList();
                }, 400);
            });
        }

        const insightJobStatusFilter = document.getElementById("insightJobStatusFilter");
        if (insightJobStatusFilter) {
            insightJobStatusFilter.addEventListener("change", (e) => {
                currentInsightJobStatus = e.target.value;
                currentInsightJobPage = 1;
                loadInsightJobList();
            });
        }

        const insightJobTargetFilter = document.getElementById("insightJobTargetFilter");
        if (insightJobTargetFilter) {
            insightJobTargetFilter.addEventListener("change", (e) => {
                currentInsightJobTargetType = e.target.value;
                currentInsightJobPage = 1;
                loadInsightJobList();
            });
        }

        // 播放排行榜搜索
        const rankingSearchInput = document.getElementById("rankingSearchInput");
        if (rankingSearchInput) {
            rankingSearchInput.addEventListener("input", (e) => {
                const val = e.target.value.trim();
                if (rankingSearchTimeout) clearTimeout(rankingSearchTimeout);
                
                rankingSearchTimeout = setTimeout(() => {
                    initRanking(currentRankingType, val);
                }, 500);
            });
        }
        document.getElementById("insightPrevPage").addEventListener("click", () => {
            if (currentInsightPage > 1) {
                currentInsightPage--;
                loadInsightList();
            }
        });
        document.getElementById("insightNextPage").addEventListener("click", () => {
            const totalPages = Math.ceil(totalInsights / insightPageSize);
            if (currentInsightPage < totalPages) {
                currentInsightPage++;
                loadInsightList();
            }
        });
        document.getElementById("closeCallLogModal").addEventListener("click", () => hideModal("callLogModal"));
        document.getElementById("insightJobPrevPage").addEventListener("click", () => {
            if (currentInsightJobPage > 1) {
                currentInsightJobPage--;
                loadInsightJobList();
            }
        });
        document.getElementById("insightJobNextPage").addEventListener("click", () => {
            const totalPages = Math.max(1, Math.ceil(totalInsightJobs / insightJobPageSize));
            if (currentInsightJobPage < totalPages) {
                currentInsightJobPage++;
                loadInsightJobList();
            }
        });
        document.getElementById("closeInsightJobModal").addEventListener("click", () => hideModal("insightJobModal"));
    });

    // 添加查看详情模态框相关函数
    // 显示模态框
    // 统一转义函数：修复 Unexpected EOF 并兼容 HTML 属性中的 JS 字符串
    function esc(s) {
        if (!s) return "";
        return String(s)
            .replace(/\\/g, '\\\\')  // 转义反斜杠
            .replace(/'/g, "\\'")   // 转义单引号 (针对 JS 字符串)
            .replace(/"/g, '&quot;') // 转义双引号 (针对 HTML 属性)
            .replace(/\n/g, '\\n')  // 转义换行
            .replace(/\r/g, '\\r');
    }

    // 显示模态框
    function showModal(modalId) {
        const el = document.getElementById(modalId);
        if (el) el.style.display = "block";
    }

    // 隐藏模态框
    function hideModal(modalId) {
        document.getElementById(modalId).style.display = "none";

        if (modalId === "insightJobModal") {
            currentInsightJobDetail = null;
        }
        
        // 增加清理逻辑
        if (modalId === 'nowPlayingInsightModal' || modalId === 'listInsightModal') {
            const contextType = modalId === 'nowPlayingInsightModal' ? 'nowPlaying' : 'list';
            const state = insightStates[contextType];
            
            // 1. 关闭 SSE
            if (state.eventSource) {
                state.eventSource.close();
                state.eventSource = null;
            }
            
            // 2. 清空 DOM
            const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
            if (streamContent) {
                streamContent.innerHTML = '<div class="loading">准备中...</div>';
            }
            
            // 3. 重置状态 (除 trackInfo 外)
            state.insight = null;
            state.allInsights = [];
            
            // 如果两个主要的都没了，兼容性的 currentTrackInsight 也可以清了
            if (!insightStates.nowPlaying.insight && !insightStates.list.insight) {
                currentTrackInsight = null;
            }

            // 切换回第一个 Tab (UI)
            switchInsightTab('summary', contextType);
        }
    }

    // 获取并显示曲目详细信息
    function escapeDetailHtml(value) {
        return String(value || "")
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }

    function buildRecentPlayDetailSection(recentPlay) {
        if (!recentPlay) {
            return "";
        }

        const playTimeText = recentPlay.playTime
            ? new Date(recentPlay.playTime).toLocaleString("zh-CN")
            : "未知";
        const sourceText = recentPlay.source || "未知";
        const scrobbledText = recentPlay.scrobbled ? "是" : "否";
        const sampledText = recentPlay.traceSampled ? "是" : "否";
        const traceIDText = recentPlay.traceId || "未记录";
        const rootSpanIDText = recentPlay.rootSpanId || "未记录";
        const resolutionStatusText = recentPlay.resolutionStatus || "未知";

        return `
            <div style="margin-top: 20px; padding: 16px 18px; border-radius: 16px; background: rgba(15, 23, 42, 0.04); border: 1px solid var(--border-color);">
              <div style="font-size: 0.95rem; font-weight: 700; margin-bottom: 12px;">最近播放链路</div>
              <p><strong>播放时间:</strong> ${escapeDetailHtml(playTimeText)}</p>
              <p><strong>播放来源:</strong> ${escapeDetailHtml(sourceText)}</p>
              <p><strong>已上报:</strong> ${scrobbledText}</p>
              <p><strong>归因状态:</strong> ${escapeDetailHtml(resolutionStatusText)}</p>
              <p><strong>Trace ID:</strong> <span style="font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace; font-size: 0.9em; word-break: break-all;">${escapeDetailHtml(traceIDText)}</span></p>
              <p><strong>Root Span ID:</strong> <span style="font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace; font-size: 0.9em; word-break: break-all;">${escapeDetailHtml(rootSpanIDText)}</span></p>
              <p><strong>Trace Sampled:</strong> ${sampledText}</p>
            </div>
        `;
    }

    async function showTrackDetails(artist, album, trackName, trackNumber = 0, discNumber = 0, recentPlay = null) {
        // 1. 彻底清空旧数据，防止数据污染
        window.currentDetailData = {
            track: null,
            insight: null,
            recommendedInsightID: 0,
            loadingTrack: true,
            loadingInsight: true,
            recentPlay: recentPlay
        };

        const modal = document.getElementById("trackDetailsModal");
        const modalContent = modal.querySelector(".modal-content");

        // 2. 初始化骨架结构
        modalContent.innerHTML = `
          <div class="module-trail"></div>
          <div class="modal-header">
            <h2 id="detailTrackTitle">${trackName}</h2>
            <span class="close" onclick="hideModal('trackDetailsModal')">×</span>
          </div>
          <div class="modal-tabs">
            <div class="modal-tab active" onclick="switchDetailTab('basic')"><span>📊</span> 基础信息</div>
            <div class="modal-tab" onclick="switchDetailTab('lyrics')"><span>📜</span> 歌曲歌词</div>
            <div class="modal-tab" onclick="switchDetailTab('ai')"><span>🤖</span> 音眸</div>
          </div>
          <div class="modal-body" id="detailModalBody">
            <div class="loading">正在加载曲目基础信息...</div>
          </div>
        `;

        showModal("trackDetailsModal");

        // 3. 异步获取基础详情 (不阻塞)
        fetch(`/api/track?artist=${encodeURIComponent(artist)}&album=${encodeURIComponent(album)}&trackName=${encodeURIComponent(trackName)}&trackNumber=${encodeURIComponent(trackNumber || 0)}&discNumber=${encodeURIComponent(discNumber || 0)}`)
            .then(r => r.ok ? r.json() : null)
            .then(data => {
                window.currentDetailData.track = data;
                window.currentDetailData.loadingTrack = false;
                // 只有当用户还在基础信息 Tab 时才自动渲染
                const activeTab = document.querySelector("#trackDetailsModal .modal-tab.active");
                if (activeTab && activeTab.innerText.includes("基础信息")) {
                    renderTrackDetailsContent('basic');
                }
            })
            .catch(e => {
                console.error("加载基础信息失败:", e);
                window.currentDetailData.loadingTrack = false;
            });

        // 4. 异步获取已有的 AI Insight (仅查询 GET)
        const insightUrl = `/api/track-insight?artist=${encodeURIComponent(artist)}&album=${encodeURIComponent(album)}&track=${encodeURIComponent(trackName)}&trackNumber=${encodeURIComponent(trackNumber || 0)}&discNumber=${encodeURIComponent(discNumber || 0)}`;
        fetch(insightUrl)
            .then(r => r.ok ? r.json() : null)

            .then(data => {
                // 兼容多种返回格式
                let insights = [];
                if (data) {
                    if (data.insights) insights = data.insights;
                    else if (data.insight) insights = [data.insight];
                    else if (Array.isArray(data)) insights = data;
                }
                window.currentDetailData.recommendedInsightID = Number(
                    data?.recommended_insight_id || data?.recommendedInsightID || 0,
                );
                window.currentDetailData.insights = insights;
                window.currentDetailData.loadingInsight = false;
                // 如果用户这时已经切换到了 AI Tab，则在加载完后刷新渲染
                const activeTab = document.querySelector("#trackDetailsModal .modal-tab.active");
                if (activeTab && activeTab.innerText.includes("AI 分析")) {
                    renderTrackDetailsContent('ai');
                }
            })
            .catch(e => {
                console.error("查询 AI 历史记录失败:", e);
                window.currentDetailData.loadingInsight = false;
            });
    }

    // Tab 切换逻辑
    function switchDetailTab(tabName) {
        const tabs = document.querySelectorAll("#trackDetailsModal .modal-tab");
        tabs.forEach(t => t.classList.remove("active"));

        if (tabName === 'basic') tabs[0].classList.add("active");
        else if (tabName === 'lyrics') tabs[1].classList.add("active");
        else tabs[2].classList.add("active");

        renderTrackDetailsContent(tabName);
    }

    // 渲染曲目详情具体内容
    function renderTrackDetailsContent(tabName) {
        const bodyContent = document.getElementById("detailModalBody");
        const data = window.currentDetailData;

        if (tabName === 'basic') {
            if (data.loadingTrack) {
                bodyContent.innerHTML = '<div class="loading">正在加载基础信息...</div>';
                return;
            }
            if (!data.track) {
                bodyContent.innerHTML = '<div class="error">未找到曲目基础信息</div>';
                return;
            }
            const track = data.track;
            const recentPlaySection = buildRecentPlayDetailSection(data.recentPlay);
            const formatDuration = (seconds) => {
                const mins = Math.floor(seconds / 60);
                const secs = Math.floor(seconds % 60);
                return `${mins}:${secs.toString().padStart(2, "0")}`;
            };

            let sourceClass = "";
            let sourceText = "";
            switch ((track.source || "").toLowerCase()) {
                case "apple music":
                    sourceClass = "source-applemusic";
                    sourceText = "AppleMusic";
                    break;
                case "audirvana":
                case "au":
                    sourceClass = "source-audirvana";
                    sourceText = "Audirvana";
                    break;
                case "roon":
                    sourceClass = "source-roon";
                    sourceText = "Roon";
                    break;
                default:
                    sourceClass = "";
                    sourceText = track.source || "Unknown";
            }

            bodyContent.innerHTML = `
            <div style="display: flex; gap: 20px; align-items: flex-start;">
              <div
                id="trackDetailArtwork"
                class="artwork-slot"
                style="width: 120px; height: 120px; border-radius: 18px; flex-shrink: 0; box-shadow: 0 12px 24px rgba(0,0,0,0.12);"
              >${renderArtworkPlaceholder(track.album || "")}</div>
              <div class="track-info" style="flex: 1; min-width: 0;">
                <p><strong>艺术家:</strong> ${track.artist}</p>
                <p><strong>专辑:</strong> ${track.album_id ? `<a href="javascript:void(0)" onclick="hideModal('trackDetailsModal'); showAlbumDetails(${track.album_id})" style="color: var(--primary-color); text-decoration: underline; cursor: pointer;">${track.album}</a>` : track.album}</p>
                <p><strong>播放次数:</strong> ${track.play_count || 0}</p>
                ${track.duration ? `<p><strong>时长:</strong> ${formatDuration(track.duration)}</p>` : ''}
                ${track.track_number ? `<p><strong>曲目编号:</strong> ${track.track_number}</p>` : ''}
                ${track.genre ? `<p><strong>流派:</strong> ${track.genre}</p>` : ''}
                <p><strong>喜欢苹果音乐:</strong> ${track.is_apple_music_fav ? "是" : "否"}</p>
                <p><strong>喜欢Last.fm:</strong> ${track.is_last_fm_fav ? "是" : "否"}</p>
                <p><strong>数据来源:</strong> <span class="play-source ${sourceClass}">${sourceText}</span></p>
                <p><strong>更新时间:</strong> ${new Date(track.updated_at).toLocaleString('zh-CN')}</p>
                ${recentPlaySection}
              </div>
            </div>
          `;
            hydrateArtworkSlot(
                document.getElementById("trackDetailArtwork"),
                {
                    albumID: track.album_id || 0,
                    artist: track.artist || "",
                    album: track.album || "",
                    coverArtURL: track.cover_art_url || "",
                },
                {
                    altText: track.album || track.track || "专辑封面",
                },
            );
        } else if (tabName === 'lyrics') {
            // 歌词 Tab
            const track = data.track;
            if (!track) {
                bodyContent.innerHTML = '<div class="loading">正在加载数据...</div>';
                return;
            }

            bodyContent.innerHTML = '<div class="loading">正在查询歌词...</div>';

            fetch(`/api/track-lyrics?artist=${encodeURIComponent(track.artist)}&album=${encodeURIComponent(track.album || '')}&track=${encodeURIComponent(track.track)}&trackNumber=${encodeURIComponent(track.track_number || 0)}&discNumber=${encodeURIComponent(track.disc_number || 0)}`)
                .then(r => r.json())
                .then(lyricsData => {
                    if (lyricsData.lyrics) {
                        bodyContent.innerHTML = "";
                        const lyricsContainer = document.createElement("div");
                        lyricsContainer.className = "lyrics-container";
                        lyricsContainer.style.padding = "20px";
                        lyricsContainer.style.fontFamily = "sans-serif";
                        lyricsContainer.style.lineHeight = "1.8";
                        lyricsContainer.style.textAlign = "center";
                        renderPlainLyrics(lyricsContainer, lyricsData.lyrics);
                        bodyContent.appendChild(lyricsContainer);
                    } else {
                        bodyContent.innerHTML = `
                  <div style="text-align: center; padding: 40px 20px;">
                    <div style="font-size: 3rem; margin-bottom: 20px;">🎶</div>
                    <p style="color: var(--text-secondary); margin-bottom: 20px;">本地暂无该曲目的歌词数据</p>
                    <button class="action-btn" onclick="triggerLyricsSearch()" style="background: var(--primary-color); color: white; padding: 10px 25px; border-radius: 20px; font-weight: 600;">🔍 立即在线查询</button>
                    <div id="lyricSearchStatus" style="margin-top: 15px; font-size: 0.9rem; color: var(--primary-color);"></div>
                  </div>
                `;
                    }
                })
                .catch(err => {
                    bodyContent.innerHTML = `<div class="error">查询失败: ${err.message}</div>`;
                });

        } else {
            // AI Tab
            if (data.loadingInsight) {
                bodyContent.innerHTML = '<div class="loading">正在查询 AI 解析历史...</div>';
                return;
            }

            const insights = data.insights || [];
            const recommendedInsightID = Number(data.recommended_insight_id || data.recommendedInsightID || 0);
            const selectedInsightIndex = recommendedInsightID > 0
                ? insights.findIndex((item) => Number(item.id || 0) === recommendedInsightID)
                : 0;
            const activeInsightIndex = selectedInsightIndex >= 0 ? selectedInsightIndex : 0;

            if (insights.length === 0) {
                bodyContent.innerHTML = `
              <div style="text-align: center; padding: 40px 20px;">
                <div style="font-size: 3rem; margin-bottom: 20px;">🔍</div>
                <p style="color: var(--text-secondary); margin-bottom: 20px;">暂无该曲目的 AI 深度分析</p>
                <div style="display: flex; justify-content: center; gap: 10px;">
                   <button class="action-btn" onclick="hideModal('trackDetailsModal'); showModelPicker({artist: window.currentDetailData.track.artist, album: window.currentDetailData.track.album, title: window.currentDetailData.track.track, track_number: window.currentDetailData.track.track_number, disc_number: window.currentDetailData.track.disc_number}, 'details');" style="background: var(--primary-color); color: white; padding: 10px 25px; border-radius: 20px; font-weight: 600;">✨ 开始分析</button>
                </div>
              </div>
            `;
                return;
            }

            let html = '';

            // 如果有多个结果，显示 Tab
            if (insights.length > 1) {
                const positiveTotal = insights.reduce((sum, i) => sum + Math.max(0, i.total_score || 0), 0);
                html += '<div class="insight-tabs">';
                insights.forEach((insight, index) => {
                    const date = new Date(insight.created_at || Date.now());
                    const timeStr = date.toLocaleString('zh-CN', {
                        month: '2-digit',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit'
                    });
                    const activeClass = index === activeInsightIndex ? 'active' : '';
                    const totalScore = insight.total_score || 0;
                    let supportRate;
                    if (positiveTotal > 0) {
                        supportRate = ((totalScore / positiveTotal) * 100).toFixed(1);
                    } else {
                        supportRate = totalScore === 0 ? '0.0' : (totalScore > 0 ? '100.0' : '-100.0');
                    }
                    html += `<div class="insight-tab ${activeClass}" onclick="switchDetailInsightTab(${index})" title="总分: ${totalScore} | 支持率: ${supportRate}%">分析 ${insights.length - index} <small>(${timeStr})</small></div>`;
                });
                html += '</div>';
            }

            html += '<div id="detail-insight-content-container">';
            insights.forEach((insight, index) => {
                const displayStyle = index === activeInsightIndex ? 'block' : 'none';
                html += `<div class="insight-item" id="detail-insight-item-${index}" style="display: ${displayStyle};">`;
                html += generateInsightContentHtml(insight);
                // 添加一致的操作按钮行
                html += `
                <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 20px; padding-top: 15px; border-top: 1px solid var(--border-color); gap: 15px;">
                  <button class="action-btn" style="flex: 1; padding: 8px 15px; border-radius: 20px; font-weight: 600;" onclick="hideModal('trackDetailsModal'); showModelPicker({artist: window.currentDetailData.track.artist, album: window.currentDetailData.track.album, title: window.currentDetailData.track.track, track_number: window.currentDetailData.track.track_number, disc_number: window.currentDetailData.track.disc_number}, 'details');">🔄 重新分析</button>
                  <button class="action-btn" style="flex: 1; padding: 8px 15px; border-radius: 20px; font-weight: 600; background: rgba(52, 152, 219, 0.1); color: var(--primary-color); border-color: var(--primary-color);" onclick="shareDetailInsight(${index})">分享图片</button>
                  <div style="display: flex; gap: 10px; flex: 1; justify-content: flex-end;">
                    <button class="action-btn" style="padding: 6px 12px; border-radius: 15px; background: rgba(46, 204, 113, 0.1); color: var(--secondary-color); border-color: var(--secondary-color);" onclick="recordAiFeedback(1)">👍 赞</button>
                    <button class="action-btn" style="padding: 6px 12px; border-radius: 15px; background: rgba(231, 76, 60, 0.1); color: var(--accent-color); border-color: var(--accent-color);" onclick="recordAiFeedback(-1)">👎 踩</button>
                  </div>
                </div>
              `;
                html += '</div>';
            });
            html += '</div>';

            bodyContent.innerHTML = html;
            // 保存 insights 到全局变量，以便切换 tab 和点赞点踩使用
            window.currentDetailData.insights = insights;
            window.currentDetailData.recommendedInsightID = recommendedInsightID;
            // 更新选中的 ID，以便全局 feedback 可用
            currentTrackInsight = insights[activeInsightIndex] || insights[0];
        }
    }
    window.showTrackDetails = showTrackDetails;

    async function triggerLyricsSearch() {
        const track = window.currentDetailData.track;
        if (!track) return;

        const statusEl = document.getElementById("lyricSearchStatus");
        statusEl.textContent = "正在通过网络搜索歌词，请稍候...";

        try {
            // 假设后端接口支持 force 参数来触发刷新，或者通过这个接口本身就能完成搜索并保存
            // 这里调用的是相同的接口，但由于之前查询结果为空，点击此按钮通常由于用户想手动触发搜索
            const resp = await fetch(`/api/track-lyrics?artist=${encodeURIComponent(track.artist)}&album=${encodeURIComponent(track.album || '')}&track=${encodeURIComponent(track.track)}&trackNumber=${encodeURIComponent(track.track_number || 0)}&discNumber=${encodeURIComponent(track.disc_number || 0)}&force=true`);
            const data = await resp.json();
            const cacheKey = buildLyricsCacheKey(track.artist, track.album || '', track.track, track.track_number, track.disc_number);
            lyricsCache[cacheKey] = { ts: Date.now(), data: data };

            if (data.lyrics) {
                statusEl.textContent = "找到歌词！正在刷新...";
                setTimeout(() => switchDetailTab('lyrics'), 1000);
            } else {
                statusEl.textContent = "抱歉，依然未找到歌词。";
            }
        } catch (e) {
            statusEl.textContent = "查询出错: " + e.message;
        }
    }

    function switchDetailInsightTab(index) {
        console.log("switchDetailInsightTab:", index);
        const container = document.getElementById("detailModalBody");
        container.querySelectorAll('.insight-tab').forEach((tab, i) => {
            if (i === index) tab.classList.add('active');
            else tab.classList.remove('active');
        });
        container.querySelectorAll('.insight-item').forEach((item, i) => {
            if (i === index) item.style.display = 'block';
            else item.style.display = 'none';
        });
        // 更新当前选中的 insight，以便点赞点踩使用正确的 ID
        if (window.currentDetailData && window.currentDetailData.insights && window.currentDetailData.insights[index]) {
            currentTrackInsight = window.currentDetailData.insights[index];
            console.log("Detail currentTrackInsight updated to index", index, ":", currentTrackInsight);
        }
    }
    window.showTrackDetails = window.showTrackDetails || showTrackDetails;

    /**
     * 显示专辑详情模态框
     * @param {number} id 专辑 ID
     */
    function showAlbumDetails(id, selectedTab = 'info') {
        if (!id) return;
        const previousAlbumID = Number(window.currentAlbumID || 0);
        window.currentAlbumID = id; // 记录当前处理的专辑 ID
        window.currentAlbumSyncStatus = 0; // 记录当前专辑的同步状态
        window.currentAlbumDetailData = { albumID: id, album: null, insights: [], recommendedInsightID: 0 };
        
        // 检测当前主题
        const isDark = document.body.classList.contains("dark-mode");
        const itemBorder = isDark ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.08)';
        const itemHover = isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.03)';
        
        const modal = document.getElementById('albumDetailsModal');
        const trackList = document.getElementById('albumTrackList');
        const titleEl = document.getElementById('albumDetailTitle');
        const nameEl = document.getElementById('ad-name');
        const artistEl = document.getElementById('ad-artist');
        const releaseEl = document.getElementById('ad-release');
        const genreEl = document.getElementById('ad-genre');
        const countryEl = document.getElementById('ad-country');
        const statusEl = document.getElementById('ad-status');
        const packagingEl = document.getElementById('ad-packaging');
        const barcodeEl = document.getElementById('ad-barcode');
        const totalDiscsEl = document.getElementById('ad-total-discs');
        const adIdEl = document.getElementById('ad-id');
        const mbidEl = document.getElementById('ad-mbid');
        const coverEl = document.getElementById('albumCoverPlaceholder');
        const mbSearchBtn = document.getElementById('mbSearchBtn');
        const mbDeepBtn = document.getElementById('mbDeepBtn');
        const candidateList = document.getElementById('mbCandidateList');
        
        // 显示模态框并重置状态
        modal.style.display = 'block';
        trackList.innerHTML = '<div class="loading">加载中...</div>';
        if (titleEl) titleEl.textContent = '专辑详情';
        nameEl.textContent = '加载中...';
        artistEl.textContent = '-';
        releaseEl.textContent = '-';
        genreEl.textContent = '-';
        if (countryEl) countryEl.textContent = '-';
        if (statusEl) statusEl.textContent = '-';
        if (packagingEl) packagingEl.textContent = '-';
        if (barcodeEl) barcodeEl.textContent = '-';
        if (totalDiscsEl) totalDiscsEl.textContent = '-';
        if (adIdEl) adIdEl.textContent = id; // 直接填充传入的 ID
        if (mbidEl) mbidEl.textContent = '-';
        if (coverEl) {
            applyArtworkPlaceholder(coverEl, "", {
                placeholderStyle: "font-size: 3em; opacity: 0.92;",
            });
        }
        mbSearchBtn.style.display = 'block';
        mbDeepBtn.style.display = 'none';
        candidateList.style.display = 'none';
        albumInsightState.albumID = id;
        albumInsightState.albumMeta = null;
        albumInsightState.insights = [];
        albumInsightState.insight = null;
        if (pendingAlbumInsightFocusID > 0) {
            albumInsightState.focusInsightID = Number(pendingAlbumInsightFocusID || 0);
        } else if (albumInsightState.focusInsightID && previousAlbumID === id) {
            albumInsightState.focusInsightID = Number(albumInsightState.focusInsightID || 0);
        } else {
            albumInsightState.focusInsightID = 0;
        }
        pendingAlbumInsightFocusID = 0;
        albumInsightState.view = 'summary';
        albumInsightState.loading = true;
        albumInsightState.generating = false;
        albumInsightState.lastError = '';
        switchAlbumDetailTab(selectedTab);
        renderAlbumInsightPanel();
        fetchAlbumInsight(false, id);

        const normalizeAlbumTrackCompareKey = (value) => {
            return String(value || '')
                .trim()
                .replaceAll('（', '(')
                .replaceAll('）', ')')
                .replaceAll('，', ',')
                .replaceAll('’', "'")
                .toLocaleLowerCase();
        };
        
        fetch(`/api/albums/${id}`)
            .then(resp => {
                if (!resp.ok) throw new Error('API 响应失败');
                return resp.json();
            })
            .then(data => {
                if (window.currentAlbumID !== id) {
                    return;
                }
                window.currentAlbumDetailData.album = data;
                albumInsightState.albumMeta = data;
                nameEl.textContent = data.name || '未知专辑';
                nameEl.title = data.name || '未知专辑';
                artistEl.textContent = data.artist || '未知艺术家';
                artistEl.title = data.artist || '未知艺术家';
                if (titleEl) {
                    titleEl.textContent = `${data.artist || '未知艺术家'} · ${data.name || '未知专辑'}`;
                }
                releaseEl.textContent = data.release_date || '-';
                releaseEl.title = data.release_date || '-';
                genreEl.textContent = data.genre || '-';
                genreEl.title = data.genre || '-';
                if (countryEl) {
                    countryEl.textContent = data.country || '-';
                    countryEl.title = data.country || '-';
                }
                if (statusEl) {
                    statusEl.textContent = data.status || '-';
                    statusEl.title = data.status || '-';
                }
                if (packagingEl) {
                    packagingEl.textContent = data.packaging || '-';
                    packagingEl.title = data.packaging || '-';
                }
                if (barcodeEl) {
                    barcodeEl.textContent = data.barcode || '-';
                    barcodeEl.title = data.barcode || '-';
                }
                if (totalDiscsEl) {
                    let text = data.total_discs ? `${data.total_discs} 碟` : '1 碟';
                    if (data.disc_infos && Object.keys(data.disc_infos).length > 0) {
                        try {
                            const parsed = JSON.parse(data.disc_infos);
                            const breakdown = Object.keys(parsed).map(k => `${parsed[k]}首`).join(' + ');
                            text += ` (${breakdown})`;
                        } catch(e) {}
                    }
                    totalDiscsEl.textContent = text;
                    totalDiscsEl.title = text;
                }
                if (mbidEl) {
                    const mbid = (data.release_mb && data.release_mb.mbid) ? data.release_mb.mbid : '-';
                    mbidEl.textContent = mbid;
                    mbidEl.title = mbid;
                }
                if (coverEl) {
                    hydrateArtworkSlot(
                        coverEl,
                        {
                            albumID: data.id || id,
                            artist: data.artist || "",
                            album: data.name || "",
                            coverArtURL: data.cover_art_url || "",
                        },
                        {
                            altText: data.name || "专辑封面",
                            placeholderStyle: "font-size: 3em; opacity: 0.92;",
                        },
                    );
                }
                
                // 根据同步状态更新 UI 按钮和候选列表
                updateMBUI(data.sync_status || 0);
                renderAlbumInsightPanel();
                
                if (data.track_album && data.track_album.length > 0) {
                    let html = '<ul style="list-style: none; padding: 0; margin: 0;">';
                    const tracksMap = {};
                    if (data.tracks) {
                        data.tracks.forEach(t => {
                            tracksMap[t.id] = t;
                        });
                    }

                    let currentDisc = -1;
                    const showDiscHeaders = data.total_discs > 1;

                    // 显式排序保证顺序稳定
                    const sortedTA = [...data.track_album].sort((a, b) => {
                        if (a.disc_number !== b.disc_number) return a.disc_number - b.disc_number;
                        return a.track_number - b.track_number;
                    });

                    sortedTA.forEach((ta, index) => {
                        const track = tracksMap[ta.track_id] || {
                            artist: data.artist,
                            album: data.name,
                            track: ta.track,
                            play_count: 0
                        };
                        
                        const mbTrackName = ta.track || '';
                        const mbRecordingID = ta.mb_recording_id || '';
                        const trackMusicBrainzID = track.music_brainz_id || '';
                        const hasMBIDMismatch = Boolean(mbRecordingID && trackMusicBrainzID && mbRecordingID !== trackMusicBrainzID);
                        const hasNameDiff = Boolean(
                            mbTrackName &&
                            track.track &&
                            normalizeAlbumTrackCompareKey(mbTrackName) !== normalizeAlbumTrackCompareKey(track.track)
                        );
                        const hasDiff = hasMBIDMismatch || (!trackMusicBrainzID && hasNameDiff);
                        const hasLink = ta.track_id && ta.track_id > 0;
                        
                        const trackDisc = ta.disc_number || 1;
                        if (showDiscHeaders && trackDisc !== currentDisc) {
                            html += `
                                <li style="padding: 15px 0 5px 0; margin-top: 5px; border-bottom: 1px solid var(--border-color); color: var(--text-primary); font-weight: 700; opacity: 0.8; font-size: 0.9em;">
                                    碟 ${trackDisc}
                                </li>
                            `;
                            currentDisc = trackDisc;
                        }

                        const trackNumDisplay = ta.track_number || (index + 1);
                        const isPlaceholder = !hasLink;

                        html += `
                            <li class="ranking-item" style="display: flex; gap: 15px; padding: 12px 0; border-bottom: 1px solid ${itemBorder}; ${isPlaceholder ? 'opacity: 0.5; filter: grayscale(0.5);' : ''}" 
                                ${hasLink ? `onclick="showTrackDetails('${esc(track.artist)}', '${esc(track.album)}', '${esc(track.track)}', ${Number(track.track_number || ta.track_number || 0)}, ${Number(track.disc_number || ta.disc_number || 0)})"` : ''}
                                onmouseover="${hasLink ? `this.style.background='${itemHover}'` : ''}" 
                                onmouseout="this.style.background='transparent'">
                                
                                <div style="width: 45px; font-weight: 700; opacity: 0.3; font-size: 0.9em; flex-shrink: 0; align-self: center; text-align: center;">${trackNumDisplay}</div>
                                
                                <div style="flex: 1; min-width: 0; overflow: hidden;">
                                    <div style="font-weight: 600; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-size: 1em;" title="${track.track || '未知条目'}">${track.track || '未知条目'}</div>
                                    <div style="font-size: 0.85em; opacity: 0.6; margin-top: 2px; color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;" title="${track.artist || '未知艺术家'}">${track.artist || '未知艺术家'}</div>
                                    <div style="font-size: 0.8em; margin-top: 4px;">
                                        ${isPlaceholder ? '<span style="color: var(--text-secondary); border: 1px solid rgba(128,128,128,0.3); padding: 0px 5px; border-radius: 3px; font-size: 0.9em;">待完善 (未播放)</span>' : `<span style="opacity: 0.5;">${track.play_count} 次播放</span>`}
                                    </div>
                                </div>

                                <div style="flex: 1; min-width: 0; font-size: 0.8em; padding-left: 15px; border-left: 1px solid ${itemBorder};">
                                    ${mbRecordingID ? `
                                    <div style="display: flex; gap: 5px; margin-bottom: 2px;">
                                        <span style="flex-shrink: 0; background: #6b46c1; color: #fff; font-size: 0.75em; padding: 1px 4px; border-radius: 3px; font-weight: 800;">MB</span>
                                        <span style="color: var(--text-primary); font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; ${hasDiff ? 'color: var(--accent-color);' : 'opacity: 0.6;'}" title="${mbTrackName}">${mbTrackName}</span>
                                    </div>
                                    <div style="font-family: monospace; opacity: 0.3; font-size: 0.85em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${mbRecordingID}</div>
                                    ` : `<div style="opacity: 0.2; font-style: italic;">尚未同步</div>`}
                                </div>

                                ${hasLink && window.currentAlbumSyncStatus !== 3 ? `
                                <div style="flex-shrink: 0; padding-left: 10px; display: flex; align-items: center;">
                                    <button onclick="event.stopPropagation(); unlinkTrackAlbum(${ta.id}, ${window.currentAlbumID}, '${esc(track.track)}')" 
                                        style="padding: 4px 8px; font-size: 0.7em; border: 1px solid var(--accent-color); background: transparent; color: var(--accent-color); border-radius: 4px; cursor: pointer; opacity: 0.6;"
                                        onmouseover="this.style.opacity=1" onmouseout="this.style.opacity=0.6">解除</button>
                                </div>
                                ` : ''}
                            </li>
                        `;
                    });
                    html += '</ul>';
                    trackList.innerHTML = html;
                } else {
                    trackList.innerHTML = '<div style="padding: 40px; text-align: center; opacity: 0.4; color: var(--text-secondary);">暂无曲目信息</div>';
                }
            })
            .catch(err => {
                if (window.currentAlbumID !== id) {
                    return;
                }
                console.error('获取专辑详情失败:', err);
                trackList.innerHTML = `<div style="padding: 20px; text-align: center; color: var(--accent-color);">加载失败: ${err.message}</div>`;
            });
    }

    /**
     * 解除 TrackAlbum 关联（人工修复用）
     */
    function unlinkTrackAlbum(trackID, albumID, trackName) {
        if (!confirm(`确定要解除曲目 "${trackName}" 的专辑关联吗？`)) {
            return;
        }
        fetch('/api/track-album/unlink', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({track_id: trackID, album_id: albumID})
        })
        .then(r => r.json())
        .then(res => {
            if (res.status === 'ok') {
                alert('已解除关联');
                showAlbumDetails(albumID); // 刷新专辑详情
            } else {
                alert('解除失败: ' + (res.error || '未知错误'));
            }
        })
        .catch(err => {
            alert('请求失败: ' + err.message);
        });
    }

    /**
     * 根据同步状态更新 UI 按钮
     */
    function updateMBUI(status) {
        window.currentAlbumSyncStatus = status; // 保存同步状态
        
        const mbSearchBtn = document.getElementById('mbSearchBtn');
        const mbDeepBtn = document.getElementById('mbDeepBtn');
        const albumLockBtn = document.getElementById('albumLockBtn');
        const albumUnlockBtn = document.getElementById('albumUnlockBtn');
        const candidateList = document.getElementById('mbCandidateList');
        const initialMsg = document.getElementById('mbInitialMsg');
        
        // 重置按钮状态
        if (mbSearchBtn) {
            mbSearchBtn.style.display = 'block';
            mbSearchBtn.disabled = false;
            mbSearchBtn.style.opacity = '1';
        }
        if (mbDeepBtn) {
            mbDeepBtn.disabled = false;
            mbDeepBtn.style.opacity = '1';
        }
        if (albumLockBtn) albumLockBtn.style.display = 'none';
        if (albumUnlockBtn) albumUnlockBtn.style.display = 'none';

        if (status === 0) {
            mbSearchBtn.textContent = '初选补全';
            mbDeepBtn.style.display = 'none';
            if (candidateList) candidateList.style.display = 'none';
            if (initialMsg) initialMsg.style.display = 'flex';
        } else {
            mbSearchBtn.textContent = '重新搜索';
            if (candidateList) candidateList.style.display = 'flex';
            if (initialMsg) initialMsg.style.display = 'none';
            
            if (status === 1) {
                mbDeepBtn.style.display = 'none';
            } else if (status === 2) {
                mbDeepBtn.style.display = 'block';
                mbDeepBtn.textContent = '精选维护';
                mbDeepBtn.style.background = 'var(--primary-color)';
                mbDeepBtn.style.color = '#fff';
            } else if (status === 3) {
                mbDeepBtn.style.display = 'block';
                mbDeepBtn.textContent = '再次精选维护';
                mbDeepBtn.style.background = 'rgba(255,255,255,0.05)';
                mbDeepBtn.style.color = 'var(--primary-color)';
                if (albumLockBtn) albumLockBtn.style.display = 'block';
            } else if (status === 4) {
                mbDeepBtn.style.display = 'block';
                mbDeepBtn.textContent = '已锁定';
                mbDeepBtn.disabled = true;
                mbDeepBtn.style.opacity = '0.5';
                mbDeepBtn.style.background = 'rgba(0,0,0,0.1)';
                mbDeepBtn.style.color = 'var(--text-secondary)';
                
                mbSearchBtn.disabled = true;
                mbSearchBtn.style.opacity = '0.5';
                
                if (albumUnlockBtn) albumUnlockBtn.style.display = 'block';
            }
            
            // 加载候选列表
            loadMBCandidates(window.currentAlbumID, status);
        }
    }

    /**
     * 加载并展示 MB 候选（解析详细 JSON 数据）
     */
    function loadMBCandidates(albumID, currentStatus) {
        const candidateList = document.getElementById('mbCandidateList');
        const candidateContent = document.getElementById('mbCandidateContent');
        
        // 检测当前主题
        const isDark = document.body.classList.contains("dark-mode");
        const cardBg = isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)';
        const cardBorder = isDark ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.08)';
        const textColor = isDark ? 'rgba(255,255,255,0.9)' : 'rgba(0,0,0,0.85)';
        const subtleBorder = isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.05)';
        
        // 如果已通过初选关联（状态>=2），则需要标出选中的版本
        let selectedMBID = '';
        
        const renderList = (candidates) => {
            if (!candidates || candidates.length === 0) {
                candidateList.style.display = 'none';
                return;
            }
            candidateList.style.display = 'block';
            let html = '<div style="display: flex; flex-direction: column; gap: 12px; padding-bottom: 20px;">';
            candidates.forEach((item, index) => {
                let detail = {};
                try {
                    detail = JSON.parse(item.json_data || '{}');
                } catch(e) {}
                
                const isSelected = (selectedMBID === item.mbid);
                const date = detail.date || '未知日期';
                const country = detail.country || '未知国家';
                // 计算所有碟的曲目总数
                let trackCount = 0;
                let trackCalcStr = "";
                if (detail.media && detail.media.length > 0) {
                    trackCount = detail.media.reduce((sum, m) => sum + (m["track-count"] || 0), 0);
                    if (detail.media.length > 1) {
                        trackCalcStr = detail.media.map(m => m["track-count"] || 0).join(' + ') + ' = ';
                    }
                }
                const trackDisplay = trackCalcStr + trackCount;
                const isCompleted = currentStatus === 3; // 精选完成后禁用确认按钮

                html += `
                    <div class="mb-candidate-card" style="padding: 15px; background: ${isSelected ? 'rgba(var(--primary-rgb), 0.15)' : cardBg}; border: 1px solid ${isSelected ? 'var(--primary-color)' : cardBorder}; border-radius: 14px; position: relative; transition: all 0.2s;">
                        <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                            <div style="flex: 1; padding-right: 15px;">
                                <div style="font-weight: 700; font-size: 1.05em; color: ${isSelected ? 'var(--primary-color)' : textColor}; margin-bottom: 6px;">
                                    ${item.name}
                                </div>
                                <div style="display: flex; flex-wrap: wrap; gap: 12px; font-size: 0.8em; opacity: 0.6;">
                                    <span>📅 ${date}</span>
                                    <span>🌍 ${country}</span>
                                    <span>🎵 ${trackDisplay}曲</span>
                                </div>
                            </div>
                            <div style="flex-shrink: 0;">
                                ${isSelected ? 
                                    '<span style="background: var(--primary-color); color: #fff; font-size: 0.7em; padding: 2px 8px; border-radius: 5px; font-weight: 800; box-shadow: 0 4px 10px rgba(var(--primary-rgb), 0.3);">已选定</span>' : 
                                    (isCompleted ? 
                                        '<span style="font-size: 0.7em; padding: 4px 10px; border-radius: 6px; opacity: 0.4; color: var(--text-secondary);">已完成</span>' :
                                        `<button class="time-filter" style="font-size: 0.7em; padding: 4px 10px; border-radius: 6px;" onclick="selectCandidate(${item.id}, '${item.mbid}')">确认此版本</button>`
                                    )
                                }
                            </div>
                        </div>
                        <div style="margin-top: 12px; padding-top: 10px; border-top: 1px solid ${subtleBorder}; display: flex; justify-content: space-between; align-items: center;">
                            <a href="javascript:void(0)" onclick="showCandidateDetail(${index})" style="font-size: 0.7em; opacity: 0.5; text-decoration: none; color: var(--text-secondary);">[VIEW DETAILS]</a>
                            <span style="font-size: 0.65em; font-family: monospace; opacity: 0.3; color: var(--text-secondary);">${item.mbid.substring(0, 18)}...</span>
                        </div>
                    </div>
                `;
            });
            html += '</div>';
            candidateContent.innerHTML = html;
        };

        // 并行处理：获取候选列表 + (可选)获取专辑当前选定的 MBID
        fetch(`/api/musicbrainz/candidates/${albumID}`)
            .then(resp => resp.json())
            .then(candidates => {
                window.currentCandidates = candidates;
                if (currentStatus >= 2) {
                    fetch(`/api/albums/${albumID}`)
                        .then(r => r.json())
                        .then(album => {
                            if (album.release_mb && album.release_mb.mbid) {
                                selectedMBID = album.release_mb.mbid;
                            }
                            renderList(candidates);
                        });
                } else {
                    renderList(candidates);
                }
            });
    }

    /**
     * 切换 MusicBrainz 详情模态框的 Tab
     */
    function switchMBDetailTab(tab) {
        const detailPane = document.getElementById('mbDetailContent');
        const jsonPane = document.getElementById('mbJsonContent');
        const detailTab = document.getElementById('tab-mb-detail');
        const jsonTab = document.getElementById('tab-mb-json');

        if (tab === 'json') {
            detailPane.style.display = 'none';
            jsonPane.style.display = 'block';
            detailTab.classList.remove('active');
            jsonTab.classList.add('active');
            detailTab.style.borderBottomColor = 'transparent';
            detailTab.style.opacity = '0.6';
            jsonTab.style.borderBottomColor = 'var(--primary-color)';
            jsonTab.style.opacity = '1';
        } else {
            detailPane.style.display = 'block';
            jsonPane.style.display = 'none';
            detailTab.classList.add('active');
            jsonTab.classList.remove('active');
            detailTab.style.borderBottomColor = 'var(--primary-color)';
            detailTab.style.opacity = '1';
            jsonTab.style.borderBottomColor = 'transparent';
            jsonTab.style.opacity = '0.6';
        }
    }

    /**
     * 显示候选版本的详细数据
     */
    function showCandidateDetail(index) {
        if (!window.currentCandidates || !window.currentCandidates[index]) return;
        const item = window.currentCandidates[index];
        let detail = {};
        try {
            detail = JSON.parse(item.json_data || '{}');
        } catch(e) { return; }

        const modal = document.getElementById('mbCandidateDetailModal');
        const content = document.getElementById('mbDetailContent');
        const jsonPre = document.getElementById('mbJsonPre');
        
        // 重置 Tab 到详情页
        switchMBDetailTab('detail');
        modal.style.display = 'block';
        
        // 渲染 JSON 源码
        jsonPre.textContent = JSON.stringify(detail, null, 4);

        // 检测主题
        const isDark = document.body.classList.contains("dark-mode");
        const subtleBorder = isDark ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.08)';

        // 从 release-events 获取国家信息
        let countryText = detail.country || '-';
        if (detail["release-events"] && detail["release-events"].length > 0) {
            const event = detail["release-events"][0];
            if (event.area && event.area.name) {
                countryText = event.area.name + (event.date ? ` (${event.date})` : '');
            }
        }

        // 渲染曲目信息
        let trackHtml = `<div style="margin-top: 15px;"><strong style="color: var(--text-primary);">曲目摘要:</strong><div style="max-height: 300px; overflow-y: auto; margin-top: 10px; font-size: 0.85em; color: var(--text-primary);">`;
        if (detail.media && detail.media.length > 0) {
            detail.media.forEach((m, mIndex) => {
                trackHtml += `<div style="margin-top: 10px; opacity: 0.7; font-weight: bold; padding-bottom: 5px; border-bottom: 1px solid ${subtleBorder};">碟 ${m.position || (mIndex+1)} (共 ${m["track-count"] || 0} 首，格式: ${m.format || '未知'})</div>`;
                if (m.tracks && m.tracks.length > 0) {
                    m.tracks.forEach(t => {
                        const trackTitle = t.title || (t.recording ? t.recording.title : '-');
                        const trackLength = t.length ? `${Math.floor(t.length/60000)}:${String(Math.floor((t.length%60000)/1000)).padStart(2, '0')}` : '-';
                        trackHtml += `<div style="padding: 4px 0; border-bottom: 1px solid ${subtleBorder}; display: flex;">
                            <span style="width: 30px; opacity: 0.5; color: var(--text-secondary);">${t.number || t.position}.</span>
                            <span style="flex: 1; color: var(--text-primary); margin-left:10px;">${trackTitle}</span>
                            <span style="opacity: 0.5; color: var(--text-secondary);">${trackLength}</span>
                        </div>`;
                    });
                } else {
                    trackHtml += '<div style="margin-top: 5px; margin-bottom: 10px; opacity: 0.5; font-style: italic; color: var(--text-secondary);">注：该搜索结果页暂不包含完整曲目单。</div>';
                }
            });
        } else {
            trackHtml += '<div style="color: var(--text-secondary);">暂无曲目信息</div>';
        }
        trackHtml += '</div></div>';

        // Label 信息
        let labelHtml = '';
        if (detail["label-info"] && detail["label-info"].length > 0) {
            labelHtml = `<div style="margin-top: 20px;"><strong style="color: var(--text-primary);">厂牌信息:</strong><div style="margin-top: 10px; display: flex; flex-wrap: wrap; gap: 8px;">`;
            detail["label-info"].forEach(li => {
                const labelName = li.label ? li.label.name : '-';
                const catalogNo = li["catalog-number"] || '-';
                labelHtml += `<div style="padding: 6px 12px; background: rgba(var(--primary-rgb), 0.1); border: 1px solid rgba(var(--primary-rgb), 0.3); border-radius: 6px; font-size: 0.85em;">
                    <span style="color: var(--primary-color); font-weight: 600;">${labelName}</span>
                    ${catalogNo !== '-' ? `<span style="opacity: 0.6; margin-left: 8px; color: var(--text-secondary);">#${catalogNo}</span>` : ''}
                </div>`;
            });
            labelHtml += '</div></div>';
        }

        // 文字表示信息
        let textRepHtml = '';
        if (detail["text-representation"]) {
            const tr = detail["text-representation"];
            const lang = tr.language || '';
            const script = tr.script || '';
            if (lang || script) {
                textRepHtml = `<div style="margin-top: 20px;"><strong style="color: var(--text-primary);">文字表示:</strong><div style="margin-top: 8px; display: flex; gap: 10px; font-size: 0.85em;">
                    ${lang ? `<span style="padding: 4px 10px; background: rgba(var(--primary-rgb), 0.1); border-radius: 4px; color: var(--text-primary);">语言: ${lang}</span>` : ''}
                    ${script ? `<span style="padding: 4px 10px; background: rgba(var(--primary-rgb), 0.1); border-radius: 4px; color: var(--text-primary);">文字: ${script}</span>` : ''}
                </div></div>`;
            }
        }

        // 封面艺术信息
        let coverArtHtml = '';
        if (detail["cover-art-archive"]) {
            const ca = detail["cover-art-archive"];
            if (ca.artwork || ca.front || ca.back) {
                coverArtHtml = `<div style="margin-top: 20px;"><strong style="color: var(--text-primary);">封面:</strong><div style="margin-top: 8px; display: flex; gap: 10px; font-size: 0.85em;">
                    ${ca.front ? '<span style="padding: 4px 10px; background: rgba(46, 204, 113, 0.2); border-radius: 4px; color: #2ecc71;">封面</span>' : ''}
                    ${ca.back ? '<span style="padding: 4px 10px; background: rgba(46, 204, 113, 0.2); border-radius: 4px; color: #2ecc71;">封底</span>' : ''}
                    ${ca.count ? `<span style="padding: 4px 10px; background: rgba(var(--primary-rgb), 0.1); border-radius: 4px; color: var(--text-secondary);">${ca.count} 张</span>` : ''}
                </div></div>`;
            }
        }

        content.innerHTML = `
            <div style="font-size: 1.1em; font-weight: 600; margin-bottom: 15px; color: var(--text-primary);">${detail.title || item.name} <span style="font-weight: normal; opacity: 0.5; font-size: 0.8em;">(${detail.date || '?'})</span></div>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; font-size: 0.9em; color: var(--text-primary);">
                <div><strong>国家/地区:</strong> ${countryText}</div>
                <div><strong>发行状态:</strong> ${detail.status || '-'}</div>
                <div><strong>包装形式:</strong> ${detail.packaging || '-'}</div>
                <div><strong>条码:</strong> ${detail.barcode || '-'}</div>
            </div>
            ${textRepHtml}
            ${coverArtHtml}
            ${trackHtml}
            ${labelHtml}
            <div style="margin-top: 20px; font-family: monospace; font-size: 0.7em; opacity: 0.3; word-break: break-all; color: var(--text-secondary);">MBID: ${item.mbid}</div>
        `;
    }
    window.showCandidateDetail = showCandidateDetail;

    /**
     * 选定 MB 版本进行关联
     */
    function selectCandidate(rmbID, mbid) {
        const albumID = window.currentAlbumID;
        fetch('/api/musicbrainz/link-album', {
            method: 'POST',
            body: JSON.stringify({ album_id: albumID, release_mb_id: rmbID, mbid: mbid })
        })
        .then(resp => resp.json())
        .then(res => {
            if (res.status === 'ok') {
                alert("关联成功，现在可以执行“精选维护”来精准核对曲目号。");
                showAlbumDetails(albumID); // 刷新 UI
            }
        });
    }

    /**
     * MusicBrainz 同步相关操作
     */
    function syncAlbumMB(mode) {
        const id = window.currentAlbumID;
        if (!id) return;

        const searchBtn = document.getElementById('mbSearchBtn');
        const deepBtn = document.getElementById('mbDeepBtn');

        if (mode === 'search') {
            searchBtn.disabled = true;
            searchBtn.textContent = '搜索中...';
            
            fetch(`/api/musicbrainz/search-releases/${id}`)
                .then(async resp => {
                    if (!resp.ok) {
                        const errData = await resp.json().catch(() => ({}));
                        throw new Error(errData.error || `服务器响应异常 (${resp.status})`);
                    }
                    return resp.json();
                })
                .then(res => {
                    searchBtn.disabled = false;
                    searchBtn.textContent = '重新搜索';
                    if (res.status === 'ok') {
                        showAlbumDetails(id); // 刷新以展示候选
                    }
                })
                .catch(err => {
                    console.error("搜索失败:", err);
                    alert("搜索失败: " + err.message);
                    searchBtn.disabled = false;
                    searchBtn.textContent = '重新搜索';
                });
        } else if (mode === 'deep') {
            deepBtn.disabled = true;
            deepBtn.textContent = '维护中...';
            fetch(`/api/musicbrainz/deep-maintenance/${id}`, { method: 'POST' })
                .then(resp => resp.json())
                .then(res => {
                    deepBtn.disabled = false;
                    deepBtn.textContent = '精选维护';
                    if (res.status === 'ok') {
                        alert("深度维护完成，曲目序号已根据 MusicBrainz 自动校正。");
                        showAlbumDetails(id); 
                    } else {
                        throw new Error(res.error || '维护失败');
                    }
                })
                .catch(err => {
                    alert("操作失败: " + err.message);
                    deepBtn.disabled = false;
                });
        }
    }

    /**
     * 锁定专辑
     */
    function lockAlbum() {
        const id = window.currentAlbumID;
        if (!id) return;

        if (!confirm('锁定后将无法修改专辑元数据和关联关系，确定要锁定吗？')) return;

        fetch(`/api/albums/${id}/lock`, { method: 'POST' })
            .then(resp => {
                if (!resp.ok) return resp.json().then(e => { throw new Error(e.error || '锁定失败'); });
                return resp.json();
            })
            .then(() => {
                if (typeof showToast === 'function') showToast('专辑已锁定');
                // 重新获取专辑信息刷新 UI
                showAlbumDetails(id);
            })
            .catch(err => {
                alert(err.message);
            });
    }

    /**
     * 解锁专辑
     */
    function unlockAlbum() {
        const id = window.currentAlbumID;
        if (!id) return;

        fetch(`/api/albums/${id}/unlock`, { method: 'POST' })
            .then(resp => {
                if (!resp.ok) return resp.json().then(e => { throw new Error(e.error || '解锁失败'); });
                return resp.json();
            })
            .then(() => {
                if (typeof showToast === 'function') showToast('专辑已解锁');
                // 重新获取专辑信息刷新 UI
                showAlbumDetails(id);
            })
            .catch(err => {
                alert(err.message);
            });
    }

    /**
     * 显示简易通知
     */
    function showToast(message) {
        const toast = document.createElement('div');
        toast.className = 'toast-notification';
        toast.style.position = 'fixed';
        toast.style.bottom = '30px';
        toast.style.right = '30px';
        toast.style.backgroundColor = 'var(--primary-color)';
        toast.style.color = '#fff';
        toast.style.padding = '12px 24px';
        toast.style.borderRadius = '12px';
        toast.style.boxShadow = '0 8px 16px rgba(0,0,0,0.15)';
        toast.style.zIndex = '9999';
        toast.style.display = 'flex';
        toast.style.alignItems = 'center';
        toast.style.gap = '10px';
        toast.style.fontSize = '0.9rem';
        toast.style.fontWeight = '600';
        toast.style.transition = 'all 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55)';
        toast.style.transform = 'translateY(100px)';
        toast.style.opacity = '0';
        
        toast.innerHTML = `<span>✨</span> ${message}`;
        
        document.body.appendChild(toast);
        
        // 触发动画
        setTimeout(() => {
            toast.style.transform = 'translateY(0)';
            toast.style.opacity = '1';
        }, 10);
        
        // 自动移除
        setTimeout(() => {
            toast.style.transform = 'translateY(20px)';
            toast.style.opacity = '0';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }
