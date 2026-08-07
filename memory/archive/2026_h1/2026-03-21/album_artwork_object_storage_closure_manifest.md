# Album Artwork Object Storage Closure（2026-03-21）

## 背景

- 原链路将播放器封面缓存到进程内 `core/artwork` map，重启后失效，且每次命中不到缓存时都要再次执行播放器指令提取封面。
- 目标是把封面能力切到 S3 兼容对象存储（本地 MinIO、未来可替换 R2），并把封面地址沉淀到 `album` 维度。

## 本次改动

1. 新增对象存储基础设施层：
   - `core/objectstorage` 增加 `Provider` 抽象与 `S3Provider` 实现。
   - 支持能力：`CheckObjectExists`、`UploadFileToObject`、`UploadBytesToObject`、`DeleteObject`、`DeleteObjects`、`GetObjectCDNURL`。
   - 启动阶段在 `main.go` 执行 `objectstorage.Init`，初始化失败时自动降级到原内存封面方案，不中断服务。

2. 新增配置：
   - `config.ObjectStorageConfig` 与 `objectStorage` YAML 节点。
   - 支持环境变量覆盖（含用户约定键）：`ENDPOINT`、`BUCKET`、`REGION`、`ACCESS_KEY_ID`、`SECRET_ACCESS_KEY`、`CDN_URL`、`BASE_PREFIX`、`ORIGINAL_PREFIX`、`THUMBNAIL_PREFIX`。

3. Scrobbler 封面链路改造：
   - `resolveArtwork` 优先按专辑维度键（`albumArtist/artist + album`）计算对象键并 `HeadObject`。
   - 已存在对象时直接返回对象服务相对路径（不带域名），不再执行提取命令。
   - 不存在时再调用播放器提取封面并上传对象存储；上传失败则回退到 `core/artwork` 内存缓存。
   - 对外字段统一返回无域名相对路径，前端按当前服务地址拼接，客户端按 Bonjour 嗅探地址拼接。

4. 专辑封面落库闭环：
   - `album` 新增字段：`cover_art_url`、`cover_art_mime`、`cover_art_object_key`。
   - 新增 DAO：`UpsertAlbumCoverByID/UpsertAlbumCoverByIDTx`。
   - 在 `HandleTrackPlaybackThreshold` 的播放落库成功后，基于 `track_play_records.album_id` 回填专辑封面字段。

5. 数据库与迁移：
   - 新增 `internal/model/schema_album_cover.go`，MySQL 启动时确保新增字段和索引存在（兼容非 `isDev` 场景）。
   - 更新 `internal/model/sql/ddl/album.sql` 对应最新实库结构。

## 验证

- `go test ./internal/model -run 'TestGetOrCreateAlbumTxPrefersCuratedAlbumWhenReleaseDateMissing|TestGetOrCreateAlbumTxReusesExistingAlbumWhenIncomingReleaseDateDiffers|TestUpdateAlbumSyncStatusTx|TestUpdateAlbumFieldsTx|TestUpsertAlbumCoverByIDTx'`
- `go test ./internal/logic/track/... ./internal/scrobbler/...`

## 注意事项

- 对象存储 URL 默认按 `endpoint/bucket/key` 生成；若配置 `cdnUrl` 则优先返回 CDN 地址。
- 当前 `CheckObjectExists` 使用 `HeadObject`；若将来需要图片压缩/缩略图，可在 `thumbnailPrefix` 下扩展异步生成链路。

## 2026-03-21 补充：封面解析兜底接口

- 新增 `GET /api/artwork/resolve`，参数支持 `albumArtist`、`artist`、`album`、`artworkKey`，并兼容可选 `album_id/albumID`。
- 解析顺序固定为：
  1) 优先查 `album` 表（`album_id` 命中，或按 `(albumArtist|artist, album)` 命中）且已有 `cover_art_url` 时直接返回；
  2) 再按 scrobbler 同款“专辑身份种子”算法（`albumArtist -> artist` + `album`）计算对象键并检查对象存储；
  3) 最后用 `artworkKey` 作为种子计算对象键兜底检查。
- 响应统一为相对路径/相对 key：`{ exists, cover_art_url, cover_art_object_key }`。
- 当请求携带 `album_id/albumID` 且通过步骤 2/3 命中对象时，会异步回填 `album.cover_art_url` 与 `album.cover_art_object_key`，收敛后续列表/详情/正在播放的封面来源。
