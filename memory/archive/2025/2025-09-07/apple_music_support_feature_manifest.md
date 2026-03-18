# Apple Music 支持特性清单

## 概述

此特性为项目添加了对 Apple Music 播放跟踪的支持，使用户能够将 Apple Music 的播放记录同步到 Last.fm。

## 功能范围

1. 实现 Apple Music 播放状态检查函数
2. 支持 Apple Music 播放信息的获取和处理
3. 将 Apple Music 播放记录保存到本地数据库
4. 支持将 Apple Music 播放记录同步到 Last.fm
5. 通过 WebSocket 实时推送 Apple Music 播放信息到前端
6. 支持检查和设置 Apple Music 中歌曲的点赞状态

## 实现要点

1. 在 `internal/scrobbler/track_check_playing.go` 中添加了 `AppleMusicCheckPlayingTrack` 函数
2. 修改了 `RoonCheckPlayingTrack` 函数以正确处理 Apple Music 和 Roon 的播放信息
3. 添加了 `mapedAppleMusic` 映射来跟踪已处理的 Apple Music 曲目，避免重复上报
4. 添加了 `isLongAppleMusic` 变量来控制 Apple Music 检查的轮询间隔
5. 在 `main.go` 中启动了 Apple Music 检查的 goroutine
6. 更新了数据库模型注释以包含 Apple Music 作为数据源
7. 在 `core/applemusic/sciprt.go` 中添加了检查和设置歌曲点赞状态的函数

## 技术细节

- 使用现有的 `exec.GetMRMediaNowPlaying()` 函数获取 Apple Music 播放信息
- 通过检查 `BundleIdentifier` 是否为 `com.apple.Music` 来识别 Apple Music 播放
- 使用与 Roon 和 Audirvana 相同的数据处理和上报逻辑
- 在 WebSocket 广播中使用 "apple music" 作为数据源标识
- 在数据库记录中使用 "Apple Music" 作为 Source 字段值
- 使用 AppleScript 命令 `favorited of current track` 检查歌曲是否被点赞
- 使用 AppleScript 命令 `set favorited of current track to true` 设置歌曲的点赞状态

## 并发安全

- 使用独立的 `mapedAppleMusic` 映射避免与其他播放源的键冲突
- 使用独立的 `isLongAppleMusic` 变量控制轮询间隔，避免影响其他播放源
- 使用现有的 `currentPlayingCache` sync.Map 来安全地存储播放缓存
- 使用原子操作 `pushCount.Add(1)` 来安全地更新计数器

## 数据存储

- 播放记录存储在 `track_play_records` 表中，Source 字段值为 "Apple Music"
- 播放统计存储在 `track_play_counts` 表中，使用现有的乐观锁机制保证并发安全
