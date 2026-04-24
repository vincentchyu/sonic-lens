function loadUnscrobbledRecords() {
        const content = document.getElementById("unscrobbledContent");
        renderAdminLoading(content);

        const offset = (currentUnscrobbledPage - 1) * unscrobbledPageSize;

        // 获取未上报记录总数
        fetch("/api/unscrobbled-records/count")
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                totalUnscrobbledRecords = data.count;
                updatePagination();
            })
            .catch((error) => {
                console.error("获取未上报记录总数失败:", error);
            });

        // 获取未上报记录
        fetch(
            `/api/unscrobbled-records?limit=${unscrobbledPageSize}&offset=${offset}`
        )
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                renderUnscrobbledRecords(data);
            })
            .catch((error) => {
                console.error("获取未上报记录失败:", error);
                renderAdminError(content, error);
            });
    }

    // 渲染未上报记录
    function renderUnscrobbledRecords(records) {
        const content = document.getElementById("unscrobbledContent");

        if (records.length === 0) {
            renderAdminEmpty(content, "暂无未上报记录");
            return;
        }

        // 检查是否处于暗色模式
        const isDark = document.body.classList.contains("dark-mode");

        // 根据主题设置颜色
        const headerBgColor = isDark ? "#3a3a3a" : "#f1f3f9";
        const borderColor = isDark ? "#444444" : "#e9ecef";
        const rowBorderColor = isDark ? "#444444" : "#e9ecef";
        const textColor = isDark ? "#f0f0f0" : "#2c3e50";

        let html = `<div style="overflow-x: auto;"><table style="width: 100%; border-collapse: collapse; color: ${textColor};">`;
        html += `<thead><tr style="background-color: ${headerBgColor}; text-align: left;">`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};"></th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">曲目</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">艺术家</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">专辑</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">播放时间</th>`;
        html += `<th style="padding: 12px; border-bottom: 2px solid ${borderColor};">来源</th>`;
        html += "</tr></thead><tbody>";

        records.forEach((record) => {
            const playTime = new Date(record.play_time);
            const timeString = playTime.toLocaleString("zh-CN");

            // 根据来源设置样式
            let sourceClass = "";
            let sourceText = "";
            switch ((record.source || "").toLowerCase()) {
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
                    sourceText = record.source || "Unknown";
            }

            html += `<tr style="border-bottom: 1px solid ${rowBorderColor};">`;
            html += `<td style="padding: 12px;"><input type="checkbox" class="record-checkbox" data-id="${record.id}"></td>`;
            html += `<td style="padding: 12px;">${record.track}</td>`;
            html += `<td style="padding: 12px;">${record.artist}</td>`;
            html += `<td style="padding: 12px;">${record.album}</td>`;
            html += `<td style="padding: 12px;">${timeString}</td>`;
            html += `<td style="padding: 12px;"><div class="play-source ${sourceClass}" style="display: inline-block;">${sourceText}</div></td>`;
            html += "</tr>";
        });

        html += "</tbody></table></div>";
        content.innerHTML = html;
    }

    // 更新分页信息
    function updatePagination() {
        const totalPages = Math.ceil(
            totalUnscrobbledRecords / unscrobbledPageSize
        );
        const pageInfo = document.getElementById("pageInfo");
        const prevBtn = document.getElementById("prevPage");
        const nextBtn = document.getElementById("nextPage");

        // 检查是否处于暗色模式
        const isDark = document.body.classList.contains("dark-mode");

        // 设置分页信息文本颜色
        if (pageInfo) {
            pageInfo.style.color = isDark ? "var(--text-primary)" : "#2c3e50";
        }

        if (totalUnscrobbledRecords === 0) {
            pageInfo.textContent = "无记录";
            prevBtn.disabled = true;
            nextBtn.disabled = true;
            return;
        }

        pageInfo.textContent = `第 ${currentUnscrobbledPage} 页，共 ${totalPages} 页 (${totalUnscrobbledRecords} 条记录)`;
        prevBtn.disabled = currentUnscrobbledPage === 1;
        nextBtn.disabled = currentUnscrobbledPage === totalPages;
    }

    // 同步选中的记录
    function syncSelectedRecords() {
        const checkboxes = document.querySelectorAll(
            ".record-checkbox:checked"
        );
        if (checkboxes.length === 0) {
            alert("请先选择要同步的记录");
            return;
        }

        const ids = Array.from(checkboxes).map((cb) => parseInt(cb.dataset.id));

        // 禁用按钮并显示加载状态
        const syncBtn = document.getElementById("syncSelectedBtn");
        const originalText = syncBtn.textContent;
        syncBtn.textContent = "同步中...";
        syncBtn.disabled = true;

        fetch("/api/unscrobbled-records/sync", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({ids: ids}),
        })
            .then((response) => {
                if (!response.ok) {
                    throw new Error("网络响应错误");
                }
                return response.json();
            })
            .then((data) => {
                if (data.failed_count > 0) {
                    alert(
                        `同步完成: ${data.success_count} 条记录同步成功, ${data.failed_count} 条记录同步失败`
                    );
                    // 显示失败的记录
                    if (data.failed_records && data.failed_records.length > 0) {
                        let failedList = "同步失败的记录:\n";
                        data.failed_records.forEach((record) => {
                            failedList += `- ${record.track} - ${record.artist}\n`;
                        });
                        console.log(failedList);
                    }
                } else {
                    alert(`同步完成: ${data.success_count} 条记录同步成功`);
                }

                // 重新加载未上报记录
                loadUnscrobbledRecords();
            })
            .catch((error) => {
                console.error("同步记录失败:", error);
                alert("同步记录失败: " + error.message);
            })
            .finally(() => {
                // 恢复按钮状态
                syncBtn.textContent = originalText;
                syncBtn.disabled = false;
            });
    }

    // --- 音眸列表相关 JS 逻辑 ---

    // 加载音眸解析列表
