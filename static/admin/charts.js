function initStats() {
        fetch("/api/dashboard/stats")
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                document.getElementById("totalPlays").textContent =
                    data.totalPlays.toLocaleString();
                document.getElementById("totalTracks").textContent =
                    data.totalTracks.toLocaleString();
                document.getElementById("totalArtists").textContent =
                    data.totalArtists.toLocaleString();
                document.getElementById("totalAlbums").textContent =
                    data.totalAlbums.toLocaleString();
            })
            .catch((error) => {
                console.error("获取统计数据失败:", error);
                // 显示错误信息
                document.getElementById("totalPlays").textContent = "错误";
                document.getElementById("totalTracks").textContent = "错误";
                document.getElementById("totalArtists").textContent = "错误";
                document.getElementById("totalAlbums").textContent = "错误";
            });
    }

    function stableSerialize(value) {
        if (value === null || typeof value !== "object") {
            return JSON.stringify(value);
        }
        if (Array.isArray(value)) {
            return "[" + value.map(stableSerialize).join(",") + "]";
        }
        const keys = Object.keys(value).sort();
        return "{" + keys.map(function (key) {
            return JSON.stringify(key) + ":" + stableSerialize(value[key]);
        }).join(",") + "}";
    }

    function getThemeSignature() {
        return isDarkMode() ? "dark" : "light";
    }

    function isLowEndMode() {
        return document.body.classList.contains("performance-mode");
    }

    function getPollingInterval(baseMs) {
        if (!isLowEndMode()) return baseMs;
        return Math.max(baseMs * 2, 120000);
    }

    function applyLowEndChartOptions(options) {
        if (!isLowEndMode()) {
            return options;
        }
        options.animation = false;
        options.animations = false;
        options.events = ["click"];
        options.devicePixelRatio = 1;
        return options;
    }

    // 全局变量存储播放来源图表实例
    let sourceChartInstance = null;
    let sourceChartSignature = "";
    let trendChartSignature = "";
    let artistChartSignature = "";
    let albumChartSignature = "";
    let genreChartSignature = "";
    let artistChartInstance = null;
    let albumChartInstance = null;

    // 未上报记录相关变量
    let currentUnscrobbledPage = 1;
    const unscrobbledPageSize = 10;
    let totalUnscrobbledRecords = 0;

    // 初始化播放来源扇形图
    function initSourceChart() {
        const canvas = document.getElementById("sourceChart");
        if (!canvas) {
            console.error("找不到ID为sourceChart的canvas元素");
            return;
        }

        // 从后端获取数据
        fetch("/api/dashboard/play-counts-by-source")
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                // 获取暗色模式配置
                const darkModeOptions = getChartDarkModeOptions();

                // 处理数据
                const sources = Object.keys(data).sort();
                const counts = sources.map((source) => data[source]);
                const nextSignature = stableSerialize({
                    theme: getThemeSignature(),
                    sources: sources,
                    counts: counts
                });

                if (sourceChartInstance && sourceChartSignature === nextSignature) {
                    return;
                }
                sourceChartSignature = nextSignature;

                const sourceVisuals = sources.map((source) => getSourceBadgeInfo(source));
                const backgroundColors = sourceVisuals.map((item) => item.backgroundColor);
                const borderColors = sourceVisuals.map((item) => item.borderColor);
                const labels = sourceVisuals.map((item) => item.sourceText);

                sourceChartInstance = upsertChart(sourceChartInstance, canvas, {
                    type: "doughnut",
                    data: {
                        labels: labels,
                        datasets: [
                            {
                                data: counts,
                                backgroundColor: backgroundColors,
                                borderColor: borderColors,
                                borderWidth: 1,
                            },
                        ],
                    },
                    options: applyLowEndChartOptions({
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                position: "right",
                                labels: {
                                    color: darkModeOptions.color, // 暗色模式下的图例文字颜色
                                    boxWidth: 12,
                                    padding: 8,
                                    font: {
                                        size: 12,
                                    },
                                },
                            },
                            tooltip: {
                                callbacks: {
                                    label: function (context) {
                                        const label = context.label || "";
                                        const value = context.raw || 0;
                                        const total = context.dataset.data.reduce(
                                            (a, b) => a + b,
                                            0
                                        );
                                        const percentage = Math.round((value / total) * 100);
                                        return `${label}: ${value} (${percentage}%)`;
                                    },
                                },
                            },
                        },
                        cutout: "60%",
                    }),
                });
            })
            .catch((error) => {
                console.error("获取播放来源数据失败:", error);
            });
    }

    // 全屏元素和状态
    let isTrendChartFullscreen = false;
    let originalChartContainer = null;
    let originalChartParent = null;
    let originalChartStyle = null;

    // 切换趋势图全屏显示
    function toggleTrendChartFullscreen() {
        const chartContainer =
            document.querySelector("#trendChart").parentElement;
        const fullscreenBtn = document.getElementById(
            "trendChartFullscreenBtn"
        );

        if (!isTrendChartFullscreen) {
            // 进入全屏模式
            // 保存原始父元素和样式
            originalChartParent = chartContainer.parentElement;
            originalChartStyle = {
                width: chartContainer.style.width,
                height: chartContainer.style.height,
                position: chartContainer.style.position,
                zIndex: chartContainer.style.zIndex,
            };

            // 创建全屏遮罩
            const fullscreenOverlay = document.createElement("div");
            fullscreenOverlay.id = "trendChartFullscreenOverlay";
            fullscreenOverlay.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.9);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 10000;
          `;

            // 创建关闭按钮
            const closeBtn = document.createElement("button");
            closeBtn.innerHTML = "×";
            closeBtn.style.cssText = `
            position: absolute;
            top: 20px;
            right: 20px;
            background: none;
            border: none;
            color: white;
            font-size: 36px;
            cursor: pointer;
            z-index: 10001;
          `;

            closeBtn.addEventListener("click", function () {
                toggleTrendChartFullscreen();
            });

            // 创建包含图表的容器
            const chartWrapper = document.createElement("div");
            // 根据当前主题设置背景色
            const isDark = document.body.classList.contains("dark-mode");
            const backgroundColor = isDark ? "#2c2c2c" : "white";
            const textColor = isDark ? "#f0f0f0" : "#2c3e50";
            chartWrapper.style.cssText = `
            width: 90%;
            height: 90%;
            background-color: ${backgroundColor};
            border-radius: 8px;
            padding: 20px;
            color: ${textColor};
          `;

            // 将图表移动到新容器中
            chartContainer.style.width = "100%";
            chartContainer.style.height = "100%";
            chartWrapper.appendChild(chartContainer);
            fullscreenOverlay.appendChild(chartWrapper);
            fullscreenOverlay.appendChild(closeBtn);
            document.body.appendChild(fullscreenOverlay);

            // 更新按钮文本
            fullscreenBtn.textContent = "退出全屏";

            // 标记为全屏状态
            isTrendChartFullscreen = true;

            // 重新渲染图表以适应新尺寸
            if (window.trendChartInstance) {
                window.trendChartInstance.resize();
            }
        } else {
            // 退出全屏模式
            // 移除全屏遮罩
            const fullscreenOverlay = document.getElementById(
                "trendChartFullscreenOverlay"
            );
            if (fullscreenOverlay) {
                // 将图表移回原始位置
                const chartWrapper = fullscreenOverlay.querySelector("div");
                if (chartWrapper) {
                    originalChartParent.appendChild(chartContainer);
                }

                // 恢复原始样式
                chartContainer.style.width = originalChartStyle.width || "";
                chartContainer.style.height = originalChartStyle.height || "";
                chartContainer.style.position = originalChartStyle.position || "";
                chartContainer.style.zIndex = originalChartStyle.zIndex || "";

                // 移除遮罩
                document.body.removeChild(fullscreenOverlay);
            }

            // 更新按钮文本
            fullscreenBtn.textContent = "全屏";

            // 标记为非全屏状态
            isTrendChartFullscreen = false;

            // 重新渲染图表以适应原始尺寸
            if (window.trendChartInstance) {
                window.trendChartInstance.resize();
            }
        }
    }

    // 初始化趋势图
    function initTrendChart(range = 30) {
        const trendCanvas = document.getElementById("trendChart");
        if (!trendCanvas) {
            console.error("找不到ID为trendChart的canvas元素");
            return;
        }
        const rangeDays = parseInt(range, 10) || 30;

        // 从后端获取数据，添加时间范围参数
        fetch(`/api/dashboard/trend?range=${range}`)
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                const nextSignature = stableSerialize({
                    theme: getThemeSignature(),
                    range: rangeDays,
                    daily: data.daily || {},
                    hourly: data.hourly || {}
                });
                if (window.trendChartInstance && trendChartSignature === nextSignature) {
                    return;
                }
                trendChartSignature = nextSignature;

                // 获取暗色模式配置
                const darkModeOptions = getChartDarkModeOptions();

                // 处理新的API响应格式，将数据转换为气泡图所需格式
                const bubbleData = [];
                // 存储每天的总播放次数，用于tooltip显示。优先使用 daily，避免小时数据窗口较短时丢天数。
                const dailyTotals = data.daily || {};
                const hourlyMap = data.hourly || {};

                // 先用 hourly 明细生成点
                Object.keys(hourlyMap).sort().forEach((date) => {
                    const hourlyData = hourlyMap[date];
                    Object.keys(hourlyData.hourly || {}).sort(function (a, b) {
                        return Number(a) - Number(b);
                    }).forEach((hour) => {
                        const playCount = hourlyData.hourly[hour];
                        if (playCount > -1) {
                            const dateObj = new Date(date);
                            dateObj.setHours(12, 0, 0, 0);
                            bubbleData.push({
                                x: dateObj,
                                y: parseInt(hour, 10),
                                r: playCount,
                                date: date,
                                isDailyFallback: false,
                            });
                        }
                    });
                });

                // 对没有小时明细（key不存在或 hourly 为空）但有 daily 总量的日期，补一个“日聚合兜底点”
                Object.keys(dailyTotals).sort().forEach((date) => {
                    const hasHourlyBreakdown =
                        hourlyMap[date] &&
                        hourlyMap[date].hourly &&
                        Object.keys(hourlyMap[date].hourly).length > 0;
                    if (hasHourlyBreakdown) {
                        return;
                    }
                    const count = Number(dailyTotals[date] || 0);
                    if (count <= 0) {
                        return;
                    }
                    const dateObj = new Date(date);
                    dateObj.setHours(12, 0, 0, 0);
                    bubbleData.push({
                        x: dateObj,
                        y: 12, // 无小时明细时放在中午位置，表示日聚合
                        r: count,
                        date: date,
                        isDailyFallback: true,
                    });
                });

                // 低端设备降帧：限制趋势气泡点位数量，降低绘制负担
                if (isLowEndMode() && bubbleData.length > 140) {
                    const step = Math.ceil(bubbleData.length / 140);
                    const sampled = [];
                    for (let i = 0; i < bubbleData.length; i += step) {
                        sampled.push(bubbleData[i]);
                    }
                    bubbleData.length = 0;
                    Array.prototype.push.apply(bubbleData, sampled);
                }

                // 计算所有数据点中的最大播放次数，用于颜色映射
                let maxPlayCount = 0;
                bubbleData.forEach((point) => {
                    if (point.r > maxPlayCount) {
                        maxPlayCount = point.r;
                    }
                });

                // 颜色插值函数：播放次数多为暖色(红色)，播放次数少为冷色(蓝色)
                function getColorForCount(count, maxCount) {
                    // 归一化值 (0-1)
                    const normalized = maxCount > 0 ? count / maxCount : 0;

                    // 暖色到冷色的渐变
                    // 红色 (255, 0, 0) 到 蓝色 (0, 0, 255)
                    // 反转逻辑：播放次数多显示红色，播放次数少显示蓝色
                    const r = Math.round(255 * normalized);
                    const g = 20;
                    const b = Math.round(255 * (1 - normalized));

                    return {
                        background: `rgba(${r}, ${g}, ${b}, 0.4)`,
                        border: `rgba(${r}, ${g}, ${b}, 1)`,
                    };
                }

                // 固定 x 轴窗口，确保选择 90 天时不会只显示 30 天范围
                const now = new Date();
                const xAxisMax = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59, 999);
                const xAxisMin = new Date(xAxisMax);
                xAxisMin.setDate(xAxisMin.getDate() - (rangeDays - 1));
                xAxisMin.setHours(0, 0, 0, 0);

                window.trendChartInstance = upsertChart(window.trendChartInstance, trendCanvas, {
                    type: "bubble",
                    data: {
                        datasets: [
                            {
                                label: "播放记录",
                                data: bubbleData,
                                backgroundColor: function (context) {
                                    const count = context.raw.r;
                                    return getColorForCount(count, maxPlayCount).background;
                                },
                                borderColor: function (context) {
                                    const count = context.raw.r;
                                    return getColorForCount(count, maxPlayCount).border;
                                },
                                borderWidth: 1,
                                pointStyle: "circle",
                            },
                        ],
                    },
                    options: applyLowEndChartOptions({
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                display: false,
                                labels: {
                                    color: darkModeOptions.color, // 暗色模式下的图例文字颜色
                                },
                            },
                            tooltip: {
                                callbacks: {
                                    title: function (context) {
                                        // 显示日期和时间
                                        const point = context[0].raw;
                                        const date = new Date(point.x);
                                        const dateString = `${date.getFullYear()}-${(
                                            date.getMonth() + 1
                                        )
                                            .toString()
                                            .padStart(2, "0")}-${date
                                            .getDate()
                                            .toString()
                                            .padStart(2, "0")}`;
                                        const dailyTotal = dailyTotals[point.date] || 0;
                                        return `${dateString} ${point.y
                                            .toString()
                                            .padStart(2, "0")}:00 ( 当日总计: ${dailyTotal} )`;
                                    },
                                    label: function (context) {
                                        return `播放次数: ${context.raw.r}`;
                                    },
                                },
                            },
                        },
                        scales: {
                            x: {
                                type: "time",
                                time: {
                                    unit: "day",
                                    tooltipFormat: "yyyy-MM-dd",
                                    displayFormats: {
                                        day: "MM-dd",
                                    },
                                    // 确保时间轴从数据的最小值开始
                                    minUnit: "day",
                                },
                                min: xAxisMin,
                                max: xAxisMax,
                                title: {
                                    display: true,
                                    text: "日期",
                                    color: darkModeOptions.color, // 暗色模式下的标题颜色
                                },
                                grid: {
                                    color: darkModeOptions.gridColor, // 暗色模式下的网格线颜色
                                },
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                    //stepSize: 1,
                                    maxRotation: 60,
                                    minRotation: 45,
                                    autoSkip: true,
                                    // 确保标签与数据点对齐
                                    source: "data",
                                    // 确保标签与数据点对齐
                                    align: "center",
                                },
                                // 添加边界设置以确保数据点完全显示
                                bounds: "data",
                            },
                            y: {
                                type: "linear",
                                min: -1,
                                max: 24.99,
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                    stepSize: isLowEndMode() ? 2 : 1,
                                    maxRotation: 60,
                                    minRotation: 45,
                                    callback: function (value) {
                                        // 显示小时格式 (HH:00)
                                        return `${value.toString().padStart(2, "0")}:00`;
                                    },
                                },
                                title: {
                                    display: true,
                                    text: "时间",
                                    color: darkModeOptions.color, // 暗色模式下的标题颜色
                                },
                                grid: {
                                    color: darkModeOptions.gridColor, // 暗色模式下的网格线颜色
                                },
                            },
                        },
                        elements: {
                            point: {
                                radius: function (context) {
                                    // 根据数据点的 r 值调整半径，添加缩放因子来调整大小
                                    const radius = context.raw.r;
                                    return isLowEndMode()
                                        ? Math.max(radius * 2, 2)
                                        : Math.max(radius * 3, 2); // 最小半径为2
                                },
                            },
                        },
                    }),
                });
            })
            .catch((error) => {
                console.error("获取趋势图数据失败:", error);
            });
    }

    // 初始化艺术家图表
    function initArtistChart(type = "plays") {
        const artistCanvas = document.getElementById("artistChart");
        if (!artistCanvas) {
            console.error("找不到ID为artistChart的canvas元素");
            return;
        }

        // 根据类型选择不同的API端点
        const apiUrl =
            type === "plays"
                ? "/api/dashboard/top-artists/plays?limit=10"
                : "/api/dashboard/top-artists/tracks?limit=10";

        // 从后端获取数据
        fetch(apiUrl)
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                const nextSignature = stableSerialize({
                    theme: getThemeSignature(),
                    type: type,
                    data: data
                });
                if (artistChartInstance && artistChartSignature === nextSignature) {
                    return;
                }
                artistChartSignature = nextSignature;

                // 处理数据
                const artists = data.map((item) => item.artist);
                const counts = data.map((item) =>
                    type === "plays" ? item.play_count : item.track_count
                );

                const label = type === "plays" ? "播放次数" : "曲目数";

                // 获取暗色模式配置
                const darkModeOptions = getChartDarkModeOptions();

                artistChartInstance = upsertChart(artistChartInstance, artistCanvas, {
                    type: "bar",
                    data: {
                        labels: artists,
                        datasets: [
                            {
                                label: label,
                                data: counts,
                                /* backgroundColor: [
                            "rgba(52, 152, 219, 0.7)",
                            "rgba(46, 204, 113, 0.7)",
                            "rgba(231, 76, 60, 0.7)",
                            "rgba(155, 89, 182, 0.7)",
                            "rgba(241, 196, 15, 0.7)",
                            "rgba(52, 152, 219, 0.7)",
                            "rgba(46, 204, 113, 0.7)",
                            "rgba(231, 76, 60, 0.7)",
                            "rgba(155, 89, 182, 0.7)",
                            "rgba(241, 196, 15, 0.7)",
                          ],
                          borderColor: [
                            "rgba(52, 152, 219, 1)",
                            "rgba(46, 204, 113, 1)",
                            "rgba(231, 76, 60, 1)",
                            "rgba(155, 89, 182, 1)",
                            "rgba(241, 196, 15, 1)",
                            "rgba(52, 152, 219, 1)",
                            "rgba(46, 204, 113, 1)",
                            "rgba(231, 76, 60, 1)",
                            "rgba(155, 89, 182, 1)",
                            "rgba(241, 196, 15, 1)",
                          ], */
                                /* backgroundColor: [
                            "rgba(231, 76, 60, 0.9)", // 1 红
                            "rgba(230, 126, 34, 0.9)", // 2 橙
                            "rgba(241, 196, 15, 0.9)", // 3 黄
                            "rgba(46, 204, 113, 0.85)", // 4 绿
                            "rgba(0, 90, 140, 0.8)", // 5 靛蓝
                            "rgba(52, 152, 219, 0.8)", // 6 蓝
                            "rgba(110, 40, 140, 0.65)", // 7 深紫
                            "rgba(165, 95, 190, 0.7)", // 8 紫
                            "rgba(127, 140, 141, 0.6)", // 9 灰蓝
                            "rgba(189, 195, 199, 0.55)", // 10 浅灰收尾
                          ],
                          borderColor: [
                            "rgba(231, 76, 60, 1)",
                            "rgba(230, 126, 34, 1)",
                            "rgba(241, 196, 15, 1)",
                            "rgba(46, 204, 113, 1)",
                            "rgba(52, 152, 219, 1)",
                            "rgba(52, 73, 94, 1)",
                            "rgba(142, 68, 173, 1)",
                            "rgba(155, 89, 182, 1)",
                            "rgba(127, 140, 141, 1)",
                            "rgba(189, 195, 199, 1)",
                          ], */
                                backgroundColor: [
                                    "rgba(231, 76, 60, 0.9)", // 1 红
                                    "rgba(230, 126, 34, 0.9)", // 2 橙
                                    "rgba(241, 196, 15, 0.9)", // 3 黄
                                    "rgba(46, 204, 113, 0.85)", // 4 绿
                                    "rgba(0, 200, 200, 0.8)", // 5 青
                                    "rgba(52, 152, 219, 0.8)", // 6 蓝
                                    "rgba(0, 90, 140, 0.75)", // 7 深蓝/靛蓝
                                    "rgba(142, 68, 173, 0.7)", // 8 紫
                                    "rgba(127, 140, 141, 0.65)", // 9 灰蓝
                                    "rgba(189, 195, 199, 0.6)", // 10 浅灰收尾
                                ],
                                borderColor: [
                                    "rgba(231, 76, 60, 1)",
                                    "rgba(230, 126, 34, 1)",
                                    "rgba(241, 196, 15, 1)",
                                    "rgba(46, 204, 113, 1)",
                                    "rgba(0, 200, 200, 1)",
                                    "rgba(52, 152, 219, 1)",
                                    "rgba(0, 90, 140, 1)",
                                    "rgba(142, 68, 173, 1)",
                                    "rgba(127, 140, 141, 1)",
                                    "rgba(189, 195, 199, 1)",
                                ],
                                borderWidth: 1,
                            },
                        ],
                    },
                    options: applyLowEndChartOptions({
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                display: false,
                                labels: {
                                    color: darkModeOptions.color, // 暗色模式下的图例文字颜色
                                },
                            },
                        },
                        scales: {
                            y: {
                                beginAtZero: true,
                                grid: {
                                    color: darkModeOptions.gridColor, // 暗色模式下的网格线颜色
                                },
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                },
                            },
                            x: {
                                grid: {
                                    display: false,
                                },
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                    autoSkip: false,
                                    maxRotation: 45, // 恢复斜体显示
                                    callback: function (value) {
                                        // 截断过长的标签文本
                                        const label = this.getLabelForValue(value);
                                        if (label && label.length > 15) {
                                            return label.substring(0, 15) + "...";
                                        }
                                        return label;
                                    },
                                },
                            },
                        },
                    }),
                });
            })
            .catch((error) => {
                console.error("获取艺术家数据失败:", error);
            });
    }

    // 初始化热门专辑图表
    function initAlbumChart(days = 7) {
        const albumCanvas = document.getElementById("albumChart");
        if (!albumCanvas) {
            console.error("找不到ID为albumChart的canvas元素");
            return;
        }

        // 从后端获取数据
        fetch(`/api/dashboard/top-albums?days=${days}&limit=10`)
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                const nextSignature = stableSerialize({
                    theme: getThemeSignature(),
                    days: Number(days) || 0,
                    data: data
                });
                if (albumChartInstance && albumChartSignature === nextSignature) {
                    return;
                }
                albumChartSignature = nextSignature;

                // 处理数据
                const albums = data.map((item) => item.album);
                const artists = data.map((item) => item.artist);
                const counts = data.map((item) => item.play_count);

                // 获取暗色模式配置
                const darkModeOptions = getChartDarkModeOptions();

                albumChartInstance = upsertChart(albumChartInstance, albumCanvas, {
                    type: "bar",
                    data: {
                        labels: albums,
                        datasets: [
                            {
                                label: "播放次数",
                                data: counts,
                                backgroundColor: [
                                    "rgba(231, 76, 60, 0.9)", // 1 红
                                    "rgba(230, 126, 34, 0.9)", // 2 橙
                                    "rgba(241, 196, 15, 0.9)", // 3 黄
                                    "rgba(46, 204, 113, 0.85)", // 4 绿
                                    "rgba(0, 200, 200, 0.8)", // 5 青
                                    "rgba(52, 152, 219, 0.8)", // 6 蓝
                                    "rgba(0, 90, 140, 0.75)", // 7 深蓝/靛蓝
                                    "rgba(142, 68, 173, 0.7)", // 8 紫
                                    "rgba(127, 140, 141, 0.65)", // 9 灰蓝
                                    "rgba(189, 195, 199, 0.6)", // 10 浅灰收尾
                                ],
                                borderColor: [
                                    "rgba(231, 76, 60, 1)",
                                    "rgba(230, 126, 34, 1)",
                                    "rgba(241, 196, 15, 1)",
                                    "rgba(46, 204, 113, 1)",
                                    "rgba(0, 200, 200, 1)",
                                    "rgba(52, 152, 219, 1)",
                                    "rgba(0, 90, 140, 1)",
                                    "rgba(142, 68, 173, 1)",
                                    "rgba(127, 140, 141, 1)",
                                    "rgba(189, 195, 199, 1)",
                                ],
                                borderWidth: 1,
                            },
                        ],
                    },
                    options: applyLowEndChartOptions({
                        onClick: (event, elements) => {
                            if (elements && elements.length > 0) {
                                const index = elements[0].index;
                                const albumData = data[index];
                                if (albumData && albumData.album_id) {
                                    // 假设跳转到专辑详情页或弹出详情框
                                    // 这里我们采用弹出详情框的逻辑，如果项目有专门的专辑页也可以修改为 location.href
                                    // 考虑到这是一个 Dashboard 仪表盘，我们跳转到 /albums/:id 或者执行现有的 showAlbumDetails(id)
                                    if (typeof showAlbumDetails === 'function') {
                                        showAlbumDetails(albumData.album_id);
                                    } else {
                                        // 如果没有现成的函数，则跳转或提示
                                        console.log("跳转到专辑 ID:", albumData.album_id);
                                        // window.location.href = `/album/${albumData.album_id}`;
                                    }
                                }
                            }
                        },
                        indexAxis: "y", // 👈 水平条形图
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                display: false,
                                labels: {
                                    color: darkModeOptions.color, // 暗色模式下的图例文字颜色
                                },
                            },
                            tooltip: {
                                callbacks: {
                                    // 移除默认 label（第一行）
                                    /*   label: function () {
                              return null;
                            }, */
                                    // 用 title 回调显示专辑名（第一行）
                                    title: function (ctx) {
                                        const index = ctx[0].dataIndex; // tooltipItems 数组的第 0 个元素
                                        const artist = artists[index]; // 拿到对应作者
                                        return "《" + ctx[0].label + `》- ${artist}`; // 专辑名
                                    },
                                    // 用 afterLabel 回调显示作者 + 播放次数（第二行）
                                    /* afterLabel: function (ctx) {
                              const artist = artists[ctx.dataIndex];
                              //const count = counts[ctx.dataIndex];
                              return `${artist}`;
                            }, */
                                },
                            },
                        },
                        scales: {
                            x: {
                                beginAtZero: true,
                                grid: {
                                    color: darkModeOptions.gridColor, // 暗色模式下的网格线颜色
                                },
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                },
                            },
                            y: {
                                grid: {display: false},
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                    autoSkip: false,
                                    //maxRotation: 60, // 恢复斜体显示
                                    //minRotation: 45, // 恢复斜体显示
                                    callback: function (value) {
                                        const label = this.getLabelForValue(value);
                                        if (label && label.length > 15) {
                                            return label.substring(0, 15) + "...";
                                        }
                                        return label;
                                    },
                                },
                            },
                        },
                    }),
                });
            })
            .catch((error) => {
                console.error("获取专辑数据失败:", error);
            });
    }

    // 全屏元素和状态 - 口味流派图表
    let isGenreChartFullscreen = false;
    let originalGenreChartContainer = null;
    let originalGenreChartParent = null;
    let originalGenreChartStyle = null;

    // 切换口味流派图表全屏显示
    function toggleGenreChartFullscreen() {
        const chartContainer =
            document.querySelector("#genreChart").parentElement;
        const fullscreenBtn = document.getElementById(
            "genreChartFullscreenBtn"
        );

        if (!isGenreChartFullscreen) {
            // 进入全屏模式
            // 保存原始父元素和样式
            originalGenreChartParent = chartContainer.parentElement;
            originalGenreChartStyle = {
                width: chartContainer.style.width,
                height: chartContainer.style.height,
                position: chartContainer.style.position,
                zIndex: chartContainer.style.zIndex,
            };

            // 创建全屏遮罩
            const fullscreenOverlay = document.createElement("div");
            fullscreenOverlay.id = "genreChartFullscreenOverlay";
            fullscreenOverlay.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.9);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 10000;
          `;

            // 创建关闭按钮
            const closeBtn = document.createElement("button");
            closeBtn.innerHTML = "×";
            closeBtn.style.cssText = `
            position: absolute;
            top: 20px;
            right: 20px;
            background: none;
            border: none;
            color: white;
            font-size: 36px;
            cursor: pointer;
            z-index: 10001;
          `;

            closeBtn.addEventListener("click", function () {
                toggleGenreChartFullscreen();
            });

            // 创建包含图表的容器
            const chartWrapper = document.createElement("div");
            // 根据当前主题设置背景色
            const isDark = document.body.classList.contains("dark-mode");
            const backgroundColor = isDark ? "#2c2c2c" : "white";
            const textColor = isDark ? "#f0f0f0" : "#2c3e50";
            chartWrapper.style.cssText = `
            width: 90%;
            height: 90%;
            background-color: ${backgroundColor};
            border-radius: 8px;
            padding: 20px;
            color: ${textColor};
          `;

            // 将图表移动到新容器中
            chartContainer.style.width = "100%";
            chartContainer.style.height = "100%";
            chartWrapper.appendChild(chartContainer);
            fullscreenOverlay.appendChild(chartWrapper);
            fullscreenOverlay.appendChild(closeBtn);
            document.body.appendChild(fullscreenOverlay);

            // 更新按钮文本
            fullscreenBtn.textContent = "退出全屏";

            // 标记为全屏状态
            isGenreChartFullscreen = true;

            // 重新渲染图表以适应新尺寸
            if (window.genreChartInstance) {
                window.genreChartInstance.resize();
            }
        } else {
            // 退出全屏模式
            // 移除全屏遮罩
            const fullscreenOverlay = document.getElementById(
                "genreChartFullscreenOverlay"
            );
            if (fullscreenOverlay) {
                // 将图表移回原始位置
                const chartWrapper = fullscreenOverlay.querySelector("div");
                if (chartWrapper) {
                    originalGenreChartParent.appendChild(chartContainer);
                }

                // 恢复原始样式
                chartContainer.style.width = originalGenreChartStyle.width || "";
                chartContainer.style.height = originalGenreChartStyle.height || "";
                chartContainer.style.position =
                    originalGenreChartStyle.position || "";
                chartContainer.style.zIndex = originalGenreChartStyle.zIndex || "";

                // 移除遮罩
                document.body.removeChild(fullscreenOverlay);
            }

            // 更新按钮文本
            fullscreenBtn.textContent = "全屏";

            // 标记为非全屏状态
            isGenreChartFullscreen = false;

            // 重新渲染图表以适应原始尺寸
            if (window.genreChartInstance) {
                window.genreChartInstance.resize();
            }
        }
    }

    // 初始化热门流派图表
    function initGenreChart() {
        const genreCanvas = document.getElementById("genreChart");
        if (!genreCanvas) {
            console.error("找不到ID为genreChart的canvas元素");
            return;
        }

        // 从后端获取数据
        fetch("/api/dashboard/top-genres?limit=10")
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                const nextSignature = stableSerialize({
                    theme: getThemeSignature(),
                    data: data
                });
                if (window.genreChartInstance && genreChartSignature === nextSignature) {
                    return;
                }
                genreChartSignature = nextSignature;

                // 处理数据
                const genres = data.map((item) => item.track_genre_name);
                const genresNameZhs = data.map((item) => item.genre_name_zh);
                const counts = data.map((item) => item.track_genre_count);

                // 定义颜色
                const backgroundColors = [
                    "rgba(231, 76, 60, 0.8)", // 红
                    "rgba(230, 126, 34, 0.8)", // 橙
                    "rgba(241, 196, 15, 0.8)", // 黄
                    "rgba(46, 204, 113, 0.8)", // 绿
                    "rgba(0, 200, 200, 0.8)", // 青
                    "rgba(52, 152, 219, 0.8)", // 蓝
                    "rgba(0, 90, 140, 0.8)", // 深蓝
                    "rgba(142, 68, 173, 0.8)", // 紫
                    "rgba(127, 140, 141, 0.8)", // 灰
                    "rgba(189, 195, 199, 0.8)", // 浅灰
                ];

                const borderColors = [
                    "rgba(231, 76, 60, 1)",
                    "rgba(230, 126, 34, 1)",
                    "rgba(241, 196, 15, 1)",
                    "rgba(46, 204, 113, 1)",
                    "rgba(0, 200, 200, 1)",
                    "rgba(52, 152, 219, 1)",
                    "rgba(0, 90, 140, 1)",
                    "rgba(142, 68, 173, 1)",
                    "rgba(127, 140, 141, 1)",
                    "rgba(189, 195, 199, 1)",
                ];

                // 获取暗色模式配置
                const darkModeOptions = getChartDarkModeOptions();

                window.genreChartInstance = upsertChart(window.genreChartInstance, genreCanvas, {
                    type: "polarArea",
                    data: {
                        labels: genres,
                        datasets: [
                            {
                                label: "播放次数",
                                data: counts,
                                backgroundColor: backgroundColors.slice(0, counts.length),
                                borderColor: borderColors.slice(0, counts.length),
                                borderWidth: 1,
                            },
                        ],
                    },
                    options: applyLowEndChartOptions({
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                position: "left",
                                labels: {
                                    color: darkModeOptions.color, // 暗色模式下的图例文字颜色
                                    font: {
                                        size: 10, // 字体变小
                                    },
                                    boxWidth: 9, // 小方块变小
                                    padding: 6, // legend 项之间的间距缩小
                                },
                            },
                            tooltip: {
                                callbacks: {
                                    title: function (ctx) {
                                        const index = ctx[0].dataIndex;
                                        const genre_name_zh = genresNameZhs[index];
                                        return ctx[0].label + `（ ${genre_name_zh} ）`;
                                    },
                                },
                            },
                        },
                        scales: {
                            r: {
                                pointLabels: {
                                    color: darkModeOptions.color, // 暗色模式下的点标签颜色
                                    display: true,
                                    centerPointLabels: true,
                                    font: {
                                        size: 8,
                                    },
                                },
                                ticks: {
                                    color: darkModeOptions.ticksColor, // 暗色模式下的刻度颜色
                                    backdropColor: isDarkMode()
                                        ? "rgba(44, 44, 44, 0.8)"
                                        : "rgba(0, 0, 0, 0.05)",
                                },
                                // max: Math.max(...shuffledCounts) * 1.1, // 扩大 20% 半径
                            },
                        },
                    }),
                });
            })
            .catch((error) => {
                console.error("获取流派数据失败:", error);
            });
    }

    let lastRecentPlaysSignature = "";
    let lastRankingSignature = "";

    function normalizeSourceName(source) {
        const lowered = (source || "").trim().toLowerCase();
        if (lowered === "apple music" || lowered === "applemusic") return "Apple Music";
        if (lowered === "audirvana" || lowered === "au") return "Audirvana";
        if (lowered === "roon") return "Roon";
        if (lowered === "foobar2000" || lowered === "foobar") return "Foobar2000";
        if (
            lowered === "netease music" ||
            lowered === "netease" ||
            lowered === "163" ||
            lowered === "163music" ||
            lowered === "com.netease.163music"
        ) {
            return "NetEase Music";
        }
        return source || "Unknown";
    }

    function getSourceBadgeInfo(source) {
        switch (normalizeSourceName(source)) {
            case "Apple Music":
                return {
                    sourceClass: "source-applemusic",
                    sourceText: "Apple Music",
                    backgroundColor: "rgba(231, 76, 60, 0.7)",
                    borderColor: "rgba(231, 76, 60, 1)"
                };
            case "Audirvana":
                return {
                    sourceClass: "source-audirvana",
                    sourceText: "Audirvana",
                    backgroundColor: "rgba(155, 89, 182, 0.7)",
                    borderColor: "rgba(155, 89, 182, 1)"
                };
            case "Roon":
                return {
                    sourceClass: "source-roon",
                    sourceText: "Roon",
                    backgroundColor: "rgba(127, 140, 141, 0.7)",
                    borderColor: "rgba(127, 140, 141, 1)"
                };
            case "Foobar2000":
                return {
                    sourceClass: "source-foobar2000",
                    sourceText: "Foobar2000",
                    backgroundColor: "rgba(245, 158, 11, 0.72)",
                    borderColor: "rgba(245, 158, 11, 1)"
                };
            case "NetEase Music":
                return {
                    sourceClass: "source-netease",
                    sourceText: "NetEase Music",
                    backgroundColor: "rgba(16, 185, 129, 0.72)",
                    borderColor: "rgba(16, 185, 129, 1)"
                };
            default:
                return {
                    sourceClass: "",
                    sourceText: normalizeSourceName(source),
                    backgroundColor: "rgba(52, 152, 219, 0.7)",
                    borderColor: "rgba(52, 152, 219, 1)"
                };
        }
    }

    function buildRecentPlaysSignature(data) {
        return JSON.stringify((data || []).map((item) => [
            item.play_time,
            item.track,
            item.artist,
            item.album,
            item.source
        ]));
    }

    function buildRankingSignature(rankingType, keyword, data) {
        return JSON.stringify({
            rankingType: rankingType,
            keyword: keyword || "",
            items: (data || []).map((item) => [item.track, item.artist, item.album, item.play_count])
        });
    }

    function renderRecentPlaysList(data, force) {
        const recentPlaysList = document.getElementById("recentPlaysList");
        const signature = buildRecentPlaysSignature(data);
        if (!force && signature === lastRecentPlaysSignature) {
            return;
        }
        lastRecentPlaysSignature = signature;
        recentPlaysList.innerHTML = "";

        if (!data || data.length === 0) {
            renderAdminEmpty(recentPlaysList, "暂无最近播放数据");
            return;
        }

        const fragment = document.createDocumentFragment();
        data.forEach((item, index) => {
            const rankingItem = document.createElement("li");
            rankingItem.className = "ranking-item";
            rankingItem.style.cursor = "pointer";
            rankingItem.dataset.artist = item.artist || "";
            rankingItem.dataset.album = item.album || "";
            rankingItem.dataset.track = item.track || "";
            rankingItem.dataset.trackNumber = String(item.track_number || 0);
            rankingItem.dataset.discNumber = String(item.disc_number || 0);
            rankingItem.dataset.playTime = item.play_time || "";
            rankingItem.dataset.source = item.source || "";
            rankingItem.dataset.scrobbled = item.scrobbled ? "1" : "0";
            rankingItem.dataset.traceId = item.trace_id || "";
            rankingItem.dataset.rootSpanId = item.root_span_id || "";
            rankingItem.dataset.traceSampled = item.trace_sampled ? "1" : "0";
            rankingItem.dataset.resolutionStatus = item.resolution_status || "";

            const playTime = new Date(item.play_time);
            const timeString = playTime.toLocaleTimeString("zh-CN", {
                hour: "2-digit",
                minute: "2-digit",
                second: "2-digit",
            });

            const sourceBadge = getSourceBadgeInfo(item.source);
            rankingItem.innerHTML = `
                <div class="rank-number rank-other">${index + 1}</div>
                <div class="track-info">
                    <div class="track-title">${item.track}</div>
                    <div class="track-artist">${item.artist} - ${item.album}</div>
                </div>
                <div style="display: flex; flex-direction: column; align-items: flex-end; margin-left: 10px;">
                  <div class="play-source ${sourceBadge.sourceClass}">${sourceBadge.sourceText}</div>
                  <div class="play-count">${timeString}</div>
                </div>
            `;
            fragment.appendChild(rankingItem);
        });
        recentPlaysList.appendChild(fragment);
        adjustRankingListsHeight();
    }

    function renderRankingList(rankingType, keyword, data, force) {
        const rankingList = document.getElementById("rankingList");
        const signature = buildRankingSignature(rankingType, keyword, data);
        if (!force && signature === lastRankingSignature) {
            return;
        }
        lastRankingSignature = signature;
        rankingList.innerHTML = "";

        if (!data || data.length === 0) {
            renderAdminEmpty(rankingList, "暂无排行榜数据");
            return;
        }

        const fragment = document.createDocumentFragment();
        data.forEach((item, index) => {
            const rankingItem = document.createElement("li");
            rankingItem.className = "ranking-item";
            rankingItem.style.cursor = "pointer";
            rankingItem.dataset.artist = item.artist || "";
            rankingItem.dataset.album = item.album || "";
            rankingItem.dataset.track = item.track || "";

            const rank = index + 1;
            let rankClass = "rank-other";
            if (rank === 1) rankClass = "rank-1";
            else if (rank === 2) rankClass = "rank-2";
            else if (rank === 3) rankClass = "rank-3";

            rankingItem.innerHTML = `
                <div class="rank-number ${rankClass}">${rank}</div>
                <div class="track-info">
                    <div class="track-title">${item.track}</div>
                    <div class="track-artist">${item.artist} - ${item.album}</div>
                </div>
                <div class="play-count">${item.play_count} 次</div>
            `;
            fragment.appendChild(rankingItem);
        });
        rankingList.appendChild(fragment);
        adjustRankingListsHeight();
    }

    // 初始化最近播放列表
    function initRecentPlays() {
        const recentPlaysList = document.getElementById("recentPlaysList");
        if (!recentPlaysList.children.length) {
            renderAdminLoading(recentPlaysList);
        }

        fetch("/api/recent-plays?limit=10")
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                renderRecentPlaysList(data, true);
            })
            .catch((error) => {
                console.error("获取最近播放数据失败:", error);
                renderAdminError(recentPlaysList, error);
            });
    }

    // 定期刷新最近播放列表
    function refreshRecentPlays() {
        fetch("/api/recent-plays?limit=10")
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                renderRecentPlaysList(data, false);
            })
            .catch((error) => {
                console.error("刷新最近播放数据失败:", error);
            });
    }

    // 初始化排行榜
    function initRanking(rankingType = "all", keyword = "") {
        currentRankingType = rankingType;
        rankingSearchKeyword = keyword;

        const rankingList = document.getElementById("rankingList");
        if (!rankingList.children.length) {
            renderAdminLoading(rankingList);
        }

        // 根据排名类型调整API请求
        let apiUrl = `/api/track-play-counts?limit=10&offset=0${keyword ? '&keyword=' + encodeURIComponent(keyword) : ''}`;
        if (rankingType === "month") {
            apiUrl = `/api/track-play-counts/period?limit=10&offset=0&period=month${keyword ? '&keyword=' + encodeURIComponent(keyword) : ''}`;
        } else if (rankingType === "week") {
            apiUrl = `/api/track-play-counts/period?limit=10&offset=0&period=week${keyword ? '&keyword=' + encodeURIComponent(keyword) : ''}`;
        }

        fetch(apiUrl)
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                renderRankingList(rankingType, keyword, data, false);
            })
            .catch((error) => {
                console.error("获取排行榜数据失败:", error);
                renderAdminError(rankingList, error);
            });
    }

    // 调整排行榜列表高度，使两个卡片保持一致
    function adjustRankingListsHeight() {
        const recentPlaysList = document.getElementById("recentPlaysList");
        const rankingList = document.getElementById("rankingList");

        // 重置高度
        recentPlaysList.style.height = "auto";
        rankingList.style.height = "auto";

        // 获取两个列表的最大高度
        const recentPlaysHeight = recentPlaysList.offsetHeight;
        const rankingHeight = rankingList.offsetHeight;
        const maxHeight = Math.max(recentPlaysHeight, rankingHeight);

        // 设置两个列表的高度为最大高度
        if (maxHeight > 0) {
            // 只在需要时设置固定高度，避免影响布局
            if (recentPlaysHeight !== rankingHeight) {
                recentPlaysList.style.height = maxHeight + "px";
                rankingList.style.height = maxHeight + "px";
            }
        }
    }

    // 加载未上报记录
