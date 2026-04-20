-- Cloudflare D1 数据库表结构
-- 简化版本,仅保留展示所需的核心字段

-- 曲目播放统计表
CREATE TABLE IF NOT EXISTS tracks
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    artist
    TEXT
    NOT
    NULL,
    album
    TEXT
    NOT
    NULL,
    album_subtitle
    TEXT,
    track
    TEXT
    NOT
    NULL,
    album_artist
    TEXT,
    play_count
    INTEGER
    DEFAULT
    0,
    genre
    TEXT,
    duration
    INTEGER,
    source
    TEXT,
    track_number
    INTEGER
    DEFAULT
    0,
    disc_number
    INTEGER
    DEFAULT
    1,
    is_apple_music_fav
    INTEGER
    DEFAULT
    0,
    is_last_fm_fav
    INTEGER
    DEFAULT
    0,
    created_at
    TEXT
    NOT
    NULL,
    updated_at
    TEXT
    NOT
    NULL,
    UNIQUE
(
    artist,
    album,
    track,
    disc_number,
    track_number
)
    );

-- 创建索引以提升查询性能
CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist);
CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album);
CREATE INDEX IF NOT EXISTS idx_tracks_genre ON tracks(genre);
CREATE INDEX IF NOT EXISTS idx_tracks_source ON tracks(source);
CREATE INDEX IF NOT EXISTS idx_tracks_play_count ON tracks(play_count DESC);

-- 播放记录表
CREATE TABLE IF NOT EXISTS track_play_records
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    artist
    TEXT
    NOT
    NULL,
    album_artist
    TEXT,
    album
    TEXT
    NOT
    NULL,
    track
    TEXT
    NOT
    NULL,
    album_id
    INTEGER
    DEFAULT
    0,
    duration
    INTEGER,
    play_time
    TEXT
    NOT
    NULL,
    scrobbled
    INTEGER
    DEFAULT
    0,
    track_number
    INTEGER
    DEFAULT
    0,
    disc_number
    INTEGER
    DEFAULT
    1,
    music_brainz_id
    TEXT,
    source
    TEXT,
    cover_art_path
    TEXT,
    resolved_track_id
    INTEGER
    DEFAULT
    0,
    resolution_status
    TEXT
    NOT
    NULL
    DEFAULT
    'pending',
    resolution_confidence
    INTEGER
    DEFAULT
    0,
    library_applied
    INTEGER
    DEFAULT
    0,
    created_at
    TEXT
    NOT
    NULL,
    updated_at
    TEXT
    NOT
    NULL
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_play_records_play_time ON track_play_records(play_time DESC);
CREATE INDEX IF NOT EXISTS idx_play_records_artist ON track_play_records(artist);
CREATE INDEX IF NOT EXISTS idx_play_records_source ON track_play_records(source);
CREATE INDEX IF NOT EXISTS idx_play_records_album_id ON track_play_records(album_id);
CREATE INDEX IF NOT EXISTS idx_play_records_music_brainz_id ON track_play_records(music_brainz_id);
CREATE INDEX IF NOT EXISTS idx_play_records_resolved_track_id ON track_play_records(resolved_track_id);
CREATE INDEX IF NOT EXISTS idx_play_records_resolution_status ON track_play_records(resolution_status);
CREATE INDEX IF NOT EXISTS idx_play_records_library_applied ON track_play_records(library_applied);

-- 流派统计表
CREATE TABLE IF NOT EXISTS genres
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    name
    TEXT
    NOT
    NULL
    UNIQUE,
    name_zh
    TEXT,
    play_count
    INTEGER
    DEFAULT
    0,
    created_at
    TEXT
    NOT
    NULL,
    updated_at
    TEXT
    NOT
    NULL
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_genres_play_count ON genres(play_count DESC);

-- 新增优化索引
CREATE INDEX IF NOT EXISTS idx_tracks_genre_play_count ON tracks(genre, play_count DESC);
CREATE INDEX IF NOT EXISTS idx_tracks_artist_track ON tracks(artist, track);

-- Dashboard 总览统计表
CREATE TABLE IF NOT EXISTS dashboard_stat
(
    id           INTEGER PRIMARY KEY,
    total_plays  INTEGER DEFAULT 0,
    total_tracks INTEGER DEFAULT 0,
    total_artist INTEGER DEFAULT 0,
    total_albums INTEGER DEFAULT 0,
    updated_at   TEXT    NOT NULL
);

-- 播放来源统计表
CREATE TABLE IF NOT EXISTS play_source_stat
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    source     TEXT    NOT NULL UNIQUE,
    count      INTEGER DEFAULT 0,
    updated_at TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_play_source_stat_count ON play_source_stat(count DESC);

-- 热门艺术家统计表
CREATE TABLE IF NOT EXISTS top_artist_stat
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    period_days  INTEGER NOT NULL DEFAULT 0,
    metric_type  TEXT    NOT NULL, -- plays|tracks
    artist       TEXT    NOT NULL,
    metric_value INTEGER DEFAULT 0,
    rank         INTEGER NOT NULL,
    updated_at   TEXT    NOT NULL,
    UNIQUE (period_days, metric_type, artist)
);
CREATE INDEX IF NOT EXISTS idx_top_artist_period_metric_rank
    ON top_artist_stat(period_days, metric_type, rank);

-- 热门专辑统计表
CREATE TABLE IF NOT EXISTS top_album_stat
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    period_days INTEGER NOT NULL,
    album_id    INTEGER DEFAULT 0,
    album       TEXT    NOT NULL,
    album_subtitle TEXT,
    artist      TEXT    DEFAULT '',
    play_count  INTEGER DEFAULT 0,
    rank        INTEGER NOT NULL,
    updated_at  TEXT    NOT NULL,
    UNIQUE (period_days, album, artist)
);
CREATE INDEX IF NOT EXISTS idx_top_album_period_rank ON top_album_stat(period_days, rank);

-- 热门流派统计表
CREATE TABLE IF NOT EXISTS top_genre_stat
(
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    genre_name        TEXT    NOT NULL UNIQUE,
    genre_name_zh     TEXT    DEFAULT '',
    track_genre_count INTEGER DEFAULT 0,
    genre_count       INTEGER DEFAULT 0,
    rank              INTEGER NOT NULL,
    updated_at        TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_top_genre_rank ON top_genre_stat(rank);

-- 趋势统计（日）
CREATE TABLE IF NOT EXISTS play_trend_daily_stat
(
    stat_date  TEXT PRIMARY KEY, -- YYYY-MM-DD
    play_count INTEGER DEFAULT 0,
    updated_at TEXT    NOT NULL
);

-- 趋势统计（小时）
CREATE TABLE IF NOT EXISTS play_trend_hourly_stat
(
    stat_date  TEXT    NOT NULL, -- YYYY-MM-DD
    hour       INTEGER NOT NULL, -- 0-23
    play_count INTEGER DEFAULT 0,
    updated_at TEXT    NOT NULL,
    PRIMARY KEY (stat_date, hour)
);
CREATE INDEX IF NOT EXISTS idx_play_trend_hourly_date ON play_trend_hourly_stat(stat_date);

-- 同步元数据表 (记录最后同步时间)
CREATE TABLE IF NOT EXISTS sync_metadata
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    table_name
    TEXT
    NOT
    NULL
    UNIQUE,
    last_sync_time
    TEXT
    NOT
    NULL,
    sync_count
    INTEGER
    DEFAULT
    0,
    last_error
    TEXT,
    created_at
    TEXT
    NOT
    NULL,
    updated_at
    TEXT
    NOT
    NULL
);
