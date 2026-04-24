// WebSocket连接
    let ws = null;
    let hasDashboardBootstrapped = false;
    let currentSectionID = "dashboard";
    let insightJobWSConnected = false;

    let currentInsightKeyword = "";
    let insightSearchTimeout = null;
    let currentInsightTargetType = "track";
    let currentInsightJobPage = 1;
    let insightJobPageSize = 20;
    let totalInsightJobs = 0;
    let currentInsightJobKeyword = "";
    let insightJobSearchTimeout = null;
    let currentInsightJobStatus = "";
    let currentInsightJobTargetType = "";
    let currentInsightJobTab = "monitor";
    let currentInsightJobDetail = null;
    let currentInsightJobRows = [];
    let currentInsightJobSummaryRows = [];
    let selectedInsightJobID = "";
    let highlightedInsightJobID = "";
    let insightJobRefreshTimer = null;
    let insightJobDurationTicker = null;
    let insightJobListCache = {};
    let insightJobDetailViewState = {};

    // 专辑列表分页及搜索变量
    let currentAlbumPage = 1;
    let albumPageSize = 20;
    let totalAlbumsCount = 0;
    let currentAlbumKeyword = "";
    let albumSearchTimeout = null;
    let pendingAlbumGroups = [];

    // 艺术家资料分页及搜索变量
    let currentArtistPage = 1;
    let artistPageSize = 20;
    let totalArtistsCount = 0;
    let currentArtistKeyword = "";
    let artistSearchTimeout = null;
    
    // 待归因工作台变量
    let currentPendingWorkTab = 'pending_groups'; // pending_groups, working_items, completed_items
    let currentPendingWorkPage = 1;
    let pendingWorkPageSize = 10;
    let totalPendingWorkCount = 0;
    let currentPendingWorkKeyword = "";
    let pendingWorkSearchTimeout = null;

    let currentPendingAlbumWorkItemID = 0;
    let currentPendingCandidates = [];
    let currentPendingSelectedMBID = "";
    let currentPendingAlbumStalePromptedWorkItemID = 0;
    let currentPendingAlbumWorkItemDetail = null;
    let currentPendingManualTrackDrafts = [];

    // 曲目列表分页及搜索变量
    let currentTrackPage = 1;
    let trackTrackPageSize = 20; // 改名避坑
    let trackPageSize = 20;
    let totalTracksCount = 0;
    let currentTrackKeyword = "";
    let trackSearchTimeout = null;

    // 音眸列表分页变量
    let currentInsightPage = 1;
    let insightPageSize = 20;
    let totalInsights = 0;

    // 自动切换日夜模式
    function toggleDarkModeBasedOnSystem() {
        const body = document.body;

        // 检查系统主题偏好
        if (
            window.matchMedia &&
            window.matchMedia("(prefers-color-scheme: dark)").matches
        ) {
            body.classList.add("dark-mode");
        } else {
            body.classList.remove("dark-mode");
        }

        // 首次初始化阶段不重复拉取图表，避免首屏双重渲染
        if (hasDashboardBootstrapped) {
            reloadAllCharts();
        }
    }

    // 监听系统主题变化
    function watchSystemThemeChange() {
        if (window.matchMedia) {
            const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
            if (mediaQuery.addEventListener) {
                mediaQuery.addEventListener("change", toggleDarkModeBasedOnSystem);
            } else if (mediaQuery.addListener) {
                mediaQuery.addListener(toggleDarkModeBasedOnSystem);
            }
        }
    }

    function enablePerformanceModeIfNeeded() {
        const nav = window.navigator || {};
        const ua = nav.userAgent || "";
        const hardwareConcurrency = nav.hardwareConcurrency || 0;
        const deviceMemory = nav.deviceMemory || 0;
        const isAndroid = /Android/i.test(ua);
        const isTvLike = /(TV|AFT|BRAVIA|MiBOX|MiTV|SmartTV|GoogleTV|HbbTV)/i.test(ua);
        const lowCpu = hardwareConcurrency > 0 && hardwareConcurrency <= 4;
        const lowMem = deviceMemory > 0 && deviceMemory <= 2;
        const prefersReducedMotion =
            window.matchMedia &&
            window.matchMedia("(prefers-reduced-motion: reduce)").matches;

        if ((isAndroid && isTvLike) || lowCpu || lowMem || prefersReducedMotion) {
            document.body.classList.add("performance-mode");
        }
    }

    // 检查是否处于暗色模式
    function isDarkMode() {
        return document.body.classList.contains("dark-mode");
    }

    // 获取图表的暗色模式配置
    function getChartDarkModeOptions() {
        if (isDarkMode()) {
            return {
                color: "#f0f0f0", // 文本颜色
                borderColor: "#444444", // 边框颜色
                backgroundColor: "#2c2c2c", // 背景颜色
                gridColor: "rgba(255, 255, 255, 0.1)", // 网格线颜色
                ticksColor: "#bdc3c7", // 刻度颜色
            };
        } else {
            return {
                color: "#2c3e50", // 文本颜色
                borderColor: "#e9ecef", // 边框颜色
                backgroundColor: "#ffffff", // 背景颜色
                gridColor: "rgba(0, 0, 0, 0.05)", // 网格线颜色
                ticksColor: "#7f8c8d", // 刻度颜色
            };
        }
    }

    // 重新加载所有图表以适应主题变化
    function reloadAllCharts() {
        if (!hasDashboardBootstrapped) {
            return;
        }

        // 重新初始化所有图表
        initSourceChart();
        initGenreChart();

        // 重新初始化当前激活的图表
        const trendRangeEl = document.querySelector(".time-filter.active[data-range]");
        const trendRange = trendRangeEl ? trendRangeEl.getAttribute("data-range") : 30;
        initTrendChart(trendRange);

        const artistTypeEl = document.querySelector(".time-filter.active[data-type]");
        const artistType = artistTypeEl ? artistTypeEl.getAttribute("data-type") : "plays";
        initArtistChart(artistType);

        const albumDaysEl = document.querySelector(".time-filter.active[data-days]");
        const albumDays = albumDaysEl ? albumDaysEl.getAttribute("data-days") : 30;
        initAlbumChart(albumDays);

        // 如果未上报页面可见，重新加载其内容以适应主题变化
        const unscrobbledContainer = document.getElementById(
            "unscrobbledContainer"
        );
        if (
            unscrobbledContainer &&
            unscrobbledContainer.style.display !== "none"
        ) {
            loadUnscrobbledRecords();
        }

        // 更新全屏模式下的图表背景色
        updateFullscreenChartBackground();
    }

    function upsertChart(instance, canvas, config) {
        if (!canvas) {
            return instance;
        }

        if (!instance) {
            const ctx = canvas.getContext("2d");
            if (!ctx) {
                return null;
            }
            return new Chart(ctx, config);
        }

        instance.config.type = config.type;
        instance.data = config.data;
        instance.options = config.options;
        instance.update("none");
        return instance;
    }

    function updateFloatingDragPosition(element, position) {
        element.style.left = position.x + "px";
        element.style.top = position.y + "px";
        element.style.right = "auto";
        element.style.bottom = "auto";
    }

    // 更新全屏模式下的图表背景色
    function updateFullscreenChartBackground() {
        // 更新趋势图全屏背景
        const trendFullscreenOverlay = document.getElementById(
            "trendChartFullscreenOverlay"
        );
        if (trendFullscreenOverlay) {
            const isDark = document.body.classList.contains("dark-mode");
            const backgroundColor = isDark ? "#2c2c2c" : "white";
            const textColor = isDark ? "#f0f0f0" : "#2c3e50";
            trendFullscreenOverlay.querySelector("div").style.backgroundColor =
                backgroundColor;
            trendFullscreenOverlay.querySelector("div").style.color = textColor;
        }

        // 更新流派图全屏背景
        const genreFullscreenOverlay = document.getElementById(
            "genreChartFullscreenOverlay"
        );
        if (genreFullscreenOverlay) {
            const isDark = document.body.classList.contains("dark-mode");
            const backgroundColor = isDark ? "#2c2c2c" : "white";
            const textColor = isDark ? "#f0f0f0" : "#2c3e50";
            genreFullscreenOverlay.querySelector("div").style.backgroundColor =
                backgroundColor;
            genreFullscreenOverlay.querySelector("div").style.color = textColor;
        }
    }

    // 连接到WebSocket服务器
