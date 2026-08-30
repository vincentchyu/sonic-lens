async function loadAlbumList(page = 1) {
        currentAlbumPage = page;
        const content = document.getElementById('albumListContent');
        renderAdminLoading(content);

        const offset = (page - 1) * albumPageSize;
        const url = `/api/albums?limit=${albumPageSize}&offset=${offset}&keyword=${encodeURIComponent(currentAlbumKeyword)}`;

        try {
            const resp = await fetch(url);
            const data = await resp.json();
            totalAlbumsCount = data.total;
            renderAlbumList(data.albums);
            updateAlbumPagination();
        } catch (err) {
            renderAdminError(content, err);
        }
    }

    function renderAlbumList(albums) {
        const content = document.getElementById('albumListContent');
        if (!albums || albums.length === 0) {
            renderAdminEmpty(content, "暂无专辑记录");
            return;
        }

        let html = '<div class="album-grid">';
        albums.forEach(album => {
            html += `
                <div class="album-card" onclick="showAlbumDetails(${album.id})">
                    <div
                        class="artwork-slot album-card-cover"
                        data-artwork-slot="1"
                        data-album-id="${album.id}"
                        data-artist="${esc(album.artist || "")}"
                        data-album="${esc(album.name || "")}"
                        data-album-subtitle="${esc(album.name_subtitle || album.album_subtitle || "")}"
                        data-cover-art-url="${esc(album.cover_art_url || "")}"
                        data-alt-text="${esc(album.name || "专辑封面")}"
                    >${renderArtworkPlaceholder(album.name || "")}</div>
                    <div class="album-card-name" title="${esc(album.name || "")}">${album.name}</div>
                    <div class="album-card-artist" title="${esc(album.artist || "")}">${album.artist}</div>
                </div>
            `;
        });
        html += '</div>';
        content.innerHTML = html;
        hydrateArtworkSlots(content);
    }

    function updateAlbumPagination() {
        const totalPages = Math.ceil(totalAlbumsCount / albumPageSize);
        document.getElementById('albumPageInfo').textContent = `第 ${currentAlbumPage} 页 / 共 ${totalPages} 页 (总计 ${totalAlbumsCount})`;
        document.getElementById('albumPrevPage').disabled = currentAlbumPage <= 1;
        document.getElementById('albumNextPage').disabled = currentAlbumPage >= totalPages;
    }

async function loadArtistList(page = 1) {
        currentArtistPage = page;
        const content = document.getElementById('artistListContent');
        renderAdminLoading(content);

        const offset = (page - 1) * artistPageSize;
        const url = `/api/artist-profiles?limit=${artistPageSize}&offset=${offset}&keyword=${encodeURIComponent(currentArtistKeyword)}`;

        try {
            const resp = await fetch(url);
            const data = await resp.json();
            if (!resp.ok) throw new Error(data.error || '加载失败');
            totalArtistsCount = Number(data.total || 0);
            renderArtistList(data.items || []);
            updateArtistPagination();
        } catch (err) {
            renderAdminError(content, err);
        }
    }

    function renderArtistList(items) {
        const content = document.getElementById('artistListContent');
        if (!items || items.length === 0) {
            renderAdminEmpty(content, "暂无艺术家资料");
            return;
        }

        let html = '<div class="album-grid">';
        items.forEach(item => {
            const avatarURL = item.avatar_url || '';
            const initial = esc(((item.artist_name || '?').trim().charAt(0) || '?').toUpperCase());
            html += `
                <div style="border-radius: 18px; border: 1px solid var(--border-color); background: var(--bg-card); padding: 18px; display: flex; flex-direction: column; gap: 12px;">
                    <div style="width: 100%; aspect-ratio: 1 / 1; border-radius: 18px; overflow: hidden; background: rgba(var(--primary-rgb), 0.08); display: flex; align-items: center; justify-content: center;">
                        ${avatarURL
                            ? `<img src="${esc(avatarURL)}" alt="${esc(item.artist_name || '艺术家头像')}" style="width: 100%; height: 100%; object-fit: cover;" />`
                            : `<div style="font-size: 2.2rem; font-weight: 800; color: var(--primary-color);">${initial}</div>`}
                    </div>
                    <div>
                        <div style="font-size: 1rem; font-weight: 700; color: var(--text-primary); word-break: break-word;">${esc(item.artist_name || '-')}</div>
                        <div style="margin-top: 6px; font-size: 0.78em; color: var(--text-secondary); word-break: break-all;">${esc(item.avatar_object_key || '未上传头像')}</div>
                        <div style="margin-top: 8px; font-size: 0.78em; color: var(--text-secondary);">MIME: ${esc(item.avatar_mime || '-')}</div>
                    </div>
                </div>
            `;
        });
        html += '</div>';
        content.innerHTML = html;
    }

    function updateArtistPagination() {
        const totalPages = Math.max(1, Math.ceil(totalArtistsCount / artistPageSize));
        document.getElementById('artistPageInfo').textContent = `第 ${currentArtistPage} 页 / 共 ${totalPages} 页 (总计 ${totalArtistsCount})`;
        document.getElementById('artistPrevPage').disabled = currentArtistPage <= 1;
        document.getElementById('artistNextPage').disabled = currentArtistPage >= totalPages;
    }

    async function loadArtistSourceOptions() {
        const select = document.getElementById('artistSourceSelect');
        if (!select || select.dataset.loaded === '1') {
            return;
        }

        try {
            const resp = await fetch('/api/artist-profiles/top-artists?limit=50');
            const data = await resp.json();
            if (!resp.ok) throw new Error(data.error || '加载热门艺术家失败');

            const items = Array.isArray(data.items) ? data.items : [];
            select.innerHTML = '<option value="">选择热门艺术家</option>' +
                items.map(name => `<option value="${esc(name)}">${esc(name)}</option>`).join('');
            select.dataset.loaded = '1';
        } catch (err) {
            select.innerHTML = '<option value="">加载热门艺术家失败</option>';
        }
    }

    async function uploadArtistAvatar() {
        const artistName = (document.getElementById('artistSourceSelect')?.value || '').trim();
        const fileInput = document.getElementById('artistAvatarFileInput');
        const hint = document.getElementById('artistUploadHint');
        const uploadBtn = document.getElementById('uploadArtistAvatarBtn');
        const file = fileInput?.files?.[0];

        if (!artistName) {
            alert('请选择热门艺术家');
            return;
        }
        if (!file) {
            alert('请选择图片文件');
            return;
        }

        uploadBtn.disabled = true;
        hint.textContent = '正在读取图片并上传...';
        try {
            const dataURL = await readFileAsDataURL(file);
            const resp = await fetch('/api/artist-profiles/avatar', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    artist_name: artistName,
                    data: dataURL,
                }),
            });
            const data = await resp.json();
            if (!resp.ok) throw new Error(data.error || '上传失败');

            hint.textContent = `上传完成：${artistName}`;
            if (fileInput) fileInput.value = '';
            await loadArtistList(currentArtistPage);
        } catch (err) {
            hint.textContent = `上传失败：${err.message || '未知错误'}`;
            alert(hint.textContent);
        } finally {
            uploadBtn.disabled = false;
        }
    }

    function readFileAsDataURL(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve(String(reader.result || ''));
            reader.onerror = () => reject(new Error('读取图片失败'));
            reader.readAsDataURL(file);
        });
    }
