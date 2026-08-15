// --- 流派权威库与归因调试器前端交互逻辑 ---

function esc(s) {
    if (!s) return "";
    return String(s)
        .replace(/\\'/g, "'")
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function showModal(id) {
    const el = document.getElementById(id);
    if (el) el.style.display = 'flex';
}

function hideModal(id) {
    const el = document.getElementById(id);
    if (el) el.style.display = 'none';
}

let currentGenrePage = 1;
const genrePageSize = 50;
let currentGenreKeyword = '';
let currentGenreSort = 'play_count';
let totalGenresCount = 0;
let genreSearchTimeout = null;
let genreTestTimeout = null;

// 加载流派分页列表
async function loadGenreList(page = 1) {
    currentGenrePage = page;
    const content = document.getElementById('genreListContent');
    if (!content) return;
    renderAdminLoading(content);

    const offset = (page - 1) * genrePageSize;
    const url = `/api/genres?limit=${genrePageSize}&offset=${offset}&keyword=${encodeURIComponent(currentGenreKeyword)}&sort=${encodeURIComponent(currentGenreSort)}`;

    try {
        const resp = await fetch(url);
        const data = await resp.json();
        if (!resp.ok) throw new Error(data.error || '加载流派列表失败');
        totalGenresCount = Number(data.total || 0);
        renderGenreList(data.genres || []);
        updateGenrePagination();
    } catch (err) {
        renderAdminError(content, err);
    }
}

// 渲染流派表格
function renderGenreList(genres) {
    const content = document.getElementById('genreListContent');
    if (!content) return;

    if (!genres || genres.length === 0) {
        renderAdminEmpty(content, "暂无流派记录");
        return;
    }

    let html = `
        <div style="overflow-x: auto;">
            <table class="data-table" style="width: 100%; border-collapse: separate; border-spacing: 0; font-size: 0.9em;">
                <thead>
                    <tr style="text-align: left; color: var(--text-secondary); border-bottom: 1px solid var(--border-color);">
                        <th style="padding: 12px 14px; width: 60px;">#</th>
                        <th style="padding: 12px 14px;">英文标准名 (Name)</th>
                        <th style="padding: 12px 14px;">规范中文名 (NameZh)</th>
                        <th style="padding: 12px 14px; text-align: right;">累计播放</th>
                        <th style="padding: 12px 14px; text-align: right; width: 150px;">操作</th>
                    </tr>
                </thead>
                <tbody>
    `;

    genres.forEach((g, idx) => {
        const rank = (currentGenrePage - 1) * genrePageSize + idx + 1;
        const nameZhDisplay = g.name_zh ? esc(g.name_zh) : '<span style="color: var(--text-secondary); opacity: 0.5;">-</span>';
        const playCount = Number(g.play_count || 0).toLocaleString();

        html += `
            <tr style="border-bottom: 1px solid rgba(255, 255, 255, 0.05); transition: background 0.2s;" onmouseenter="this.style.background='rgba(255,255,255,0.03)'" onmouseleave="this.style.background='transparent'">
                <td style="padding: 12px 14px; font-weight: 600; color: var(--text-secondary);">${rank}</td>
                <td style="padding: 12px 14px;">
                    <span style="font-weight: 700; color: var(--text-primary); cursor: pointer;" onclick="testSingleGenre('${esc(g.name)}')" title="点击测试归因">${esc(g.name)}</span>
                </td>
                <td style="padding: 12px 14px; color: #68d391; font-weight: 500;">
                    ${nameZhDisplay}
                </td>
                <td style="padding: 12px 14px; text-align: right; font-weight: 600; color: #63b3ed;">
                    ${playCount}
                </td>
                <td style="padding: 12px 14px; text-align: right;">
                    <button class="time-filter" style="padding: 4px 8px; font-size: 0.8em; margin-right: 6px;" onclick="openEditGenreModal(${g.id}, '${esc(g.name)}', '${esc(g.name_zh || '')}')">编辑</button>
                    <button class="time-filter" style="padding: 4px 8px; font-size: 0.8em; color: #e53e3e; border-color: rgba(229, 62, 62, 0.3);" onclick="confirmDeleteGenre(${g.id}, '${esc(g.name)}')">删除</button>
                </td>
            </tr>
        `;
    });

    html += `
                </tbody>
            </table>
        </div>
    `;

    content.innerHTML = html;
}

// 分页器刷新
function updateGenrePagination() {
    const totalPages = Math.max(1, Math.ceil(totalGenresCount / genrePageSize));
    const pageInfo = document.getElementById('genrePageInfo');
    const prevBtn = document.getElementById('genrePrevPage');
    const nextBtn = document.getElementById('genreNextPage');

    if (pageInfo) pageInfo.textContent = `第 ${currentGenrePage} 页 / 共 ${totalPages} 页 (总计 ${totalGenresCount} 个流派)`;
    if (prevBtn) prevBtn.disabled = currentGenrePage <= 1;
    if (nextBtn) nextBtn.disabled = currentGenrePage >= totalPages;
}

// 打开新增弹窗
function openCreateGenreModal() {
    document.getElementById('genreModalTitle').textContent = '新增标准流派';
    document.getElementById('genreFormId').value = '';
    const nameInput = document.getElementById('genreFormName');
    nameInput.value = '';
    nameInput.disabled = false;
    document.getElementById('genreFormNameZh').value = '';
    showModal('genreModal');
}

// 打开编辑弹窗
function openEditGenreModal(id, name, nameZh) {
    document.getElementById('genreModalTitle').textContent = `编辑流派：${name}`;
    document.getElementById('genreFormId').value = id;
    const nameInput = document.getElementById('genreFormName');
    nameInput.value = name;
    nameInput.disabled = false;
    document.getElementById('genreFormNameZh').value = nameZh || '';
    showModal('genreModal');
}

// 提交表单（新增/修改）
async function submitGenreForm() {
    const id = document.getElementById('genreFormId').value.trim();
    const name = document.getElementById('genreFormName').value.trim();
    const nameZh = document.getElementById('genreFormNameZh').value.trim();
    const submitBtn = document.getElementById('genreFormSubmitBtn');

    if (!name) {
        alert('请输入英文标准名');
        return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = '保存中...';

    try {
        const isEdit = Boolean(id);
        const url = isEdit ? `/api/genres/${id}` : '/api/genres';
        const method = isEdit ? 'PUT' : 'POST';

        const resp = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: name,
                name_zh: nameZh
            })
        });

        const data = await resp.json();
        if (!resp.ok) throw new Error(data.error || '操作失败');

        hideModal('genreModal');
        await loadGenreList(currentGenrePage);
    } catch (err) {
        alert('保存失败：' + (err.message || '未知错误'));
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = '保存';
    }
}

