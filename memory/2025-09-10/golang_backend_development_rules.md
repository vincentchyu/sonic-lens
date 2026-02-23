# Go后端开发规范

## 规范摘要

定义Go后端开发的分层架构规范，确保业务逻辑、数据访问和API接口的职责分离。

## 规范详情

### 1. 分层架构原则

项目采用清晰的分层架构，各层职责如下：

1. **API层** (`api/`)
    - 负责处理HTTP请求和响应
    - 参数验证和错误处理
    - 调用logic层服务处理业务逻辑
    - 不应包含业务逻辑和数据库操作

2. **Logic层** (`internal/logic/`)
    - 实现业务逻辑
    - 协调多个model操作
    - 处理复杂的业务流程
    - 为API层提供服务接口

3. **Model层** (`internal/model/`)
    - 负责数据访问和持久化
    - 实现ORM操作和数据库查询
    - 数据验证和转换
    - 为logic层提供数据访问接口

### 2. 开发规范

#### 2.1 API层规范

- API层只负责HTTP请求处理，不应包含业务逻辑
- 所有业务逻辑必须委托给logic层处理
- 数据库操作必须通过model层完成
- 统一错误处理和日志记录

#### 2.2 Logic层规范

- 实现具体的业务逻辑
- 调用model层进行数据操作
- 处理业务规则和流程控制
- 为API层提供清晰的服务接口

#### 2.3 Model层规范

- 实现数据访问逻辑
- 使用GORM进行数据库操作
- 提供原子性的数据操作方法
- 处理数据验证和转换

### 3. 重构示例

#### 问题代码（API层直接操作数据库）：

```go
r.POST("/api/unscrobbled-records/sync", func(c *gin.Context) {
    // 直接在API层操作数据库
    var records []*model.TrackPlayRecord
    err := model.GetDB().WithContext(ctx).Where("id IN ? AND scrobbled = ?", req.IDs, false).Find(&records).Error
    
    // 直接调用第三方API
    _, err := lastfm.PushTrackScrobble(ctx, lastfmReq)
    
    // 直接更新数据库状态
    if err := model.BatchUpdateScrobbledStatus(ctx, successIDs, true); err != nil {
        // 错误处理
    }
})
```

#### 修复后代码：

```go
// API层 - 只负责HTTP请求处理
r.POST("/api/unscrobbled-records/sync", func(c *gin.Context) {
    // 调用logic层方法处理业务逻辑
    successCount, failedRecords, err := trackService.SyncSelectedUnscrobbledRecords(ctx, req.IDs)
    if err != nil {
        // 错误处理
    }
    // 返回响应
})

// Logic层 - 实现业务逻辑
func (s *TrackServiceImpl) SyncSelectedUnscrobbledRecords(ctx context.Context, ids []uint) (successCount int, failedRecords []*model.TrackPlayRecord, err error) {
    return model.SyncSelectedUnscrobbledRecords(ctx, ids)
}

// Model层 - 实现数据访问
func SyncSelectedUnscrobbledRecords(ctx context.Context, ids []uint) (successCount int, failedRecords []*TrackPlayRecord, err error) {
    // 数据库查询
    // 调用第三方API
    // 数据库更新
    // 返回结果
}
```

### 4. 规范优势

1. **职责分离**：各层职责明确，便于维护和扩展
2. **可测试性**：各层可以独立测试
3. **可复用性**：logic层和model层可以在不同API中复用
4. **可维护性**：代码结构清晰，便于理解和修改