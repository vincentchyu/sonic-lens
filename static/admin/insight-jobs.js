async function loadInsightList() {
        const content = document.getElementById("insightListContent");
        renderAdminLoading(content);

        // 实时从输入框获取最新关键词，确保刷新按钮点击时也能带上
        const searchInput = document.getElementById("insightSearchInput");
        if (searchInput) {
            currentInsightKeyword = searchInput.value.trim();
        }

        try {
            const offset = (currentInsightPage - 1) * insightPageSize;
            const url = `/api/insights/all?limit=${insightPageSize}&offset=${offset}&keyword=${encodeURIComponent(currentInsightKeyword)}&analysis_target_type=${encodeURIComponent(currentInsightTargetType)}`;
            const response = await fetch(url);
            const data = await response.json();

            totalInsights = data.total;
            renderInsightList(data.insights || []);
            updateInsightPagination();
        } catch (error) {
            console.error("加载音眸列表失败:", error);
            renderAdminError(content, error);
        }
    }

    // 渲染音眸列表
    function renderInsightList(insights) {
        const content = document.getElementById("insightListContent");
        if (insights.length === 0) {
            const emptyText = currentInsightTargetType === 'album' ? '暂无专辑解析记录' : '暂无曲目解析记录';
            renderAdminEmpty(content, emptyText);
            return;
        }

        const isDark = isDarkMode();
        const headerBgColor = isDark ? "#3a3a3a" : "#f1f3f9";
        const borderColor = isDark ? "#444444" : "#e9ecef";
        const textColor = isDark ? "#f0f0f0" : "#2c3e50";

        let html = `<div style="overflow-x: auto;"><table style="width: 100%; border-collapse: collapse; color: ${textColor};">`;
        const headerLabel = currentInsightTargetType === 'album' ? '艺术家 / 专辑' : '艺术家 / 曲目';

        html += `<thead style="background-color: ${headerBgColor}; position: sticky; top: 0; z-index: 10;"><tr>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">${headerLabel}</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">模型</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">对象类型</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">创建于</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor}; text-align: center;">状态</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor}; text-align: center;">操作</th>`;
        html += "</tr></thead><tbody>";

        insights.forEach(ins => {
            const time = new Date(ins.created_at).toLocaleString("zh-CN");
            const statusText = ins.is_disabled ? "已禁用" : "已启用";
            const statusClass = ins.is_disabled ? "negative" : "positive";
            const toggleText = ins.is_disabled ? "启用" : "禁用";
            const targetType = ins.analysis_target_type || currentInsightTargetType || 'track';
            const isAlbum = targetType === 'album';
            const displayTitle = isAlbum ? (ins.album || '-') : (ins.track || '-');
            const displaySubtitle = isAlbum ? (ins.artist || '-') : `${ins.artist || '-'} · ${ins.album || '-'}`;
            const typeBadge = isAlbum
                ? '<span style="display:inline-flex;align-items:center;padding:3px 8px;border-radius:999px;background:rgba(99,102,241,0.16);color:#818cf8;font-size:0.75em;font-weight:700;">专辑</span>'
                : '<span style="display:inline-flex;align-items:center;padding:3px 8px;border-radius:999px;background:rgba(16,185,129,0.16);color:#34d399;font-size:0.75em;font-weight:700;">曲目</span>';

            html += `<tr style="border-bottom: 1px solid ${borderColor};">`;
            html += `<td style="padding: 12px;">
                        <div style="font-weight: bold;">${escapeHtmlText(displayTitle)}</div>
                        <div style="font-size: 0.85em; opacity: 0.8;">${escapeHtmlText(displaySubtitle)}</div>
                     </td>`;
            html += `<td style="padding: 12px;">${ins.llm_provider || '-'}</td>`;
            html += `<td style="padding: 12px;">${typeBadge}</td>`;
            html += `<td style="padding: 12px;">${time}</td>`;
            html += `<td style="padding: 12px; text-align: center;">
                        <span class="stat-change ${statusClass}" style="float:none; padding: 2px 8px; border-radius: 4px;">${statusText}</span>
                     </td>`;
            html += `<td style="padding: 12px; text-align: center;">
                        <div style="display: flex; gap: 8px; justify-content: center;">
                            ${isAlbum ? `
                                <button class="time-filter" onclick="toggleInsightStatus(${ins.id}, '${targetType}')" style="padding: 4px 10px;">${toggleText}</button>
                                <button class="time-filter" onclick="showInsightDetailsById(${ins.id}, '${targetType}')" style="padding: 4px 10px;">详情</button>
                                <button class="time-filter" onclick="showAlbumInsightDetailsByAlbumId(${ins.album_id || 0})" style="padding: 4px 10px;">专辑</button>
                                <button class="time-filter" onclick="showInsightCallLogs(${ins.id}, '${esc(ins.artist)}', '${esc(ins.album)}', '${targetType}')" style="padding: 4px 10px;">流水</button>
                                <button class="time-filter" onclick="deleteInsight(${ins.id}, '${targetType}')" style="padding: 4px 10px; background-color: #e74c3c; color: white; border-color: #e74c3c;">删除</button>
                            ` : `
                                <button class="time-filter" onclick="toggleInsightStatus(${ins.id}, '${targetType}')" style="padding: 4px 10px;">${toggleText}</button>
                                <button class="time-filter" onclick="showInsightDetailsById(${ins.id}, '${targetType}')" style="padding: 4px 10px;">详情</button>
                                <button class="time-filter" onclick="showInsightCallLogs(${ins.id}, '${esc(ins.artist)}', '${esc(ins.track)}', '${targetType}')" style="padding: 4px 10px;">流水</button>
                                <button class="time-filter" onclick="deleteInsight(${ins.id}, '${targetType}')" style="padding: 4px 10px; background-color: #e74c3c; color: white; border-color: #e74c3c;">删除</button>
                            `}
                        </div>
                     </td>`;
            html += "</tr>";
        });

        html += "</tbody></table></div>";
        content.innerHTML = html;
    }

    // 更新音眸列表分页
    function updateInsightPagination() {
        const totalPages = Math.ceil(totalInsights / insightPageSize);
        const info = document.getElementById("insightPageInfo");
        const targetLabel = currentInsightTargetType === 'album' ? '专辑' : '曲目';
        info.textContent = `第 ${currentInsightPage} 页 / 共 ${totalPages} 页 (${totalInsights} 条 ${targetLabel})`;
        document.getElementById("insightPrevPage").disabled = currentInsightPage === 1;
        document.getElementById("insightNextPage").disabled = currentInsightPage >= totalPages;
    }

    function normalizeInsightJobStatus(status) {
        return String(status || "queued").toLowerCase();
    }

    function renderInsightJobPhaseBadge(status) {
        const normalized = normalizeInsightJobStatus(status);
        const textMap = {
            queued: "已排队",
            running: "运行中",
            completed: "已完成",
            failed: "失败",
            canceled: "已取消"
        };
        return `<span class="queue-phase-badge" data-phase="${normalized}">${textMap[normalized] || "已排队"}</span>`;
    }

    function renderInsightJobTargetBadge(targetType) {
        if (targetType === "album") {
            return '<span class="queue-badge queue-badge-album">专辑</span>';
        }
        return '<span class="queue-badge queue-badge-track">曲目</span>';
    }

    function mergeInsightJobSnapshot(job) {
        if (!job || !job.id) return job;
        return {
            ...(insightJobListCache[job.id] || {}),
            ...(currentInsightJobDetail && currentInsightJobDetail.id === job.id ? currentInsightJobDetail : {}),
            ...job
        };
    }

    function getInsightJobDetailStateKey(jobID, target) {
        return `${jobID || ""}::${target || "inspector"}`;
    }

    function captureInsightJobDetailViewState(jobID, target) {
        if (!jobID) return;
        const normalizedTarget = target === "modal" ? "modal" : "inspector";
        const container = normalizedTarget === "modal"
            ? document.getElementById("insightJobModalContent")
            : document.getElementById("insightJobInspector");
        if (!container) return;

        const openDetails = [];
        container.querySelectorAll("details[data-detail-key]").forEach((item) => {
            if (item.open) {
                openDetails.push(item.dataset.detailKey);
            }
        });

        insightJobDetailViewState[getInsightJobDetailStateKey(jobID, normalizedTarget)] = {
            scrollTop: container.scrollTop || 0,
            openDetails
        };
    }

    function restoreInsightJobDetailViewState(jobID, target) {
        if (!jobID) return;
        const normalizedTarget = target === "modal" ? "modal" : "inspector";
        const container = normalizedTarget === "modal"
            ? document.getElementById("insightJobModalContent")
            : document.getElementById("insightJobInspector");
        if (!container) return;

        const state = insightJobDetailViewState[getInsightJobDetailStateKey(jobID, normalizedTarget)];
        if (!state) return;

        const openSet = new Set(Array.isArray(state.openDetails) ? state.openDetails : []);
        container.querySelectorAll("details[data-detail-key]").forEach((item) => {
            item.open = openSet.has(item.dataset.detailKey);
        });

        if (typeof state.scrollTop === "number" && state.scrollTop > 0) {
            container.scrollTop = state.scrollTop;
        }
    }

    function getCurrentInsightJobDetailTarget() {
        const modal = document.getElementById("insightJobModal");
        if (modal && modal.style.display === "block") {
            return "modal";
        }
        return "inspector";
    }

    function formatInsightJobSubject(job) {
        if (!job) return { title: "-", subtitle: "-" };
        if (job.analysis_target_type === "album") {
            return {
                title: job.album || `专辑 #${job.album_id || 0}`,
                subtitle: `${job.artist || "-"} · album_id=${job.album_id || 0}`
            };
        }
        return {
            title: job.track || "-",
            subtitle: `${job.artist || "-"} · ${job.album || "-"}`
        };
    }

    function parseJobDate(value) {
        if (!value) return null;
        const date = new Date(value);
        return Number.isNaN(date.getTime()) ? null : date;
    }

    function formatJobDateTime(value) {
        const date = parseJobDate(value);
        return date ? date.toLocaleString("zh-CN") : "-";
    }

    function formatDurationBetween(startValue, endValue) {
        const start = parseJobDate(startValue);
        const end = parseJobDate(endValue) || new Date();
        if (!start || !end) return "未记录";
        const diffMs = Math.max(0, end.getTime() - start.getTime());
        const totalSeconds = Math.floor(diffMs / 1000);
        const hours = Math.floor(totalSeconds / 3600);
        const minutes = Math.floor((totalSeconds % 3600) / 60);
        const seconds = totalSeconds % 60;
        if (hours > 0) return `${hours}h ${minutes}m`;
        if (minutes > 0) return `${minutes}m ${seconds}s`;
        return `${seconds}s`;
    }

    function deriveInsightJobMetrics(job) {
        const status = normalizeInsightJobStatus(job.status);
        const createdAt = parseJobDate(job.created_at);
        const startedAt = parseJobDate(job.started_at);
        const finishedAt = parseJobDate(job.finished_at);
        const updatedAt = parseJobDate(job.updated_at);
        const now = new Date();
        const queuedEndValue = status === "queued" ? now : (updatedAt || now);
        const runningEndValue = status === "running" ? now : (finishedAt || updatedAt || now);
        const waitDurationText = startedAt
            ? formatDurationBetween(job.created_at, job.started_at)
            : status === "queued"
                ? `${formatDurationBetween(job.created_at, queuedEndValue)} 待分配`
                : "未开始";
        const runDurationText = startedAt
            ? formatDurationBetween(job.started_at, runningEndValue)
            : "未执行";
        const finalTimeText = finishedAt
            ? `${formatJobDateTime(job.finished_at)} 结束`
            : updatedAt
                ? `${formatJobDateTime(job.updated_at)} 最近更新`
                : "暂无时间";
        const displayPhaseGroup = status === "running"
            ? "active"
            : status === "failed"
                ? "blocked"
                : status === "queued"
                    ? "queued"
                    : "done";
        return {
            createdAt,
            startedAt,
            finishedAt,
            updatedAt,
            waitDurationText,
            runDurationText,
            finalTimeText,
            displayPhaseGroup,
            isActionable: status === "queued" || status === "running" || status === "failed"
        };
    }

    function getInsightJobSortWeight(job) {
        const status = normalizeInsightJobStatus(job.status);
        if (status === "running") return 0;
        if (status === "failed") return 1;
        if (status === "queued") return 2;
        return 3;
    }

    function compareInsightJobs(a, b) {
        const weightDiff = getInsightJobSortWeight(a) - getInsightJobSortWeight(b);
        if (weightDiff !== 0) return weightDiff;
        const aTime = parseJobDate(a.updated_at)?.getTime() || 0;
        const bTime = parseJobDate(b.updated_at)?.getTime() || 0;
        return bTime - aTime;
    }

    function isInsightJobInspectorAvailable() {
        return window.matchMedia("(min-width: 1181px)").matches;
    }

    function updateInsightJobTabLayout() {
        const workspace = document.getElementById("insightJobWorkspace");
        const monitorView = document.getElementById("insightJobMonitorView");
        const listView = document.getElementById("insightJobListView");
        const toolbarControls = document.querySelector(".queue-toolbar-controls");

        document.querySelectorAll(".insight-job-tab").forEach(tab => {
            tab.classList.toggle("active", tab.dataset.tab === currentInsightJobTab);
        });

        if (workspace) {
            workspace.classList.toggle("is-monitor", currentInsightJobTab === "monitor");
            workspace.classList.toggle("is-list", currentInsightJobTab === "list");
        }

        if (monitorView) {
            monitorView.style.display = currentInsightJobTab === "monitor" ? "flex" : "none";
        }

        if (listView) {
            listView.style.display = currentInsightJobTab === "list" ? "flex" : "none";
        }

        if (toolbarControls) {
            toolbarControls.classList.toggle("is-hidden", currentInsightJobTab !== "list");
        }
    }

    function switchInsightJobTab(tabName) {
        if (!tabName || tabName === currentInsightJobTab) {
            updateInsightJobTabLayout();
            syncInsightJobWorkspaceHeight();
            return;
        }
        currentInsightJobTab = tabName;
        updateInsightJobTabLayout();
        syncInsightJobWorkspaceHeight();
    }

    function insightJobNeedsLiveTick(job) {
        const status = normalizeInsightJobStatus(job && job.status);
        return status === "queued" || status === "running";
    }

    function refreshInsightJobDurationsLocally() {
        if (currentSectionID !== "insightJobList") {
            return;
        }
        const hasLiveJobs = currentInsightJobRows.some(insightJobNeedsLiveTick)
            || currentInsightJobSummaryRows.some(insightJobNeedsLiveTick)
            || (currentInsightJobDetail && insightJobNeedsLiveTick(currentInsightJobDetail));
        if (!hasLiveJobs) {
            return;
        }
        renderInsightJobList(currentInsightJobRows);
        renderInsightJobPriorityStrip(currentInsightJobSummaryRows);
        if (selectedInsightJobID) {
            const selected = (currentInsightJobDetail && currentInsightJobDetail.id === selectedInsightJobID)
                ? currentInsightJobDetail
                : insightJobListCache[selectedInsightJobID];
            if (selected) {
                renderInsightJobDetail(selected, {
                    target: isInsightJobInspectorAvailable() ? "inspector" : "modal"
                });
            }
        }
    }

    function updateInsightJobRealtimeIndicator() {
        const indicator = document.getElementById("insightJobRealtimeIndicator");
        if (!indicator) return;
        indicator.classList.toggle("queue-chip-live", insightJobWSConnected);
        indicator.classList.toggle("queue-chip-polling", !insightJobWSConnected);
        indicator.innerHTML = `
            <span class="queue-chip-dot"></span>
            <span>${insightJobWSConnected ? "实时通道在线，状态会自动回流" : "实时通道断开，当前降级为轮询"}</span>
        `;
    }

    function scheduleInsightJobListRefresh() {
        if (currentSectionID !== "insightJobList") {
            return;
        }
        if (insightJobRefreshTimer) {
            clearTimeout(insightJobRefreshTimer);
        }
        insightJobRefreshTimer = setTimeout(() => {
            loadInsightJobList(currentInsightJobPage);
        }, 250);
    }

    async function loadInsightJobSummarySnapshot() {
        try {
            const params = new URLSearchParams({
                limit: "200",
                offset: "0"
            });
            const response = await fetch(`/api/insight-jobs?${params.toString()}`);
            if (!response.ok) return;
            const data = await response.json();
            currentInsightJobSummaryRows = Array.isArray(data.jobs) ? data.jobs.slice() : [];
            renderInsightJobSummaryBar(currentInsightJobSummaryRows, data.total || currentInsightJobSummaryRows.length);
            renderInsightJobPriorityStrip(currentInsightJobSummaryRows);
        } catch (error) {
            console.error("加载音眸任务摘要失败:", error);
        }
    }

    async function loadInsightJobList(page = 1) {
        currentInsightJobPage = page;
        const content = document.getElementById("insightJobListContent");
        if (!content) return;
        content.innerHTML = '<div class="queue-empty">加载中...</div>';

        const searchInput = document.getElementById("insightJobSearchInput");
        const statusFilter = document.getElementById("insightJobStatusFilter");
        const targetFilter = document.getElementById("insightJobTargetFilter");
        if (searchInput) currentInsightJobKeyword = searchInput.value.trim();
        if (statusFilter) currentInsightJobStatus = statusFilter.value;
        if (targetFilter) currentInsightJobTargetType = targetFilter.value;

        try {
            const offset = (currentInsightJobPage - 1) * insightJobPageSize;
            const params = new URLSearchParams({
                limit: String(insightJobPageSize),
                offset: String(offset)
            });
            if (currentInsightJobKeyword) params.set("keyword", currentInsightJobKeyword);
            if (currentInsightJobStatus) params.set("status", currentInsightJobStatus);
            if (currentInsightJobTargetType) params.set("analysis_target_type", currentInsightJobTargetType);

            const response = await fetch(`/api/insight-jobs?${params.toString()}`);
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }
            const data = await response.json();
            totalInsightJobs = data.total || 0;
            currentInsightJobRows = (data.jobs || []).map((job) => mergeInsightJobSnapshot(job)).slice().sort(compareInsightJobs);
            currentInsightJobRows.forEach((job) => {
                insightJobListCache[job.id] = job;
            });
            renderInsightJobList(currentInsightJobRows);
            updateInsightJobPagination();
            updateInsightJobRealtimeIndicator();
            await loadInsightJobSummarySnapshot();

            if (selectedInsightJobID) {
                const selected = (currentInsightJobDetail && currentInsightJobDetail.id === selectedInsightJobID)
                    ? mergeInsightJobSnapshot(currentInsightJobDetail)
                    : insightJobListCache[selectedInsightJobID];
                if (selected) {
                    renderInsightJobDetail(selected, { target: isInsightJobInspectorAvailable() ? "inspector" : "modal" });
                }
            }
        } catch (error) {
            console.error("加载音眸任务失败:", error);
            content.innerHTML = `<div class="error">加载失败: ${escapeHtmlText(error.message)}</div>`;
        }
    }

    function renderInsightJobSummaryBar(jobs, totalCount) {
        const container = document.getElementById("insightJobSummaryBar");
        if (!container) return;
        const counts = {
            all: totalCount || jobs.length,
            queued: 0,
            running: 0,
            failed: 0,
            canceled:0,
            completed: 0
        };
        jobs.forEach((job) => {
            const status = normalizeInsightJobStatus(job.status);
            if (status === "queued") counts.queued += 1;
            else if (status === "running") counts.running += 1;
            else if (status === "failed") counts.failed += 1;
            else if (status === "canceled") counts.canceled += 1;
            else if (status === "completed") counts.completed += 1;
        });

        container.innerHTML = `
            <div class="queue-summary-card">
                <div class="queue-summary-label">全部任务</div>
                <div class="queue-summary-value">${counts.all}</div>
                <div class="queue-summary-note">当前检索视图共 ${totalInsightJobs} 条</div>
            </div>
            <div class="queue-summary-card" data-tone="queued">
                <div class="queue-summary-label">排队中</div>
                <div class="queue-summary-value">${counts.queued}</div>
                <div class="queue-summary-note">等待 worker 处理</div>
            </div>
            <div class="queue-summary-card" data-tone="running">
                <div class="queue-summary-label">运行中</div>
                <div class="queue-summary-value">${counts.running}</div>
                <div class="queue-summary-note">优先跟进超时或阻塞</div>
            </div>
            <div class="queue-summary-card" data-tone="failed">
                <div class="queue-summary-label">失败待处理</div>
                <div class="queue-summary-value">${counts.failed}</div>
                <div class="queue-summary-note">建议优先重试或排障</div>
            </div>
             <div class="queue-summary-card" data-tone="canceled">
                <div class="queue-summary-label">已取消</div>
                <div class="queue-summary-value">${counts.canceled}</div>
                <div class="queue-summary-note">不建议重启</div>
            </div>
            <div class="queue-summary-card" data-tone="success">
                <div class="queue-summary-label">已完成</div>
                <div class="queue-summary-value">${counts.completed}</div>
                <div class="queue-summary-note">结果可回看与跳转</div>
            </div>
        `;
    }

    function renderInsightJobPriorityLane(title, note, jobs, emptyText) {
        if (!jobs.length) {
            return `
                <div class="queue-priority-lane">
                    <div class="queue-priority-head">
                        <strong>${title}</strong>
                        <span>${note}</span>
                    </div>
                    <div class="queue-priority-list">
                        <div class="queue-empty">${emptyText}</div>
                    </div>
                </div>
            `;
        }

        const items = jobs.map((job) => {
            const subject = formatInsightJobSubject(job);
            const metrics = deriveInsightJobMetrics(job);
            const status = normalizeInsightJobStatus(job.status);
            const hasResult = !!job.result_insight_id && !!job.result_available;
            const canDelete = status === "failed" || status === "canceled";
            const secondary = normalizeInsightJobStatus(job.status) === "running"
                ? `执行 ${metrics.runDurationText}`
                : `最近更新 ${formatJobDateTime(job.updated_at)}`;
            return `
                <div class="queue-priority-item">
                    <div class="queue-priority-meta">
                        <div class="queue-row-topline">
                            ${renderInsightJobTargetBadge(job.analysis_target_type)}
                            ${renderInsightJobPhaseBadge(job.status)}
                        </div>
                        <div class="queue-priority-title" title="${escapeHtmlText(subject.title)}">${escapeHtmlText(subject.title)}</div>
                        <div class="queue-priority-subtitle" title="${escapeHtmlText(subject.subtitle)}">${escapeHtmlText(subject.subtitle)} · ${escapeHtmlText(secondary)}</div>
                    </div>
                    <div class="queue-priority-actions">
                        <button class="queue-btn" onclick="showInsightJobDetails('${esc(job.id)}')">详情</button>
                        ${hasResult
                            ? `<button class="queue-btn queue-btn-primary" onclick="openInsightJobResult('${esc(job.id)}')">结果</button>`
                            : status === "failed"
                                ? `<button class="queue-btn" onclick="retryInsightJob('${esc(job.id)}')">重试</button>${canDelete ? `<button class="queue-btn queue-btn-danger" onclick="deleteInsightJob('${esc(job.id)}')">删除</button>` : ""}`
                                : status === "running" || status === "queued"
                                    ? `<button class="queue-btn queue-btn-danger" onclick="cancelInsightJob('${esc(job.id)}')">取消</button>`
                                    : canDelete
                                        ? `<button class="queue-btn queue-btn-danger" onclick="deleteInsightJob('${esc(job.id)}')">删除</button>`
                                        : `<button class="queue-btn" disabled>等待</button>`
                        }
                    </div>
                </div>
            `;
        }).join("");

        return `
            <div class="queue-priority-lane">
                <div class="queue-priority-head">
                    <strong>${title}</strong>
                    <span>${note}</span>
                </div>
                <div class="queue-priority-list">${items}</div>
            </div>
        `;
    }

    function renderInsightJobPriorityStrip(jobs) {
        const container = document.getElementById("insightJobPriorityStrip");
        if (!container) return;
        const sorted = jobs.slice().sort(compareInsightJobs);
        const running = sorted.filter((job) => normalizeInsightJobStatus(job.status) === "running").slice(0, 5);
        const failed = sorted.filter((job) => normalizeInsightJobStatus(job.status) === "failed").slice(0, 5);
        const canceled = sorted.filter((job) => normalizeInsightJobStatus(job.status) === "canceled").slice(0, 5);

        container.innerHTML = `
            ${renderInsightJobPriorityLane("正在执行", "优先观察长时运行与结果回流", running, "暂无运行中的任务")}
            ${renderInsightJobPriorityLane("失败待处理", "优先恢复阻塞任务", failed, "暂无失败任务")}
            ${renderInsightJobPriorityLane("已取消待清理", "可直接删除无效任务记录", canceled, "暂无已取消任务")}
        `;
    }

    function syncInsightJobWorkspaceHeight() {
        const workspace = document.getElementById("insightJobWorkspace");
        const listContent = document.getElementById("insightJobListContent");
        const monitorContent = document.getElementById("insightJobPriorityStrip");
        const inspector = document.getElementById("insightJobInspector");
        if (!workspace) return;

        const minHeight = 640;
        const maxHeight = Math.max(minHeight, window.innerHeight - 240);
        const activeContent = currentInsightJobTab === "list" ? listContent : monitorContent;
        const contentHeight = activeContent ? (activeContent.scrollHeight || activeContent.offsetHeight || 0) : 0;
        const inspectorHeight = inspector ? (inspector.scrollHeight || inspector.offsetHeight || 0) : 0;
        const nextHeight = Math.min(Math.max(Math.max(contentHeight, inspectorHeight) + 140, minHeight), maxHeight);
        workspace.style.setProperty("--insight-job-workspace-height", `${nextHeight}px`);
    }

    function renderInsightJobRow(job) {
        const subject = formatInsightJobSubject(job);
        const metrics = deriveInsightJobMetrics(job);
        const hasResult = !!job.result_insight_id && !!job.result_available;
        const canCancel = normalizeInsightJobStatus(job.status) === "queued" || normalizeInsightJobStatus(job.status) === "running";
        const canDelete = normalizeInsightJobStatus(job.status) === "failed" || normalizeInsightJobStatus(job.status) === "canceled";
        const provider = escapeHtmlText(job.provider_display_name || job.provider || "-");
        const model = escapeHtmlText(job.model_display_name || job.model || "-");
        const clientPlatform = escapeHtmlText(job.client_platform || "-");
        const resultBadge = hasResult
            ? '<span class="queue-result-badge is-ready">结果可查看</span>'
            : '<span class="queue-result-badge">结果未就绪</span>';
        const statusLine = normalizeInsightJobStatus(job.status) === "queued"
            ? `已等待 ${metrics.waitDurationText}`
            : normalizeInsightJobStatus(job.status) === "running"
                ? `执行时长 ${metrics.runDurationText}`
                : metrics.finalTimeText;

        return `
            <div
                class="queue-row${selectedInsightJobID === job.id ? " is-selected" : ""}${highlightedInsightJobID === job.id ? " is-updated" : ""}"
                data-job-id="${escapeHtmlText(job.id)}"
            >
                <div class="queue-row-main">
                    <div class="queue-row-topline">
                        ${renderInsightJobTargetBadge(job.analysis_target_type)}
                        <span class="queue-id">${escapeHtmlText(job.id || "-")}</span>
                    </div>
                    <div class="queue-row-title" title="${escapeHtmlText(subject.title)}">${escapeHtmlText(subject.title)}</div>
                    <div class="queue-row-subtitle" title="${escapeHtmlText(subject.subtitle)}">${escapeHtmlText(subject.subtitle)}</div>
                    <div class="queue-row-meta">
                        ${job.error_message ? `<span style="color: var(--queue-failed);" title="${escapeHtmlText(job.error_message)}">错误已记录</span>` : '<span>无异常</span>'}
                    </div>
                </div>
                <div class="queue-meta-block">
                    <div class="queue-meta-strong">${model}</div>
                    <div>${provider}</div>
                    <div>${clientPlatform}</div>
                </div>
                <div class="queue-status-block">
                    <div class="queue-status-note">
                        ${renderInsightJobPhaseBadge(job.status)}
                        ${resultBadge}
                    </div>
                    <div class="queue-status-note">${escapeHtmlText(statusLine)}</div>
                    <div class="queue-status-note">最近更新 ${escapeHtmlText(formatJobDateTime(job.updated_at))}</div>
                </div>
                <div class="queue-actions">
                    <button class="queue-btn" onclick="showInsightJobDetails('${esc(job.id)}')">详情</button>
                    ${hasResult
                        ? `<button class="queue-btn queue-btn-primary" onclick="openInsightJobResult('${esc(job.id)}')">查看结果</button>`
                        : `<button class="queue-btn" onclick="retryInsightJob('${esc(job.id)}')">重试</button>`
                    }
                    ${canCancel
                        ? `<button class="queue-btn queue-btn-danger" onclick="cancelInsightJob('${esc(job.id)}')">取消</button>`
                        : `<button class="queue-btn" disabled>取消</button>`
                    }
                    ${canDelete
                        ? `<button class="queue-btn queue-btn-danger" onclick="deleteInsightJob('${esc(job.id)}')">删除</button>`
                        : `<button class="queue-btn" disabled>删除</button>`
                    }
                </div>
            </div>
        `;
    }

    function bindInsightJobRowEvents() {
        document.querySelectorAll("#insightJobListContent .queue-row").forEach((row) => {
            if (row.dataset.bound === "1") return;
            row.addEventListener("click", (event) => {
                if (event.target.closest("button")) return;
                const jobID = row.dataset.jobId;
                if (jobID) {
                    showInsightJobDetails(jobID, { preferModal: false });
                }
            });
            row.dataset.bound = "1";
        });
    }

    function renderInsightJobList(jobs) {
        const content = document.getElementById("insightJobListContent");
        if (!content) return;
        if (!jobs || jobs.length === 0) {
            content.innerHTML = '<div class="queue-empty">暂无符合条件的音眸任务</div>';
            renderInsightJobInspectorEmpty();
            syncInsightJobWorkspaceHeight();
            return;
        }
        content.innerHTML = jobs.map((job) => renderInsightJobRow(job)).join("");
        bindInsightJobRowEvents();
        syncInsightJobWorkspaceHeight();
        if (highlightedInsightJobID) {
            window.setTimeout(() => {
                highlightedInsightJobID = "";
                renderInsightJobList(currentInsightJobRows);
            }, 1200);
        }
    }

    function updateInsightJobPagination() {
        const totalPages = Math.max(1, Math.ceil(totalInsightJobs / insightJobPageSize));
        const info = document.getElementById("insightJobPageInfo");
        if (info) {
            info.textContent = `第 ${currentInsightJobPage} 页 / 共 ${totalPages} 页 (${totalInsightJobs} 条任务)`;
        }
        const prev = document.getElementById("insightJobPrevPage");
        const next = document.getElementById("insightJobNextPage");
        if (prev) prev.disabled = currentInsightJobPage <= 1;
        if (next) next.disabled = currentInsightJobPage >= totalPages;
    }

    function buildInsightJobTimeline(job) {
        const status = normalizeInsightJobStatus(job.status);
        const items = [
            { label: "任务创建", time: formatJobDateTime(job.created_at), tone: "success", note: "任务已写入队列" },
            { label: "等待执行", time: job.started_at ? `${formatJobDateTime(job.started_at)} 开始` : `${deriveInsightJobMetrics(job).waitDurationText}`, tone: "queued", note: "等待 worker 获取任务" }
        ];
        if (job.started_at) {
            items.push({
                label: "开始运行",
                time: formatJobDateTime(job.started_at),
                tone: "running",
                note: `执行时长 ${deriveInsightJobMetrics(job).runDurationText}`
            });
        }
        if (status === "completed" || status === "failed" || status === "canceled") {
            items.push({
                label: status === "completed" ? "执行完成" : status === "failed" ? "执行失败" : "任务已取消",
                time: formatJobDateTime(job.finished_at || job.updated_at),
                tone: status === "completed" ? "success" : status === "failed" ? "failed" : "queued",
                note: status === "completed"
                    ? (job.result_available ? "结果已可查看" : "任务完成但暂无结果")
                    : status === "failed"
                        ? "查看错误信息后可直接重试"
                        : "任务被手动或系统取消"
            });
        } else {
            items.push({
                label: "持续跟进",
                time: formatJobDateTime(job.updated_at),
                tone: status === "running" ? "running" : "queued",
                note: "队列仍在推进"
            });
        }
        return items.map((item) => `
            <div class="queue-timeline-item">
                <div class="queue-timeline-dot" data-tone="${item.tone}"></div>
                <div>
                    <div class="queue-timeline-label">${item.label}</div>
                    <div class="queue-timeline-time">${escapeHtmlText(item.time)}</div>
                    <div class="queue-timeline-time">${escapeHtmlText(item.note)}</div>
                </div>
            </div>
        `).join("");
    }

    function buildInsightCallLogCard(log) {
        if (!log) return "";
        const time = formatJobDateTime(log.created_at);
        const isError = log.status !== "success" && log.status !== "ok";
        const statusColor = isError ? "#e74c3c" : "#2ecc71";
        const targetType = log.analysis_target_type || "track";
        const targetKey = escapeHtmlText(log.target_key || log.track_info || "-");
        const targetMetadata = prettyJsonLike(log.target_metadata);
        const requestJson = prettyJsonLike(log.request_json);
        const responseJson = prettyJsonLike(log.response_json);
        const requestSize = log.request_json ? String(log.request_json.length) : "0";
        const responseSize = log.response_json ? String(log.response_json.length) : "0";
        const objectBadge = targetType === "album"
            ? '<span style="display:inline-flex;align-items:center;padding:3px 8px;border-radius:999px;background:rgba(99,102,241,0.16);color:#818cf8;font-size:0.75em;font-weight:700;">专辑</span>'
            : '<span style="display:inline-flex;align-items:center;padding:3px 8px;border-radius:999px;background:rgba(16,185,129,0.16);color:#34d399;font-size:0.75em;font-weight:700;">曲目</span>';

        return `
            <div style="border: 1px solid #444; border-radius: 12px; padding: 14px; background: rgba(255,255,255,0.03); box-shadow: 0 8px 24px rgba(0,0,0,0.08);">
                <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 10px;">
                    <div style="min-width: 0;">
                        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">
                            <span style="font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${escapeHtmlText(log.provider)} (${escapeHtmlText(log.model)})</span>
                            ${objectBadge}
                        </div>
                        <div style="font-size: 0.8em; opacity: 0.72;">${escapeHtmlText(time)}</div>
                    </div>
                    <div style="text-align: right; font-size: 0.82em; line-height: 1.5; opacity: 0.86;">
                        <div>状态: <span style="color: ${statusColor}; font-weight: 700;">${escapeHtmlText(log.status || "-")}</span> | 耗时: ${escapeHtmlText(String(log.duration_ms || 0))}ms</div>
                        <div>请求 ${escapeHtmlText(requestSize)} 字符 | 响应 ${escapeHtmlText(responseSize)} 字符</div>
                    </div>
                </div>
                <div style="margin-bottom: 10px; font-size: 0.85em; opacity: 0.84; word-break: break-all;">
                    对象键: <span style="font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;">${targetKey}</span>
                </div>
                ${targetMetadata ? `
                <details data-detail-key="log-${escapeHtmlText(String(log.id || 0))}-metadata" style="margin-bottom: 10px;">
                    <summary style="cursor: pointer; opacity: 0.84; font-size: 0.9em;">对象元数据</summary>
                    <pre style="margin: 8px 0 0 0; white-space: pre-wrap; background: #1a1a1a; padding: 10px; border-radius: 8px; overflow-x: auto;">${escapeHtmlText(targetMetadata)}</pre>
                </details>` : ""}
                ${log.error_msg ? `<div style="margin-bottom: 10px; color: #e74c3c; font-size: 0.86em; line-height: 1.6;">错误: ${escapeHtmlText(log.error_msg)}</div>` : ""}
                <details data-detail-key="log-${escapeHtmlText(String(log.id || 0))}-payload">
                    <summary style="cursor: pointer; opacity: 0.84; font-size: 0.9em;">查看原始请求与响应</summary>
                    <div style="margin-top: 10px; display: flex; flex-direction: column; gap: 10px; font-size: 0.8em;">
                        <div>
                            <div style="font-weight: 700; margin-bottom: 6px;">Request</div>
                            <pre style="white-space: pre-wrap; background: #1a1a1a; padding: 10px; border-radius: 8px; overflow-x: auto;">${escapeHtmlText(requestJson)}</pre>
                        </div>
                        <div>
                            <div style="font-weight: 700; margin-bottom: 6px;">Response</div>
                            <pre style="white-space: pre-wrap; background: #1a1a1a; padding: 10px; border-radius: 8px; overflow-x: auto;">${escapeHtmlText(responseJson)}</pre>
                        </div>
                    </div>
                </details>
            </div>
        `;
    }

    function renderInsightCallLogList(logs, emptyText = "暂无关联流水记录") {
        if (!Array.isArray(logs) || logs.length === 0) {
            return `<div class="queue-empty">${escapeHtmlText(emptyText)}</div>`;
        }
        return `<div style="display: flex; flex-direction: column; gap: 15px;">${logs.map((log) => buildInsightCallLogCard(log)).join("")}</div>`;
    }

    function buildInsightJobDetailHtml(job, options = {}) {
        const subject = formatInsightJobSubject(job);
        const hasResult = !!job.result_insight_id && !!job.result_available;
        const canCancel = normalizeInsightJobStatus(job.status) === "queued" || normalizeInsightJobStatus(job.status) === "running";
        const canDelete = normalizeInsightJobStatus(job.status) === "failed" || normalizeInsightJobStatus(job.status) === "canceled";
        const metrics = deriveInsightJobMetrics(job);
        const includeClose = options.includeClose === true;
        const callLogs = Array.isArray(job.call_logs) ? job.call_logs : [];
        return `
            <div class="queue-inspector-shell">
                <div class="queue-inspector-top">
                    <div class="queue-row-topline">
                        ${renderInsightJobTargetBadge(job.analysis_target_type)}
                        ${renderInsightJobPhaseBadge(job.status)}
                        ${hasResult ? '<span class="queue-result-badge is-ready">结果可查看</span>' : '<span class="queue-result-badge">结果未就绪</span>'}
                    </div>
                    <div class="queue-inspector-title">${escapeHtmlText(subject.title)}</div>
                    <div class="queue-inspector-subtitle">${escapeHtmlText(subject.subtitle)}</div>
                    <div class="queue-inspector-actions">
                        <button class="queue-btn" onclick="retryInsightJob('${esc(job.id)}')">重试</button>
                        ${canCancel ? `<button class="queue-btn queue-btn-danger" onclick="cancelInsightJob('${esc(job.id)}')">取消</button>` : ""}
                        ${canDelete ? `<button class="queue-btn queue-btn-danger" onclick="deleteInsightJob('${esc(job.id)}')">删除</button>` : ""}
                        ${hasResult ? `<button class="queue-btn queue-btn-primary" onclick="openInsightJobResult('${esc(job.id)}')">查看结果</button>` : ""}
                        ${includeClose ? `<button class="queue-btn" onclick="hideModal('insightJobModal')">关闭</button>` : ""}
                    </div>
                </div>

                <div class="queue-section">
                    <div class="queue-section-title">状态时间线</div>
                    <div class="queue-timeline">${buildInsightJobTimeline(job)}</div>
                </div>

                <div class="queue-section">
                    <div class="queue-section-title">诊断信息</div>
                    <div class="queue-kv-grid">
                        <div class="queue-kv-card">
                            <strong>任务 ID</strong>
                            <span class="queue-id">${escapeHtmlText(job.id || "-")}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>目标键</strong>
                            <span>${escapeHtmlText(job.target_key || "-")}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>模型</strong>
                            <span>${escapeHtmlText(job.provider_display_name || job.provider || "-")} / ${escapeHtmlText(job.model_display_name || job.model || "-")}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>客户端</strong>
                            <span>${escapeHtmlText(job.client_platform || "-")}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>等待时长</strong>
                            <span>${escapeHtmlText(metrics.waitDurationText)}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>执行时长</strong>
                            <span>${escapeHtmlText(metrics.runDurationText)}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>创建时间</strong>
                            <span>${escapeHtmlText(formatJobDateTime(job.created_at))}</span>
                        </div>
                        <div class="queue-kv-card">
                            <strong>最近更新时间</strong>
                            <span>${escapeHtmlText(formatJobDateTime(job.updated_at))}</span>
                        </div>
                    </div>
                </div>

                <div class="queue-section">
                    <div class="queue-section-title">错误信息</div>
                    <div class="queue-error-box">${escapeHtmlText(job.error_message || "无")}</div>
                </div>

                <div class="queue-section">
                    <div class="queue-section-title">关联流水</div>
                    ${renderInsightCallLogList(callLogs)}
                </div>
            </div>
        `;
    }

    function renderInsightJobInspectorEmpty() {
        const inspector = document.getElementById("insightJobInspector");
        if (!inspector) return;
        inspector.innerHTML = '<div class="queue-inspector-empty">选择一条任务后，这里会显示状态时间线、关联流水、错误信息和结果跳转。</div>';
    }

    function renderInsightJobDetail(job, options = {}) {
        const previousTarget = getCurrentInsightJobDetailTarget();
        if (currentInsightJobDetail && currentInsightJobDetail.id) {
            captureInsightJobDetailViewState(currentInsightJobDetail.id, previousTarget);
        }
        currentInsightJobDetail = job;
        selectedInsightJobID = job.id;
        const target = options.target || (isInsightJobInspectorAvailable() && options.preferModal !== true ? "inspector" : "modal");
        if (target === "modal") {
            const content = document.getElementById("insightJobModalContent");
            const title = document.getElementById("insightJobModalTitle");
            if (!content || !title) return;
            title.textContent = `音眸任务详情 · ${formatInsightJobSubject(job).title}`;
            content.innerHTML = buildInsightJobDetailHtml(job, { includeClose: true });
            showModal("insightJobModal");
            restoreInsightJobDetailViewState(job.id, "modal");
        } else {
            const inspector = document.getElementById("insightJobInspector");
            if (!inspector) return;
            inspector.innerHTML = buildInsightJobDetailHtml(job);
            const modal = document.getElementById("insightJobModal");
            if (modal) modal.style.display = "none";
            restoreInsightJobDetailViewState(job.id, "inspector");
        }
        renderInsightJobList(currentInsightJobRows);
        syncInsightJobWorkspaceHeight();
    }

    async function showInsightJobDetails(jobID, options = {}) {
        if (!jobID) return;
        const preferModal = options.preferModal === true || !isInsightJobInspectorAvailable();
        if (preferModal) {
            const content = document.getElementById("insightJobModalContent");
            if (content) {
                content.innerHTML = '<div class="queue-empty">加载中...</div>';
            }
            showModal("insightJobModal");
        }

        try {
            const response = await fetch(`/api/insight-jobs/${encodeURIComponent(jobID)}`);
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }
            const data = await response.json();
            const detailJob = {
                ...(data.job || {}),
                call_logs: Array.isArray(data.call_logs) ? data.call_logs : []
            };
            insightJobListCache[jobID] = detailJob;
            renderInsightJobDetail(detailJob, { target: preferModal ? "modal" : "inspector", preferModal });
        } catch (error) {
            console.error("加载音眸任务详情失败:", error);
            if (preferModal) {
                const content = document.getElementById("insightJobModalContent");
                if (content) {
                    content.innerHTML = `<div class="error">加载失败: ${escapeHtmlText(error.message)}</div>`;
                }
            } else {
                const inspector = document.getElementById("insightJobInspector");
                if (inspector) {
                    inspector.innerHTML = `<div class="queue-inspector-empty">加载失败: ${escapeHtmlText(error.message)}</div>`;
                }
            }
        }
    }

    async function cancelInsightJob(jobID) {
        if (!jobID) return;
        if (!confirm("确定要取消这个音眸任务吗？后台中的已发起调用不会立刻回滚，但任务状态会收敛为已取消。")) {
            return;
        }

        try {
            const response = await fetch(`/api/insight-jobs/${encodeURIComponent(jobID)}/cancel`, { method: "POST" });
            const data = await response.json().catch(() => ({}));
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }
            insightJobListCache[jobID] = data.job;
            if (selectedInsightJobID === jobID || document.getElementById("insightJobModal").style.display === "block") {
                renderInsightJobDetail(data.job, { target: isInsightJobInspectorAvailable() ? "inspector" : "modal" });
            }
            scheduleInsightJobListRefresh();
        } catch (error) {
            console.error("取消音眸任务失败:", error);
            alert(`取消失败: ${error.message}`);
        }
    }

    async function deleteInsightJob(jobID) {
        if (!jobID) return;
        if (!confirm("确定要删除这条失败或已取消的任务吗？这会同时清理该任务关联的调用流水，且无法恢复。")) {
            return;
        }

        try {
            const response = await fetch(`/api/insight-jobs/${encodeURIComponent(jobID)}`, { method: "DELETE" });
            const data = await response.json().catch(() => ({}));
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }

            delete insightJobListCache[jobID];
            if (selectedInsightJobID === jobID) {
                selectedInsightJobID = "";
                currentInsightJobDetail = null;
                renderInsightJobInspectorEmpty();
            }
            await loadInsightJobList(currentInsightJobPage);
        } catch (error) {
            console.error("删除音眸任务失败:", error);
            alert(`删除失败: ${error.message}`);
        }
    }

    async function retryInsightJob(jobID) {
        if (!jobID) return;
        try {
            const response = await fetch(`/api/insight-jobs/${encodeURIComponent(jobID)}/retry`, { method: "POST" });
            const data = await response.json().catch(() => ({}));
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }
            insightJobListCache[jobID] = data.job;
            if (selectedInsightJobID === jobID || document.getElementById("insightJobModal").style.display === "block") {
                renderInsightJobDetail(data.job, { target: isInsightJobInspectorAvailable() ? "inspector" : "modal" });
            }
            scheduleInsightJobListRefresh();
        } catch (error) {
            console.error("重试音眸任务失败:", error);
            alert(`重试失败: ${error.message}`);
        }
    }

    function openInsightJobResult(jobID) {
        if (!jobID) return;
        const job = currentInsightJobDetail && currentInsightJobDetail.id === jobID
            ? currentInsightJobDetail
            : insightJobListCache[jobID];
        if (!job || !job.result_insight_id) {
            alert("当前任务还没有可查看的结果");
            return;
        }
        hideModal("insightJobModal");
        showInsightDetailsById(job.result_insight_id, job.analysis_target_type || "track");
    }

    function handleInsightJobRealtimeUpdate(job) {
        if (!job) return;
        const mergedJob = mergeInsightJobSnapshot(job);
        insightJobListCache[job.id] = mergedJob;
        highlightedInsightJobID = job.id;
        const index = currentInsightJobRows.findIndex((item) => item.id === job.id);
        if (index >= 0) {
            currentInsightJobRows.splice(index, 1, mergedJob);
            currentInsightJobRows.sort(compareInsightJobs);
            renderInsightJobList(currentInsightJobRows);
        }
        const summaryIndex = currentInsightJobSummaryRows.findIndex((item) => item.id === job.id);
        if (summaryIndex >= 0) {
            currentInsightJobSummaryRows.splice(summaryIndex, 1, mergedJob);
        } else {
            currentInsightJobSummaryRows.unshift(mergedJob);
        }
        currentInsightJobSummaryRows = currentInsightJobSummaryRows
            .slice()
            .sort(compareInsightJobs)
            .slice(0, 200);
        renderInsightJobSummaryBar(currentInsightJobSummaryRows, Math.max(totalInsightJobs, currentInsightJobSummaryRows.length));
        renderInsightJobPriorityStrip(currentInsightJobSummaryRows);
        if (selectedInsightJobID === job.id) {
            renderInsightJobDetail(mergedJob, { target: isInsightJobInspectorAvailable() ? "inspector" : "modal" });
        }
        if (currentSectionID === "insightJobList" && index < 0) {
            scheduleInsightJobListRefresh();
        }
        syncInsightJobWorkspaceHeight();
    }

    function switchInsightListTarget(targetType) {
        if (!targetType || targetType === currentInsightTargetType) return;
        currentInsightTargetType = targetType;
        currentInsightPage = 1;
        updateInsightListTargetTabs();
        updateInsightSearchPlaceholder();
        loadInsightList();
    }

    function updateInsightListTargetTabs() {
        document.querySelectorAll("#insightListTargetTabs .insight-tab").forEach((tab) => {
            const targetType = tab.getAttribute("data-target-type");
            tab.classList.toggle("active", targetType === currentInsightTargetType);
        });
    }

    function updateInsightSearchPlaceholder() {
        const input = document.getElementById("insightSearchInput");
        if (!input) return;
        input.placeholder = currentInsightTargetType === 'album' ? "按专辑模糊搜索..." : "按曲目模糊搜索...";
    }

    // 切换禁用/启用状态
    async function toggleInsightStatus(id, targetType = currentInsightTargetType) {
        try {
            const normalizedTargetType = targetType === 'album' ? 'album' : 'track';
            const res = await fetch(`/api/insights/${id}/toggle-status?analysis_target_type=${encodeURIComponent(normalizedTargetType)}`, { method: 'POST' });
            if (res.ok) loadInsightList();
            else alert("操作失败");
        } catch (e) {
            console.error(e);
            alert("切换状态出错");
        }
    }

    function showAlbumInsightDetailsByAlbumId(albumID) {
        if (!albumID) return;
        hideModal('listInsightModal');
        showAlbumDetails(albumID, 'insight');
    }
    window.showAlbumInsightDetailsByAlbumId = showAlbumInsightDetailsByAlbumId;





    function focusAlbumInsightByID(insightID) {
        albumInsightState.focusInsightID = Number(insightID || 0);
    }

    // 展示详情（从列表点击，使用 list 上下文）
    async function showInsightDetailsById(id, targetType = 'track') {
        const normalizedTargetType = targetType === 'album' ? 'album' : 'track';
        if (normalizedTargetType === 'album') {
            const contextType = 'list';
            try {
                showModal(`${contextType}InsightModal`);
                configureInsightModal(contextType, 'album');
                switchInsightTab('summary', contextType);

                const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
                const footer = document.getElementById(`${contextType}-aiActionFooter`);
                if (streamContent) streamContent.innerHTML = '<div class="loading">加载专辑解析详情中...</div>';
                if (footer) footer.style.display = 'none';

                const response = await fetch(`/api/insights/${id}?analysis_target_type=album`);
                if (!response.ok) {
                    throw new Error("加载专辑解析详情失败");
                }
                const insight = await response.json();

                insightStates[contextType].insight = insight;
                insightStates[contextType].allInsights = [insight];
                insightStates[contextType].trackInfo = {
                    artist: insight.artist || '',
                    album: insight.album || '',
                    title: insight.album || ''
                };

                renderAlbumInsightDetailsInModal(insight, contextType);
            } catch (e) {
                console.error(e);
                alert("加载详情失败");
            }
            return;
        }

        const contextType = 'list';
        try {
            configureInsightModal(contextType, 'track');
            // 首先切换到“解析详情”页签并展示 Loading
            showModal(`${contextType}InsightModal`);
            switchInsightTab('summary', contextType);
            
            const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
            if (streamContent) streamContent.innerHTML = '<div class="loading">加载中...</div>';
            
            const footer = document.getElementById(`${contextType}-aiActionFooter`);
            if (footer) footer.style.display = "none";

            const response = await fetch(`/api/insights/${id}?analysis_target_type=track`);
            if (!response.ok) {
                throw new Error("加载详情失败");
            }
            const insight = await response.json();

            // 更新上下文状态
            insightStates[contextType].insight = insight;
            insightStates[contextType].allInsights = [insight];
            insightStates[contextType].trackInfo = {
                artist: insight.artist,
                album: insight.album,
                title: insight.track
            };
            currentTrackInsight = insight;
            
            // 渲染主要解析内容
            renderInsight(insight, contextType);
            
            // 显示底部操作栏
            if (footer) footer.style.display = "flex";
            
        } catch (e) {
            console.error(e);
            alert("加载详情失败");
        }
    }
    window.showInsightDetailsById = showInsightDetailsById;

    // 删除解析记录
    async function deleteInsight(id, targetType = currentInsightTargetType) {
        const normalizedTargetType = targetType === 'album' ? 'album' : 'track';
        const targetLabel = normalizedTargetType === 'album' ? '专辑' : '曲目';
        const confirmed = confirm(
            `确定要永久删除这条${targetLabel}解析吗？\n\n删除后将同时清空这条解析对应的调用流水，且无法恢复。`
        );
        if (!confirmed) return;
        
        try {
            const res = await fetch(`/api/insights/${id}?analysis_target_type=${encodeURIComponent(normalizedTargetType)}`, { method: 'DELETE' });
            if (res.ok) {
                if (document.getElementById("listInsightModal")?.style.display === "block") {
                    hideModal("listInsightModal");
                }
                loadInsightList();
            } else {
                const data = await res.json();
                alert("删除失败: " + (data.error || "未知错误"));
            }
        } catch (e) {
            console.error(e);
            alert("删除记录出错");
        }
    }

    // 展示流水日志
    async function showInsightCallLogs(id, artist, track, targetType = 'track') {
        const modal = document.getElementById("callLogModal");
        const content = document.getElementById("callLogContent");
        document.getElementById("callLogTitle").textContent = `流水日志: ${artist} - ${track}`;
        
        showModal("callLogModal");
        content.innerHTML = '<div class="loading">加载中...</div>';
        
        try {
            const normalizedTargetType = targetType === 'album' ? 'album' : 'track';
            const res = await fetch(`/api/insights/${id}/logs?analysis_target_type=${encodeURIComponent(normalizedTargetType)}`);
            const data = await res.json();
            content.innerHTML = renderInsightCallLogList(data.logs);
        } catch (e) {
            console.error(e);
            content.innerHTML = `<div class="error">加载流水日志失败</div>`;
        }
    }

    async function showAlbumInsightCallLogs(albumID, artist, album) {
        const content = document.getElementById("callLogContent");
        document.getElementById("callLogTitle").textContent = `流水日志: ${artist} - ${album}`;

        showModal("callLogModal");
        content.innerHTML = '<div class="loading">加载中...</div>';

        try {
            const res = await fetch(`/api/album-insights/${albumID}/logs`);
            const data = await res.json();
            content.innerHTML = renderInsightCallLogList(data.logs);
        } catch (e) {
            console.error(e);
            content.innerHTML = `<div class="error">加载流水日志失败</div>`;
        }
    }

    // 定时更新函数
