# MusicBrainz 多碟已听曲目对齐修复

## 背景

`DeepingMaintenance` 在处理“已听过曲目”时，旧逻辑优先读取 `track` 主表上的 `disc_number/track_number`，再退回按曲名匹配。  
这在多碟专辑里会产生两个问题：

1. `track` 主表坐标可能不是当前专辑内最可靠的物理位置
2. 本地曲名与 MusicBrainz 曲名可能存在繁简、全角/半角括号、大小写差异，退回名称匹配时容易把第二碟错误吸到第一碟

典型表现：

- 已听过曲目被错误写回到 `disc 1`
- 正确的 `(disc_number, track_number)` 位置被当成“未听过曲目”
- `track_album` 被额外插入 `track_id = 0` 的占位行

## 本次修复

`internal/logic/musicbrainz/service.go`

- 新增 `normalizeMBTrackLookupKey`
  - 统一 `UnityFixAll`
  - 繁体转简体
  - 英文统一转小写
- 全角/半角括号归一已上移到 `common.UnityPunctuationMarksFix`
  - 避免 MusicBrainz 单独维护一套标点归一规则
  - 播放器入库链路与 MusicBrainz 对齐链路共享同一标题规范
- 新增 `findMBTrackForHeardTrack`
  - 对齐顺序改为：
    1. `track_album` 的 `(disc_number, track_number)`
    2. `track` 的 `(disc_number, track_number)`
    3. `track_album.track` 标准化名称
    4. `track.track` 标准化名称
  - 当 `track_album` 与 `track` 的位置已经一起写错，但 `track.track` 仍保留更具体标题时，允许标题匹配反向修正错误位置
- 已听曲目写回 `track_album` 时改为复用模型层 upsert
  - 如果正确位置上已经存在 `track_id = 0` 的占位符，会在修正真实曲目时一并回收，避免遗留脏占位数据
  - 如果历史脏数据已经造成“同一物理位置存在两条真实记录”，且当前记录本来就在目标位置，则允许先原地保存，等待后续错位记录在同一事务内迁走，不能提前报硬冲突中断整张专辑修复
- `DeepingMaintenance` 的日志新增 `match_source`
  - 便于确认一次对齐是按专辑物理位置命中，还是被迫退回名称匹配

## 测试

新增 `internal/logic/musicbrainz/service_test.go`：

- 验证繁简 + 括号 + 大小写归一
- 验证在 `track` 主表坐标错误、`track_album` 坐标正确时，仍然优先命中 `track_album` 的物理位置

## 后续规则

1. 多碟专辑的已听曲目对齐，必须把 `track_album` 视为“当前专辑内物理位置”的最高优先级来源
2. `track` 主表位置只能作为次级兜底，不能覆盖 `track_album`
3. 名称匹配必须使用统一标准化 key，不能再直接 `strings.ToLower(track)` 裸比对
