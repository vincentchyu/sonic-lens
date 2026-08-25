function escJs(s) {
    if (!s) return "";
    return String(s)
        .replace(/\\/g, '\\\\')   // 转义反斜杠
        .replace(/'/g, "\\'")     // 转义单引号
        .replace(/&/g, '&amp;')   // 转义 &
        .replace(/"/g, '&quot;')  // 转义双引号，防止闭合 HTML 属性
        .replace(/</g, '&lt;')    // 转义 <
        .replace(/>/g, '&gt;');   // 转义 >
}

async function loadPendingAlbumList() {
        if (currentPendingWorkTab === 'pending_groups' || currentPendingWorkTab === 'uncreated_groups') {
            const content = document.getElementById('pendingAlbumListContent');
            renderAdminLoading(content);
            try {
                const filter = currentPendingWorkTab === 'uncreated_groups' ? 'uncreated' : '';
                const resp = await fetch(`/api/pending-albums?limit=100&filter=${filter}`);
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
            'uncreated_groups': 'tabUncreatedGroups',
            'working_items': 'tabWorkingItems',
            'completed_items': 'tabCompletedItems'
        };
        document.getElementById(tabIdMap[tab]).classList.add('active');
        
        // 控制搜索栏和分页容器显示
        const searchContainer = document.getElementById('pendingWorkSearchContainer');
        const paginationContainer = document.getElementById('pendingWorkPagination');
        
        if (tab === 'pending_groups' || tab === 'uncreated_groups') {
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
                return '待处理';
            case 'mb_selected':
                return '已选版本';
            case 'staged':
                return '草稿就绪';
            case 'applying':
                return '应用中';
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
        if (statusName === '草稿就绪') {
            return `<span style="display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; background: rgba(243, 156, 18, 0.16); color: #e67e22; font-weight: 700; letter-spacing: 0.02em;">${statusName}</span>`;
        }
        if (statusName === '待处理' || statusName === '已选版本' || statusName === '应用中') {
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
                        <button class="time-filter" onclick="openPendingAlbumWorkItem('${escJs(group.identity_key)}')" style="flex-shrink: 0;">进入维护</button>
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
        const manualSubInput = document.getElementById('pendingManualAlbumSubtitle');
        if (manualSubInput) manualSubInput.value = workItem.album_subtitle || '';
        const manualRtInput = document.getElementById('pendingManualReleaseType');
        if (manualRtInput) manualRtInput.value = '';
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
        const albumSubtitle = (document.getElementById('pendingManualAlbumSubtitle')?.value || '').trim();
        const releaseType = (document.getElementById('pendingManualReleaseType')?.value || '').trim();
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
                album_subtitle: albumSubtitle,
                release_type: releaseType,
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
        if (workItem.status === 'staged') {
            resultText = `
                <div style="display: flex; flex-direction: column; gap: 8px; margin-top: 4px;">
                    <span style="color: #e67e22; font-weight: 700;">预审草稿就绪（待管理员审核确认）</span>
                    <button class="time-filter" onclick="openPendingAlbumDiffPreviewModal()" style="background: #e67e22; color: #fff; border: none; font-weight: 700; padding: 8px 14px; border-radius: 8px; cursor: pointer; text-align: center;">👉 查看 / 修改预审草稿</button>
                </div>
            `;
        } else if (workItem.status === 'completed') {
            resultText = '<span style="color: #27ae60; font-weight: 700;">维护完成落库</span>';
        } else if (workItem.last_error) {
            resultText = `<span style="color: #e74c3c;">最近错误: ${esc(workItem.last_error)}</span>`;
        }
        document.getElementById('pendingAlbumExecutionReport').innerHTML = resultText;

        const isCompleted = workItem.status === 'completed';
        document.getElementById('pendingAlbumDeepBtn').disabled = false;
        document.getElementById('pendingManualSubmitBtn').disabled = isCompleted;

        // 自动装载候选版本列表
        await loadPendingAlbumCandidates();
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
        const accordion = document.getElementById('pendingManualAccordion');
        if (!candidates || candidates.length === 0) {
            renderAdminEmpty(container, "未搜到 MB 候选版本（可展开下方手动创建表单）");
            if (accordion) accordion.open = true;
            return;
        }
        if (accordion) accordion.open = false;
        const isCompleted = currentPendingAlbumWorkItemDetail?.work_item?.status === 'completed';
        container.innerHTML = candidates.map((item, index) => `
            <div style="padding: 12px; border: 1px solid var(--border-color); border-radius: 12px;">
                <div style="display: flex; justify-content: space-between; gap: 10px; align-items: flex-start;">
                    <div style="min-width: 0; flex: 1;">
                        <div style="font-weight: 700; color: var(--text-primary);">${esc(item.name)}</div>
                        <div style="font-size: 0.8em; opacity: 0.65; color: var(--text-secondary);">${esc(item.mbid)}</div>
                    </div>
                    <div style="display: flex; gap: 6px;">
                        ${isCompleted 
                            ? (item.mbid === currentPendingSelectedMBID ? '<button class="time-filter" style="background: rgba(var(--primary-rgb), 0.18); color: var(--primary-color); font-weight: 700;" onclick="openPendingAlbumDiffPreviewModal(' + (item.id || 0) + ', \'' + esc(item.mbid) + '\')">查看预审草稿</button>' : '<span style="font-size: 0.82em; opacity: 0.5; color: var(--text-secondary); align-self: center;">归因已完成</span>')
                            : (item.mbid === currentPendingSelectedMBID
                                ? '<button class="time-filter" style="background: rgba(var(--primary-rgb), 0.18); color: var(--primary-color); font-weight: 700;" onclick="openPendingAlbumDiffPreviewModal(' + (item.id || 0) + ', \'' + esc(item.mbid) + '\')">预览与维护</button>'
                                : `<button class="time-filter" onclick="selectPendingCandidateAndPreview(${item.id || 0}, '${esc(item.mbid)}')">选定并预览</button>`)
                        }
                    </div>
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

    async function selectPendingCandidateAndPreview(releaseMBID, mbid) {
        await selectPendingCandidate(releaseMBID, mbid);
        await openPendingAlbumDiffPreviewModal(releaseMBID, mbid);
    }

    let currentPendingDiffPreview = null;
    let currentContextAlbumID = 0;

    async function openAlbumMBDiffPreviewModal(albumID, releaseMBID = 0, mbid = '', forceRefresh = false) {
        if (!albumID) return;
        currentContextAlbumID = albumID;

        showModal('pendingAlbumDiffPreviewModal');

        const footerStatus = document.getElementById('pendingDiffFooterStatus');
        const saveBtn = document.getElementById('pendingDiffSaveDraftBtn');
        const applyBtn = document.getElementById('pendingDiffApplyBtn');
        if (footerStatus) footerStatus.textContent = forceRefresh ? '正在重新从 MusicBrainz 拉取最新元数据...' : '正在获取 MB 精选维护差异预审...';
        if (applyBtn) applyBtn.disabled = true;
        if (saveBtn) {
            saveBtn.style.display = 'inline-block';
            saveBtn.disabled = false;
        }

        try {
            const url = `/api/albums/${albumID}/musicbrainz/preview?release_mb_id=${releaseMBID || 0}&mbid=${encodeURIComponent(mbid)}&force_refresh=${forceRefresh ? 1 : 0}`;
            const resp = await fetch(url);
            const data = await resp.json();
            if (!resp.ok) {
                throw new Error(data.error || '获取 MB 精选维护快照失败');
            }
            currentPendingDiffPreview = data;
            renderPendingDiffPreview(data);
            if (footerStatus) footerStatus.textContent = forceRefresh ? '✅ 已成功从 MB 重新拉取最新数据！' : '精选维护对比已就绪，审查无误后可点击“仅保存草稿”或“确认审核并落库数据”。';
        } catch (err) {
            if (footerStatus) footerStatus.textContent = '预审失败: ' + (err.message || '网络或数据错误');
            alert('获取预审快照失败: ' + (err.message || err));
        } finally {
            if (applyBtn) applyBtn.disabled = false;
            if (saveBtn) saveBtn.disabled = false;
        }
    }

    async function refreshPendingDiffPreviewFromMB() {
        if (!confirm('重新从 MusicBrainz 拉取将刷新当前对比内容，是否继续？')) {
            return;
        }
        if (currentContextAlbumID > 0) {
            await openAlbumMBDiffPreviewModal(currentContextAlbumID, 0, '', true);
        } else {
            await openPendingAlbumDiffPreviewModal(0, '', true);
        }
    }

    async function openPendingAlbumDiffPreviewModal(releaseMBID = 0, mbid = '', forceRefresh = false) {
        if (!currentPendingAlbumWorkItemID) return;
        currentContextAlbumID = 0;
        const targetMBID = mbid || currentPendingSelectedMBID || '';

        showModal('pendingAlbumDiffPreviewModal');

        const footerStatus = document.getElementById('pendingDiffFooterStatus');
        const saveBtn = document.getElementById('pendingDiffSaveDraftBtn');
        const applyBtn = document.getElementById('pendingDiffApplyBtn');
        if (saveBtn) saveBtn.style.display = 'inline-block';
        if (footerStatus) footerStatus.textContent = forceRefresh ? '正在重新从 MusicBrainz 拉取最新元数据...' : '正在获取 MB 差异预审草稿...';
        if (applyBtn) applyBtn.disabled = true;
        if (saveBtn) saveBtn.disabled = true;

        const isCompleted = currentPendingAlbumWorkItemDetail?.work_item?.status === 'completed';

        try {
            const url = `/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/musicbrainz/preview?release_mb_id=${releaseMBID || 0}&mbid=${encodeURIComponent(targetMBID)}&force_refresh=${forceRefresh ? 1 : 0}`;
            const resp = await fetch(url);
            const data = await resp.json();
            if (!resp.ok) {
                throw new Error(data.error || '获取 MB 预审快照失败');
            }
            currentPendingDiffPreview = data;
            renderPendingDiffPreview(data);
            if (isCompleted) {
                if (footerStatus) footerStatus.textContent = '✅ 维护已完成并已落库（当前为只读审查模式）。';
            } else {
                if (footerStatus) footerStatus.textContent = forceRefresh ? '✅ 已成功从 MB 重新拉取最新草稿！' : '草稿对比已就绪，审查无误后可保存草稿或点击应用落库。';
            }
        } catch (err) {
            if (footerStatus) footerStatus.textContent = '预审失败: ' + (err.message || '网络或数据错误');
            alert('获取预审快照失败: ' + (err.message || err));
        } finally {
            if (saveBtn) saveBtn.disabled = isCompleted;
            if (applyBtn) applyBtn.disabled = isCompleted;
        }
    }

    function setPendingDiffGenre(val) {
        const input = document.getElementById('pendingDiffGenre');
        if (input) input.value = val;
    }

window.setPendingDiffGenre = setPendingDiffGenre;

function setPendingDiffReleaseType(val) {
    const input = document.getElementById('pendingDiffReleaseType');
    if (input) input.value = val;
}

window.setPendingDiffReleaseType = setPendingDiffReleaseType;

    function renderPendingDiffPreview(preview) {
        if (!preview) return;
        const album = preview.album_preview || {};
        document.getElementById('pendingDiffAlbumName').value = album.name || '';
        const diffSubInput = document.getElementById('pendingDiffAlbumSubtitle');
        if (diffSubInput) diffSubInput.value = album.album_subtitle || '';
        const diffRtInput = document.getElementById('pendingDiffReleaseType');
        const mbRt = (album.mb_release_type || '').toLowerCase();
        const evRt = (album.evidence_release_type || '').toLowerCase();
        const chosenRt = (album.release_type || album.mb_release_type || album.evidence_release_type || '').toLowerCase();
        if (diffRtInput) diffRtInput.value = chosenRt;

        const rtHint = document.getElementById('pendingDiffReleaseTypeHint');
        if (rtHint) {
            let hintHtml = '';
            if (mbRt || evRt) {
                hintHtml = `MB: <b>${esc(mbRt || '-')}</b> | 监听源: <b>${esc(evRt || '-')}</b>`;
                if (evRt && evRt !== mbRt) {
                    hintHtml += ` <a href="javascript:void(0)" onclick="setPendingDiffReleaseType('${esc(evRt)}')" style="color: var(--primary-color); text-decoration: underline; margin-left: 6px; font-weight: bold;">[用监听源]</a>`;
                }
                if (mbRt && evRt && evRt !== mbRt) {
                    hintHtml += ` <a href="javascript:void(0)" onclick="setPendingDiffReleaseType('${esc(mbRt)}')" style="color: #e67e22; text-decoration: underline; margin-left: 6px; font-weight: bold;">[用 MB]</a>`;
                }
                if (mbRt && evRt && mbRt !== evRt) {
                    hintHtml += ` <span style="color: #e67e22; font-weight: bold; margin-left: 6px;">⚠️ 存在差异</span>`;
                }
            }
            rtHint.innerHTML = hintHtml;
        }

        document.getElementById('pendingDiffAlbumArtist').value = album.album_artist || '';
        
        const mbGenre = album.genre || '';
        const evGenre = album.evidence_genre || '';
        const chosenGenre = album.genre || album.evidence_genre || '';
        document.getElementById('pendingDiffGenre').value = chosenGenre;

        const genreHint = document.getElementById('pendingDiffGenreHint');
        if (genreHint) {
            let hintHtml = '';
            if (mbGenre || evGenre) {
                hintHtml = `MB 流派: <b>${esc(mbGenre || '-')}</b> | 监听源流派: <b>${esc(evGenre || '-')}</b>`;
                if (evGenre && evGenre !== mbGenre) {
                    hintHtml += ` <a href="javascript:void(0)" onclick="setPendingDiffGenre('${esc(evGenre)}')" style="color: var(--primary-color); text-decoration: underline; margin-left: 6px; font-weight: bold;">[用监听源流派]</a>`;
                }
                if (mbGenre && evGenre && evGenre !== mbGenre) {
                    hintHtml += ` <a href="javascript:void(0)" onclick="setPendingDiffGenre('${esc(mbGenre)}')" style="color: #e67e22; text-decoration: underline; margin-left: 6px; font-weight: bold;">[用 MB 流派]</a>`;
                }
            }
            genreHint.innerHTML = hintHtml;
        }

        document.getElementById('pendingDiffReleaseDate').value = album.release_date || '';
        document.getElementById('pendingDiffOriginalReleaseDate').value = album.original_release_date || '';
        document.getElementById('pendingDiffCountry').value = album.country || '';

        const badge = document.getElementById('pendingDiffBadge');
        if (badge) {
            const count = preview.diff_track_count || 0;
            badge.textContent = count > 0 ? `有 ${count} 首曲名差异` : '曲名完全匹配';
            badge.style.background = count > 0 ? 'rgba(243, 156, 18, 0.2)' : 'rgba(46, 204, 113, 0.2)';
            badge.style.color = count > 0 ? '#e67e22' : '#27ae60';
            badge.style.borderColor = count > 0 ? 'rgba(243, 156, 18, 0.4)' : 'rgba(46, 204, 113, 0.4)';
        }

        const tbody = document.getElementById('pendingDiffTrackBody');
        const tracks = preview.track_previews || [];
        if (!tracks.length) {
            tbody.innerHTML = '<tr><td colspan="4" style="text-align: center; padding: 20px; color: var(--text-secondary);">无曲目数据</td></tr>';
            return;
        }

        tbody.innerHTML = tracks.map((track, idx) => {
            const hasDiff = !!track.has_diff;
            const rowBg = hasDiff ? 'rgba(243, 156, 18, 0.06)' : 'transparent';
            const mbTitle = track.mb_title || '';
            const evTitle = track.evidence_title || (track.evidence_titles && track.evidence_titles[0]) || '';
            const chosenTitle = track.title || mbTitle || evTitle || '';
            const chosenGenre = track.genre || track.evidence_genre || '';

            return `
                <tr style="background: ${rowBg}; border-bottom: 1px solid var(--border-color);" data-track-idx="${idx}">
                    <td style="padding: 10px 12px; font-weight: 600; color: var(--text-secondary);">
                        CD ${track.disc_number || 1} / #${track.track_number}
                    </td>
                    <td style="padding: 10px 12px; color: var(--text-primary);">
                        <div>${esc(mbTitle || '-')}</div>
                        <div style="font-size: 0.78em; opacity: 0.6; color: var(--text-secondary);">时长: ${Math.floor(track.duration / 60)}:${String(track.duration % 60).padStart(2, '0')}</div>
                    </td>
                    <td style="padding: 10px 12px;">
                        ${evTitle ? `
                            <div style="color: var(--text-primary);">${esc(evTitle)}</div>
                            ${hasDiff ? '<span style="font-size: 0.72em; background: #e67e22; color: #fff; padding: 1px 6px; border-radius: 4px; font-weight: bold; display: inline-block; margin-top: 4px;">不一致</span>' : ''}
                        ` : '<span style="opacity: 0.5; color: var(--text-secondary);">-</span>'}
                    </td>
                    <td style="padding: 10px 12px;">
                        <div style="display: flex; flex-direction: column; gap: 6px;">
                            <input type="text" class="search-input diff-chosen-title" data-idx="${idx}" value="${esc(chosenTitle)}" placeholder="曲名" style="width: 100%; font-size: 0.9em; padding: 6px 10px;">
                            <div style="display: flex; align-items: center; gap: 6px;">
                                <span style="font-size: 0.76em; opacity: 0.75; color: var(--text-secondary); white-space: nowrap;">曲目流派:</span>
                                <input type="text" class="search-input diff-chosen-genre" data-idx="${idx}" value="${esc(chosenGenre)}" placeholder="曲目流派 (选填)" style="width: 100%; font-size: 0.8em; padding: 4px 8px;">
                            </div>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');
    }

    function setAllDiffTitleChoice(choice) {
        if (!currentPendingDiffPreview) return;
        const tracks = currentPendingDiffPreview.track_previews || [];
        const inputs = document.querySelectorAll('.diff-chosen-title');
        inputs.forEach(input => {
            const idx = Number(input.getAttribute('data-idx'));
            const track = tracks[idx];
            if (!track) return;
            if (choice === 'mb') {
                input.value = track.mb_title || track.title || '';
            } else if (choice === 'evidence') {
                input.value = track.evidence_title || (track.evidence_titles && track.evidence_titles[0]) || track.mb_title || '';
            }
        });
    }

    async function savePendingDiffDraft() {
        if ((!currentPendingAlbumWorkItemID && !currentContextAlbumID) || !currentPendingDiffPreview) return;
        const footerStatus = document.getElementById('pendingDiffFooterStatus');
        const saveBtn = document.getElementById('pendingDiffSaveDraftBtn');

        const albumName = document.getElementById('pendingDiffAlbumName').value.trim();
        const albumSubtitle = (document.getElementById('pendingDiffAlbumSubtitle')?.value || '').trim();
        const releaseType = (document.getElementById('pendingDiffReleaseType')?.value || '').trim();
        const albumArtist = document.getElementById('pendingDiffAlbumArtist').value.trim();
        if (!albumName || !albumArtist) {
            alert('专辑名和专辑艺术家不能为空');
            return;
        }

        const tracks = currentPendingDiffPreview.track_previews || [];
        const chosenInputs = document.querySelectorAll('.diff-chosen-title');
        const chosenGenreInputs = document.querySelectorAll('.diff-chosen-genre');
        const manualTracks = [];

        for (let i = 0; i < tracks.length; i++) {
            const track = tracks[i];
            const chosenTitle = (chosenInputs[i] ? chosenInputs[i].value : '').trim() || track.mb_title;
            const chosenGenre = (chosenGenreInputs[i] ? chosenGenreInputs[i].value : '').trim();
            if (!chosenTitle) {
                alert(`第 ${i + 1} 首曲目的标题不能为空`);
                return;
            }
            manualTracks.push({
                disc_number: Number(track.disc_number || 1),
                track_number: Number(track.track_number || (i + 1)),
                title: chosenTitle,
                artist: track.artist || albumArtist,
                genre: chosenGenre,
                duration: Number(track.duration || 0),
                composer: track.composer || '',
                music_brainz_id: track.music_brainz_id || '',
                evidence_titles: track.evidence_titles || []
            });
        }

        const draftPayload = {
            work_item_id: Number(currentPendingAlbumWorkItemID),
            release_mb_id: Number(currentPendingDiffPreview.release_mb_id || 0),
            mbid: currentPendingDiffPreview.mbid || '',
            album_preview: {
                name: albumName,
                album_subtitle: albumSubtitle,
                release_type: releaseType,
                album_artist: albumArtist,
                display_artist: albumArtist,
                genre: document.getElementById('pendingDiffGenre').value.trim(),
                evidence_genre: currentPendingDiffPreview.album_preview?.evidence_genre || '',
                evidence_release_type: currentPendingDiffPreview.album_preview?.evidence_release_type || '',
                mb_release_type: currentPendingDiffPreview.album_preview?.mb_release_type || '',
                release_date: document.getElementById('pendingDiffReleaseDate').value.trim(),
                original_release_date: document.getElementById('pendingDiffOriginalReleaseDate').value.trim(),
                country: document.getElementById('pendingDiffCountry').value.trim(),
                status: currentPendingDiffPreview.album_preview?.status || '',
                packaging: currentPendingDiffPreview.album_preview?.packaging || '',
                barcode: currentPendingDiffPreview.album_preview?.barcode || '',
                cover_art_url: currentPendingDiffPreview.album_preview?.cover_art_url || ''
            },
            track_previews: tracks.map((track, i) => ({
                ...track,
                title: (chosenInputs[i] ? chosenInputs[i].value : '').trim() || track.mb_title,
                genre: (chosenGenreInputs[i] ? chosenGenreInputs[i].value : '').trim()
            })),
            diff_track_count: currentPendingDiffPreview.diff_track_count || 0,
            suggested_input: {
                manual_album: {
                    name: albumName,
                    album_subtitle: albumSubtitle,
                    release_type: releaseType,
                    album_artist: albumArtist,
                    display_artist: albumArtist,
                    genre: document.getElementById('pendingDiffGenre').value.trim(),
                    release_date: document.getElementById('pendingDiffReleaseDate').value.trim(),
                    original_release_date: document.getElementById('pendingDiffOriginalReleaseDate').value.trim(),
                    country: document.getElementById('pendingDiffCountry').value.trim(),
                    status: currentPendingDiffPreview.album_preview?.status || '',
                    packaging: currentPendingDiffPreview.album_preview?.packaging || '',
                    barcode: currentPendingDiffPreview.album_preview?.barcode || '',
                    cover_art_url: currentPendingDiffPreview.album_preview?.cover_art_url || ''
                },
                manual_tracks: manualTracks
            }
        };

        if (footerStatus) footerStatus.textContent = '正在保存改动为草稿...';
        if (saveBtn) saveBtn.disabled = true;

        try {
            let draftUrl = `/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/musicbrainz/draft`;
            if (currentContextAlbumID > 0) {
                draftUrl = `/api/albums/${currentContextAlbumID}/musicbrainz/draft`;
            }
            const resp = await fetch(draftUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(draftPayload)
            });
            const data = await resp.json();
            if (!resp.ok) {
                throw new Error(data.error || '保存草稿失败');
            }
            currentPendingDiffPreview = draftPayload;
            if (footerStatus) footerStatus.textContent = '✅ 草稿已微调并成功保存！';
            if (typeof showToast === 'function') showToast('✅ 预审草稿已成功保存！');
            if (currentPendingAlbumWorkItemID > 0) {
                await loadPendingAlbumWorkItemDetail();
                await loadPendingAlbumList();
            }
        } catch (err) {
            if (footerStatus) footerStatus.textContent = '保存草稿失败: ' + (err.message || err);
            alert('保存草稿失败: ' + (err.message || err));
        } finally {
            if (saveBtn) saveBtn.disabled = false;
        }
    }

    async function submitPendingDiffMaintenance() {
        if ((!currentPendingAlbumWorkItemID && !currentContextAlbumID) || !currentPendingDiffPreview) return;

        const albumName = document.getElementById('pendingDiffAlbumName').value.trim();
        const albumSubtitle = (document.getElementById('pendingDiffAlbumSubtitle')?.value || '').trim();
        const releaseType = (document.getElementById('pendingDiffReleaseType')?.value || '').trim();
        const albumArtist = document.getElementById('pendingDiffAlbumArtist').value.trim();
        if (!albumName || !albumArtist) {
            alert('专辑名和专辑艺术家不能为空');
            return;
        }

        const tracks = currentPendingDiffPreview.track_previews || [];
        const chosenInputs = document.querySelectorAll('.diff-chosen-title');
        const chosenGenreInputs = document.querySelectorAll('.diff-chosen-genre');
        const manualTracks = [];

        for (let i = 0; i < tracks.length; i++) {
            const track = tracks[i];
            const chosenTitle = (chosenInputs[i] ? chosenInputs[i].value : '').trim() || track.mb_title;
            const chosenGenre = (chosenGenreInputs[i] ? chosenGenreInputs[i].value : '').trim();
            if (!chosenTitle) {
                alert(`第 ${i + 1} 首曲目的标题不能为空`);
                return;
            }
            manualTracks.push({
                disc_number: Number(track.disc_number || 1),
                track_number: Number(track.track_number || (i + 1)),
                title: chosenTitle,
                artist: track.artist || albumArtist,
                genre: chosenGenre,
                duration: Number(track.duration || 0),
                composer: track.composer || '',
                music_brainz_id: track.music_brainz_id || '',
                evidence_titles: track.evidence_titles || []
            });
        }

        const payload = {
            release_mb_id: currentPendingDiffPreview?.release_mb_id || 0,
            mbid: currentPendingDiffPreview?.mbid || '',
            manual_album: {
                name: albumName,
                album_subtitle: albumSubtitle,
                release_type: releaseType,
                album_artist: albumArtist,
                display_artist: albumArtist,
                release_date: document.getElementById('pendingDiffReleaseDate').value.trim(),
                original_release_date: document.getElementById('pendingDiffOriginalReleaseDate').value.trim(),
                genre: document.getElementById('pendingDiffGenre').value.trim(),
                country: document.getElementById('pendingDiffCountry').value.trim(),
                status: (currentPendingDiffPreview.album_preview || {}).status || 'official',
                packaging: (currentPendingDiffPreview.album_preview || {}).packaging || '',
                barcode: (currentPendingDiffPreview.album_preview || {}).barcode || '',
                cover_art_url: (currentPendingDiffPreview.album_preview || {}).cover_art_url || '',
            },
            manual_tracks: manualTracks
        };

        const applyBtn = document.getElementById('pendingDiffApplyBtn');
        const footerStatus = document.getElementById('pendingDiffFooterStatus');
        applyBtn.disabled = true;
        applyBtn.textContent = '提交审核落库中...';
        if (footerStatus) footerStatus.textContent = '提交事务中...';

        try {
            if (currentContextAlbumID > 0) {
                const resp = await fetch(`/api/albums/${currentContextAlbumID}/musicbrainz/apply-maintenance`, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                const data = await resp.json();
                if (!resp.ok) {
                    throw new Error(data.error || '精选维护落库失败');
                }
                hideModal('pendingAlbumDiffPreviewModal');
                if (typeof showToast === 'function') showToast('✨ 专辑精选维护成功并已同步落库！');
                if (typeof showAlbumDetails === 'function') showAlbumDetails(currentContextAlbumID);
            } else {
                if (!currentPendingAlbumWorkItemID) return;
                const resp = await fetch(`/api/pending-albums/work-items/${currentPendingAlbumWorkItemID}/manual-maintenance`, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                const data = await resp.json();
                if (!resp.ok) {
                    throw new Error(data.error || '执行失败');
                }
                hideModal('pendingAlbumDiffPreviewModal');
                document.getElementById('pendingAlbumExecutionReport').innerHTML = renderPendingMaintenanceReport(data.report || {});
                await loadPendingAlbumWorkItemDetail();
                await loadPendingAlbumList();
                alert('审核落库成功！');
            }
        } catch (err) {
            alert('应用失败: ' + err.message);
        } finally {
            applyBtn.disabled = false;
            applyBtn.textContent = '确认审核并落库数据';
        }
    }

    async function runPendingAlbumDeepMaintenance() {
        if (!currentPendingAlbumWorkItemID) return;
        if (!currentPendingSelectedMBID) {
            alert('请先在上方搜寻并选定一个 MusicBrainz 候选版本，或使用下方手动维护。');
            return;
        }
        await openPendingAlbumDiffPreviewModal(0, currentPendingSelectedMBID);
    }

    // --- 新编：曲目列表加载逻辑 ---
