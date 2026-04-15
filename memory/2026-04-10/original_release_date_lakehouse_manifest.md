# 专辑原始发布日期落库特性清单

## 特性摘要

`album.original_release_date` 作为独立事实字段正式落库，保持与现有 `release_date` 的语义分离。MusicBrainz 深度维护优先写入 `release-group.first-release-date`，Audirvana/exiftool 链路则将 `OriginalDate` 透传到 `TrackMetadata` 后再写入专辑，资料库同步也同步暴露该字段，避免把当前发行日误当作首发日期。

## 变更范围

- `internal/model/album.go` 增加 `OriginalReleaseDate`，并在合并时只允许空值被非空值填充。
- `internal/model/track.go`、`common/types.go`、`core/exec/exec.go`、`internal/scrobbler/*`、`api/server.go` 和 `common/converters/track_converter.go` 补齐原始发布日期从播放器/文件元数据到 `TrackMetadata` 的透传链路。
- `internal/logic/musicbrainz/service.go` 与 `internal/logic/pendingalbum/service.go` 在深度维护与手动维护时同步写入该字段，但不把它纳入专辑身份键。
- `internal/model/library_index.go`、`internal/model/init.go`、`internal/model/schema_album_original_release_date.go` 与 `internal/model/sql/ddl/album.sql` 补齐资料库索引、MySQL 修复与表结构定义。
- `internal/model/sql/repair/06_album_original_release_date.sql` 提供旧库手工补列脚本；相关测试 fixture 也同步补齐 `album` 表结构。

## 关键约束

- `original_release_date` 只能在来源明确提供时写入，不能用 `release_date` 直接冒充。
- 该字段不参与专辑身份匹配，仍然以 `artist + name + name_subtitle + release_date` 作为唯一性约束。
- 资料库同步与前端展示可以读取该字段，但不能依赖它反向推导专辑版本身份。

## 验证

- 已通过定点测试：
  - `TestGetOrCreateAlbumTxPrefersCuratedAlbumWhenReleaseDateMissing`
  - `TestGetOrCreateAlbumTxCreatesNewAlbumWhenIncomingReleaseDateDiffers`
  - `TestGetAlbumIndexRowsIncludesCoverFields`
  - `TestGetAlbumIndexRowsByIDsIncludesCoverFields`

## 说明

- 这次变更的核心是把首发日期从“展示/推断值”升级成“单独事实字段”，避免后续封面、列表、待归因和 MusicBrainz 回写链路继续混用当前发行日。
