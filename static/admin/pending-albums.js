async function loadPendingAlbumList() {
        if (currentPendingWorkTab === 'pending_groups') {
            const content = document.getElementById('pendingAlbumListContent');
            renderAdminLoading(content);
            try {
                const resp = await fetch('/api/pending-albums?limit=100');
                const data = await resp.json();
                pendingAlbumGroups = data.groups || [];
                renderPendingAlbumList(pendingAlbumGroups);
            } catch (err) {
                renderAdminError(content, err);
            }
        } else {
            loadPendingWorkItems(1);
        }
    }

    function switchPendingWorkTab(tab) {
        currentPendingWorkTab = tab;
        
        // 更新 UI 激活状态
        document.querySelectorAll('#pendingAlbumListContainer .time-filter').forEach(el => {
            if (el.id.startsWith('tab')) {
                el.classList.remove('active');
            }
        });
        
        const tabIdMap = {
            'pending_groups': 'tabPendingGroups',
            'working_items': 'tabWorkingItems',
            'completed_items': 'tabCompletedItems'
        };
        document.getElementById(tabIdMap[tab]).classList.add('active');
        
        // 控制搜索栏和分页容器显示
        const searchContainer = document.getElementById('pendingWorkSearchContainer');
        const paginationContainer = document.getElementById('pendingWorkPagination');
        
        if (tab === 'pending_groups') {
            searchContainer.style.display = 'none';
            paginationContainer.style.display = 'none';
            loadPendingAlbumList();
        } else {
            searchContainer.style.display = 'flex';
            paginationContainer.style.display = 'flex';
            currentPendingWorkPage = 1;
            loadPendingWorkItems(1);
        }
    }

    async function loadPendingWorkItems(page = 1) {
        currentPendingWorkPage = page;
        const content = document.getElementById('pendingAlbumListContent');
        renderAdminLoading(content);
        
        const statusGroup = currentPendingWorkTab === 'working_items' ? 'open' : 'completed';
        const offset = (page - 1) * pendingWorkPageSize;
        
        try {
            const url = `/api/pending-albums/work-items?limit=${pendingWorkPageSize}&offset=${offset}&status_group=${statusGroup}&keyword=${encodeURIComponent(currentPendingWorkKeyword)}`;
            const resp = await fetch(url);
            const data = await resp.json();
            
            totalPendingWorkCount = data.total || 0;
            renderPendingWorkItems(data.items || []);
            updatePendingWorkPagination();
        } catch (err) {
            renderAdminError(content, err);
        }
    }

    function renderPendingWorkItems(items) {
        const content = document.getElementById('pendingAlbumListContent');
        if (!items || items.length === 0) {
            renderAdminEmpty(content, "暂无维护项记录");
            return;
        }

        let html = '<div style="display: flex; flex-direction: column; gap: 14px;">';
        items.forEach(item => {
            const dateStr = new Date(item.created_at).toLocaleString();
            const isCompleted = item.status === 'completed';
            const statusLabel = renderPendingAlbumWorkItemStatusBadge(item.status);
            
            html += `
                <div style="padding: 16px; border: 1px solid var(--border-color); border-radius: 14px; background: var(--bg-card);">
                    <div style="display: flex; justify-content: space-between; gap: 16px; align-items: flex-start;">
                        <div
                            class="artwork-slot"
                            data-artwork-slot="1"
                            data-resolved-album-id="${Number(item.resolved_album_id || 0)}"
                            data-artist="${esc(item.artist || "")}"
                            data-album-artist="${esc(item.album_artist || "")}"
                            data-album="${esc(item.album || "")}"
                            data-artwork-compact="1"
                            data-alt-text="${esc(item.album || "专辑封面")}"
                            style="width: 64px; height: 64px; flex-shrink: 0; border-radius: 12px;"
                        >${renderArtworkPlaceholder(item.album || "", true)}</div>
                        <div style="flex: 1; min-width: 0;">
                            <div style="font-size: 1.05em; font-weight: 700; color: var(--text-primary);">${item.album}</div>
                            <div style="margin-top: 4px; opacity: 0.65; color: var(--text-secondary);">${item.album_artist || item.artist || '-'}</div>
                            <div style="margin-top: 10px; font-size: 0.82em; color: var(--text-secondary);">
                                ID: #${item.id} · 状态: ${statusLabel} · 创建时间: ${dateStr}
                            </div>
                        </div>
                        <div style="display: flex; gap: 8px; flex-shrink: 0;">
                            <button class="time-filter" onclick="resumePendingAlbumWorkItem(${item.id})">查看详情</button>
                            ${isCompleted && item.resolved_album_id > 0 ? 
                                `<button class="time-filter" onclick="showAlbumDetails(${item.resolved_album_id})">查看专辑详情</button>` : 
                                ''
                            }
                        </div>
                    </div>
                </div>
            `;
        });
        html += '</div>';
        content.innerHTML = html;
        hydrateArtworkSlots(content);
    }

    function updatePendingWorkPagination() {
        const totalPages = Math.ceil(totalPendingWorkCount / pendingWorkPageSize);
        document.getElementById('pendingWorkPageInfo').textContent = `第 ${currentPendingWorkPage} 页 / 共 ${totalPages} 页 (总计 ${totalPendingWorkCount})`;
        document.getElementById('pendingWorkPrevPage').disabled = currentPendingWorkPage <= 1;
        document.getElementById('pendingWorkNextPage').disabled = currentPendingWorkPage >= totalPages || totalPages === 0;
    }

    function resumePendingAlbumWorkItem(id) {
        currentPendingAlbumWorkItemID = id;
        currentPendingAlbumStalePromptedWorkItemID = 0;
        showModal('pendingAlbumWorkItemModal');
        loadPendingAlbumWorkItemDetail();
    }

    function formatPendingAlbumWorkItemStatus(status) {
        switch (status) {
            case 'not_created':
                return '尚未建单';
            case 'open':
            case 'mb_selected':
            case 'deep_maintaining':
            case 'applying':
                return '维护中';
            case 'completed':
                return '已完成';
            case 'failed':
                return '失败';
            default:
                return status || '-';
        }
    }

    function renderPendingAlbumWorkItemStatusBadge(status) {
        const statusName = formatPendingAlbumWorkItemStatus(status);
        if (statusName === '尚未建单') {
            return `<span style="display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; background: rgba(127, 140, 141, 0.14); color: #7f8c8d; font-weight: 600; letter-spacing: 0.02em;">${statusName}</span>`;
        }
        if (statusName === '维护中') {
            return `<span style="display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; background: rgba(var(--primary-rgb), 0.16); color: var(--primary-color); font-weight: 700; letter-spacing: 0.02em;">${statusName}</span>`;
        }
        if (statusName === '已完成') {
            return `<span style="display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; background: rgba(39, 174, 96, 0.14); color: #27ae60; font-weight: 700; letter-spacing: 0.02em;">${statusName}</span>`;
        }
        if (statusName === '失败') {
            return `<span style="display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; background: rgba(231, 76, 60, 0.14); color: #e74c3c; font-weight: 700; letter-spacing: 0.02em;">${statusName}</span>`;
        }
        return `<span style="display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; background: rgba(var(--text-secondary-rgb), 0.12); color: var(--text-secondary); font-weight: 600;">${statusName}</span>`;
    }

    function renderPendingAlbumList(groups) {
        const content = document.getElementById('pendingAlbumListContent');
        if (!groups || groups.length === 0) {
            renderAdminEmpty(content, "当前没有待归因专辑");
            return;
        }

        let html = '<div style="display: flex; flex-direction: column; gap: 14px;">';
        groups.forEach(group => {
            const sourceText = (group.sources || []).join(' / ') || '-';
            const trackPreview = (group.track_names || []).slice(0, 4).join('、');
            const workItemText = group.open_work_item_id
                ? `工作项 #${group.open_work_item_id} · ${renderPendingAlbumWorkItemStatusBadge(group.open_work_item_status)}`
                : `${renderPendingAlbumWorkItemStatusBadge('not_created')}`;
            html += `
                <div style="padding: 16px; border: 1px solid var(--border-color); border-radius: 14px; background: var(--bg-card);">
                    <div style="display: flex; justify-content: space-between; gap: 16px; align-items: flex-start;">
                        <div
                            class="artwork-slot"
                            data-artwork-slot="1"
                            data-artist="${esc(group.artist || "")}"
                            data-album-artist="${esc(group.album_artist || "")}"
                            data-album="${esc(group.album || "")}"
                            data-artwork-compact="1"
                            data-alt-text="${esc(group.album || "专辑封面")}"
                            style="width: 64px; height: 64px; flex-shrink: 0; border-radius: 12px;"
                        >${renderArtworkPlaceholder(group.album || "", true)}</div>
                        <div style="flex: 1; min-width: 0;">
                            <div style="font-size: 1.05em; font-weight: 700; color: var(--text-primary);">${group.album}</div>
                            <div style="margin-top: 4px; opacity: 0.65; color: var(--text-secondary);">${group.album_artist || group.artist || '-'}</div>
                            <div style="margin-top: 10px; font-size: 0.82em; color: var(--text-secondary);">
                                待处理播放 ${group.play_record_count} 条 · 点赞事件 ${group.favorite_event_count} 条 · 来源 ${sourceText}
                            </div>
                            <div style="margin-top: 6px; font-size: 0.8em; opacity: 0.6; color: var(--text-secondary);">${workItemText}</div>
                            <div style="margin-top: 8px; font-size: 0.85em; color: var(--text-primary);">${trackPreview || '暂无曲目摘要'}</div>
                        </div>
                        <button class="time-filter" onclick="openPendingAlbumWorkItem('${esc(group.identity_key)}')" style="flex-shrink: 0;">进入维护</button>
                    </div>
                </div>
            `;
        });
        html += '</div>';
        content.innerHTML = html;
        hydrateArtworkSlots(content);
    }

    function normalizePendingManualTitle(value) {
        return String(value || '').trim().toLowerCase().replace(/\s+/g, ' ');
    }

    function uniqPendingStrings(values) {
        const result = [];
        (values || []).forEach(value => {
            const trimmed = String(value || '').trim();
            if (!trimmed || result.includes(trimmed)) {
                return;
            }
            result.push(trimmed);
        });
        return result;
    }

    function renderPendingMaintenanceReport(report) {
        if (!report || !report.resolved_album_id) {
            return '尚未执行';
        }
        const modeText = report.mode === 'manual' ? '手动维护' : 'MusicBrainz 维护';
        return `
            <div style="font-weight: 700; color: var(--text-primary); margin-bottom: 8px;">${modeText}完成</div>
            目标专辑 #${report.resolved_album_id || '-'}<br>
            复用已听曲 ${report.reused_heard_tracks || 0} 首<br>
            新建曲目 ${report.created_tracks || 0} 首<br>
            写入 track_album ${report.track_album_writes || 0} 条<br>
            回填播放流水 ${report.applied_play_records || 0} 条<br>
            回填点赞事件 ${report.applied_favorite_events || 0} 条
        `;
    }

    function collectPendingManualTrackDrafts(detail) {
        const rows = new Map();
        let looseIndex = 0;

        function ensureRow(key, seed) {
            if (!rows.has(key)) {
                rows.set(key, {
                    discNumber: seed.discNumber || 0,
                    trackNumber: seed.trackNumber || 0,
                    title: seed.title || '',
                    artist: seed.artist || '',
                    duration: seed.duration || 0,
                    composer: seed.composer || '',
                    musicBrainzId: seed.musicBrainzId || '',
                    evidenceTitles: [],
                    sourceHints: [],
                    needsPosition: !seed.trackNumber,
                });
            }
            return rows.get(key);
        }

        function absorb(item, sourceType) {
            const title = String(item.track || '').trim();
            const rawTrackNumber = Number(item.track_number || 0);
            const rawDiscNumber = Number(item.disc_number || 0);
            const discNumber = rawTrackNumber > 0 ? (rawDiscNumber || 1) : 0;
            const key = rawTrackNumber > 0
                ? `pos:${discNumber}|${rawTrackNumber}`
                : `loose:${normalizePendingManualTitle(title) || 'untitled'}:${looseIndex++}`;
            const row = ensureRow(key, {
                discNumber,
                trackNumber: rawTrackNumber,
                title,
                artist: item.artist || '',
                duration: Number(item.duration || 0),
                composer: item.composer || '',
                musicBrainzId: item.music_brainz_id || '',
            });
            if (!row.title && title) {
                row.title = title;
            }
            if (!row.artist && item.artist) {
                row.artist = item.artist;
            }
            if (!row.duration && Number(item.duration || 0) > 0) {
                row.duration = Number(item.duration || 0);
            }
            if (!row.musicBrainzId && item.music_brainz_id) {
                row.musicBrainzId = item.music_brainz_id;
            }
            row.needsPosition = row.needsPosition || !(row.trackNumber > 0 && row.discNumber > 0);
            row.evidenceTitles = uniqPendingStrings([...(row.evidenceTitles || []), title]);
            row.sourceHints = uniqPendingStrings([
                ...(row.sourceHints || []),
                `${sourceType} · ${item.source || '-'} · #${item.id || '-'}`
            ]);
        }

        (detail?.play_records || []).forEach(item => absorb(item, '播放流水'));
        (detail?.favorite_events || []).forEach(item => absorb(item, '点赞事件'));

        return Array.from(rows.values()).sort((left, right) => {
            return comparePendingManualTrackDrafts(left, right);
        });
    }

    function comparePendingManualTrackDrafts(left, right) {
        const leftTrack = Number(left.trackNumber || 0);
        const rightTrack = Number(right.trackNumber || 0);
        if (leftTrack > 0 && rightTrack === 0) return -1;
        if (leftTrack === 0 && rightTrack > 0) return 1;
        const leftDisc = Number(left.discNumber || 0);
        const rightDisc = Number(right.discNumber || 0);
        if (leftDisc !== rightDisc) return leftDisc - rightDisc;
        if (leftTrack !== rightTrack) return leftTrack - rightTrack;
        return String(left.title || '').localeCompare(String(right.title || ''));
    }

    function sortPendingManualTrackDrafts() {
        currentPendingManualTrackDrafts.sort(comparePendingManualTrackDrafts);
    }

    function renderPendingManualTrackTable() {
        const container = document.getElementById('pendingManualTrackTable');
        if (!container) return;
        if (!currentPendingManualTrackDrafts.length) {
            renderAdminEmpty(container, "当前没有可编辑曲目，请点击“新增曲目”。");
            return;
        }

        container.innerHTML = currentPendingManualTrackDrafts.map((item, index) => {
            const sourceText = (item.sourceHints || []).join(' ｜ ') || '手动新增';
            const evidenceText = (item.evidenceTitles || []).join('、') || '无';
            return `
                <div style="padding: 12px; border: 1px solid var(--border-color); border-radius: 12px; background: rgba(255,255,255,0.02);">
                    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 10px; margin-bottom: 10px;">
                        <div style="font-weight: 700; color: var(--text-primary);">曲目 ${index + 1}</div>
                        <button class="time-filter" onclick="removePendingManualTrackRow(${index})" style="padding: 5px 10px;">删除</button>
                    </div>
                    <div style="display: grid; grid-template-columns: 80px 80px minmax(0, 1.4fr) minmax(0, 1fr); gap: 8px;">
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">碟号</span>
                            <input class="search-input" type="number" min="1" value="${Number(item.discNumber || 0) || ''}" oninput="updatePendingManualTrackField(${index}, 'discNumber', this.value)">
                        </label>
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">曲序</span>
                            <input class="search-input" type="number" min="1" value="${Number(item.trackNumber || 0) || ''}" oninput="updatePendingManualTrackField(${index}, 'trackNumber', this.value)">
                        </label>
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">曲名</span>
                            <input class="search-input" type="text" value="${esc(item.title || '')}" oninput="updatePendingManualTrackField(${index}, 'title', this.value)">
                        </label>
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">曲目艺人</span>
                            <input class="search-input" type="text" value="${esc(item.artist || '')}" oninput="updatePendingManualTrackField(${index}, 'artist', this.value)">
                        </label>
                    </div>
                    <div style="display: grid; grid-template-columns: minmax(0, 0.8fr) minmax(0, 1fr) minmax(0, 1fr); gap: 8px; margin-top: 8px;">
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">时长(秒)</span>
                            <input class="search-input" type="number" min="0" value="${Number(item.duration || 0) || ''}" oninput="updatePendingManualTrackField(${index}, 'duration', this.value)">
                        </label>
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">Composer</span>
                            <input class="search-input" type="text" value="${esc(item.composer || '')}" oninput="updatePendingManualTrackField(${index}, 'composer', this.value)">
                        </label>
                        <label style="display: flex; flex-direction: column; gap: 6px;">
                            <span style="font-size: 0.75em; color: var(--text-secondary);">MusicBrainz ID</span>
                            <input class="search-input" type="text" value="${esc(item.musicBrainzId || '')}" oninput="updatePendingManualTrackField(${index}, 'musicBrainzId', this.value)">
                        </label>
                    </div>
                    <div style="margin-top: 10px; font-size: 0.75em; line-height: 1.6; color: var(--text-secondary);">
                        默认值来源: ${esc(sourceText)}<br>
                        归因证据标题: ${esc(evidenceText)}
                        ${item.needsPosition ? '<br><span style="color: #e67e22; font-weight: 700;">该行没有可靠位置，请补齐碟号和曲序后再提交。</span>' : ''}
                    </div>
                </div>
            `;
        }).join('');
    }

    function updatePendingManualTrackField(index, field, value) {
        const row = currentPendingManualTrackDrafts[index];
        if (!row) return;
        if (field === 'discNumber' || field === 'trackNumber' || field === 'duration') {
            row[field] = Number(value || 0);
        } else {
            row[field] = String(value || '');
        }
        row.needsPosition = !(Number(row.discNumber || 0) > 0 && Number(row.trackNumber || 0) > 0);
        if (field === 'discNumber' || field === 'trackNumber') {
            sortPendingManualTrackDrafts();
            renderPendingManualTrackTable();
        }
    }

    function addPendingManualTrackRow() {
        currentPendingManualTrackDrafts.push({
            discNumber: 1,
            trackNumber: 0,
            title: '',
            artist: document.getElementById('pendingManualDisplayArtist')?.value || '',
            duration: 0,
            composer: '',
            musicBrainzId: '',
            evidenceTitles: [],
            sourceHints: ['手动新增'],
            needsPosition: true,
        });
        sortPendingManualTrackDrafts();
        renderPendingManualTrackTable();
    }

    function removePendingManualTrackRow(index) {
        currentPendingManualTrackDrafts.splice(index, 1);
        sortPendingManualTrackDrafts();
        renderPendingManualTrackTable();
    }

    function resetPendingManualFormFromContext() {
        const detail = currentPendingAlbumWorkItemDetail || {};
        const workItem = detail.work_item || {};
        document.getElementById('pendingManualAlbumName').value = workItem.album || '';
        document.getElementById('pendingManualAlbumArtist').value = workItem.album_artist || workItem.artist || '';
        document.getElementById('pendingManualDisplayArtist').value = workItem.artist || '';
        document.getElementById('pendingManualReleaseDate').value = '';
        document.getElementById('pendingManualGenre').value = '';
        document.getElementById('pendingManualCountry').value = '';
        document.getElementById('pendingManualStatus').value = '';
        document.getElementById('pendingManualPackaging').value = '';
        document.getElementById('pendingManualBarcode').value = '';
        document.getElementById('pendingManualCoverArtURL').value = '';
        currentPendingManualTrackDrafts = collectPendingManualTrackDrafts(detail);
        sortPendingManualTrackDrafts();
        renderPendingManualTrackTable();
    }

    function buildPendingManualPayload() {
        const albumName = document.getElementById('pendingManualAlbumName').value.trim();
        const albumArtist = document.getElementById('pendingManualAlbumArtist').value.trim();
        const displayArtist = document.getElementById('pendingManualDisplayArtist').value.trim();
        if (!albumName) {
            throw new Error('专辑名不能为空');
        }
        if (!albumArtist) {
            throw new Error('专辑艺术家不能为空');
        }
        if (!currentPendingManualTrackDrafts.length) {
            throw new Error('至少需要一首曲目');
        }

        sortPendingManualTrackDrafts();
        const positionSet = new Set();
        const manualTracks = currentPendingManualTrackDrafts.map((row, index) => {
            const title = String(row.title || '').trim();
            const discNumber = Number(row.discNumber || 0);
            const trackNumber = Number(row.trackNumber || 0);
            if (!title) {
                throw new Error(`第 ${index + 1} 首曲目的曲名不能为空`);
            }
            if (!(discNumber > 0 && trackNumber > 0)) {
                throw new Error(`第 ${index + 1} 首曲目需要补齐碟号和曲序`);
            }
            const posKey = `${discNumber}|${trackNumber}`;
            if (positionSet.has(posKey)) {
                throw new Error(`曲目位置重复：CD ${discNumber} / Track ${trackNumber}`);
            }
            positionSet.add(posKey);

            return {
                disc_number: discNumber,
                track_number: trackNumber,
                title,
                artist: String(row.artist || '').trim(),
                duration: Number(row.duration || 0),
                composer: String(row.composer || '').trim(),
                music_brainz_id: String(row.musicBrainzId || '').trim(),
                evidence_titles: uniqPendingStrings(row.evidenceTitles || []),
            };
        });

        return {
            manual_album: {
                name: albumName,
                album_artist: albumArtist,
                display_artist: displayArtist,
                release_date: document.getElementById('pendingManualReleaseDate').value.trim(),
                genre: document.getElementById('pendingManualGenre').value.trim(),
                country: document.getElementById('pendingManualCountry').value.trim(),
                status: document.getElementById('pendingManualStatus').value.trim(),
                packaging: document.getElementById('pendingManualPackaging').value.trim(),
                barcode: document.getElementById('pendingManualBarcode').value.trim(),
                cover_art_url: document.getElementById('pendingManualCoverArtURL').value.trim(),
            },
            manual_tracks: manualTracks,
        };
    }

    async function submitPendingManualMaintenance() {
        if (!currentPendingAlbumWorkItemID) return;
        const button = document.getElementById('pendingManualSubmitBtn');
        let payload;
        try {
            payload = buildPendingManualPayload();
        } catch (err) {
            alert(err.message);
            return;
        }

        button.disabled = true;
        button.textContent = '手动维护处理中...';
        try {
            const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/manual-maintenance`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(payload)
            });
            const data = await resp.json();
            if (!resp.ok) {
                throw new Error(data.error || '执行失败');
            }
            document.getElementById('pendingAlbumExecutionReport').innerHTML = renderPendingMaintenanceReport(data.report || {});
            await loadPendingAlbumWorkItemDetail();
            await loadPendingAlbumList();
        } catch (err) {
            document.getElementById('pendingAlbumExecutionReport').innerHTML = `<span style="color: #e74c3c;">${esc(err.message || '执行失败')}</span>`;
        } finally {
            button.disabled = false;
            button.textContent = '手动创建并应用上下文';
        }
    }

    async function openPendingAlbumWorkItem(identityKey) {
        const resp = await fetch('/api/pending-albums/work-items', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({identity_key: identityKey})
        });
        const data = await resp.json();
        if (!resp.ok) {
            alert(data.error || '创建工作项失败');
            return;
        }
        currentPendingAlbumWorkItemID = data.id;
        currentPendingAlbumWorkItemDetail = null;
        currentPendingManualTrackDrafts = [];
        currentPendingAlbumStalePromptedWorkItemID = 0;
        currentPendingCandidates = [];
        currentPendingSelectedMBID = "";
        renderAdminEmpty(document.getElementById('pendingAlbumCandidates'), "点击“搜索候选”加载 MusicBrainz 候选版本");
        document.getElementById('pendingAlbumWorkItemModal').style.display = 'block';
        await loadPendingAlbumWorkItemDetail();
    }

    async function loadPendingAlbumWorkItemDetail() {
        if (!currentPendingAlbumWorkItemID) return;
        const staleBanner = document.getElementById('pendingAlbumContextStaleBanner');
        if (staleBanner) {
            staleBanner.style.display = 'none';
        }
        const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}`);
        const data = await resp.json();
        if (!resp.ok) {
            alert(data.error || '加载工作项失败');
            return;
        }

        currentPendingAlbumWorkItemDetail = data;
        const workItem = data.work_item || {};
        currentPendingSelectedMBID = workItem.selected_mbid || "";
        document.getElementById('pendingAlbumWorkItemMeta').textContent =
            `${workItem.album || '-'} · ${workItem.album_artist || workItem.artist || '-'} · 状态 ${workItem.status || '-'}`;
        document.getElementById('pendingAlbumSelectedMBID').textContent =
            workItem.selected_mbid ? `当前候选: ${workItem.selected_mbid}` : '未选择候选版本';

        const contextEl = document.getElementById('pendingAlbumContextTracks');
        const contextTracks = data.context_tracks || [];
        contextEl.innerHTML = contextTracks.length ? contextTracks.map(item => `
            <div style="padding: 10px 12px; border-radius: 10px; background: rgba(var(--primary-rgb), 0.08);">
                <div style="font-weight: 600; color: var(--text-primary);">${item.track || '未知曲目'}</div>
                <div style="margin-top: 4px; font-size: 0.8em; opacity: 0.7; color: var(--text-secondary);">
                    播放 ${item.play_record_count} · 点赞 ${item.favorite_count} · ${((item.sources || []).join(' / ')) || '-'}
                </div>
            </div>
        `).join('') : '<div class="no-data">暂无上下文曲目</div>';

        const playEl = document.getElementById('pendingAlbumPlayRecords');
        const playRecords = data.play_records || [];
        playEl.innerHTML = playRecords.length ? playRecords.map(item => `
            <div style="padding: 10px 12px; border-bottom: 1px solid var(--border-color);">
                <div style="font-weight: 600; color: var(--text-primary);">${item.track}</div>
                <div style="font-size: 0.8em; opacity: 0.65; color: var(--text-secondary);">
                    #${item.id} · ${item.source || '-'} · ${item.play_time || '-'} · ${item.resolution_status || '-'}
                </div>
            </div>
        `).join('') : '<div class="no-data">暂无播放流水</div>';

        const favoriteEl = document.getElementById('pendingAlbumFavoriteEvents');
        const favoriteEvents = data.favorite_events || [];
        favoriteEl.innerHTML = favoriteEvents.length ? favoriteEvents.map(item => `
            <div style="padding: 10px 12px; border-bottom: 1px solid var(--border-color);">
                <div style="font-weight: 600; color: var(--text-primary);">${item.track}</div>
                <div style="font-size: 0.8em; opacity: 0.65; color: var(--text-secondary);">
                    #${item.id} · ${item.source || '-'} · liked=${item.provider_favorite} · ${item.resolution_status || '-'}
                </div>
            </div>
        `).join('') : '<div class="no-data">暂无点赞事件</div>';

        const staleText = document.getElementById('pendingAlbumContextStaleText');
        const liveGroup = data.live_group || null;
        const frozenPlayCount = playRecords.length;
        const frozenFavoriteCount = favoriteEvents.length;
        if (data.context_stale) {
            staleBanner.style.display = 'block';
            staleText.textContent = liveGroup
                ? `实时列表与冻结上下文不一致。实时播放 ${liveGroup.play_record_count || 0} 条 · 点赞 ${liveGroup.favorite_event_count || 0} 条，当前冻结播放 ${frozenPlayCount} 条 · 点赞 ${frozenFavoriteCount} 条。`
                : `实时列表里已经找不到这组上下文。当前冻结播放 ${frozenPlayCount} 条 · 点赞 ${frozenFavoriteCount} 条。`;
            if (currentPendingAlbumStalePromptedWorkItemID !== currentPendingAlbumWorkItemID) {
                currentPendingAlbumStalePromptedWorkItemID = currentPendingAlbumWorkItemID;
                const shouldRefresh = window.confirm(
                    liveGroup
                        ? `当前上下文与实时列表有出入，是否刷新冻结上下文？\n\n实时播放 ${liveGroup.play_record_count || 0} 条，冻结播放 ${frozenPlayCount} 条。\n实时点赞 ${liveGroup.favorite_event_count || 0} 条，冻结点赞 ${frozenFavoriteCount} 条。`
                        : '当前上下文已与实时列表脱节，是否刷新冻结上下文？'
                );
                if (shouldRefresh) {
                    await refreshPendingAlbumWorkItemContext();
                    return;
                }
            }
        } else {
            staleBanner.style.display = 'none';
            staleText.textContent = '-';
            currentPendingAlbumStalePromptedWorkItemID = 0;
        }

        resetPendingManualFormFromContext();

        let resultText = '尚未执行';
        if (workItem.status === 'completed') {
            resultText = '<span style="color: #27ae60;">执行完成</span>';
        } else if (workItem.last_error) {
            resultText = `<span style="color: #e74c3c;">最近错误: ${esc(workItem.last_error)}</span>`;
        }
        document.getElementById('pendingAlbumExecutionReport').innerHTML = resultText;

        const isCompleted = workItem.status === 'completed';
        document.getElementById('pendingAlbumDeepBtn').disabled = (isCompleted || !workItem.selected_mbid);
        document.getElementById('pendingManualSubmitBtn').disabled = isCompleted;
        if (currentPendingCandidates.length > 0) {
            renderPendingAlbumCandidates(currentPendingCandidates);
        }
    }

    function dismissPendingAlbumContextStale() {
        const staleBanner = document.getElementById('pendingAlbumContextStaleBanner');
        if (staleBanner) {
            staleBanner.style.display = 'none';
        }
        currentPendingAlbumStalePromptedWorkItemID = currentPendingAlbumWorkItemID;
    }

    async function refreshPendingAlbumWorkItemContext() {
        if (!currentPendingAlbumWorkItemID) return;
        const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/refresh-context`, {
            method: 'POST'
        });
        const data = await resp.json();
        if (!resp.ok) {
            alert(data.error || '刷新冻结上下文失败');
            return;
        }
        currentPendingAlbumStalePromptedWorkItemID = 0;
        await loadPendingAlbumWorkItemDetail();
        await loadPendingAlbumList();
    }

    async function loadPendingAlbumCandidates() {
        if (!currentPendingAlbumWorkItemID) return;
        const container = document.getElementById('pendingAlbumCandidates');
        renderAdminLoading(container, "搜索候选中...");
        const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/musicbrainz/candidates`);
        const data = await resp.json();
        if (!resp.ok) {
            renderAdminError(container, data.error || '未知错误', "搜索失败");
            return;
        }
        currentPendingCandidates = data || [];
        renderPendingAlbumCandidates(currentPendingCandidates);
    }

    function renderPendingAlbumCandidates(candidates) {
        const container = document.getElementById('pendingAlbumCandidates');
        if (!candidates || candidates.length === 0) {
            renderAdminEmpty(container, "没有搜索到候选版本");
            return;
        }
        container.innerHTML = candidates.map((item, index) => `
            <div style="padding: 12px; border: 1px solid var(--border-color); border-radius: 12px;">
                <div style="display: flex; justify-content: space-between; gap: 10px; align-items: flex-start;">
                    <div style="min-width: 0; flex: 1;">
                        <div style="font-weight: 700; color: var(--text-primary);">${item.name}</div>
                        <div style="font-size: 0.8em; opacity: 0.65; color: var(--text-secondary);">${item.mbid}</div>
                    </div>
                    ${item.mbid === currentPendingSelectedMBID
                        ? '<span style="padding: 7px 12px; border-radius: 10px; background: rgba(var(--primary-rgb), 0.18); color: var(--primary-color); font-weight: 700;">已选定</span>'
                        : `<button class="time-filter" onclick="selectPendingCandidate(${item.id || 0}, '${esc(item.mbid)}')">选定</button>`}
                </div>
                <div style="margin-top: 8px;">
                    <a href="javascript:void(0)" onclick="window.currentCandidates = currentPendingCandidates; showCandidateDetail(${index})" style="font-size: 0.78em; color: var(--primary-color); text-decoration: none;">查看详情</a>
                </div>
            </div>
        `).join('');
    }

    async function selectPendingCandidate(releaseMBID, mbid) {
        if (!currentPendingAlbumWorkItemID) return;
        const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/musicbrainz/link`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({release_mb_id: releaseMBID, mbid: mbid})
        });
        const data = await resp.json();
        if (!resp.ok) {
            alert(data.error || '绑定候选失败');
            return;
        }
        await loadPendingAlbumWorkItemDetail();
    }

    async function runPendingAlbumDeepMaintenance() {
        if (!currentPendingAlbumWorkItemID) return;
        const button = document.getElementById('pendingAlbumDeepBtn');
        button.disabled = true;
        button.textContent = '处理中...';
        const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/deep-maintenance`, {
            method: 'POST'
        });
        const data = await resp.json();
        if (!resp.ok) {
            document.getElementById('pendingAlbumExecutionReport').innerHTML = `<span style="color: #e74c3c;">${esc(data.error || '执行失败')}</span>`;
            button.disabled = false;
            button.textContent = '深度维护并应用上下文';
            return;
        }

        const report = data.report || {};
        document.getElementById('pendingAlbumExecutionReport').innerHTML = renderPendingMaintenanceReport(report);
        button.textContent = '已完成';
        await loadPendingAlbumWorkItemDetail();
        await loadPendingAlbumList();
    }

    // --- 新编：曲目列表加载逻辑 ---
