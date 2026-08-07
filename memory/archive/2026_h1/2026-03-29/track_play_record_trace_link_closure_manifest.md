# 播放流水 Trace 关联闭环

- **日期**: 2026-03-29
- **范围**: `internal/scrobbler/`、`internal/logic/track/`、`internal/model/`

## 背景

Scrobbler 当前已经形成“单首歌一个根 span”的播放链路，但播放流水 `track_play_record` 与观测链路仍然是割裂的：

1. 当某次 scrobble、收藏同步、封面回填或归因出现异常时，只能先从 Jaeger/SigNoz 找 trace，再人工比对播放时间和曲目信息。
2. 后台 replay、数据排查和客服定位无法直接从业务记录跳到对应的 trace。
3. 即使库里已经有完整的播放事实，仍然缺少“这条记录对应哪条观测链路”的稳定锚点。

## 本次收敛

### 1. 播放事件输入追加 trace 标识

1. `internal/logic/track.PlaybackEventInput` 新增：
   - `trace_id`
   - `root_span_id`
   - `trace_sampled`
2. 这些字段不参与业务归因，只表示“当前播放会话在观测系统中的定位信息”。

### 2. Scrobbler 根 span 标识透传到播放事件

1. `BasePlayerChecker` 从当前活动曲目的根 span 读取：
   - `TraceID`
   - 根 span `SpanID`
   - `IsSampled`
2. `playingTrackSnapshot` 在转成 `PlaybackEventInput` 时透传这些标识。
3. 阈值处理、当前播放、副作用同步现在都可以拿到当前歌曲根 span 的 trace 上下文快照。

### 3. 播放流水落库保存 trace 信息

1. `track_play_records` 新增字段：
   - `trace_id`
   - `root_span_id`
   - `trace_sampled`
2. `HandleTrackPlaybackThreshold` 在创建 `TrackPlayRecord` 时会把这些字段一并写入。
3. 这样每条播放流水都可以直接反查对应的播放链路，而不是再靠时间和标题模糊比对。

### 4. Schema 与档案同步

1. MySQL 老库新增 `ensureTrackPlayRecordTraceSchema(...)`：
   - 补列 `trace_id`
   - 补列 `root_span_id`
   - 补列 `trace_sampled`
   - 补索引 `idx_track_play_records_trace_id`
2. `internal/model/sql/ddl/track_play_records.sql` 已同步更新到最新表结构。
3. `track_resolution_test.go` 的 SQLite 手工建表基座已补齐新字段，避免回归测试被模型字段漂移打断。

## 约束

1. `track_play_record.trace_id` 只是观测定位锚点，不得被当作业务主键或幂等键。
2. `root_span_id` 必须保存当前歌曲根 span，而不是阈值子 span；否则无法从播放流水稳定定位到“单首歌根节点”。
3. 当链路未采样或上下文无效时，允许 `trace_id/root_span_id` 为空，但 `trace_sampled` 必须准确反映采样结果，避免误判“库里有 trace id 但平台查不到”。
4. 后续若有播放相关的后台任务、问题工单或调试页需要深链观测，应优先复用 `track_play_record` 上的 trace 字段，不要再重复设计第二套关联锚点。

## 验证

1. `go test ./internal/logic/track/... ./internal/scrobbler/...` 通过。
2. `go test ./internal/model -run 'TrackPlayRecord|ProcessTrackPlayRecord|ReplayTrackPlayRecords|ResolveTrackPlayRecord'` 通过。
3. `go test ./internal/model/...` 仍然失败，但失败点仍是仓库内既有：
   - `TestCleanupDuplicateAlbumsMergesRelationsIntoCanonicalAlbum`
   - `TestCleanupDuplicateAlbumsContinueOnErrorSkipsConflictedGroup`
   - `TestLinkAlbumToMBIDTx`
   - `TestLinkAlbumToMBIDUsesContextDB`

## 结果

SonicLens 现在已经具备“从播放流水反查观测链路”的闭环：

1. Scrobbler 根 span 标识进入播放事件输入
2. 阈值落库时写入 `track_play_record`
3. 后台排障时可从业务事实直达对应 trace

这为后续的异常工单定位、replay 排查和调试页深链提供了稳定基础。
