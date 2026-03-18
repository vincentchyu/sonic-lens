# Redis链路跟踪和日志记录功能特性清单

## 日期

2025-09-19

## 特性摘要

为Redis客户端实现完整的链路跟踪和日志记录功能，包括自定义日志钩子、OpenTelemetry集成、慢查询检测等，与数据库模块保持一致的设计和实现。

## 功能详情

### 1. Redis日志记录器

- 实现了专门的Redis日志记录器(`RedisLogger`)，与数据库模块设计保持一致
- 提供三种日志记录方法：
    - `LogCommand`: 记录单个Redis命令执行
    - `LogPipeline`: 记录Redis管道操作执行
    - `LogDial`: 记录Redis连接建立
- 集成项目现有的日志库(`github.com/vincentchyu/sonic-lens/core/log`)
- 复用链路ID进行日志关联

### 2. 慢查询检测

- 单个命令慢查询阈值：100ms
- 管道操作慢查询阈值：200ms
- 超过阈值时自动记录警告日志
- 记录详细的执行时间和命令信息

### 3. Redis钩子实现

- 实现go-redis v9的钩子接口(`redis.Hook`)
- 三种钩子方法：
    - `DialHook`: 监控Redis连接建立
    - `ProcessHook`: 监控单个命令处理
    - `ProcessPipelineHook`: 监控管道命令处理
- 与OpenTelemetry集成，为每个操作创建独立的span
- 记录命令内容、执行时间等属性
- 正确处理错误状态和链路关系

### 4. 链路跟踪集成

- 使用项目现有的链路跟踪框架(`github.com/vincentchyu/sonic-lens/core/telemetry`)
- 为每个Redis操作创建带有语义化名称的span
- 记录关键属性如命令内容、执行时间、连接信息等
- 正确处理错误状态码和错误记录

### 5. 错误处理

- 优雅处理Redis连接失败情况
- 详细记录各种错误信息，包括Redis特定错误
- 不影响主程序运行的错误处理机制
- 区分处理`redis.Nil`等特殊错误

### 6. 性能优化

- 合理的慢查询阈值设置
- 高效的日志记录和链路跟踪实现
- 非阻塞的监控机制
- 对于大量命令的管道操作，智能截断日志记录

## 技术实现

### 核心文件

1. `core/redis/logger.go` - Redis日志记录器实现
2. `core/redis/hook.go` - Redis钩子实现
3. `core/redis/redis.go` - Redis客户端初始化更新

### 依赖库

- `github.com/redis/go-redis/v9` - Redis客户端
- `github.com/redis/go-redis/extra/redisotel/v9` - Redis OpenTelemetry集成
- `go.opentelemetry.io/otel` - OpenTelemetry SDK
- `go.uber.org/zap` - 日志库

### 集成点

- 与项目现有的日志系统集成
- 与项目现有的链路跟踪系统集成
- 与项目现有的错误处理机制集成

## 设计原则

### 1. 一致性原则

- 与数据库模块的设计和实现保持一致
- 使用相同的日志格式和链路ID复用机制
- 统一的慢查询检测和阈值设置

### 2. 可观测性原则

- 完整的操作监控覆盖
- 详细的错误记录和链路跟踪
- 合理的性能指标收集

### 3. 容错性原则

- 优雅处理连接失败等异常情况
- 不影响主程序运行的监控实现
- 详细的错误信息记录便于问题排查

## 测试验证

- 创建测试程序验证日志记录功能
- 验证链路跟踪功能正常工作
- 测试慢查询检测机制
- 验证错误处理机制