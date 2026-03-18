# 播放记录同步功能特性清单

## 特性摘要

为 TrackPlayRecord 模型增加分页查询未同步记录的功能，并实现批量同步到 Last.fm 的逻辑。

## 功能详情

### 1. 新增模型方法

在 `internal/model/track_play_record.go` 中添加了以下方法：

1. `GetUnscrobbledRecordsWithPagination` - 支持分页查询未同步的播放记录
2. `GetUnscrobbledRecordsCount` - 获取未同步播放记录的总数
3. `BatchUpdateScrobbledStatus` - 批量更新播放记录的同步状态
4. `SyncUnscrobbledRecords` - 同步未上报的数据到 Last.fm 并更新状态

### 2. 新增服务接口

在 `internal/logic/track/service.go` 中添加了对应的接口方法：

1. `GetUnscrobbledRecordsWithPagination` - 分页获取未同步到 Last.fm 的播放记录
2. `GetUnscrobbledRecordsCount` - 获取未同步到 Last.fm 的播放记录总数
3. `SyncUnscrobbledRecords` - 同步未上报的数据到 Last.fm 并更新状态

### 3. 新增API接口

在 `api/server.go` 中添加了以下API接口：

1. `GET /api/unscrobbled-records` - 获取未同步到Last.fm的播放记录（分页）
2. `GET /api/unscrobbled-records/count` - 获取未同步到Last.fm的播放记录总数
3. `POST /api/unscrobbled-records/sync` - 同步选中的未同步记录到Last.fm

### 4. 前端页面

在 `templates/dashboard.html` 中添加了未同步记录的展示和同步功能。

### 5. 功能特点

- 支持分页查询，避免一次性加载大量数据
- 批量处理同步操作，提高效率
- 返回同步失败的记录，便于后续处理
- 保持与现有代码的一致性和兼容性

## 实现要点

- 使用 GORM 的分页查询功能实现分页
- 通过 Last.fm 客户端库实现数据上报
- 使用批量更新优化数据库操作
- 正确处理同步失败的情况，返回失败记录供重试

## 使用示例

```go
// 分页查询未同步记录
records, err := trackService.GetUnscrobbledRecordsWithPagination(ctx, 10, 0)

// 获取未同步记录总数
count, err := trackService.GetUnscrobbledRecordsCount(ctx)

// 同步未上报的数据
failedRecords, err := trackService.SyncUnscrobbledRecords(ctx, 10)
```