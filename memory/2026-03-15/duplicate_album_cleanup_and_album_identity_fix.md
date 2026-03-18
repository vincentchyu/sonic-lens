# 重复专辑清洗与专辑身份修复

## 特性概述

- 修复 `album` 以 `(artist, name, release_date)` 作为唯一键时，被曲目级 `release_date` 持续拆分出重复专辑的问题。
- 新增重复专辑清洗命令，将 `track_album`、`album_release_mb` 等关联统一迁移到保留专辑并删除冗余专辑。

## 功能要点

- `getOrCreateAlbumTx` 在精确命中失败后，会回退到同 `artist + name` 的现有专辑，避免因不同曲目日期继续裂变。
- 当外部客户端缺少专辑发布时间时，优先复用 `sync_status=3` 的已深度维护专辑。
- 新增 `cleanup-duplicate-albums` 子命令，支持 `--dry-run`、`--apply`、`--limit`、`--artist`、`--album`。

## 实现细节

- model 层新增重复专辑组选取、主专辑决策、`track_album` 合并、`album_release_mb` 迁移与冗余 `album` 删除事务。
- 主专辑优先级按 `sync_status`、已确认 MusicBrainz 关联数、已挂曲目数、是否有发布时间、`id` 顺序决策。
- 回归测试覆盖：
  - 缺失发布时间时优先复用精选维护专辑。
  - 曲目日期不同但专辑同名同作者时复用已有专辑，不再新建重复行。
  - 重复专辑清洗时关联迁移不丢失，`dry-run` 不改库。

## 风险提示

- 该清洗策略按 `(artist, name)` 合并，默认将同名同作者下的历史版本视为脏数据；若未来要保留真正多版本专辑，需要引入更稳定的版本身份，而不能继续依赖播放器上报日期。