// 删除流派
async function confirmDeleteGenre(id, name) {
    if (!confirm(`确定要删除流派【${name}】吗？删除后该流派将从权威库移除，并自动刷新缓存。`)) {
        return;
    }

    try {
        const resp = await fetch(`/api/genres/${id}`, {
            method: 'DELETE'
        });
        const data = await resp.json();
        if (!resp.ok) throw new Error(data.error || '删除失败');

        await loadGenreList(currentGenrePage);
    } catch (err) {
        alert('删除失败：' + (err.message || '未知错误'));
    }
}

// 全量流派对账
async function triggerGenreReconcile() {
    const btn = document.getElementById('btnReconcileGenres');
    if (btn) {
        btn.disabled = true;
        btn.textContent = '对账中...';
    }

    try {
        const resp = await fetch('/api/genres/reconcile', {
            method: 'POST'
        });
        const data = await resp.json();
        if (!resp.ok) throw new Error(data.error || '对账请求失败');

        alert(data.message || '全量流派对账完成');
        await loadGenreList(1);
    } catch (err) {
        alert('流派对账失败：' + (err.message || '未知错误'));
    } finally {
        if (btn) {
            btn.disabled = false;
            btn.textContent = '🔄 流派对账';
        }
    }
}

// 流派沙盒解析测试
async function runGenreResolveTest(rawTag) {
    const clean = (rawTag || '').trim();
    const resultArea = document.getElementById('genreTestResultArea');
    if (!clean) {
        if (resultArea) resultArea.style.display = 'none';
        return;
    }

    try {
        const resp = await fetch('/api/genres/resolve-test', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ raw_genre: clean })
        });
        const data = await resp.json();
        if (!resp.ok) throw new Error(data.error || '测试解析失败');

        if (resultArea) {
            resultArea.style.display = 'block';
            document.getElementById('genreTestSegment').textContent = data.segment || '-';
            document.getElementById('genreTestCanonicalEng').textContent = data.canonical_eng || '(未命中权威英文)';
            document.getElementById('genreTestCanonicalZh').textContent = data.canonical_zh || '(无对应中文)';
            document.getElementById('genreTestNormalized').textContent = data.normalized || '-';

            const badge = document.getElementById('genreTestMatchBadge');
            if (badge) {
                if (data.is_matched) {
                    badge.className = 'queue-chip queue-chip-completed';
                    badge.textContent = '已认证匹配 (Matched)';
                    badge.style.background = 'rgba(72, 187, 120, 0.15)';
                    badge.style.color = '#68d391';
                } else {
                    badge.className = 'queue-chip queue-chip-failed';
                    badge.textContent = '未认证/未归因 (Unmatched)';
                    badge.style.background = 'rgba(245, 101, 101, 0.15)';
                    badge.style.color = '#fc8181';
                }
            }
        }
    } catch (err) {
        console.error('Genre resolve test failed:', err);
    }
}

