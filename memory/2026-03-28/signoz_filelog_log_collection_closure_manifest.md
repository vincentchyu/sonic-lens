# SigNoz filelog 日志采集闭环

- **日期**: 2026-03-28
- **范围**: `config/otelcol/`、`config/config.yaml.example`、`README.md`、`GEMINI.md`

## 背景

项目已经完成 SigNoz trace + metrics 直连闭环，但应用日志仍由 `core/log` 使用 `zap + lumberjack` 写入本地文件。

当前用户场景是：

1. 应用侧日志文件轮转策略保持现状，不改 `core/log/log.go`
2. 通过本地 OpenTelemetry Collector 采集日志文件并转发到 SigNoz
3. SigNoz 自身日志 retention 单独收紧到 1 天，避免本地 ClickHouse 日志存储与索引长期膨胀

## 本次收敛

### 1. Collector 示例配置

新增 `config/otelcol/filelog_signoz.yaml.example`，提供面向 SonicLens 当前日志格式的本地采集模板：

1. `filelog` receiver 采集 `./.logs/go_lastfm-scrobbler.log*`
2. 排除 `.gz` 压缩备份，避免重读历史归档
3. 使用 `start_at: end`，避免首轮把历史文件全量灌入 SigNoz
4. 使用 `file_storage` 扩展持久化 offset，避免 Collector 重启后重复读取
5. 通过 `json_parser` 解析 zap JSON 日志，并把 `msg` 提升为 `body`
6. 通过 `resource` processor 统一补齐 `service.name=sonic-lens`
7. 最终通过 OTLP exporter 转发到本地 SigNoz `4317`

### 2. 配置与文档说明

1. `config/config.yaml.example` 补充本地日志目录示例，并明确：
   - 应用当前只直连 trace + metrics
   - 日志建议继续写本地文件
   - 再由 `filelog` Collector 采集
2. `README.md` 新增 SigNoz 日志采集章节，说明“应用本地日志保留”和“SigNoz retention”是解耦策略
3. `GEMINI.md` 补充长期记忆：日志上报 SigNoz 优先走 `filelog`，不要在业务代码里先叠加第二套应用内 logs exporter

## 约束

1. `filelog` 只负责采集路径，不负责决定 SigNoz retention
2. SigNoz retention 由管理端或 SigNoz 存储侧配置决定，不要误改应用日志轮转来替代 SigNoz retention
3. 当前 SonicLens 应用日志写盘策略仍保持 `zap + lumberjack`
4. 若后续需要日志与 trace 更强关联，优先在 Collector 解析阶段保留 `trace_id` / `span_id` 字段，不要先侵入业务日志调用方式

## 验证

1. Collector 模板已覆盖当前 SonicLens JSON 日志文件路径与基础解析规则
2. 仓库示例配置与 README 已明确 trace/metrics 直连、logs 走 `filelog` 的职责边界

## 结果

SonicLens 现在形成了更完整的本地 SigNoz 可观测性闭环：

1. 应用内 trace + metrics 直连 OTLP gRPC
2. 应用日志继续本地文件轮转
3. 本地 Collector 用 `filelog` 采集日志文件
4. SigNoz 日志 retention 可独立收紧到 1 天，不影响应用本地日志保留策略
