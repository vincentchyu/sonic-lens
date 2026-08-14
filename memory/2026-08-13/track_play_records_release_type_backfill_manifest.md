# 听歌流水 `track_play_records.release_type` 字段丢失与修补归因对账特性清单

## 1. 特性背景与问题分析
在排查数据库 SQL `SELECT * FROM multimedia.track_play_records WHERE release_type = ''` 时，发现最近播放上报及归因到的听歌流水中，`release_type` 字段普遍为 `''`（空字符串）。

### 根本原因
1. **归因回填遗漏**：在播放流水上报时（`RecordPlayback`），若输入的专辑名没有带 Apple Music 连字符格式后缀（如 ` - EP`），则解析得到的 `releaseType` 为 `""`。在随后的 `ProcessTrackPlayRecord` 事务中，流水虽然成功归因到了数据库中的 `Album` 实体（如 `album_id = 201`, `ReleaseType = "album"`），但在调用 `buildTrackPlayRecordResolvedFields` 组合更新字段时，仅处理了 `album_subtitle` 和 `genre`，**遗漏了 `release_type` 的回填**。
2. **修补引擎扫描与继承缺失**：自动数据修补引擎 `RepairAndReconcileTrackPlayRecordsTx` 的扫描条件没有覆盖 `release_type` 缺失或空白的场景，且在从同名原型流水（Pass 1）及 Album 实体（Pass 3）继承元数据时，未包含 `release_type` 的双向纠偏。

---

## 2. 核心修改与修补逻辑

### 2.1 模型与归因逻辑修改 ([`internal/model/track_play_record.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/internal/model/track_play_record.go))
- **`buildTrackPlayRecordResolvedFields`**:
  - 当 `albumID > 0` 且查找到 `albumObj` 时，若 `albumObj.ReleaseType != ""` 且流水记录的 `release_type` 为空，自动回填 `fields["release_type"] = albumObj.ReleaseType`。
  - 若流水记录携带有 `release_type` 但 `albumObj.ReleaseType` 为空，反向更新 `albumObj.ReleaseType`。
- **`RepairAndReconcileTrackPlayRecordsReport` & `RepairAndReconcileTrackPlayRecordsTx`**:
  - 报告增加 `RepairedReleaseType` 统计指标。
  - 查询条件补充对 `release_type IS NULL OR (release_type = '' AND album_id > 0)` 的扫描。
  - 在 Pass 1 / Pass 3 阶段引入 `release_type` 的原型继承与 `Album` 实体绑定；当已归因绑定专辑且仍无类型后缀时，自动兜底补全为 `"album"`（全长专辑）。

### 2.2 CLI 提示优化 ([`cmd/replay_track_play_records.go`](file:///Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/cmd/replay_track_play_records.go))
- CLI 命令 `go run . replay-track-play-records --repair` 输出中新增 `成功补全ReleaseType=%d` 维度。

---

## 3. 验证与数据库校对结果
1. **单元测试验证**：
   - 更新并增加 `internal/model/track_play_record_repair_test.go` 对 `ReleaseType` 修补与断言验证。
   - 运行 `go test -v ./internal/model/... -run "TestRepairAndReconcileTrackPlayRecords|TestProcessTrackPlayRecord"` 结果 **PASS**。
2. **物理数据库对账纠偏**：
   - 对 Docker MySQL `multimedia` 数据库中历史 53 条 `release_type = ''` 的已归因流水完成了全量对账与纠偏，`WHERE release_type = ''` 降为 **0** 条。
