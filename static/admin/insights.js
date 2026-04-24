async function handleAiInsightClick() {
        // 始终尝试 fetchTrackInsight(false) 即查询模式，默认上下文为 'nowPlaying'
        fetchTrackInsight(false, null, 'nowPlaying');
    }

    // 2. 显示模型选择弹窗
    let pendingAnalysisContext = 'nowPlaying'; // 暂存当前的上下文类型

    // 2. 显示模型选择弹窗
    function showModelPicker(targetTrack = null, contextType = 'nowPlaying') {
        // 如果传入了 targetTrack（比如从详情页传来），则暂存它
        pendingAnalysisMode = 'track';
        pendingAnalysisAlbumID = 0;
        pendingAnalysisTrack = targetTrack || currentTrackInfo;
        pendingAnalysisContext = contextType;
        showModal("aiModelPickerModal");
    }

    function showAlbumInsightModelPicker() {
        if (!window.currentAlbumID) {
            alert("当前没有可分析的专辑");
            return;
        }
        pendingAnalysisMode = 'album';
        pendingAnalysisAlbumID = window.currentAlbumID;
        pendingAnalysisTrack = null;
        pendingAnalysisContext = 'album';
        showModal("aiModelPickerModal");
    }

    let availableAIPlatforms = [];
    let aiModelsByPlatform = {};

    function getSelectedAIRequest() {
        const providerSelect = document.getElementById('aiProviderSelect');
        const modelSelect = document.getElementById('aiModelSelect');
        return {
            provider: providerSelect ? providerSelect.value : '',
            model: modelSelect ? modelSelect.value : ''
        };
    }

    async function loadModelsForPlatform(platformId, preferredModel = '') {
        const select = document.getElementById('aiModelSelect');
        if (!select || !platformId) {
            return;
        }

        if (!aiModelsByPlatform[platformId]) {
            const resp = await fetch(`/api/ai-models/${encodeURIComponent(platformId)}/models`);
            if (!resp.ok) {
                throw new Error('加载模型目录失败');
            }
            const data = await resp.json();
            aiModelsByPlatform[platformId] = Array.isArray(data.models) ? data.models : [];
        }

        const models = aiModelsByPlatform[platformId];
        select.innerHTML = '';
        models.forEach(item => {
            const opt = document.createElement('option');
            opt.value = item.id;
            opt.textContent = item.display_name || item.id;
            if ((preferredModel && item.id === preferredModel) || (!preferredModel && item.is_default)) {
                opt.selected = true;
            }
            select.appendChild(opt);
        });

        if (!select.value && models.length > 0) {
            select.value = models[0].id;
        }
    }

    // 3. 用户选择模型后点击“开始分析”
    function startAnalysisWithSelectedModel() {
        console.log("startAnalysisWithSelectedModel triggered");
        hideModal("aiModelPickerModal");
        if (pendingAnalysisMode === 'album') {
            fetchAlbumInsight(true, pendingAnalysisAlbumID);
            return;
        }
        // 使用暂存的曲目信息和上下文开启强制分析流
        console.log("Calling fetchTrackInsight(true, ...)");
        fetchTrackInsight(true, pendingAnalysisTrack, pendingAnalysisContext);
    }

    // 4. 发送对 AI 歌词解析的点赞 / 点踩反馈
    function recordAiFeedback(score, contextType = 'nowPlaying') {
        const state = insightStates[contextType];
        const insight = state.insight || currentTrackInsight;

        console.log("recordAiFeedback called with score:", score, "insight:", insight, "context:", contextType);
        if (!insight) {
            alert("暂无分析结果");
            return;
        }

        const insightId = currentTrackInsight.id || currentTrackInsight.ID;
        if (!insightId) {
            console.warn("Insight object missing ID property:", currentTrackInsight);
            alert("未找到解析 ID，无法提交反馈");
            return;
        }

        let comment = "";
        if (score === -1) {
            console.log("Triggering prompt for dislike comment");
            comment = prompt("我们很抱歉这次分析没能让你满意，请提供你的改进建议，以便我们优化：");
            console.log("User comment prompt result:", comment);
            if (comment === null) return; // 用户取消了输入
        }

        fetch(`/api/track-insight/${insightId}/feedback`, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify({score: score, comment: comment}),
        })
            .then(r => r.json())
            .then(data => {
                if (data.status === "ok") {
                    alert(score === 1 ? "感谢你的点赞！" : "感谢反馈，我们会持续改进。");
                } else {
                    console.error("反馈失败:", data);
                    alert("反馈失败: " + (data.error || "未知错误"));
                }
            })
            .catch(error => {
                console.error("发送反馈失败:", error);
                alert("发送反馈失败: " + error.message);
            });
    }

    async function shareInsight(contextType = 'nowPlaying') {
        const state = insightStates[contextType];
        const insight = state.insight || currentTrackInsight;

        if (!insight) {
            alert("暂无分析结果可分享");
            return;
        }

        const trackInfo = state.trackInfo || currentTrackInfo || pendingAnalysisTrack;
        if (!trackInfo) {
            alert("无法获取曲目信息");
            return;
        }

        const shareBtn = document.getElementById(`${contextType}-aiShareBtn`);
        const originalText = shareBtn ? shareBtn.textContent : "分享图片";
        if (shareBtn) {
            shareBtn.textContent = "生成中...";
            shareBtn.disabled = true;
        }

        try {
            const container = document.getElementById("shareInsightContainer");
            const titleEl = document.getElementById("shareTrackTitle");
            const metaEl = document.getElementById("shareTrackMeta");
            const contentEl = document.getElementById("shareInsightContent");
            const modelEl = document.getElementById("shareModelName");

            titleEl.textContent = trackInfo.title || trackInfo.track || "未知曲目";
            metaEl.textContent = `${trackInfo.artist || "未知艺术家"} · ${trackInfo.album || "未知专辑"}`;

            const modelName = insight.llm_provider || "AI";
            modelEl.textContent = modelName.charAt(0).toUpperCase() + modelName.slice(1);

            contentEl.innerHTML = generateShareInsightHtml(insight);

            container.style.position = "fixed";
            container.style.left = "0";
            container.style.top = "0";
            container.style.zIndex = "-1";

            await new Promise(resolve => setTimeout(resolve, 100));

            // 临时禁用不支持 oklch 的外部 CSS (Tailwind/DaisyUI)
            // 分享容器使用全内联样式，不依赖外部 CSS
            const links = document.querySelectorAll('link[rel="stylesheet"]');
            const targetLinks = Array.from(links).filter(l => l.href.includes('full.min.css'));
            targetLinks.forEach(l => l.disabled = true);

            try {
                const canvas = await html2canvas(container, {
                    scale: 3,
                    useCORS: true,
                    backgroundColor: null,
                    logging: false
                });

                container.style.position = "absolute";
                container.style.left = "-9999px";
                container.style.top = "-9999px";

                canvas.toBlob(async (blob) => {
                    if (!blob) {
                        alert("生成图片失败");
                        return;
                    }
                    try {
                        await navigator.clipboard.write([
                            new ClipboardItem({"image/png": blob})
                        ]);
                        alert("图片已复制到剪贴板！");
                    } catch (err) {
                        console.error("复制失败:", err);
                        const link = document.createElement("a");
                        link.download = `insight-${trackInfo.artist}-${trackInfo.title}.png`;
                        link.href = URL.createObjectURL(blob);
                        link.click();
                        URL.revokeObjectURL(link.href);
                    }
                }, "image/png");
            } finally {
                // 恢复样式
                targetLinks.forEach(l => l.disabled = false);
            }

        } catch (error) {
            console.error("分享失败:", error);
            alert("生成分享图片失败: " + error.message);
        } finally {
            if (shareBtn) {
                shareBtn.textContent = originalText;
                shareBtn.disabled = false;
            }
        }
    }

    let detailInsights = [];

    async function shareDetailInsight(index) {
        if (!window.currentDetailData || !window.currentDetailData.track) {
            alert("无法获取曲目信息");
            return;
        }

        const insights = window.currentDetailData.insights || [];
        const insight = insights[index];
        if (!insight) {
            alert("暂无分析结果可分享");
            return;
        }

        const track = window.currentDetailData.track;
        const trackInfo = {
            artist: track.artist,
            album: track.album,
            title: track.track
        };

        try {
            const container = document.getElementById("shareInsightContainer");
            const titleEl = document.getElementById("shareTrackTitle");
            const metaEl = document.getElementById("shareTrackMeta");
            const contentEl = document.getElementById("shareInsightContent");
            const modelEl = document.getElementById("shareModelName");

            titleEl.textContent = trackInfo.title || "未知曲目";
            metaEl.textContent = `${trackInfo.artist || "未知艺术家"} · ${trackInfo.album || "未知专辑"}`;

            const modelName = insight.llm_provider || "AI";
            modelEl.textContent = modelName.charAt(0).toUpperCase() + modelName.slice(1);

            contentEl.innerHTML = generateShareInsightHtml(insight);

            container.style.position = "fixed";
            container.style.left = "0";
            container.style.top = "0";
            container.style.zIndex = "-1";

            await new Promise(resolve => setTimeout(resolve, 100));

            // 临时禁用不支持 oklch 的外部 CSS (Tailwind/DaisyUI)
            const links = document.querySelectorAll('link[rel="stylesheet"]');
            const targetLinks = Array.from(links).filter(l => l.href.includes('full.min.css'));
            targetLinks.forEach(l => l.disabled = true);

            try {
                const canvas = await html2canvas(container, {
                    scale: 3,
                    useCORS: true,
                    backgroundColor: null,
                    logging: false
                });

                container.style.position = "absolute";
                container.style.left = "-9999px";
                container.style.top = "-9999px";

                canvas.toBlob(async (blob) => {
                    if (!blob) {
                        alert("生成图片失败");
                        return;
                    }
                    try {
                        await navigator.clipboard.write([
                            new ClipboardItem({"image/png": blob})
                        ]);
                        alert("图片已复制到剪贴板！");
                    } catch (err) {
                        console.error("复制失败:", err);
                        const link = document.createElement("a");
                        link.download = `insight-${trackInfo.artist}-${trackInfo.title}.png`;
                        link.href = URL.createObjectURL(blob);
                        link.click();
                        URL.revokeObjectURL(link.href);
                    }
                }, "image/png");
            } finally {
                // 恢复样式
                targetLinks.forEach(l => l.disabled = false);
            }

        } catch (error) {
            console.error("分享失败:", error);
            alert("生成分享图片失败: " + error.message);
        }
    }

    function parseTaggedContent(text, tagName) {
        const regex = new RegExp(`<${tagName}>([\\s\\S]*?)<(?:\\/)?${tagName}>`, 'g');
        const results = [];
        let match;
        while ((match = regex.exec(text)) !== null) {
            results.push(match[1].trim());
        }
        return results;
    }

    function parseLyricsTranslation(text) {
        if (!text) return [];
        const cleanText = text.replace(/\\n/g, '\n');
        const originals = parseTaggedContent(cleanText, 'original');
        const translations = parseTaggedContent(cleanText, 'translation');
        const result = [];
        const maxLen = Math.max(originals.length, translations.length);
        for (let i = 0; i < maxLen; i++) {
            result.push({
                original: originals[i] || '',
                translation: translations[i] || ''
            });
        }
        return result;
    }

    function parseAppreciateAnalysis(text) {
        if (!text) return [];
        const cleanText = text.replace(/\\n/g, '\n');

        const result = [];
        let currentGroup = { sectionTitle: '', items: [], explain: '' };

        const regex = /<(original|translation|explain)>([\s\S]*?)<(?:\/)?\1>/g;
        let match;
        let lastIndex = 0;

        while ((match = regex.exec(cleanText)) !== null) {
            const tag = match[0];
            const tagName = match[1];
            const content = match[2].trim();

            const textBeforeTag = cleanText.slice(lastIndex, match.index).trim();
            if (textBeforeTag && textBeforeTag.includes('段')) {
                currentGroup.sectionTitle = textBeforeTag.split('\n')[0].trim();
            }

            if (tagName === 'original') {
                currentGroup.items.push({original: content, translation: ''});
            } else if (tagName === 'translation') {
                if (currentGroup.items.length > 0) {
                    currentGroup.items[currentGroup.items.length - 1].translation = content;
                }
            } else if (tagName === 'explain') {
                currentGroup.explain = content;
                if (currentGroup.items.length > 0 || currentGroup.explain) {
                    result.push(currentGroup);
                }
                currentGroup = { sectionTitle: '', items: [], explain: '' };
            }
            lastIndex = match.index + tag.length;
        }

        if (currentGroup.items.length > 0 || currentGroup.explain) {
            const textAfter = cleanText.slice(lastIndex).trim();
            if (textAfter && textAfter.includes('段')) {
                currentGroup.sectionTitle = textAfter.split('\n')[0].trim();
            }
            result.push(currentGroup);
        }

        return result;
    }

    function generateShareInsightHtml(insight) {
        let html = '';

        if (insight.lyrics_translation) {
            const lyrics = parseLyricsTranslation(insight.lyrics_translation);
            if (lyrics.length > 0) {
                html += `<div style="margin-bottom: 20px;">
              <h3 style="font-size: 13px; font-weight: 600; margin: 0 0 10px 0; opacity: 0.9;">📝 歌词翻译</h3>
              <div style="font-size: 12px; line-height: 1.8;">`;
                lyrics.forEach(item => {
                    html += `<div style="margin-bottom: 8px;">
                <div style="opacity: 0.7; font-style: italic; white-space: pre-wrap;">${item.original}</div>
                ${item.translation ? `<div style="font-weight: 500; white-space: pre-wrap;">${item.translation}</div>` : ''}
              </div>`;
                });
                html += `</div></div>`;
            }
        }

        if (insight.analysis_summary) {
            const summary = insight.analysis_summary.replace(/\\n/g, '\n');
            html += `<div style="margin-bottom: 20px;">
            <h3 style="font-size: 13px; font-weight: 600; margin: 0 0 10px 0; opacity: 0.9;">💡 曲目解读</h3>
            <div style="font-size: 12px; line-height: 1.8; white-space: pre-wrap;">${summary}</div>
          </div>`;
        }

        try {
            let sections = insight.analysis_by_section;
            if (typeof sections === 'string' && sections.trim() !== '') {
                try {
                    sections = JSON.parse(sections);
                } catch (e) {
                    sections = null;
                }
            }
            if (sections && Object.keys(sections).length > 0) {
                const sectionTitles = {
                    "literary_analysis": "文学解读",
                    "musical_analysis": "乐评分析",
                    "cultural_context": "文化背景",
                    "translation_notes": "翻译说明",
                    "appreciate_analysis": "分句赏析"
                };
                html += `<div style="margin-bottom: 16px;">
              <h3 style="font-size: 13px; font-weight: 600; margin: 0 0 10px 0; opacity: 0.9;">🧩 深度解析</h3>`;
                Object.entries(sections).forEach(([section, content]) => {
                    const title = sectionTitles[section] || section;
                    const contentStr = typeof content === 'string' ? content.replace(/\\n/g, '\n') : JSON.stringify(content);

                    if (section === 'appreciate_analysis') {
                        const parsed = parseAppreciateAnalysis(contentStr);
                        if (parsed.length > 0) {
                            html += `<div style="margin-bottom: 12px; padding-left: 10px; border-left: 2px solid rgba(255,255,255,0.4);">
                        <h4 style="font-size: 12px; font-weight: 600; margin: 0 0 8px 0;">${title}</h4>`;
                            parsed.forEach((group, idx) => {
                                if (group.sectionTitle) {
                                    html += `<div style="margin: 10px 0 6px 0; font-size: 11px; font-weight: 600; color: #e2e8f0; opacity: 0.9;">${group.sectionTitle}</div>`;
                                }
                                group.items.forEach(item => {
                                    html += `<div style="margin-bottom: 4px; font-size: 11px;">
                                <div style="opacity: 0.7; font-style: italic; white-space: pre-wrap;">${item.original}</div>
                                ${item.translation ? `<div style="font-weight: 500; white-space: pre-wrap;">${item.translation}</div>` : ''}
                              </div>`;
                                });
                                if (group.explain) {
                                    html += `<div style="margin: 6px 0 12px 0; padding: 8px 10px; background: rgba(255,255,255,0.1); border-radius: 6px; font-size: 11px; line-height: 1.7; font-weight: 500;">${group.explain}</div>`;
                                }
                            });
                            html += `</div>`;
                        }
                    } else {
                        html += `<div style="margin-bottom: 12px; padding-left: 10px; border-left: 2px solid rgba(255,255,255,0.4);">
                    <h4 style="font-size: 12px; font-weight: 600; margin: 0 0 6px 0;">${title}</h4>
                    <div style="font-size: 11px; line-height: 1.7; white-space: pre-wrap; opacity: 0.9;">${contentStr}</div>
                  </div>`;
                    }
                });
                html += `</div>`;
            }
        } catch (err) {
            console.error("渲染分段解析时出错:", err);
        }

        if (insight.background_info && insight.background_info.length > 5) {
            html += `<div style="margin-bottom: 20px;">
            <h3 style="font-size: 13px; font-weight: 600; margin: 0 0 10px 0; opacity: 0.9;">🎨 创作背景</h3>
            <div style="font-size: 12px; line-height: 1.8; white-space: pre-wrap;">${insight.background_info.replace(/\\n/g, '\n')}</div>
          </div>`;
        }

        if (insight.era_context) {
            html += `<div style="margin-bottom: 16px;">
            <h3 style="font-size: 13px; font-weight: 600; margin: 0 0 10px 0; opacity: 0.9;">🌍 时代语境</h3>
            <div style="font-size: 12px; line-height: 1.8; white-space: pre-wrap;">${insight.era_context.replace(/\\n/g, '\n')}</div>
          </div>`;
        }

        if (!html) {
            html = `<div style="text-align: center; opacity: 0.7; padding: 30px;">
            <div style="font-size: 32px; margin-bottom: 10px;">🎵</div>
            <div style="font-size: 13px;">暂无详细分析内容</div>
          </div>`;
        }

        return html;
    }

    function getRandomLoadingHtml() {
        const texts = [
            "正在为您捕捉旋律中的灵魂...",
            "正在为您解读词句间的深意...",
            "正在为您连接音乐与情感的彼岸...",
            "正在为您寻觅旋律背后的故事...",
            "正在为您雕琢每一句诗词意境..."
        ];
        const text = "音眸" + texts[Math.floor(Math.random() * texts.length)];
        return `
          <div class="ai-loading-wrapper">
            <div class="ai-ripple"><div></div><div></div></div>
            <div class="ai-loading-text">${text}</div>
          </div>
        `;
    }

    // 5. 核心分析驱动函数
    async function fetchTrackInsight(force = false, targetTrack = null, contextType = 'nowPlaying') {
        // 这里的逻辑：传入了 targetTrack 就用传入的，否则用全局 currentTrackInfo
        const trackInfo = targetTrack || currentTrackInfo;
        if (!trackInfo) {
            alert("无法获取曲目元数据，请确保页面已加载曲目信息");
            return;
        }

        const state = insightStates[contextType];
        state.trackInfo = trackInfo;

        const aiButton = document.getElementById("aiInsightButton");
        const sseEnabled = document.getElementById("aiModelSseToggle").checked;
        const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
        const actionFooter = document.getElementById(`${contextType}-aiActionFooter`);
        const reanalyzeBtn = document.getElementById(`${contextType}-aiReanalyzeBtn`);

        // 基础参数
        const artist = trackInfo.artist;
        const album = trackInfo.album;
        const track = trackInfo.title || trackInfo.track;
        const trackNumber = trackInfo.track_number || trackInfo.trackNumber || 0;
        const discNumber = trackInfo.disc_number || trackInfo.discNumber || 0;

        // 如果是强制分析，必须有已选模型
        let provider = "";
        let modelName = "";
        if (force) {
            const selection = getSelectedAIRequest();
            provider = selection.provider;
            modelName = selection.model;
            console.log("Force analysis started. Provider:", provider, "Model:", modelName, "SSE:", sseEnabled);
            if (aiButton) aiButton.disabled = true;
            if (streamContent) streamContent.innerHTML = getRandomLoadingHtml();
            if (actionFooter) actionFooter.style.display = "none";
            
            console.log("Showing modal immediately:", `${contextType}InsightModal`);
            showModal(`${contextType}InsightModal`);
        } else {
            // 仅查询模式
            if (aiButton) aiButton.disabled = true;
        }

        if (force && sseEnabled) {
            // --- SSE 流式模式 ---
            // 如果已有连接先关闭
            if (state.eventSource) state.eventSource.close();
            
            const url = `/api/track-insight-stream?artist=${encodeURIComponent(artist)}&album=${encodeURIComponent(album)}&track=${encodeURIComponent(track)}&trackNumber=${encodeURIComponent(trackNumber)}&discNumber=${encodeURIComponent(discNumber)}&force=true&provider=${encodeURIComponent(provider)}&model=${encodeURIComponent(modelName)}`;
            const eventSource = new EventSource(url);
            state.eventSource = eventSource;
            let fullText = "";

            eventSource.onmessage = function (event) {
                if (streamContent) {
                    if (streamContent.querySelector(".ai-loading-wrapper")) streamContent.innerHTML = "";
                    fullText += event.data;
                    streamContent.textContent = fullText;
                    streamContent.scrollTop = streamContent.scrollHeight;
                }
            };

            eventSource.onerror = function (err) {
                eventSource.close();
                state.eventSource = null;
                if (aiButton) aiButton.disabled = false;
                if (fullText.length > 0) {
                    // 分析完成，执行非强制查询以刷新结果展示排版
                    fetchTrackInsight(false, trackInfo, contextType);
                } else {
                    // 分析失败
                    streamContent.innerHTML = '<div class="error">对话意外中断或该模型暂不支持流式返回。</div>';
                    actionFooter.style.display = "flex";
                    reanalyzeBtn.textContent = "开始分析";
                }
            };
        } else if (force) {
            // --- 强制执行分析模式 (POST) ---
            console.log("Starting POST analysis fetch...");
            const body = {artist, album, track, track_number: trackNumber, disc_number: discNumber, provider, model: modelName};
            fetch("/api/track-insight", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify(body)
            }).then(response => {
                console.log("POST analysis response received. Status:", response.status);
                const aiButton = document.getElementById("aiInsightButton");
                if (aiButton) aiButton.disabled = false;
                return response.json().then(data => {
                    if (!response.ok) throw new Error(data.error || "分析失败");
                    const insights = data.insights || (data.insight ? [data.insight] : []);
                    const recommendedInsightID = Number(data.recommended_insight_id || data.recommendedInsightID || 0);
                    console.log("POST analysis logic success, calling handleInsightResponse");
                    handleInsightResponse(insights, true, contextType, recommendedInsightID);
                });
            }).catch(err => {
                console.error("POST analysis fetch error:", err);
                handleInsightError(err, true, contextType);
            });
        } else {
            // --- 纯查询模式 (GET) ---
            try {
                const url = `/api/track-insight?artist=${encodeURIComponent(artist)}&album=${encodeURIComponent(album)}&track=${encodeURIComponent(track)}&trackNumber=${encodeURIComponent(trackNumber)}&discNumber=${encodeURIComponent(discNumber)}`;
                const response = await fetch(url);
                if (aiButton) aiButton.disabled = false;
                const data = await response.json();

                const insights = data.insights || [];
                const recommendedInsightID = Number(data.recommended_insight_id || data.recommendedInsightID || 0);
                if (insights.length > 0) {
                    handleInsightResponse(insights, false, contextType, recommendedInsightID);
                } else {
                    handleNoInsight(contextType);
                }
            } catch (err) {
                handleInsightError(err, false, contextType);
            }
        }
    }

    // 辅助：处理解析成功响应
    function handleInsightResponse(insights, isFromForce, contextType, recommendedInsightID = 0) {
        const aiButton = document.getElementById("aiInsightButton");
        const actionFooter = document.getElementById(`${contextType}-aiActionFooter`);
        const reanalyzeBtn = document.getElementById(`${contextType}-aiReanalyzeBtn`);

        configureInsightModal(contextType, 'track');
        const recommendedInsight = recommendedInsightID > 0
            ? insights.find((item) => Number(item.id || 0) === recommendedInsightID)
            : null;
        insightStates[contextType].insight = recommendedInsight || insights[0];
        insightStates[contextType].allInsights = insights;
        currentTrackInsight = insightStates[contextType].insight || insights[0]; // 保持兼容

        renderInsight(insights, contextType);
        
        if (actionFooter) actionFooter.style.display = "flex";
        if (reanalyzeBtn) reanalyzeBtn.textContent = "重新分析";
        
        console.log("handleInsightResponse: ensuring modal is visible", `${contextType}InsightModal`);
        showModal(`${contextType}InsightModal`);
    }

    // 辅助：处理无解析数据
    function handleNoInsight(contextType) {
        const aiButton = document.getElementById("aiInsightButton");
        const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
        const actionFooter = document.getElementById(`${contextType}-aiActionFooter`);
        const reanalyzeBtn = document.getElementById(`${contextType}-aiReanalyzeBtn`);

        configureInsightModal(contextType, 'track');
        if (streamContent) renderAdminEmpty(streamContent, "该曲目暂无分析数据，请点击下方按钮开始分析。");
        if (actionFooter) actionFooter.style.display = "flex";
        if (reanalyzeBtn) reanalyzeBtn.textContent = "开始分析";
        showModal(`${contextType}InsightModal`);
    }

    // 辅助：处理错误
    function handleInsightError(err, isFromForce, contextType) {
        const aiButton = document.getElementById("aiInsightButton");
        const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
        const actionFooter = document.getElementById(`${contextType}-aiActionFooter`);

        configureInsightModal(contextType, 'track');
        console.error("AI Insight Error:", err);
        if (aiButton) aiButton.disabled = false;
        if (streamContent) streamContent.innerHTML = `<div class="error">${err.message}</div>`;
        if (actionFooter) actionFooter.style.display = "flex";
        
        console.log("handleInsightError: ensuring modal is visible", `${contextType}InsightModal`);
        showModal(`${contextType}InsightModal`);
    }

    // 6. 初始化 AI 模型
    async function initAIModels() {
        try {
            const resp = await fetch("/api/ai-models");
            if (!resp.ok) return;
            const data = await resp.json();
            const providerSelect = document.getElementById("aiProviderSelect");
            const platforms = Array.isArray(data.platforms) ? data.platforms : [];
            if (providerSelect && platforms.length > 0) {
                availableAIPlatforms.splice(0, availableAIPlatforms.length, ...platforms);
                providerSelect.innerHTML = "";
                platforms.forEach(item => {
                    const opt = document.createElement("option");
                    opt.value = item.id;
                    opt.textContent = item.display_name || item.id;
                    if (item.id === "ollama") opt.selected = true;
                    providerSelect.appendChild(opt);
                });
                const preferredPlatform = providerSelect.value || platforms[0].id;
                await loadModelsForPlatform(preferredPlatform, platforms.find(item => item.id === preferredPlatform)?.default_model || "");
            }
        } catch (e) {
            console.error("加载模型失败:", e);
        }
    }

    initAIModels();

    let allInsights = [];

    // 渲染结构化解析结果
    function renderInsight(data, contextType = 'nowPlaying') {
        const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
        if (!data || !streamContent) return;

        // 标准化输入为数组
        let insights = [];
        if (Array.isArray(data)) {
            insights = data;
        } else if (data.insights && Array.isArray(data.insights)) {
            insights = data.insights;
        } else {
            insights = [data];
        }

        if (insights.length === 0) {
            streamContent.innerHTML = '<div class="error">暂无分析数据</div>';
            return;
        }

        const state = insightStates[contextType];
        state.allInsights = insights;
        state.insight = insights[0];
        currentTrackInsight = insights[0]; // 兼容

        // 设置为正常布局
        streamContent.style.whiteSpace = "normal";

        let html = '';

        // 如果有多个结果，显示 Tab
        if (insights.length > 1) {
            // 增加安全检查，防止 total_score 为空导致 NaN
            const positiveTotal = insights.reduce((sum, i) => {
                const score = Number(i.total_score || 0);
                return sum + (isNaN(score) ? 0 : Math.max(0, score));
            }, 0);
            
            html += '<div class="insight-tabs sub-tabs" style="margin-bottom: 15px; border-bottom: 1px solid var(--border-color);">';
            insights.forEach((insight, index) => {
                const date = new Date(insight.created_at || Date.now());
                const timeStr = date.toLocaleString('zh-CN', {
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit'
                });
                const activeClass = index === 0 ? 'active' : '';
                const totalScore = insight.total_score || 0;
                let supportRate;
                if (positiveTotal > 0) {
                    supportRate = ((totalScore / positiveTotal) * 100).toFixed(1);
                } else {
                    supportRate = totalScore === 0 ? '0.0' : (totalScore > 0 ? '100.0' : '-100.0');
                }
                html += `<div class="insight-tab ${activeClass}" onclick="switchInsightTrackTab(${index}, '${contextType}')" title="总分: ${totalScore} | 支持率: ${supportRate}%">分析 ${insights.length - index} <small>(${timeStr})</small></div>`;
            });
            html += '</div>';
        }

        html += `<div id="${contextType}-insight-content-container">`;
        insights.forEach((insight, index) => {
            const displayStyle = index === 0 ? 'block' : 'none';
            html += `<div class="insight-item" id="${contextType}-insight-item-${index}" style="display: ${displayStyle};">`;
            html += generateInsightContentHtml(insight);
            html += '</div>';
        });
        html += '</div>';

        streamContent.innerHTML = html;
        streamContent.scrollTop = 0;
    }

    function switchInsightTrackTab(index, contextType) {
        const container = document.getElementById(`${contextType}-aiInsightStreamContent`);
        if (!container) return;
        
        container.querySelectorAll('.insight-tab').forEach((tab, i) => {
            if (i === index) tab.classList.add('active');
            else tab.classList.remove('active');
        });
        container.querySelectorAll('.insight-item').forEach((item, i) => {
            if (i === index) item.style.display = 'block';
            else item.style.display = 'none';
        });
        
        const state = insightStates[contextType];
        if (state.allInsights[index]) {
            state.insight = state.allInsights[index];
            currentTrackInsight = state.insight; // 兼容
        }
    }

    function configureInsightModal(contextType = 'nowPlaying', targetType = 'track') {
        const modal = document.getElementById(`${contextType}InsightModal`);
        if (!modal) return;

        const normalizedTargetType = targetType === 'album' ? 'album' : 'track';
        const state = insightStates[contextType];
        if (state) {
            state.targetType = normalizedTargetType;
        }

        const topTabsContainer = modal.querySelector('.modal-body > .insight-tabs');
        const feedbackTabButton = topTabsContainer
            ? Array.from(topTabsContainer.querySelectorAll('.insight-tab')).find((tab) => tab.textContent.includes('反馈'))
            : null;
        const footer = document.getElementById(`${contextType}-aiActionFooter`);

        if (feedbackTabButton) {
            feedbackTabButton.style.display = normalizedTargetType === 'album' ? 'none' : '';
        }

        if (footer && normalizedTargetType === 'album') {
            footer.style.display = 'none';
        }
    }

    function renderAlbumInsightDetailsInModal(insight, contextType = 'list') {
        const streamContent = document.getElementById(`${contextType}-aiInsightStreamContent`);
        if (!streamContent) return;

        streamContent.innerHTML = generateAlbumInsightContentHtml(insight);
        streamContent.scrollTop = 0;
    }

    function switchInsightTab(tabName, contextType = 'nowPlaying') {
        console.log("switchInsightTab:", tabName, "context:", contextType);
        const modal = document.getElementById(`${contextType}InsightModal`);
        if (!modal) return;
        const state = insightStates[contextType];
        const normalizedTargetType = state && state.targetType === 'album' ? 'album' : 'track';

        if (normalizedTargetType === 'album' && tabName === 'feedback') {
            tabName = 'summary';
        }

        // 仅切换顶层标签页 UI (解析详情/用户反馈/原始数据)
        // 使用针对性的选择器只寻找直接隶属于 modal-body > insight-tabs 的项，避免误伤内容区的小 Tab
        const topTabsContainer = modal.querySelector('.modal-body > .insight-tabs');
        if (topTabsContainer) {
            topTabsContainer.querySelectorAll('.insight-tab').forEach(tab => {
                const label = tab.textContent;
                if (label.includes('详情') && tabName === 'summary') tab.classList.add('active');
                else if (label.includes('反馈') && tabName === 'feedback') tab.classList.add('active');
                else if (label.includes('原始') && tabName === 'raw') tab.classList.add('active');
                else tab.classList.remove('active');
            });
        }

        // 切换内容区域
        const summaryTab = modal.querySelector(`#${contextType}-insightSummaryTab`);
        const feedbackTab = modal.querySelector(`#${contextType}-insightFeedbackTab`);
        const rawTab = modal.querySelector(`#${contextType}-insightRawTab`);

        if (summaryTab) summaryTab.style.display = tabName === 'summary' ? "block" : "none";
        if (feedbackTab) feedbackTab.style.display = tabName === 'feedback' ? "block" : "none";
        if (rawTab) rawTab.style.display = tabName === 'raw' ? "block" : "none";

        if (tabName === 'feedback') {
            loadInsightFeedback(contextType);
        } else if (tabName === 'raw') {
            const content = modal.querySelector(`#${contextType}-insightRawContent pre`);
            if (content && state.insight) {
                content.textContent = JSON.stringify(state.insight, null, 2);
            }
        }
    }

    async function loadInsightFeedback(contextType) {
        const state = insightStates[contextType];
        const insight = state.insight || currentTrackInsight;
        const container = document.getElementById(`${contextType}-insightFeedbackContent`);
        if (!container) return;

        if (!insight) {
            container.innerHTML = '<div class="info">暂无分析结果</div>';
            return;
        }

        const insightId = insight.id || insight.ID;
        try {
            const resp = await fetch(`/api/insights/${insightId}/feedbacks`);
            if (!resp.ok) throw new Error("加载反馈失败");
            const data = await resp.json();
            const feedbacks = data.feedbacks || [];
            
            if (feedbacks.length === 0) {
                container.innerHTML = '<div class="info">暂无用户反馈</div>';
                return;
            }

            let html = '<div class="feedback-list" style="display: flex; flex-direction: column; gap: 12px;">';
            feedbacks.forEach(f => {
                const date = new Date(f.created_at).toLocaleString();
                html += `
                    <div class="feedback-item" style="padding: 12px; background: rgba(255,255,255,0.05); border-radius: 8px; border: 1px solid var(--border-color);">
                        <div style="display: flex; justify-content: space-between; margin-bottom: 8px;">
                            <span style="font-weight: 600; color: ${f.score > 0 ? '#2ecc71' : '#e74c3c'};">${f.score > 0 ? '👍 赞' : '👎 踩'}</span>
                            <span style="font-size: 0.8rem; color: var(--text-secondary);">${date}</span>
                        </div>
                        ${f.comment ? `<div style="font-size: 0.9rem; line-height: 1.5;">${f.comment}</div>` : '<i style="font-size: 0.85rem; color: var(--text-secondary);">无评论</i>'}
                    </div>
                `;
            });
            html += '</div>';
            container.innerHTML = html;
        } catch (e) {
            container.innerHTML = `<div class="error">${e.message}</div>`;
        }
    }

    function generateInsightContentHtml(insight) {
        let html = '<div class="insight-container">';

        const lyrics = parseLyricsTranslation(insight.lyrics_translation);
        if (lyrics.length > 0) {
            html += `
            <div class="insight-section">
              <h3><span>📝</span> 歌词翻译</h3>
              <div class="lyrics-grid">
                ${(() => {
                let lyricsHtml = '';
                lyrics.forEach(item => {
                    lyricsHtml += `
                      <div class="lyrics-line">
                        <div class="lyrics-original">${item.original}</div>
                        ${item.translation ? `<div class="lyrics-translated">${item.translation}</div>` : ''}
                      </div>
                    `;
                });
                return lyricsHtml;
            })()}
              </div>
            </div>
          `;
        }

        if (insight.analysis_summary) {
            html += `
            <div class="insight-section">
              <h3><span>💡</span> 曲目解读</h3>
              <div class="analysis-text" style="white-space: pre-wrap; line-height: 1.8;">${insight.analysis_summary.replace(/\\n/g, '\n')}</div>
            </div>
          `;
        }

        try {
            let sections = insight.analysis_by_section;
            if (typeof sections === 'string' && sections.trim() !== '') {
                try {
                    sections = JSON.parse(sections);
                } catch (e) {
                    sections = null;
                }
            }

            if (sections && Object.keys(sections).length > 0) {
                const sectionTitles = {
                    "literary_analysis": "文学翻译家的深度解读",
                    "musical_analysis": "乐评人的专业评价",
                    "cultural_context": "文化史学家的背景与时代分析",
                    "translation_notes": "翻译难点说明或语言特色分析",
                    "appreciate_analysis": "分句进行赏析和解读"
                };

                let sectionsHtml = '';
                Object.entries(sections).forEach(([section, content]) => {
                    const title = sectionTitles[section] || section;
                    const contentStr = typeof content === 'string' ? content.replace(/\\n/g, '\n') : JSON.stringify(content);

                    if (section === 'appreciate_analysis') {
                        const parsed = parseAppreciateAnalysis(contentStr);
                        if (parsed.length > 0) {
                            let appreciateHtml = '';
                            parsed.forEach((group, idx) => {
                                if (group.sectionTitle) {
                                    appreciateHtml += `
                                        <div class="section-title-block" style="margin: 16px 0 8px 0; padding: 8px 12px; background: rgba(102, 126, 234, 0.08); border-radius: 6px; border-left: 3px solid var(--primary-color);">
                                            <span class="section-title-text" style="font-weight: 600; color: var(--primary-color); font-size: 0.95rem;">${group.sectionTitle}</span>
                                        </div>
                                    `;
                                }
                                group.items.forEach(item => {
                                    appreciateHtml += `
                                        <div class="lyrics-line">
                                            <div class="lyrics-original">${item.original}</div>
                                            ${item.translation ? `<div class="lyrics-translated">${item.translation}</div>` : ''}
                                        </div>
                                    `;
                                });
                                if (group.explain) {
                                    appreciateHtml += `
                                        <div class="explain-block">
                                            <div class="explain-content">${group.explain}</div>
                                        </div>
                                    `;
                                }
                            });
                            sectionsHtml += `
                                <div class="section-item appreciate-item">
                                    <span class="section-name">${title}</span>
                                    <div class="appreciate-content">${appreciateHtml}</div>
                                </div>
                            `;
                        }
                    } else {
                        sectionsHtml += `
                            <div class="section-item">
                                <span class="section-name">${title}</span>
                                <div class="section-content" style="white-space: pre-wrap; line-height: 1.8;">${contentStr}</div>
                            </div>
                        `;
                    }
                });

                html += `
                    <div class="insight-section">
                        <h3><span>🧩</span> 分段解析</h3>
                        <div class="analysis-sections">${sectionsHtml}</div>
                    </div>
                `;
            }
        } catch (err) {
            console.error("渲染分段解析时出错:", err);
        }

        if (insight.background_info && insight.background_info.length > 5) {
            html += `
            <div class="insight-section">
              <h3><span>🎨</span> 创作背景</h3>
              <div class="analysis-text">${insight.background_info.replace(/\\n/g, '\n')}</div>
            </div>
          `;
        }

        if (insight.era_context) {
            html += `
            <div class="insight-section">
              <h3><span>🌍</span> 时代语境</h3>
              <div class="analysis-text">${insight.era_context.replace(/\\n/g, '\n')}</div>
            </div>
          `;
        }

        html += `
            <div style="margin-top: 10px; font-size: 0.75rem; color: var(--text-secondary); text-align: right; opacity: 0.7;">
              由 ${insight.llm_provider || 'AI'} 提供深度分析
            </div>
          </div>
        `;
        return html;
    }

    function escapeHtmlText(value) {
        if (value === null || value === undefined) return "";
        return String(value)
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }

    function prettyJsonLike(value) {
        if (value === null || value === undefined || value === '') {
            return '';
        }
        if (typeof value === 'object') {
            return JSON.stringify(value, null, 2);
        }
        if (typeof value === 'string') {
            try {
                return JSON.stringify(JSON.parse(value), null, 2);
            } catch (e) {
                return value;
            }
        }
        return String(value);
    }

    function parseAlbumInsightMetadata(rawMetadata) {
        if (!rawMetadata) return {};
        if (typeof rawMetadata === 'object') return rawMetadata;
        if (typeof rawMetadata !== 'string') return {};
        try {
            const parsed = JSON.parse(rawMetadata);
            return parsed && typeof parsed === 'object' ? parsed : {};
        } catch (e) {
            return {};
        }
    }

    function parseInsightSections(rawSections) {
        if (!rawSections) return null;
        if (typeof rawSections === 'object') return rawSections;
        if (typeof rawSections !== 'string' || rawSections.trim() === '') return null;
        try {
            const parsed = JSON.parse(rawSections);
            return parsed && typeof parsed === 'object' ? parsed : null;
        } catch (e) {
            return null;
        }
    }

    function formatAlbumInsightSectionTitle(sectionKey) {
        const sectionTitles = {
            album_positioning: "作品定位",
            theme_and_narrative: "主题与叙事",
            literary_analysis: "文学解读",
            musical_analysis: "听感与编排",
            author_motivation: "作者动机",
            philosophical_reflection: "哲学反思",
            key_tracks: "关键曲目",
            listening_guide: "聆听指南",
        };
        if (sectionTitles[sectionKey]) return sectionTitles[sectionKey];
        return String(sectionKey || "")
            .replace(/_/g, " ")
            .replace(/\b\w/g, c => c.toUpperCase());
    }

    function getAlbumInsightSectionDefinitions() {
        return [
            {
                key: 'album_positioning',
                icon: '🧭',
                title: '作品定位',
                description: '专辑在艺术家生涯和作品谱系中的位置',
            },
            {
                key: 'theme_and_narrative',
                icon: '🪡',
                title: '主题与叙事',
                description: '整张专辑的主题母题、叙事线与情绪推进',
            },
            {
                key: 'literary_analysis',
                icon: '✍️',
                title: '文学解读',
                description: '意象、修辞、象征与文本组织层面的文学解读',
            },
            {
                key: 'musical_analysis',
                icon: '🎼',
                title: '听感与编排',
                description: '曲风、编排、听感推进与曲序设计',
            },
            {
                key: 'author_motivation',
                icon: '🎙️',
                title: '作者动机',
                description: '创作者动机、创作处境与表达意图',
            },
            {
                key: 'philosophical_reflection',
                icon: '🪞',
                title: '哲学反思',
                description: '专辑折射出的价值观、存在主题与哲学反思',
            },
            {
                key: 'key_tracks',
                icon: '🎯',
                title: '关键曲目',
                description: '关键曲目及其在整张专辑中的作用',
            },
            {
                key: 'listening_guide',
                icon: '🎧',
                title: '聆听指南',
                description: '欣赏整张专辑的切入角度和收听建议',
            },
        ];
    }

    function formatAlbumInsightMetaValue(value) {
        if (Array.isArray(value)) {
            return value.length > 0 ? value.join(", ") : "-";
        }
        if (value && typeof value === 'object') {
            try {
                return JSON.stringify(value, null, 2);
            } catch (e) {
                return String(value);
            }
        }
        if (value === null || value === undefined || value === "") {
            return "-";
        }
        return String(value);
    }

    function buildAlbumInsightStatusHtml(title, description, actionHtml = "") {
        return `
            <div class="album-insight-status">
                <strong>${title}</strong>
                <div style="max-width: 420px; line-height: 1.7;">${description}</div>
                ${actionHtml ? `<div style="margin-top: 16px;">${actionHtml}</div>` : ""}
            </div>
        `;
    }

    function switchAlbumDetailTab(tabName) {
        currentAlbumDetailTab = tabName;

        const infoTab = document.getElementById('albumDetailTabInfo');
        const insightTab = document.getElementById('albumDetailTabInsight');
        const infoBtn = document.getElementById('albumDetailTabBtnInfo');
        const insightBtn = document.getElementById('albumDetailTabBtnInsight');

        if (infoTab) infoTab.classList.toggle('active', tabName === 'info');
        if (insightTab) insightTab.classList.toggle('active', tabName === 'insight');
        if (infoBtn) infoBtn.classList.toggle('active', tabName === 'info');
        if (insightBtn) insightBtn.classList.toggle('active', tabName === 'insight');

        if (tabName === 'insight') {
            if (albumInsightState.albumID !== window.currentAlbumID && window.currentAlbumID) {
                fetchAlbumInsight(false, window.currentAlbumID);
            } else {
                renderAlbumInsightPanel();
            }
        }
    }

    function generateAlbumInsightContentHtml(insight) {
        const metadata = parseAlbumInsightMetadata(insight.metadata);
        const sections = parseInsightSections(insight.analysis_by_section);
        const sectionDefinitions = getAlbumInsightSectionDefinitions();
        const totalTracks = metadata.total_tracks || (albumInsightState.albumMeta && Array.isArray(albumInsightState.albumMeta.track_album) ? albumInsightState.albumMeta.track_album.length : "-");
        const analyzedTracks = metadata.analyzed_tracks || (Array.isArray(metadata.selected_track_insight_ids) ? metadata.selected_track_insight_ids.length : "-");
        const createdAt = insight.created_at ? new Date(insight.created_at).toLocaleString('zh-CN') : '-';
        const title = insight.album || (albumInsightState.albumMeta && albumInsightState.albumMeta.name) || '当前专辑';
        const artist = insight.artist || (albumInsightState.albumMeta && albumInsightState.albumMeta.artist) || '未知艺术家';

        let html = '<div class="album-insight-shell">';
        html += `
            <div class="album-insight-overview">
                <div class="album-insight-eyebrow">Album Insight</div>
                <h3 class="album-insight-title">${escapeHtmlText(title)}</h3>
                <div class="album-insight-meta-line">${escapeHtmlText(artist)} · ${escapeHtmlText(insight.llm_provider || 'AI')} · 生成于 ${escapeHtmlText(createdAt)}</div>
                <div class="album-insight-kpi-row">
                    <div class="album-insight-kpi">
                        <span class="album-insight-kpi-label">曲目总数</span>
                        <span class="album-insight-kpi-value">${escapeHtmlText(totalTracks)}</span>
                    </div>
                    <div class="album-insight-kpi">
                        <span class="album-insight-kpi-label">纳入分析</span>
                        <span class="album-insight-kpi-value">${escapeHtmlText(analyzedTracks)}</span>
                    </div>
                    <div class="album-insight-kpi">
                        <span class="album-insight-kpi-label">来源模型</span>
                        <span class="album-insight-kpi-value">${escapeHtmlText(insight.llm_provider || 'AI')}</span>
                    </div>
                </div>
            </div>
        `;

        if (sections && Object.keys(sections).length > 0) {
            const consumedKeys = new Set();
            const primarySectionCards = sectionDefinitions.map((sectionDef) => {
                const rawValue = sections[sectionDef.key];
                const content = formatAlbumInsightMetaValue(rawValue);
                const hasContent = content !== "-";
                consumedKeys.add(sectionDef.key);
                return `
                    <div class="album-analysis-card ${hasContent ? '' : 'album-analysis-card--empty'}">
                        <div class="album-analysis-card-header">
                            <h4 class="album-analysis-card-title"><span>${sectionDef.icon}</span> ${escapeHtmlText(sectionDef.title)}</h4>
                            <p class="album-analysis-card-desc">${escapeHtmlText(sectionDef.description)}</p>
                        </div>
                        <div class="album-analysis-card-body">${hasContent ? escapeHtmlText(content).replace(/\n/g, '<br>') : '<span style="color: var(--text-secondary);">当前结果未返回这一分区内容。</span>'}</div>
                    </div>
                `;
            }).join('');

            const extraSectionEntries = Object.entries(sections).filter(([key]) => !consumedKeys.has(key));

            html += `
                <div class="insight-section">
                    <h3><span>🧩</span> 结构化分区</h3>
                    <div class="album-analysis-grid">${primarySectionCards}</div>
                </div>
            `;

            if (extraSectionEntries.length > 0) {
                const extraSectionsHtml = extraSectionEntries.map(([sectionKey, sectionValue]) => `
                    <div class="section-item">
                        <span class="section-name">${escapeHtmlText(formatAlbumInsightSectionTitle(sectionKey))}</span>
                        <div class="section-content">${escapeHtmlText(formatAlbumInsightMetaValue(sectionValue)).replace(/\n/g, '<br>')}</div>
                    </div>
                `).join('');

                html += `
                    <div class="insight-section">
                        <h3><span>➕</span> 扩展分区</h3>
                        <div class="analysis-sections">${extraSectionsHtml}</div>
                    </div>
                `;
            }
        }

        if (insight.analysis_summary) {
            html += `
                <div class="insight-section">
                    <h3><span>💿</span> 专辑总评</h3>
                    <div class="analysis-text">${escapeHtmlText(insight.analysis_summary).replace(/\n/g, '<br>')}</div>
                </div>
            `;
        }

        if (insight.background_info) {
            html += `
                <div class="insight-section">
                    <h3><span>🎨</span> 创作背景</h3>
                    <div class="analysis-text">${escapeHtmlText(insight.background_info).replace(/\n/g, '<br>')}</div>
                </div>
            `;
        }

        if (insight.era_context) {
            html += `
                <div class="insight-section">
                    <h3><span>🌍</span> 时代语境</h3>
                    <div class="analysis-text">${escapeHtmlText(insight.era_context).replace(/\n/g, '<br>')}</div>
                </div>
            `;
        }

        html += '</div>';
        return html;
    }

    function renderAlbumInsightPanel() {
        const container = document.getElementById('albumInsightContent');
        const refreshBtn = document.getElementById('albumInsightRefreshBtn');
        const analyzeBtn = document.getElementById('albumInsightAnalyzeBtn');
        if (!container) return;

        if (refreshBtn) {
            refreshBtn.disabled = !albumInsightState.albumID || albumInsightState.loading || albumInsightState.generating;
            refreshBtn.style.opacity = refreshBtn.disabled ? '0.55' : '1';
            refreshBtn.style.cursor = refreshBtn.disabled ? 'not-allowed' : 'pointer';
        }

        if (analyzeBtn) {
            analyzeBtn.disabled = !albumInsightState.albumID || albumInsightState.generating;
            analyzeBtn.style.opacity = analyzeBtn.disabled ? '0.55' : '1';
            analyzeBtn.style.cursor = analyzeBtn.disabled ? 'not-allowed' : 'pointer';
            if (albumInsightState.generating) {
                analyzeBtn.textContent = '分析中...';
            } else if (albumInsightState.insights.length > 0) {
                analyzeBtn.textContent = '重新分析';
            } else {
                analyzeBtn.textContent = '开始分析';
            }
        }

        if (albumInsightState.loading && albumInsightState.insights.length === 0) {
            container.innerHTML = buildAlbumInsightStatusHtml("正在加载专辑音眸", "正在查询这张专辑已有的聚合分析结果，请稍候。");
            return;
        }

        if (albumInsightState.generating) {
            container.innerHTML = buildAlbumInsightStatusHtml("正在生成专辑音眸", "正在按曲序聚合已有曲目音眸，返回整张专辑的主题、叙事与听感结构。");
            return;
        }

        if (albumInsightState.lastError && albumInsightState.insights.length === 0) {
            container.innerHTML = buildAlbumInsightStatusHtml(
                "暂时无法返回专辑分析",
                escapeHtmlText(albumInsightState.lastError),
                `<button class="time-filter" onclick="showAlbumInsightModelPicker()" style="background: var(--primary-color); color: #fff;">开始分析</button>`
            );
            return;
        }

        if (albumInsightState.insights.length === 0 || !albumInsightState.insight) {
            container.innerHTML = buildAlbumInsightStatusHtml(
                "还没有专辑音眸",
                "这里会返回专辑级 analysis_summary、analysis_by_section、background_info、era_context 和 metadata。生成前请先确保这张专辑至少已有部分曲目音眸。",
                `<button class="time-filter" onclick="showAlbumInsightModelPicker()" style="background: var(--primary-color); color: #fff;">开始分析</button>`
            );
            return;
        }

        let html = '';
        if (albumInsightState.lastError) {
            html += `
                <div style="margin-bottom: 14px; padding: 12px 14px; border-radius: 10px; border: 1px solid rgba(244, 63, 94, 0.35); background: rgba(244, 63, 94, 0.08); color: var(--text-primary); line-height: 1.6;">
                    <strong style="display: block; margin-bottom: 4px;">最近一次生成失败</strong>
                    <span style="color: var(--text-secondary);">${escapeHtmlText(albumInsightState.lastError)}</span>
                </div>
            `;
        }

        if (albumInsightState.insights.length > 1) {
            html += '<div class="insight-tabs">';
            albumInsightState.insights.forEach((insight, index) => {
                const date = new Date(insight.created_at || Date.now());
                const timeStr = date.toLocaleString('zh-CN', {
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit'
                });
                const activeClass = albumInsightState.insight === insight ? 'active' : '';
                html += `<div class="insight-tab ${activeClass}" onclick="switchAlbumInsightHistory(${index})">分析 ${albumInsightState.insights.length - index} <small>(${timeStr})</small></div>`;
            });
            html += '</div>';
        }

        html += `
            <div class="insight-tabs" style="margin-bottom: 16px;">
                <div class="insight-tab ${albumInsightState.view === 'summary' ? 'active' : ''}" onclick="switchAlbumInsightTab('summary')">解析详情</div>
                <div class="insight-tab ${albumInsightState.view === 'raw' ? 'active' : ''}" onclick="switchAlbumInsightTab('raw')">原始数据</div>
            </div>
        `;

        if (albumInsightState.view === 'raw') {
            html += `<pre class="album-insight-raw">${escapeHtmlText(JSON.stringify(albumInsightState.insight, null, 2))}</pre>`;
        } else {
            html += generateAlbumInsightContentHtml(albumInsightState.insight);
        }

        container.innerHTML = html;
        container.scrollTop = 0;
    }

    function switchAlbumInsightHistory(index) {
        if (!albumInsightState.insights[index]) return;
        albumInsightState.insight = albumInsightState.insights[index];
        albumInsightState.focusInsightID = Number(albumInsightState.insight.id || 0);
        renderAlbumInsightPanel();
    }

    function switchAlbumInsightTab(tabName) {
        albumInsightState.view = tabName;
        renderAlbumInsightPanel();
    }

    async function fetchAlbumInsight(force = false, albumID = window.currentAlbumID) {
        if (!albumID) {
            return;
        }

        albumInsightState.albumID = albumID;
        albumInsightState.loading = !force;
        albumInsightState.generating = force;
        albumInsightState.lastError = '';
        if (!albumInsightState.view) {
            albumInsightState.view = 'summary';
        }
        renderAlbumInsightPanel();

        const selection = force ? getSelectedAIRequest() : {provider: '', model: ''};
        const url = force ? '/api/album-insight' : `/api/album-insight?albumID=${encodeURIComponent(albumID)}`;
        const options = force ? {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({album_id: albumID, provider: selection.provider, model: selection.model})
        } : {};

        try {
            const response = await fetch(url, options);
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || '获取专辑分析失败');
            }
            if (albumInsightState.albumID !== albumID) {
                return;
            }

            const insights = data.insights || (data.insight ? [data.insight] : []);
            const recommendedInsightID = Number(data.recommended_insight_id || data.recommendedInsightID || 0);
            albumInsightState.insights = insights;
            const focusInsightID = Number(albumInsightState.focusInsightID || 0);
            const recommendedInsight = recommendedInsightID > 0
                ? insights.find((item) => Number(item.id || 0) === recommendedInsightID)
                : null;
            const focusedInsight = focusInsightID > 0
                ? insights.find((item) => Number(item.id || 0) === focusInsightID)
                : null;
            albumInsightState.insight = focusedInsight || recommendedInsight || insights[0] || null;
            albumInsightState.focusInsightID = Number(albumInsightState.insight ? albumInsightState.insight.id : 0);
            albumInsightState.view = albumInsightState.view || 'summary';
            window.currentAlbumDetailData = window.currentAlbumDetailData || {};
            window.currentAlbumDetailData.insights = insights;
            window.currentAlbumDetailData.recommendedInsightID = recommendedInsightID;
        } catch (error) {
            if (albumInsightState.albumID !== albumID) {
                return;
            }
            albumInsightState.lastError = error.message || '获取专辑分析失败';
        } finally {
            if (albumInsightState.albumID === albumID) {
                albumInsightState.loading = false;
                albumInsightState.generating = false;
                renderAlbumInsightPanel();
            }
        }
    }

    // 点赞按钮点击事件处理
