# 单元测试零外部依赖改造与项目单测专属 Skill 建设特性清单

## 1. 特性背景

在新会话启动或进入单测环节时，由于部分测试直接依赖宿主 macOS 进程（如 Music.app、Audirvana）、公网外部请求或未规范初始化的实体数据库，导致 `go test` 运行报错、反复重试，造成额度与上下文浪费。

## 2. 核心架构与落地改造

### 2.1 外部依赖全面隔离
- **`core/applemusic/sciprt_test.go`**:
  - 增加 `//go:build integration` 标签，隔离宿主 AppleScript 与 Music.app 真实调用。
  - 清理包含无效外网请求 `client.Get("google.com")` 的 `TestName` 废弃代码。
- **`core/applemusic/sciprt_unit_test.go`**:
  - 新增纯单元测试，测试 `ToTrackMetadata()` 字段转换、时长与日期解析等纯逻辑。
- **`core/audirvana/sciprt_test.go`**:
  - 增加 `//go:build integration` 标签，隔离 Audirvana 真实进程。

### 2.2 通用测试数据库辅助体系 (`internal/testutil`)
- **`testutil.NewMemoryDB`**:
  - 基于 SQLite `:memory:` 共享缓存模式提供秒级轻量 GORM 实例，支持传入 DDL 建表，测试结束通过 `t.Cleanup` 自动释放。
- **`testutil.NewMockDB`**:
  - 封装 GORM MySQL 驱动与 `sqlmock`，统一模拟 SQL 行为。
- **`testutil.SetupTestGlobalMySQL`**:
  - 临时安全替换 `model.GlobalDBForMysql` 并在测试结束后自动还原。

### 2.3 沉淀项目专属单元测试技能
- 在 `.agents/skills/soniclens-unit-test/SKILL.md` 中沉淀标准规范：
  - 默认单测 100% 独立运行原则；
  - macOS asdf 环境 `BypassSandbox: true` 运行红线；
  - Model / Logic / API 分层测试范式与代码生成模版；
  - 常见单测报错排查速查表。

## 3. 验证情况
- `go test -count=1 ./...` 全项目零外部依赖 100% 秒级通过。
- `internal/testutil` 单测全部通过。
