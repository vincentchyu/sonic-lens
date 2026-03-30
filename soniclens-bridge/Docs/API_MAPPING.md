# API Mapping

## Home
- Stats: GET /api/dashboard/stats
- Trend: GET /api/dashboard/trend?range=7d|30d|365d
- Top Artists (plays): GET /api/dashboard/top-artists/plays?limit=10（含 avatar_url / avatar_object_key）
- Top Artists (tracks): GET /api/dashboard/top-artists/tracks?limit=10（含 avatar_url / avatar_object_key）
- Top Albums: GET /api/dashboard/top-albums?days=30&limit=10
- Top Genres: GET /api/dashboard/top-genres?limit=10
- Recent Plays: GET /api/recent-plays?limit=20

## Library
- Albums: GET /api/albums?limit=50&offset=0
- Album Detail: GET /api/albums/:id
- Tracks: GET /api/tracks?limit=50&offset=0
- Insights: GET /api/insights/all?limit=50&offset=0&analysis_target_type=track|album
- Insight Detail: GET /api/insights/:id?analysis_target_type=track|album
- Insight Logs: GET /api/insights/:id/logs?analysis_target_type=track|album
- Insight Delete: DELETE /api/insights/:id?analysis_target_type=track|album
- Unscrobbled: GET /api/unscrobbled-records?limit=50&offset=0

## Player
- Now Playing (WS): ws://<host>/ws
- Insight Job Updates (WS event): `insight_job_updated`，终态 payload 会带 `result_insight_id`
- Artwork Resolve: GET /api/artwork/resolve?albumArtist=...&artist=...&album=...&artworkKey=...
- Lyrics: GET /api/track-lyrics?artist=...&track=...&album=...
- Insight Job Create: POST /api/insight-jobs { target_type, artist?, album?, track?, track_number?, disc_number?, album_id?, provider, model, client_platform }
- Insight Job Status: GET /api/insight-jobs/:id
- Insight Live Activity Token: POST /api/insight-jobs/:id/live-activity-token { token }
- Insight: GET /api/track-insight?artist=...&track=...&album=...
- Album Insight: GET /api/album-insight?albumID=...
- Async Completion Read: 任务终态优先使用 GET /api/insights/:id?analysis_target_type=track|album，只有缺失 `result_insight_id` 时才回退身份查询
- Album Insight Generate: POST /api/album-insight { album_id, provider, model }
- Legacy Album Logs: GET /api/album-insights/:id/logs
- Insight Stream: GET /api/track-insight-stream?artist=...&track=...&album=...
- AI Platforms: GET /api/ai-models
- AI Platform Models: GET /api/ai-models/:platform/models
- Favorite: POST /api/favorite
- Terminal Header: `X-SonicLens-Terminal: web|mac|ipad|iphone`

## Health
- GET /health
