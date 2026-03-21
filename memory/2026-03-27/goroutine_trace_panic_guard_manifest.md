# 异步协程 trace 与 panic 保护闭环

- **日期**: 2026-03-27
- **范围**: `core/telemetry`、`main.go`、`api/`、`internal/logic/`、`internal/model/`、`internal/scrobbler/`、`internal/sync/`、`core/ai/`、`core/bonjour/`

## 背景

项目里长期存在裸 `go func` / `go xxx(...)` 启动点。它们会带来两个直接问题：

1. 子协程没有显式新 span，链路追踪会把异步执行阶段揉进父 span，或直接丢失异步边界。
2. 子协程没有统一 `recover`，一旦后台任务 panic，会直接把整个进程打崩。

## 本次收敛

1. 在 `core/telemetry/goroutine.go` 新增统一异步入口：
   - `telemetry.GoSafe(ctx, spanName, fn)`：保留父 `ctx` 的取消语义，自动创建子 span，并统一 `recover panic`。
   - `telemetry.GoSafeDetached(ctx, spanName, fn)`：基于 `context.WithoutCancel(ctx)` 保留 trace 父子关系，但脱离上游取消，适合异步落库、日志补写、回填这类收尾任务。
   - `telemetry.GoOnlySafe(ctx, fn)`：只负责起协程与 `recover panic`，不主动创建 span，适合长生命周期循环。
2. `recover` 统一记录：
   - span `RecordError` + `codes.Error`
   - 中文错误日志
   - panic 值与堆栈
3. 把项目内现存 goroutine 启动点统一切到 helper：
   - `main.go` 启动的 HTTP Server 与调度器
   - `api/server.go` 的 WebSocket 消息处理
   - `internal/sync/*` 的后台循环和启动任务
   - `internal/scrobbler` 的播放器检查协程
   - `internal/logic/insight` 的流式转发与异步落库
   - `core/ai/*` 的流式读取与 `BaseProvider` 异步调用流水落库
   - `internal/model/library_change_log.go` 的资料库变更广播
   - `core/bonjour` 与 `internal/cache/genre_cache` 的后台循环
   - `internal/logic/artwork` 的异步封面回填
4. 删除了 `internal/model/genre.go` 中三个实际什么都没做的空协程，避免继续制造噪音。

## 约束

1. 后续新增异步协程时，禁止直接写裸 `go func` / `go xxx(...)`。
2. 需要跟随上游取消并创建异步子 span 时，用 `telemetry.GoSafe`。
3. 需要“即使请求结束也继续完成收尾任务”时，用 `telemetry.GoSafeDetached`。
4. 需要常驻循环但不希望整个循环挂在一个长期 trace 里时，用 `telemetry.GoOnlySafe`，再在每次工作单元里自己 `StartSpan`。
5. 若异步任务内部还要再拆子协程，也必须继续走 helper，保持 span 树与 panic 保护完整。

## 验证

1. 新增 `core/telemetry/goroutine_test.go`，覆盖：
   - 子协程 span 继承父 trace id
   - panic 被 recover 且 span 状态标记为 error
2. `go test ./core/telemetry/...` 通过。
3. `go test ./internal/model -run '^$'` 通过，确认本次改动未引入 `internal/model` 编译回归。

## 风险提示

- `go test ./...` 目前仍被 `internal/model` 既有失败用例阻塞，集中在 `album_cleanup_*` 与 `album_release_mb_*`，不属于本次异步协程治理范围。
