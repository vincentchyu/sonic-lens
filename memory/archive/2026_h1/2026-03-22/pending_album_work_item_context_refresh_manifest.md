# 待归因专辑工作项上下文刷新特性清单

## 日期
2026-03-22

## 特性摘要
- 待归因专辑工作台进入维护后，不再把冻结上下文误认为实时数据，而是明确区分“工作项冻结快照”和“当前实时列表”。
- 工作项详情接口新增实时对比结果，前端在发现上下文过期时提示用户刷新冻结上下文，用户可选择保持现状或显式刷新。
- 冻结上下文刷新改为独立写接口，避免在查询接口里隐式修改工作项，降低调试和回放时的副作用。

## 后端改动
- `internal/model/pending_album_work_item.go`
  - `PendingAlbumWorkItemDetail` 新增 `live_group` 与 `context_stale`。
  - `GetPendingAlbumWorkItemDetail` 会对比工作项冻结的播放/点赞 ID 与当前实时聚合结果。
  - 新增 `RefreshPendingAlbumWorkItemContext`，用于用当前实时列表重置工作项冻结 ID。
- `internal/logic/pendingalbum/service.go`
  - 暴露工作项上下文刷新能力给 API 层。
- `api/server.go`
  - 新增 `POST /api/pending-albums/work-items/:id/refresh-context`。

## 前端改动
- `templates/dashboard.html`
  - 在工作项详情页增加“冻结上下文已变化”的提示条。
  - 当详情判断上下文过期时，前端弹出确认框，允许用户选择刷新或继续使用冻结快照。
  - 新增手动刷新按钮，避免用户必须先触发二次操作才能同步上下文。

## 验证
- 新增模型测试，覆盖“建单后新增播放记录导致上下文过期”以及“刷新后恢复一致”。
- 已执行 `gofmt`。

