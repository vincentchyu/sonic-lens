async function loadTrackList(page = 1) {
        currentTrackPage = page;
        const content = document.getElementById('trackListContent');
        renderAdminLoading(content);

        const offset = (page - 1) * trackPageSize;
        const url = `/api/tracks?limit=${trackPageSize}&offset=${offset}&keyword=${encodeURIComponent(currentTrackKeyword)}`;

        try {
            const resp = await fetch(url);
            const data = await resp.json();
            totalTracksCount = data.total;
            renderTrackList(data.tracks);
            updateTrackPagination();
        } catch (err) {
            renderAdminError(content, err);
        }
    }

    function renderTrackList(tracks) {
        const content = document.getElementById('trackListContent');
        if (!tracks || tracks.length === 0) {
            renderAdminEmpty(content, "暂无曲目记录");
            return;
        }

        let html = '<ul class="ranking-list">';
        tracks.forEach((track, index) => {
            const trackName = track.track || track.title || "未知曲目";
            const discText = track.disc_number > 1 ? `CD ${track.disc_number} - ` : "";
            html += `
                <li class="ranking-item" onclick="showTrackDetails('${esc(track.artist)}', '${esc(track.album)}', '${esc(trackName)}', ${Number(track.track_number || 0)}, ${Number(track.disc_number || 0)})" style="cursor: pointer;">
                    <div class="rank-number">${(currentTrackPage-1)*trackPageSize + index + 1}</div>
                    <div class="track-info">
                        <div class="track-title">${trackName}</div>
                        <div class="track-artist">${track.artist} - 《${track.album}》 (${discText}${track.track_number})</div>
                    </div>
                </li>
            `;
        });
        html += '</ul>';
        content.innerHTML = html;
    }

    function updateTrackPagination() {
        const totalPages = Math.ceil(totalTracksCount / trackPageSize);
        document.getElementById('trackPageInfo').textContent = `第 ${currentTrackPage} 页 / 共 ${totalPages} 页 (总计 ${totalTracksCount})`;
        document.getElementById('trackPrevPage').disabled = currentTrackPage <= 1;
        document.getElementById('trackNextPage').disabled = currentTrackPage >= totalPages;
    }

    // 绑定分页与搜索
    document.getElementById('albumPrevPage')?.addEventListener('click', () => loadAlbumList(currentAlbumPage - 1));
    document.getElementById('albumNextPage')?.addEventListener('click', () => loadAlbumList(currentAlbumPage + 1));
    document.getElementById('refreshAlbumListBtn')?.addEventListener('click', () => loadAlbumList(1));
    document.getElementById('refreshPendingAlbumListBtn')?.addEventListener('click', () => loadPendingAlbumList());
    document.getElementById('albumSearchInput')?.addEventListener('input', (e) => {
        currentAlbumKeyword = e.target.value;
        clearTimeout(albumSearchTimeout);
        albumSearchTimeout = setTimeout(() => loadAlbumList(1), 500);
    });

    document.getElementById('artistPrevPage')?.addEventListener('click', () => loadArtistList(currentArtistPage - 1));
    document.getElementById('artistNextPage')?.addEventListener('click', () => loadArtistList(currentArtistPage + 1));
    document.getElementById('refreshArtistListBtn')?.addEventListener('click', () => {
        document.getElementById('artistSourceSelect').dataset.loaded = '';
        loadArtistSourceOptions();
        loadArtistList(1);
    });
    document.getElementById('artistSearchInput')?.addEventListener('input', (e) => {
        currentArtistKeyword = e.target.value.trim();
        clearTimeout(artistSearchTimeout);
        artistSearchTimeout = setTimeout(() => loadArtistList(1), 500);
    });
    document.getElementById('uploadArtistAvatarBtn')?.addEventListener('click', uploadArtistAvatar);

    document.getElementById('trackPrevPage')?.addEventListener('click', () => loadTrackList(currentTrackPage - 1));
    document.getElementById('trackNextPage')?.addEventListener('click', () => loadTrackList(currentTrackPage + 1));
    document.getElementById('refreshTrackListBtn')?.addEventListener('click', () => loadTrackList(1));
    document.getElementById('trackSearchInput')?.addEventListener('input', (e) => {
        currentTrackKeyword = e.target.value;
        clearTimeout(trackSearchTimeout);
        trackSearchTimeout = setTimeout(() => loadTrackList(1), 500);
    });

    // --- 新增内容见上文 ---

    const NAVBAR_ANIMATION_MS = 280;
    let navbarAnimationTimer = null;

    function syncNavbarBodySpacing(collapsed) {
        document.body.classList.toggle('collapsed-sidebar', collapsed);
    }

    // 侧边栏菜单展开/折叠逻辑
    function toggleNavbar() {
        const navbar = document.getElementById('mainNavbar');
        const isCollapsed = navbar.classList.contains('collapsed');
        const nextCollapsed = !isCollapsed;
        const details = document.getElementById('libraryMenuDetails');
        const navToggle = document.getElementById('navToggle');
        
        document.body.classList.add('sidebar-animating');
        clearTimeout(navbarAnimationTimer);

        if (isCollapsed) {
            navbar.classList.remove('collapsed', 'w-20');
            navbar.classList.add('w-60');
            localStorage.setItem('navbarCollapsed', 'false');
            if (details) {
                const librarySections = new Set(['albumList', 'artistList', 'pendingAlbumList', 'trackList']);
                details.open = librarySections.has(currentSectionID);
            }
            if (navToggle) navToggle.textContent = '◀';
        } else {
            navbar.classList.add('collapsed', 'w-20');
            navbar.classList.remove('w-60');
            localStorage.setItem('navbarCollapsed', 'true');
            // 折叠时关闭所有子菜单以节省空间
            if (details) details.open = false;
            if (navToggle) navToggle.textContent = '▶';
        }

        navbarAnimationTimer = setTimeout(() => {
            syncNavbarBodySpacing(nextCollapsed);
            document.body.classList.remove('sidebar-animating');
        }, NAVBAR_ANIMATION_MS);
    }

    // 初始化按钮点击
    const navToggleBtn = document.getElementById('navToggle');
    if (navToggleBtn) {
        navToggleBtn.addEventListener('click', toggleNavbar);
    }

    // 初始化侧边栏状态
    (function initNavbar() {
        const navbar = document.getElementById('mainNavbar');
        const collapsed = localStorage.getItem('navbarCollapsed') === 'true';
        if (collapsed) {
            navbar.classList.add('collapsed', 'w-20');
            navbar.classList.remove('w-60');
            if (navToggleBtn) navToggleBtn.textContent = '▶';
        } else {
            navbar.classList.remove('collapsed', 'w-20');
            navbar.classList.add('w-60');
            if (navToggleBtn) navToggleBtn.textContent = '◀';
        }
        syncNavbarBodySpacing(collapsed);
        document.body.classList.remove('sidebar-animating');
    })();
