# Track模型增强和参数优化功能特性清单

## 日期

2025-09-13

## 特性摘要

增强Track数据模型以支持更多元数据字段，并优化函数参数传递方式，遵循Go开发规范当参数超过5个时使用结构体封装。

## 功能详情

### 1. Track模型增强

- 扩展Track模型结构，增加以下元数据字段：
    - AlbumArtist: 专辑艺术家
    - TrackNumber: 曲目编号
    - Duration: 持续时间(秒)
    - Genre: 流派
    - Composer: 作曲家
    - ReleaseDate: 发布日期
    - MusicBrainzID: MusicBrainz ID
    - Source: 数据来源
    - BundleID: 应用标识符
    - UniqueID: 唯一标识符

### 2. 元数据处理功能

- 创建TrackMetadata结构体封装元数据字段
- 在internal/model包中添加JSON到Track结构的转换函数
- 实现exiftool和media-control两种数据源的解析和转换
- 在exec包中增加media-control调用函数

### 3. 函数参数优化

- 遵循Go开发规范，当函数参数超过5个时使用结构体封装
- 创建IncrementTrackPlayCountParams结构体封装IncrementTrackPlayCount函数参数
- 创建SetFavoriteParams结构体封装SetAppleMusicFavorite和SetLastFmFavorite函数参数
- 更新所有相关函数签名和调用点

### 4. 数据持久化增强

- 更新SetAppleMusicFavorite和SetLastFmFavorite方法，使其在创建新记录时也能保存完整的元数据
- 更新IncrementTrackPlayCount方法，使其在创建新记录时也能保存完整的元数据

## 技术实现

### 数据模型

```go
// TrackMetadata represents metadata for a music track
type TrackMetadata struct {
    AlbumArtist   string `json:"album_artist"`   // 专辑艺术家
    TrackNumber   int64  `json:"track_number"`   // 曲目编号
    Duration      int64  `json:"duration"`       // 持续时间(秒)
    Genre         string `json:"genre"`          // 流派
    Composer      string `json:"composer"`       // 作曲家
    ReleaseDate   string `json:"release_date"`   // 发布日期
    MusicBrainzID string `json:"musicbrainz_id"` // MusicBrainz ID
    Source        string `json:"source"`         // 数据来源
    BundleID      string `json:"bundle_id"`      // 应用标识符
    UniqueID      string `json:"unique_id"`      // 唯一标识符
}

// IncrementTrackPlayCountParams represents parameters for IncrementTrackPlayCount function
type IncrementTrackPlayCountParams struct {
    Ctx           context.Context
    Artist        string
    Album         string
    Track         string
    TrackMetadata TrackMetadata
}

// SetFavoriteParams represents parameters for SetAppleMusicFavorite and SetLastFmFavorite functions
type SetFavoriteParams struct {
    Ctx           context.Context
    Artist        string
    Album         string
    Track         string
    IsFavorite    bool
    TrackMetadata TrackMetadata
}
```

### 转换函数

```go
// ConvertExiftoolInfoToTrack converts ExiftoolInfo to Track model
func ConvertExiftoolInfoToTrack(exiftoolInfo *exec.ExiftoolInfo, source string) *Track

// ConvertMediaControlInfoToTrack converts MediaControlNowPlayingInfo to Track model
func ConvertMediaControlInfoToTrack(mediaInfo *exec.MediaControlNowPlayingInfo, source string) *Track
```

### 参数优化示例

```go
// 优化前
func SetAppleMusicFavorite(ctx context.Context, artist, album, track string, isFavorite bool, trackNumber int64, duration int64, genre string, composer string, releaseDate string, musicBrainzID string) error

// 优化后
func SetAppleMusicFavorite(params SetFavoriteParams) error
```

## 文件变更

### 新增文件

- internal/model/track_converter.go: JSON到Track结构的转换函数
- common/converters/track_converter.go: 转换函数包（解决循环依赖问题）

### 修改文件

- internal/model/track.go: 扩展Track模型，更新函数签名
- internal/logic/track/service.go: 更新服务接口和实现
- internal/scrobbler/scrobbler_player_checker.go: 更新函数调用
- api/server.go: 更新API调用
- core/exec/exec.go: 增加media-control调用函数

## 依赖关系

- 依赖common包进行参数验证
- 依赖core/exec包进行元数据提取
- 依赖gorm进行数据持久化

## 测试验证

- 验证项目可以成功编译
- 确保所有函数调用点都已正确更新
- 确保数据模型扩展不会影响现有功能