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
    track
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
    source
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

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_play_records_play_time ON track_play_records(play_time DESC);
CREATE INDEX IF NOT EXISTS idx_play_records_artist ON track_play_records(artist);
CREATE INDEX IF NOT EXISTS idx_play_records_source ON track_play_records(source);

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
