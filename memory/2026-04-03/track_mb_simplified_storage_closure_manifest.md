# 曲目 MB 深度维护简体写入闭环特性清单

## 特性摘要

MusicBrainz 深度维护与待归因专辑的精选回写链路，现已在 `track` 写入前统一执行繁简与标点归一，避免把 MusicBrainz 返回的繁体标题、艺人名和专辑名直接落入数据库。`GetOrCreateTrackByIdentityTx` 还补了简体身份回退匹配，确保旧有简体存量行不会因为新规则而被重复创建。

## 变更范围

- `internal/model/track.go` 新增统一的 `track` 写入归一化工具
- `GetOrCreateTrackByIdentityTx` 新增简体身份回退与创建前归一化
- `UpdateTrackCuratedMetadataTx` 在回写 `artist` / `album` / `track` 时统一做简体化
- `UpdateTrackWithTrackMetadata` 和播放写入路径同步收口字符串归一化
- `internal/model/track_dao_test.go` 增加繁体到简体的写入回归测试

## 关键约束

- MusicBrainz 维护回写到 `track` 时，标题、艺人、专辑、专辑艺人、流派、作曲人都必须写入简体并统一标点
- 只允许在 DAO 层处理数据库写入归一化，不要把简体化逻辑散落到 dashboard 或 logic 里
- 对已经存在的简体存量 `track`，创建前必须先做简体身份回退匹配，避免重复建行

## 验证

- `go test ./internal/model -run 'TestUpdateTrackCuratedMetadataTxNormalizesChineseText|TestGetOrCreateTrackByIdentityTxNormalizesCreatedTrackText'`
- `go test ./internal/model -run 'TestGetTrackByIDTx|TestGetTrackByIDTxPropagatesNotFound|TestUpdateTrackMusicBrainzPositionTx|TestUpdateTrackMusicBrainzMetadataTx|TestUpdateTrackCuratedMetadataTxNormalizesChineseText|TestGetOrCreateTrackByIdentityTxNormalizesCreatedTrackText'`
- `go test ./internal/logic/musicbrainz ./internal/logic/pendingalbum`