// 点击表格行快速填入调试测试器
function testSingleGenre(name) {
    const input = document.getElementById('genreTestInput');
    if (input) {
        input.value = name;
        runGenreResolveTest(name);
        input.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
}

// DOM 事件监听初始化
document.addEventListener('DOMContentLoaded', () => {
    // 分页
    document.getElementById('genrePrevPage')?.addEventListener('click', () => {
        if (currentGenrePage > 1) loadGenreList(currentGenrePage - 1);
    });
    document.getElementById('genreNextPage')?.addEventListener('click', () => {
        loadGenreList(currentGenrePage + 1);
    });

    // 刷新
    document.getElementById('refreshGenreListBtn')?.addEventListener('click', () => {
        loadGenreList(1);
    });

    // 搜索框防抖
    document.getElementById('genreSearchInput')?.addEventListener('input', (e) => {
        currentGenreKeyword = e.target.value.trim();
        clearTimeout(genreSearchTimeout);
        genreSearchTimeout = setTimeout(() => {
            loadGenreList(1);
        }, 400);
    });

    // 排序选择
    document.getElementById('genreSortSelect')?.addEventListener('change', (e) => {
        currentGenreSort = e.target.value;
        loadGenreList(1);
    });

    // 打开新增弹窗
    document.getElementById('btnOpenCreateGenreModal')?.addEventListener('click', openCreateGenreModal);

    // 一键对账
    document.getElementById('btnReconcileGenres')?.addEventListener('click', triggerGenreReconcile);

    // 解析测试按钮与输入框即时防抖
    const testInput = document.getElementById('genreTestInput');
    const testBtn = document.getElementById('btnTestGenreResolve');
    
    if (testBtn && testInput) {
        testBtn.addEventListener('click', () => {
            runGenreResolveTest(testInput.value);
        });
        testInput.addEventListener('input', (e) => {
            clearTimeout(genreTestTimeout);
            genreTestTimeout = setTimeout(() => {
                runGenreResolveTest(e.target.value);
            }, 500);
        });
        testInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                runGenreResolveTest(testInput.value);
            }
        });
    }
});
