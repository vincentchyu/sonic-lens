# 播放完成并发重复上报与双层防重拦截特性清单

## 1. 特性背景与问题排查
用户在数据库中发现单次播放结束时，同一首歌曲会被重复插入 2 条听歌流水记录（如 `id: 9064` 与 `id: 9065`，`id: 9062` 与 `id: 9063`），播放时间几乎一致（相差 1~2 秒），且一条记录 `release_type` 为 `'album'`，另一条为 `''`。

### 根本原因
1. **Scrobbler 防重标志更新时序漏洞**：
   在 `internal/scrobbler/scrobbler_player_checker.go` 的 `handleTrackScrobble` 中，先同步执行了包含了网络 IO（Last.fm Scrobble 接口调用，耗时 1~3s）与 GORM 事务的 `HandleTrackPlaybackThreshold`，在**执行结束后才将 `scrobbledTracks[trackKey]` 设为 `true`**。
   导致在 2 秒后的下一次轮询中，由于上一次的网络/事务尚未返回，防重标志依然为 `false`，从而误判为“达到阈值但尚未上报”，触发了第二次重复上报。
2. **并发竞争导致 `release_type` 空白**：
   由于第一次上报（`id: 9064`）在事务中锁定了 `Album` 实体并补充了 `release_type`，而第二次重复上报（`id: 9065`）在并发冲刷时拿到了未提交前的快照且触发了竞争，导致第二次插入的 `release_type` 保持为空。

---

## 2. 核心修改与双层防重方案

### 2.1 客户端/Scrobbler 侧防重（前端即时锁定）([`internal/scrobbler/scrobbler_player_checker.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/scrobbler/scrobbler_player_checker.go))
- 在 `handleTrackScrobble` 函数入口处，**先判定并立即锁定 `b.scrobbledTracks[snapshot.trackKey] = true`**，而后才发起 `HandleTrackPlaybackThreshold` 异步/同步处理。彻底切断了轮询间隔内因接口等待导致的并发重复触发。

### 2.2 服务端/Logic 侧防重（后端 60s 内窗兜底）([`internal/logic/track/playback.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/logic/track/playback.go) & [`internal/model/track_play_record.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/model/track_play_record.go))
- 在 `internal/model/track_play_record.go` 引入 `IsDuplicateTrackPlayRecord` 函数，查询指定时间窗口内是否存在同源同曲目的物理流水。
- 在 `HandleTrackPlaybackThreshold` 准备插入 `modelInsertTrackPlayRecord` 之前，增加 60 秒内窗防重拦截。若判定为短时间内重复上报，打印警告日志并优雅跳过数据库写库与统计重复累加。

---

## 3. 验证与数据库对账结果
1. **单元测试**：
   运行 `go test ./internal/scrobbler/... ./internal/model/... ./internal/logic/track/...` 全部 **PASS**。
2. **物理数据库对账纠偏**：
   对用户反馈的 9063, 9065 等历史重复记录完成 `release_type` 更正与播放数对账校对，数据库 `WHERE release_type = ''` 保持为 **0** 条。
