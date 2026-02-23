# Apple Music AppleScript 支持

## 概述

此特性为项目添加了对 Apple Music 的 AppleScript 接口支持，替换了原有的 nowplaying-cli 实现，以提供更稳定和准确的播放信息获取。

## 功能范围

1. 实现 Apple Music 的 AppleScript 接口访问
2. 支持通过 AppleScript 获取 Apple Music 的播放状态
3. 支持通过 AppleScript 获取当前播放曲目的详细信息（标题、专辑、艺术家、时长、播放位置）
4. 替换原有的 nowplaying-cli 实现，提供更可靠的播放信息获取方式

## 实现要点

1. 创建了 `core/applemusic` 包，包含 AppleScript 接口实现
2. 实现了 `IsRunning` 函数，用于检查 Apple Music 应用是否正在运行
3. 实现了 `GetState` 函数，用于获取 Apple Music 的播放状态
4. 实现了 `GetNowPlayingTrackInfo` 函数，用于获取当前播放曲目的详细信息
5. 修改了 `internal/scrobbler/track_check_playing.go` 中的 `AppleMusicCheckPlayingTrack` 函数，使用新的 AppleScript 实现
6. 添加了相应的单元测试

## 技术细节

- 使用 `github.com/andybrewer/mack` 库执行 AppleScript 命令
- 通过 `tell application "Music"` 命令与 Apple Music 应用交互
- 获取的播放信息包括：曲目标题、专辑、艺术家、时长（秒）和当前播放位置（秒）
- 时长和播放位置都以秒为单位，便于计算播放进度百分比

## AppleScript 命令

- 检查应用是否运行：`tell application "System Events" to name of every application process`
- 获取播放状态：`tell application "Music" to get player state`
- 获取播放信息：`tell application "Music" to get {name, album, artist, duration, player position} of current track`

## 并发安全

- 与项目中其他播放源（Audirvana、Roon）一样，使用独立的映射和变量来跟踪状态
- 使用现有的并发安全机制处理播放信息缓存和 WebSocket 广播

## 数据存储

- 播放记录存储在 `track_play_records` 表中，Source 字段值为 "Apple Music"
- 播放统计存储在 `track_play_counts` 表中，使用现有的乐观锁机制保证并发安全