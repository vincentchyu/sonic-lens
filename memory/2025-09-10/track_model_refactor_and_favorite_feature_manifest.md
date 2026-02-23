# Track Model重构和喜欢状态功能实现

## 特性摘要

本次更新重构了TrackPlayCount模型为新的Track模型，并增加了对Apple Music和Last.fm喜欢状态的支持。

## 功能变更

### 1. 数据库模型变更

- 创建了新的`Track`模型，替代原有的`TrackPlayCount`模型
- 在新模型中增加了两个布尔字段：
    - `IsAppleMusicFav`: 表示曲目是否在Apple Music中被标记为喜欢
    - `IsLastFmFav`: 表示曲目是否在Last.fm中被标记为喜欢
- 表名从`track_play_counts`变更为`tracks`
- 保留了原有的播放统计功能（PlayCount）

### 2. 数据库迁移

- 更新了`init.go`文件，将自动迁移的模型从`TrackPlayCount`改为`Track`
- 保留了`TrackPlayRecord`模型不变

### 3. Model层功能增强

- 在新的`Track`模型中实现了以下方法：
    - `SetAppleMusicFavorite`: 更新Apple Music喜欢状态
    - `SetLastFmFavorite`: 更新Last.fm喜欢状态
    - `GetAppleMusicFavorite`: 获取Apple Music喜欢状态
    - `GetLastFmFavorite`: 获取Last.fm喜欢状态
- 所有更新操作都使用乐观锁机制保证并发安全

### 4. Service层接口扩展

- 在`TrackService`接口中增加了以下方法：
    - `SetAppleMusicFavorite`
    - `SetLastFmFavorite`
    - `GetAppleMusicFavorite`
    - `GetLastFmFavorite`
- 在`TrackServiceImpl`中实现了这些新方法

### 5. 向后兼容性

- 将原有的`TrackPlayCount`模型标记为已弃用（DEPRECATED）
- 保留了原有的方法实现，但添加了弃用标记和说明
- 现有代码可以继续使用，但建议迁移到新的Track模型

## 实现细节

### 乐观锁机制

所有更新操作（播放次数增加、喜欢状态更新）都使用乐观锁机制：

1. 查询时获取当前记录及其版本号
2. 更新时同时更新数据和版本号
3. 更新时添加版本号条件，确保只有版本号匹配时才能更新成功
4. 如果更新失败（版本号不匹配），则重试整个操作

### 数据验证

所有涉及艺术家、专辑、曲目信息的操作都通过`common.ValidateTrackInfo`进行验证，确保数据完整性。

## 使用示例

```go
// 增加播放次数
err := model.IncrementTrackPlayCount(ctx, "Artist Name", "Album Name", "Track Name")

// 设置Apple Music喜欢状态
err := model.SetAppleMusicFavorite(ctx, "Artist Name", "Album Name", "Track Name", true)

// 设置Last.fm喜欢状态
err := model.SetLastFmFavorite(ctx, "Artist Name", "Album Name", "Track Name", true)

// 获取喜欢状态
isAppleMusicFav, err := model.GetAppleMusicFavorite(ctx, "Artist Name", "Album Name", "Track Name")
isLastFmFav, err := model.GetLastFmFavorite(ctx, "Artist Name", "Album Name", "Track Name")
```