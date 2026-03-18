# SQL 结构与初始化脚本同步清单

## 日期

2026-03-15

## 背景

仓库 `internal/model/sql/ddl/` 与 `internal/model/sql/dml/` 中长期混有旧版 MySQL、SQLite 片段、手工补丁和历史数据快照，已经和当前 `multimedia` 实库结构出现漂移：

- DDL 中部分唯一索引、字段类型、索引定义与当前 MySQL 实库不一致
- `track_insight_feedbacks` 缺少独立 DDL 文件
- DML 中保留了大量历史业务数据，语义从“初始化脚本”漂移成“数据快照”

## 本次改动

### 1. 按当前 MySQL 实库同步 DDL

- 使用 `docker exec -i my-mysql mysql -uroot -p66243766 multimedia` 获取当前表结构
- 同步更新以下 DDL 文件：
  - `album.sql`
  - `album_release_mb.sql`
  - `genre.sql`
  - `llm_call_log.sql`
  - `release_mb.sql`
  - `stat.sql`
  - `track.sql`
  - `track_album.sql`
  - `track_insight.sql`
  - `track_lyrics.sql`
  - `track_play_records.sql`
  - `track_rank_stat.sql`
- 新增 `track_insight_feedbacks.sql`

### 2. 清理 DML 历史数据

- 删除 `init_track.sql`、`init_track_play_records.sql`、`init_genre.sql` 中的历史业务数据快照
- DML 改为只表达“初始化动作”与“是否需要预置数据”的约束

### 3. 保留必要初始化语义

- `init.sql` 仅负责建库、切库和初始化 `dashboard_stat` 单行基线
- `init_album_data_mysql.sql` 保留 `track -> album -> track_album` 的幂等回填逻辑
- 其他表明确标注“无静态初始化数据”

## 新约束

- `internal/model/sql/ddl/` 以当前 MySQL 实库结构为准，不再混写 SQLite 片段和旧版补丁
- `internal/model/sql/dml/` 只存初始化动作，不存历史业务数据样本
- 若后续再调整生产或开发 MySQL 表结构，应同步更新对应 DDL 文件，而不是只改 GORM 模型或在线库

## 验证

使用临时库 `multimedia_sql_check` 验证：

```bash
docker exec -i my-mysql mysql -uroot -p66243766
```

- 全量 DDL 可创建成功
- 全量 DML 初始化脚本可执行成功
