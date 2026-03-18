# 曲目五元组与专辑闭环修复特性清单

## 背景

`track` 主表的唯一身份已经从旧三元组 `(artist, album, track)` 扩展为五元组 `(artist, album, track, disc_number, track_number)`，但 `track_album`、专辑初始化和 MusicBrainz 深度维护链路仍保留旧的按歌名匹配逻辑。在多碟专辑中，如果不同碟存在同名且同曲序的曲目，旧逻辑会出现占位符误绑定、`track_album` 物理位置错位以及专辑详情关联不稳定的问题。

## 本次修复

### 1. 曲目与专辑关联闭环统一为“物理位置优先”

- 在 `internal/model/track_album.go` 中新增：
  - `TrackAlbumPlaceholderLookup`
  - `FindTrackAlbumByPositionTx`
  - `FindTrackAlbumPlaceholderTx`
  - `upsertTrackAlbumTx`
- 新策略统一为：
  1. 优先按 `(album_id, disc_number, track_number)` 匹配同一专辑内的物理位置
  2. 仅在物理位置缺失时，才退回按 `track` 名称兜底
  3. 已有占位符被真实曲目命中时，优先“转正”占位符并回收重复行
  4. 如果同一专辑物理位置已被其他真实曲目占用，则显式报冲突

### 2. 播放写入链路与专辑创建闭环打通

- `internal/model/track.go` 中：
  - 专辑创建不再直接按 `artist + name` `FirstOrCreate`
  - 统一走 `getOrCreateAlbumTx`
  - 优先使用 `track_album` 中已有的物理位置占位符来补全 `track_number` / `disc_number`
  - 绑定 `track_album` 时不再直接按歌名查占位符，而是优先按碟号+曲号定位
- 修复了一个旧 bug：
  - `DiscNumber` 回填条件误写成了 `ta.TrackNumber <= 0`
  - 现已更正为 `ta.DiscNumber <= 0`

### 3. 专辑初始化与 MusicBrainz 维护不再丢碟号

- `internal/logic/musicbrainz/service.go` 中：
  - `InitializeAlbums` 建立 `track_album` 时同步写入：
    - `Track`
    - `TrackNumber`
    - `DiscNumber`
    - `MusicBrainzRecordingID`
  - `DeepingMaintenance` 在对齐未听过曲目的占位符时：
    - 先按 `(album_id, disc_number, track_number)` 查找占位记录
    - 找不到时才退回按歌名兜底

### 4. Album 创建策略统一为“精确优先，模糊收口”

- `internal/model/album.go` 中：
  - `GetOrCreateAlbum` 改为：
    1. 优先按 `(artist, name, release_date)` 精确命中
    2. 再按 `(artist, name)` 回收已有专辑
    3. 如果已有记录缺少 `release_date/genre/country/status/packaging/barcode/disc_infos`，则补齐
- 这避免了“日常播放创建 album”与“后续 MusicBrainz 初始化补数创建 album”拆成两条记录

### 5. Schema 与 DDL 同步

- `internal/model/schema_track_identity.go`
  - 运行时为 `track_album` 增加辅助索引 `idx_ta_album_disc_track (album_id, disc_number, track_number)`
- `internal/model/sql/ddl/track_album.sql`
  - 同步增加上述索引

## 设计约束说明

当前 `track_album` 允许 `TrackID = 0` 作为 MusicBrainz 占位符，因此**不适合直接给 `(track_id, album_id)` 加数据库唯一键**。否则同一专辑下多个未听过曲目的占位行都会冲突。  
因此本次对“物理位置唯一”的约束，采取的是：

- 数据库层提供 `(album_id, disc_number, track_number)` 辅助索引
- 应用层 `upsertTrackAlbumTx` 执行冲突检查和占位符回收

如果未来要上更强的数据库唯一约束，需要先把占位符设计从 `track_id = 0` 升级为可区分的独立实体或允许 `NULL track_id` 配合条件约束。

## 后续维护规则

后续凡是涉及专辑曲目绑定、占位符消耗、MusicBrainz 深度维护、详情页曲目展示排序的改动，必须遵守以下规则：

1. 同一专辑内的曲目身份优先使用 `disc_number + track_number`
2. `track` 名称只能做兜底，不可作为多碟专辑内的主匹配条件
3. 所有 `track_album` 写入都应复用模型层统一 helper，不能在逻辑层直接散写 SQL
