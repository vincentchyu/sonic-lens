# IA & Page Specs

## Navigation
- Tab Bar
  - Home
  - Library
- Global Mini Player (always visible, above Tab Bar)
- Fullscreen Player (modal/push)

## Home (Dashboard)
Purpose: showcase SonicLens server capabilities.

Sections:
- Stats cards (plays, tracks, albums, artists, insights, etc.)
- Trend chart (range switcher)
- Rankings
  - Top artists (by plays / by tracks)
  - Top albums
  - Top genres
- Recent plays

Primary data sources:
- /api/dashboard/stats
- /api/dashboard/trend
- /api/dashboard/top-artists/plays
- /api/dashboard/top-artists/tracks
- /api/dashboard/top-albums
- /api/dashboard/top-genres
- /api/recent-plays

## Library
Purpose: structured access to your music data.

Sections:
- Albums list
- Tracks list
- Insights list (音眸)
- Unscrobbled list (未上报)

Primary data sources:
- /api/albums
- /api/tracks
- /api/insights/all
- /api/unscrobbled-records

## Mini Player
Purpose: always-on playback context.

Content:
- Artwork
- Track title + artist
- Current position / duration (if available)
- Tap to open Fullscreen Player

Primary data sources:
- WebSocket /ws (now_playing)

## Fullscreen Player
Purpose: immersive playback and analysis.

Content:
- Artwork + core metadata
- Lyrics panel
- AI Insight panel (音眸)
- Favorite / action buttons

Primary data sources:
- /api/track-lyrics
- /api/track-insight
- /api/track-insight-stream
- /api/favorite
