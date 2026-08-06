# 专辑发行格式后缀（- EP/Single/LP）整改特性清单

---

## 1. 业务背景
在 Apple Music 上报曲目时，单曲与 EP 的专辑字段常带有连字符后缀（如 `Tell Them, I'm Here - EP` 或 `Calzaghe - Single`）。这种不规范的命名会在系统中引发一连串副作用：
- 数据库中以脏名写入 `album` / `track` 表，污染元数据。
- 检索 MusicBrainz 时无法查找到精确的结果（MusicBrainz 的专辑名一般为干净主名，发行格式由 primarytype/type 表征）。
- 归因分析错误，无法将带有后缀的播放数据归并到已精选完毕的干净专辑下。

---

## 2. 解决方案与修改模块

### 2.1 层次一：防增量（实时解析与 Scrobbler 第一关）
- **`common/album_title_v3.go`**:
  在 `ParseAlbumTitleMetadata` 中增加支持连字符类型后缀（` - EP`, ` - Single`, ` - LP`）的提取。剥离后缀后的部分即为 `OfficialTitle`。
- **`internal/scrobbler/scrobbler_player_apple_music.go`**:
  `GetAlbum()` 入口改用 `ParseAlbumTitleMetadata().OfficialTitle`。这是 Apple Music 实时上报播放历史的第一道关卡，保证数据流入 `track_play_records` 前就已经去除了格式后缀。
- **`internal/model/track_play_record.go`** / **`internal/logic/track/playback.go`**:
  在播放流水表 `track_play_records` 中，新增了 `release_type`（VARCHAR(20)）字段，在 Handle 播放落库时同步写入从上报专辑中裁剪出的格式类型。这确保了流水在归因成功（resolved）前，能够完备地保留最初上报时的发行类型这一核心分析线索。

### 2.2 层次二：数据库 DAO 层自动规一
- **`internal/model/album.go`**:
  在 `Album` 表中新增 `release_type`（VARCHAR(20)）字段以承接发行格式枚举。在 `getOrCreateAlbumTx` 内部，通过 `normalizeAlbumReleaseTypeSuffix` 工具实现落库前自动剥离，让 `name` 保存为干净标题，`release_type` 承载实际格式。
- **`internal/model/track.go`**:
  `normalizeTrackForStorage` 与 `normalizeTrackIdentity` 同步更新，针对 `Album` 做同样的剥离，确保 track 写入和查询策略的规范一致。

### 2.3 层次三：MusicBrainz 检索与待归因工作台适配
- **`internal/logic/musicbrainz/service.go`**:
  封装包级别的通用两阶段 Releases 检索方法 `SearchMBReleases`：若具有明确的 `release_type`，第一阶段使用 `primarytype` 约束条件精准搜索（防止 EP 匹配到同名全长 Album）；若第一阶段无结果或本来就没有格式限制，第二阶段降级为宽松检索。
- **`internal/logic/pendingalbum/service.go`**:
  - `SearchPendingAlbumMBReleases` 检索动作直接复用 `musicbrainz.SearchMBReleases`，精简代码，保证两阶段检索的单一事实源。
  - `resolveMusicBrainzMaterial` 预览数据生成时，将原始 `detail.WorkItem.Album` 后缀剥离后输出干净的候选专辑名并填充 `ReleaseType`，防止预览比对画面因未剥离而造成混淆。
  - `resolveManualMaterial` 手动维护阶段，修复用户因前端表单不含有发行格式属性导致 `ReleaseType` 存储为空的问题，通过从原始 `detail.WorkItem.Album` 提取发行后缀自动赋给 `AlbumCandidate.ReleaseType` 从而完成落库。

### 2.4 层次四：历史数据清洗工具与锁定绕过
- **`internal/model/album_cleanup.go`**:
  实现一键清洗逻辑 `CleanupReleaseTypeSuffixes`。该工具扫描历史带有连字符发行格式的专辑，剥离后缀后寻找目标干净专辑。
- **`internal/model/track_album.go`**:
  重构 `upsertTrackAlbumTx` 引入 `force` 控制标志。在常规播放写库时，SyncStatus=3/4 的精选专辑受到 `layoutLocked` 保护禁止改写；而在清洗重构合并时，传入 `force=true` 强行绕过该拦截以将同名专辑曲目完成合并迁移。
- **`cmd/album_cleanup.go`**:
  注册 CLI 命令 `cleanup-release-type-suffixes`，支持 `--apply` 和 `--limit`。

### 2.5 层次五：展示与渲染闭环
- **`templates/admin/modals.html`** / **`static/admin/init.js`** (Web):
  在详情面板中新增 `ad-release-type` 发行格式 DOM 节点，并在获取响应的回调中将 `data.release_type` 进行大写反显。
- **`soniclens-bridge/SoniclensCore/Models/AlbumDetailModels.swift`** / **`soniclens-bridge/SoniclensBridge/Views/AlbumDetailView.swift`** (iOS/macOS 客户端):
  在 Swift `AlbumDetail` 模型中追加 `releaseType` 字段实现数据映射解析；并在 `AlbumDetailView` 详情面板的 `metaFlow` 中，追加 `AlbumMetaChip` 卡片来反显大写格式类型。

---

## 3. 架构约束与陷阱规避
1. **防增量第一关**：新增上报组件或解析组件时，必须统一在最前端（如 Wrapper 侧）采用 `ParseAlbumTitleMetadata` 处理，禁止直接以 raw string 入库。
2. **重构绕过**：对于精选锁定（SyncStatus=3）的专辑物理布局保护，合并逻辑通过 `upsertTrackAlbumTx` 的 `force` 标志显示声明绕过，避免直接用 GORM Save 造成重复坐标记录，同时也避开了只读拦截导致的静默抛弃。
3. **单元测试 Schema 完整性**：测试环境中手写 GORM `CREATE TABLE` 或 sqlmock 期望的 Expectations UPDATE/INSERT 必须同步补充 `release_type` 字段以及锁定 SELECT 的判定，避免 schema 倾斜造成单元测试报错。
