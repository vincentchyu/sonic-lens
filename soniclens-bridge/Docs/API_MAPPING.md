# API Mapping

## Home
- Stats: GET /api/dashboard/stats
- Trend: GET /api/dashboard/trend?range=7d|30d|365d
- Top Artists (plays): GET /api/dashboard/top-artists/plays?limit=10
- Top Artists (tracks): GET /api/dashboard/top-artists/tracks?limit=10
- Top Albums: GET /api/dashboard/top-albums?days=30&limit=10
- Top Genres: GET /api/dashboard/top-genres?limit=10
- Recent Plays: GET /api/recent-plays?limit=20

## Library
- Albums: GET /api/albums?limit=50&offset=0
- Album Detail: GET /api/albums/:id
- Tracks: GET /api/tracks?limit=50&offset=0
- Insights: GET /api/insights/all?limit=50&offset=0
- Unscrobbled: GET /api/unscrobbled-records?limit=50&offset=0

## Player
- Now Playing (WS): ws://<host>/ws
- Lyrics: GET /api/track-lyrics?artist=...&track=...&album=...
- Insight: GET /api/track-insight?artist=...&track=...&album=...
- Insight Stream: GET /api/track-insight-stream?artist=...&track=...&album=...
- Favorite: POST /api/favorite

## Health
- GET /health
