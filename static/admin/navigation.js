// --- 新增：导航切换逻辑 ---
    function showSection(sectionId) {
        currentSectionID = sectionId;
        // 隐藏所有主内容区域
        const sections = [
            '.stats-container', '.charts-container', '.rankings-container',
            '#unscrobbledContainer', '#insightListContainer', '#insightJobListContainer',
            '#albumListContainer', '#artistListContainer', '#pendingAlbumListContainer', '#trackListContainer'
        ];
        
        sections.forEach(selector => {
            const el = document.querySelector(selector);
            if (el) el.style.display = 'none';
        });

        // 取消所有导航激活状态
        // 搜索框防抖
        const pendingWorkSearchInput = document.getElementById('pendingWorkSearchInput');
        if (pendingWorkSearchInput) {
            pendingWorkSearchInput.addEventListener('input', (e) => {
                currentPendingWorkKeyword = e.target.value.trim();
                clearTimeout(pendingWorkSearchTimeout);
                pendingWorkSearchTimeout = setTimeout(() => {
                    loadPendingWorkItems(1);
                }, 500);
            });
        }

        // 分页按钮
        const prevBtn = document.getElementById('pendingWorkPrevPage');
        const nextBtn = document.getElementById('pendingWorkNextPage');
        if (prevBtn) {
            prevBtn.onclick = () => {
                if (currentPendingWorkPage > 1) loadPendingWorkItems(currentPendingWorkPage - 1);
            };
        }
        if (nextBtn) {
            nextBtn.onclick = () => {
                const totalPages = Math.ceil(totalPendingWorkCount / pendingWorkPageSize);
                if (currentPendingWorkPage < totalPages) loadPendingWorkItems(currentPendingWorkPage + 1);
            };
        }

        document.querySelectorAll('.menu a').forEach(a => {
            a.classList.remove('active');
        });

        const librarySections = new Set(['albumList', 'artistList', 'pendingAlbumList', 'trackList']);
        const details = document.getElementById('libraryMenuDetails');
        if (details) {
            details.open = librarySections.has(sectionId);
        }
        const librarySummary = document.querySelector('.library-summary');
        if (librarySummary) {
            librarySummary.classList.toggle('active', librarySections.has(sectionId));
        }

        // 根据 sectionId 显示对应区域
        if (sectionId === 'dashboard') {
            document.querySelector('.stats-container').style.display = 'grid';
            document.querySelector('.charts-container').style.display = 'grid';
            document.querySelector('.rankings-container').style.display = 'grid';
            document.getElementById('dashboardTab').classList.add('active');
        } else {
            const container = document.getElementById(sectionId + 'Container');
            if (container) container.style.display = 'block';
            
            const tab = document.getElementById(sectionId + 'Tab');
            if (tab) tab.classList.add('active');

            // 特殊逻辑：如果是特定列表，则加载数据
            if (sectionId === 'albumList') loadAlbumList();
            if (sectionId === 'artistList') {
                loadArtistList();
                loadArtistSourceOptions();
            }
            if (sectionId === 'pendingAlbumList') loadPendingAlbumList();
            if (sectionId === 'trackList') loadTrackList();
            if (sectionId === 'insightList') loadInsightList();
            if (sectionId === 'insightJobList') {
                updateInsightJobTabLayout();
                loadInsightJobList();
            }
            if (sectionId === 'unscrobbled') loadUnscrobbledRecords();
        }
    }

    // 处理侧边栏导航点击 (DaisyUI 版)
    document.querySelectorAll('.menu a').forEach(a => {
        a.addEventListener('click', (e) => {
            const id = a.id;
            if (!id) return;
            
            // 只有 Tab 类项才执行切换
            if (id.endsWith('Tab')) {
                e.preventDefault();
                const sectionMap = {
                    'dashboardTab': 'dashboard',
                    'albumListTab': 'albumList',
                    'artistListTab': 'artistList',
                    'pendingAlbumListTab': 'pendingAlbumList',
                    'trackListTab': 'trackList',
                    'insightListTab': 'insightList',
                    'insightJobListTab': 'insightJobList',
                    'unscrobbledTab': 'unscrobbled'
                };
                
                const section = sectionMap[id];
                if (section) {
                    showSection(section);
                    // 更新子导航激活状态 (DaisyUI 类)
                    document.querySelectorAll('.menu a').forEach(el => el.classList.remove('active'));
                    a.classList.add('active');
                }
            }
        });
    });

    const librarySummary = document.querySelector('.library-summary');
    if (librarySummary) {
        librarySummary.addEventListener('click', (e) => {
            if (!window.matchMedia('(max-width: 768px)').matches) return;
            e.preventDefault();
            showSection('albumList');
        });
    }

    // 默认激活仪表盘
    showSection('dashboard');

    // --- 新编：专辑列表加载逻辑 ---
