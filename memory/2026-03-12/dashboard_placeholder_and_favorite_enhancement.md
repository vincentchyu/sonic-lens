# 2026-03-12 特性清单：专辑占位曲目显示与收藏系统三态增强

## 1. 专辑曲目展示完整性 (Dashboard)

### 1.1 核心变更
- **遍历逻辑重构**：修改 `templates/dashboard.html` 中的 `showAlbumDetails` 函数，将渲染基础从 `data.tracks` 切换为 `data.track_album`，确保未播放但已存在的物理曲目能够被完整展示。
- **占位符处理**：针对 `track_id: 0` 的曲目，应用灰度滤镜（grayscale）与半透明效果，并标注“待完善 (未播放)”状态，实现视觉区分。
- **安全加固**：引入 `esc(s)` 函数处理动态 JS 参数转义，解决了因曲目名含单双引号导致的 `Unexpected EOF` 前端崩溃问题。
- **代码除重**：识别并清理了 `dashboard.html` 中第 6817 行附件的重复定义。

## 2. 收藏系统三态增强 (Lyrics Live)

### 2.1 歌词页星星状态展示
- **三态逻辑映射**：
    - **无收藏**：空心 SVG 星标。
    - **半星 (左实右虚)**：仅收藏了 Apple Music 或 Last.fm 其中之一。
    - **全实心星**：双平台均已收藏。
- **交互优化**：允许在半星状态下点击按钮进行“补全收藏”，提升双平台同步率。

### 2.2 后端逻辑对齐
- **状态统一返回**：重构 `internal/logic/track/service.go` 中的 `SetTrackFavorite`，确保无论操作源为何，均主动查询并返回 Apple Music 与 Last.fm 的最新布尔对，不再返回伪造的 `false`。

## 3. 数据库与元数据加固

### 3.1 索引与匹配优化
- **Track 表索引**：适配 `track_number` 与 `disc_number` 复合索引，更新了相关更新/写入逻辑的 `Where` 条件。
- **MusicBrainz 匹配**：将 MB 候选映射 Key 升级为 `disc|number|name` 复合形式，解决了多碟版专辑中同名曲目的冲突覆盖。
