# subtitle 身份索引闭环特性清单

## 特性摘要

专辑与曲目的 `subtitle` 这次不再只是展示层字段，而是正式进入身份索引与资料库联结条件。`album`、`track`、`track_favorite_event`、`track_play_records` 的 MySQL 唯一键/查询索引已经同步升级，资料库索引、最近播放 fallback join 也一起改成 subtitle-aware，避免 Deluxe / Remaster / Anniversary 等不同版本继续被串绑到同一条专辑或曲目关系上。

## 变更范围

- `internal/model/album.go` 与 `internal/model/sql/ddl/album.sql` 将专辑身份索引升级为 `artist + name + name_subtitle + release_date`。
- `internal/model/track.go` 与 `internal/model/sql/ddl/track.sql` 将曲目唯一索引升级为包含 `album_subtitle` 的版本。
- `internal/model/track_favorite_event.go`、`internal/model/track_play_record.go` 以及对应 DDL 补齐 subtitle-aware 复合索引。
- `internal/model/library_index.go` 与最近播放 fallback join 改为按 `album_subtitle` 联结，避免同名不同版本的播放统计和封面回退再次混版。
- `internal/model/schema_*.go` 和 `internal/model/sql/repair/05_subtitle_identity_indexes.sql` 补齐 MySQL 自动升级与手工修复脚本。

## 关键约束

- `release_date` 不能被简单视为可删字段；在专辑身份里它仍然是版本区分的一部分，不能因为加了 `name_subtitle` 就把不同发行日期的版本合并掉。
- `track`、`track_play_records`、`track_favorite_event` 的 subtitle 字段都必须参与身份/回填条件，不能再只按 `artist + album + track` 做弱匹配。
- 资料库列表、最近播放和收藏待归因都必须复用 subtitle-aware 的 join / index 组合，否则展示层还会继续出现错版聚合。

## 验证

- 已通过定点测试：
  - `TestGetAlbumIndexRowsIncludesCoverFields`
  - `TestGetAlbumIndexRowsByIDsIncludesCoverFields`
  - `TestGetRecentPlayRecordsReturnsCoverArtPath`
  - `TestGetOrCreateAlbumTxPrefersCuratedAlbumWhenReleaseDateMissing`
  - `TestGetOrCreateAlbumTxCreatesNewAlbumWhenIncomingReleaseDateDiffers`

## 说明

- 这次修复的重点不是“多加一个 subtitle 字段”，而是把 subtitle 变成真正能防串版的身份约束，并把 MySQL 迁移、DDL、测试和记忆一起补齐。
