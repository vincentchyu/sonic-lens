# 播放归因置信度与弱来源止血

## 特性概述

- 将播放器元数据按来源置信度分层，阻止弱来源直接写坏 `album`、`track`、`track_album` 主实体。
- 收紧 `track` 身份解析，移除不安全的 `unique_id` 直连与低置信三元组回退。

## 功能要点

- `TrackMetadata` 新增 `player_type`、`confidence`、`release_year`。
- Apple Music 流媒体与 Roon 归为低置信来源，只允许命中已存在稳定曲目并增加播放次数，不再创建专辑结构。
- Audirvana 保持高置信，可继续驱动资料库实体补全。
- `track_play_record` 自动回填 `album_id` 时改为严格身份解析，不再对低置信播放做三元组误绑定。

## 实现细节

- `resolveTrackIdentity` 改为带选项解析：
  - `unique_id` 仅作为安全提示，要求候选唯一且 `artist/album/track` 一致。
  - 时长匹配要求结果唯一。
  - 低置信来源禁用宽松三元组回退。
- `IncrementTrackPlayCount` 分流：
  - 高置信来源走原有建模链路。
  - 低置信来源走 `resolved-only` 分支，只在稳定命中既有曲目时做 `play_count + 1`。
- scrobbler 层统一构建 `TrackMetadata`，Apple Music 根据 `kind` 区分本地/资料库/流媒体置信度。
- `track_play_record` 新增 `resolved_track_id / resolution_status / resolution_confidence / library_applied`。
- scrobbler 实时链路已收口为 `ProcessTrackPlayRecord`：
  - 先读取播放流水；
  - 再统一执行资料库写入；
  - 最后回填归因状态与 `library_applied`。
- `IncrementTrackPlayCount` 已降为薄入口，事务内具体写库动作分别下沉到 `applyTrackPlayMutationTx`、`applyPlayToLibraryTx` 与 `incrementExistingTrackPlayCountTx`，为后续后台补归因复用同一条写入路径做准备。
- model 层新增 `ReplayTrackPlayRecords` 与 `GetReplayableTrackPlayRecords`，支持扫描 `pending/unresolved` 或 `library_applied=false` 的播放流水，并按最保守来源置信度规则重放。
- CLI 新增 `replay-track-play-records`，可先 `dry-run` 预览，再批量 `--apply` 执行后台补归因。
- 新增 `playReplay` 配置段和 `StartTrackPlayReplayScheduler`，可按配置开启常驻自动补归因任务，默认关闭，避免未验数前直接影响现网写入节奏。

## 风险提示

- 自动补归因调度已具备，但上线前仍建议先用命令手动 `dry-run` / `--apply` 小批量验数，再打开 `playReplay.enabled`。
