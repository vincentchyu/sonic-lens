# Last.fm 收藏功能实现特性清单

## 概述

本次特性开发实现了Last.fm音乐收藏（喜爱）功能，包括检查歌曲是否被收藏和设置歌曲收藏状态的功能。

## 功能实现

### 1. IsFavorite 函数

- **功能**: 检查指定艺术家和歌曲是否在Last.fm中被标记为喜爱
- **参数**:
    - ctx: 上下文
    - artist: 艺术家名称
    - track: 歌曲名称
- **返回值**:
    - bool: 是否被收藏
    - error: 错误信息
- **实现细节**:
    - 调用Last.fm API的`track.getInfo`方法获取歌曲信息
    - 检查返回结果中的`UserLoved`字段判断是否被收藏
    - 添加了API初始化检查，防止空指针异常
    - 添加了详细的日志记录

### 2. SetFavorite 函数

- **功能**: 设置指定艺术家和歌曲在Last.fm中的收藏状态
- **参数**:
    - ctx: 上下文
    - artist: 艺术家名称
    - track: 歌曲名称
    - favorited: 收藏状态（true为收藏，false为取消收藏）
- **返回值**:
    - error: 错误信息
- **实现细节**:
    - 根据favorited参数调用Last.fm API的`track.love`或`track.unlove`方法
    - 添加了API初始化检查，防止空指针异常
    - 添加了详细的日志记录

## 代码改进

### 1. 错误处理增强

- 为所有Last.fm API调用函数添加了API初始化检查
- 防止在API未初始化时出现空指针异常

### 2. 日志记录完善

- 为所有函数添加了详细的日志记录
- 包括函数调用、参数信息和错误信息

## 测试

### 1. 单元测试

- 为IsFavorite和SetFavorite函数创建了单元测试
- 测试考虑了API未初始化的情况，确保在测试环境中不会失败
- 保持了与现有测试的一致性

### 2. 现有功能保护

- 为现有的PushTrackScrobble和TrackUpdateNowPlaying函数也添加了API初始化检查
- 更新了相关测试以适应新的错误处理机制

## 文件变更

1. `core/lastfm/lastfm.go`:
    - 实现了IsFavorite和SetFavorite函数
    - 为现有函数添加了API初始化检查

2. `core/lastfm/lastfm_favorites_test.go`:
    - 新增了IsFavorite和SetFavorite的单元测试

3. `core/lastfm/lastfm_test.go`:
    - 更新了PushTrackScrobble测试以适应新的错误处理机制
    - 改进了配置文件加载的错误处理

## 使用示例

```go
// 检查歌曲是否被收藏
isLoved, err := lastfm.IsFavorite(ctx, "Coldplay", "Viva La Vida")
if err != nil {
    // 处理错误
}

// 收藏歌曲
err = lastfm.SetFavorite(ctx, "Coldplay", "Viva La Vida", true)
if err != nil {
    // 处理错误
}

// 取消收藏歌曲
err = lastfm.SetFavorite(ctx, "Coldplay", "Viva La Vida", false)
if err != nil {
    // 处理错误
}
```