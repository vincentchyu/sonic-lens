# IA & Page Specs

## Navigation

- macOS Sidebar / iOS Tab Bar
    - Home (`Ctrl + 1`)
    - Albums (`Ctrl + 2`)
    - Tracks (`Ctrl + 3`)
    - Unreported (`Ctrl + 4`)
    - SonicLens Insights (`Ctrl + 5`)
    - Future Features (`Ctrl + 6`)
- Global Navigation Shortcuts (macOS)
    - Back: `Command + [`
    - Forward: `Command + ]`
    - Now Playing Toggle / Close: `Command + J`
- Global Mini Player (always visible, above bottom / sidebar)
- Fullscreen / Immersive Player (modal/overlay)

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
- /api/insights/:id?analysis_target_type=track|album
- /api/insights/:id/logs?analysis_target_type=track|album
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
