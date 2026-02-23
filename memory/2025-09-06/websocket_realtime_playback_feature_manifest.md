# WebSocket实时播放信息推送功能特性清单

## 特性概述

本特性实现了通过WebSocket实时推送当前播放信息到前端页面的功能，用户可以在首页的悬浮窗口中实时查看Audirvana或Roon正在播放的音乐信息。

## 功能范围

1. 在后端实现WebSocket服务器端点
2. 在播放检查逻辑中添加实时信息广播
3. 在前端实现WebSocket客户端连接
4. 在首页添加悬浮窗口展示实时播放信息

## 技术实现要点

### 后端实现

- 创建`core/websocket`包来处理WebSocket连接和消息广播，避免循环导入问题
- 在`api/server.go`中添加WebSocket端点`/ws`
- 实现WebSocket连接池管理
- 实现消息广播功能`BroadcastMessage`
- 在`internal/scrobbler/track_check_playing.go`中添加播放信息广播代码
- 当获取到Audirvana或Roon播放信息时，实时向所有连接的客户端推送消息

### 前端实现

- 在`templates/index.html`中添加WebSocket客户端连接逻辑
- 实现消息接收和处理
- 添加悬浮窗口展示当前播放信息
- 实现连接断开后的自动重连机制

## 数据结构

### 广播消息格式

```json
{
  "type": "now_playing",
  "source": "audirvana|roon",
  "data": {
    // Audirvana或Roon的播放信息对象
  }
}
```

## 文件变更列表

1. `api/server.go` - 添加WebSocket端点和广播功能
2. `internal/scrobbler/track_check_playing.go` - 添加播放信息广播代码
3. `templates/index.html` - 添加前端WebSocket客户端和悬浮窗口
4. `core/websocket/websocket.go` - 新增WebSocket处理包，避免循环导入

## 解决的循环导入问题

在实现过程中，我们遇到了`api`包和`scrobbler`包之间的循环导入问题：

- `api`包导入了`scrobbler`包（通过`server.go`中的导入）
- `scrobbler`包需要调用`api`包中的广播函数

为了解决这个问题，我们创建了一个独立的`core/websocket`包来处理WebSocket相关功能，避免了循环导入。

## 测试要点

1. WebSocket连接建立和断开处理
2. 消息广播功能正确性
3. 前端悬浮窗口显示效果
4. 自动重连机制有效性
5. 多客户端连接支持

## 注意事项

1. 确保WebSocket端点的跨域访问配置正确
2. 处理网络异常情况下的连接重试
3. 避免频繁的消息推送影响性能
4. 确保连接池的线程安全