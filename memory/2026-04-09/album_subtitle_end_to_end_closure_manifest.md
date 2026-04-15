# 专辑副标题贯通闭环特性清单

## 特性摘要

专辑“版本说明/副标题”现在正式从展示层临时字符串，升级为资料库与同步链路的稳定字段。后端新增 `album.name_subtitle` 及多张关联表的 `album_subtitle`，播放流水、待归因专辑、排行榜、D1 同步、Bridge 本地索引和三端展示统一消费该字段，避免 Deluxe / Remaster / Anniversary 等信息只存在于原始专辑名里，导致列表、详情、最近播放和当前播放对同一张专辑显示不一致。

## 变更范围

- `common/album_title_v3.go` 收敛专辑标题解析，识别常见 `Deluxe / Remaster / Anniversary / Live / Soundtrack / Mix / Version` 版本尾缀，并保留不应拆分的括号内容。
- `internal/model/schema_album_title_subtitle.go` 为 `album`、`track`、`track_play_records`、`track_favorite_event`、`pending_album_work_item`、`top_album_stat` 补齐副标题字段。
- `internal/model/album.go`、`track.go`、`track_play_record.go`、`pending_album_work_item.go` 将副标题纳入专辑/播放/待归因上下文的持久化与回填流程。
- `internal/model/dashboard_stat.go` 和 `internal/sync/d1_sync.go` 让热门专辑统计、最近播放、D1 侧镜像表同步带上 `album_subtitle`。
- Bridge 共享模型与本地索引补齐 `name_subtitle` / `album_subtitle`，`AlbumGridView`、`AlbumDetailView`、`LibraryView`、`NowPlayingView` 等入口统一展示格式化后的专辑名。

## 关键约束

- 专辑稳定主名与版本说明必须拆开保存：主名落 `album.name`，版本说明落 `name_subtitle` / `album_subtitle`，禁止后续再把两者重新糊回单字段作为唯一事实源。
- 播放链路、收藏待归因、排行榜统计和 D1 同步都必须透传副标题；不能只在详情页做前端字符串拼接，否则不同入口会再次出现专辑名漂移。
- Bridge 资料库本地 SQLite 与服务端 `/api/library/sync` 必须同步维护 `name_subtitle`，列表和详情统一使用展示名计算属性，而不是各页面各自手搓拼接规则。
- 当前播放、最近播放、热门专辑等轻量摘要接口必须优先消费结构化副标题字段，避免再次回退到从原始标题字符串里临时猜测版本信息。

## 验证

- 这次只更新记忆文件，未额外运行测试。

## 说明

- 这条记忆的核心不是“某个页面显示了括号”，而是专辑副标题已经成为跨后端、同步层和 Bridge 客户端的正式数据契约，后续新增接口或资料库功能时都应默认带上这组字段。
