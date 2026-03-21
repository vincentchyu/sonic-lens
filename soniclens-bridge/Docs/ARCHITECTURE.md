# Architecture

## Overview
- `soniclens-bridge` is a three-product client line:
  - `SoniclensBridgeMac`
  - `SoniclensBridgePad`
  - `SoniclensBridgePhone`
- The architecture follows a split between:
  - shared core/state/data-loading layers
  - platform-specific app shells and experience layers
- Shared logic should be pushed into `SoniclensCore` and shared `ViewModels` first, then composed differently by Mac / iPad / iPhone containers.

## Entry And Container Topology
- macOS:
  - `SoniclensBridgeApp.swift`
  - `RootView -> MacRootView -> AppLayoutView`
- iPad:
  - `SoniclensBridgePadApp.swift`
  - `PadRootView -> PadAppLayoutView`
- iPhone:
  - `SoniclensBridgePhoneApp.swift`
  - `PhoneRootView -> PhoneAppLayoutView`

Notes:
- All app entries create and inject a shared `AppStore`.
- iPad and iPhone do not route through the macOS `RootView`.
- App shell differences belong to platform containers, not to shared core modules.

## Shared Core

### SoniclensCore
- `Networking`
  - `APIClient`: REST JSON requests
  - `NowPlayingService`: playback-related fetches
  - `WebSocketClient`: real-time updates
- `Discovery`
  - Bonjour discovery plus manual server override
- `Models`
  - API response and client-facing model types
- `Store`
  - `AppStore`: global app state
  - `LibraryIndexStore`: local library index access
  - `LibrarySyncService`: incremental library sync orchestration
  - `SnapshotCache`: lightweight UI snapshot cache

### Shared ViewModels
- `HomeViewModel`
- `LibraryViewModel`
- `PlayerViewModel`
- `AlbumDetailViewModel`
- `TrackDetailViewModel`

Rules:
- `AppStore` owns cross-screen app state such as server selection, connection, now playing, favorites, and library update notifications.
- Page-local `ViewModel` types own loading state, refresh logic, pagination state, and view-facing derived state.
- Expensive IO and orchestration stay out of SwiftUI `body`.

## Experience Layers

### Shared Pages And Components
- Shared pages:
  - `LibraryView`
  - `AlbumDetailView`
  - `TrackDetailView`
- Shared visual system:
  - `Theme.swift`
  - `GlassStyles.swift`
  - shared controls inside `Views/`
- Shared dashboard pieces:
  - `DashboardTrendSection`
  - `TrendHeatmapView`
  - ranking / stats / genre / album cards in `HomeView.swift`

### Platform-Specific Layers
- macOS:
  - window and split-view semantics
  - immersive now playing behaviors
- iPad:
  - tablet navigation shell
  - iPad-specific home and now-playing composition
- iPhone:
  - compact tab shell
  - compact home and now-playing composition
  - iPhone-specific dashboard presentation strategy when a shared data model needs a different layout density

## Home Dashboard Data Flow
- Home dashboard data is loaded by shared `HomeViewModel`.
- Current home composition pulls:
  - dashboard stats
  - top artists
  - top albums
  - top genres
  - top tracks
  - recent plays
  - trend / heatmap data from `/api/dashboard/trend`
- Trend data flow is:
  - shared fetch and model decoding in `HomeViewModel`
  - shared visualization primitives in `DashboardTrendSection` and `TrendHeatmapView`
  - platform-specific range, label density, and action placement decided by the container view

Current platform split:
- Mac / iPad continue to use the broader shared dashboard presentation.
- iPhone home uses a compact 30-day heatmap presentation.
- iPhone exposes 90-day heatmap browsing in a dedicated detail surface instead of forcing the full timeline into the home card.

This is an experience-layer divergence, not a separate data architecture.

## Library Architecture
- Library is local-first on the Bridge side.
- The supported model is:
  - local SQLite lightweight index
  - FTS5 search
  - `/api/library/sync` incremental sync
  - `library_updated(version)` WebSocket push
  - lazy detail loading

Rules:
- Do not regress to remote pagination plus large in-memory filtering/sorting for library pages.
- Sync, indexing, and cache invalidation logic should stay in shared core/store layers.

## Networking
- REST: JSON over `URLSession`
- WebSocket: `URLSessionWebSocketTask`
- Base URL comes from discovery or manual server selection
- WebSocket reconnect uses retry/backoff logic in the networking layer

## Documentation Boundary
- Update this file when:
  - product-line topology changes
  - shared vs platform-private responsibility changes
  - core data-flow architecture changes
  - local-first library architecture changes
- Do not use this file for:
  - per-screen spacing tweaks
  - one-off visual polish decisions
  - heatmap label angle / button placement / card alignment details

For platform ownership and impact analysis, also maintain:
- `Docs/CLIENT_MODULE_BOUNDARY.md`
