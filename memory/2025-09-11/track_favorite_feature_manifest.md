# 曲目收藏功能特性清单

## 特性概述

实现曲目收藏功能，允许用户通过前端界面收藏正在播放的曲目，并在Apple Music和Last.fm上同步收藏状态。

## 功能详情

### 后端功能

1. 在WebSocket消息结构中增加喜欢状态标志
    - 在`WsTrackInfo`结构中添加`apple_music`和`lastfm`字段
    - 在发送WebSocket消息时查询并填充这些字段

2. 新增API端点
    - 添加`POST /api/favorite`端点处理收藏请求
    - 实现收藏逻辑，支持Apple Music和Last.fm同步收藏

3. 业务逻辑层实现
    - 在`TrackService`接口中添加`SetTrackFavorite`方法
    - 实现`SetTrackFavorite`方法，处理Apple Music和Last.fm的收藏逻辑

### 前端功能

1. UI更新
    - 在正在播放悬浮窗中显示喜欢状态指示器
    - 根据不同喜欢状态显示不同颜色：
        - 仅Apple Music喜欢：红色
        - 仅Last.fm喜欢：粉色
        - 两个都喜欢：橙色
    - 添加点赞按钮，使用爱心图标表示

2. 交互逻辑
    - 实现点赞按钮点击事件处理
    - 提供即时视觉反馈（按钮颜色变化）
    - 与后端API交互，发送收藏请求
    - 根据后端响应更新UI状态

## 技术实现

### 数据模型

- 利用现有的`Track`模型存储喜欢状态
    - `IsAppleMusicFav`字段存储Apple Music喜欢状态
    - `IsLastFmFav`字段存储Last.fm喜欢状态

### API接口

- `POST /api/favorite`
    - 请求参数：
        - `artist`: 艺术家名称
        - `album`: 专辑名称
        - `track`: 曲目名称
        - `source`: 播放来源
        - `favorite`: 收藏状态（true/false）
    - 响应数据：
        - `apple_music`: Apple Music喜欢状态
        - `lastfm`: Last.fm喜欢状态

### 前端实现

- 使用WebSocket接收实时播放信息和喜欢状态
- 使用CSS实现不同喜欢状态的颜色显示
- 使用JavaScript处理点赞按钮交互和API调用

## 测试要点

1. 验证WebSocket消息中正确包含喜欢状态
2. 验证前端正确显示不同喜欢状态的颜色指示器
3. 验证点赞按钮交互和视觉反馈
4. 验证API端点正确处理收藏请求
5. 验证Apple Music和Last.fm收藏状态同步

## 依赖项

- 现有的Track模型和相关数据库操作
- 现有的Apple Music和Last.fm API集成
- WebSocket实时通信机制