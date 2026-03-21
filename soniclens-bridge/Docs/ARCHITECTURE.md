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
- Shared share infrastructure:
  - `SoniclensBridge/ShareKit/Domain`, `Builder`, `Render`, `Action`, `Analytics`
  - shared share payload assembly, tagged insight parsing reuse, poster rendering, temp file export, photo save/share sheet coordination
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
  - iPhone-specific share poster templates and preview flow under `ShareKit/Template/iPhone`

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
- Library sort/filter/query state stays page-local; dense lists should receive only derived inputs and static container data, not broad high-frequency store updates.

## Networking
- REST: JSON over `URLSession`
- WebSocket: `URLSessionWebSocketTask`
- Base URL comes from discovery or manual server selection
- WebSocket reconnect uses retry/backoff logic in the networking layer

## Share Architecture
- Share capability uses a layered split:
  - `ShareKit/Builder`: transforms `Track` / `AlbumDetail` / lyrics / insight / favorite state into scene payloads
  - `ShareKit/Template/iPhone`: iPhone-only poster layout and preview UI
  - `ShareKit/Render`: poster measurement, single-image rendering, long-poster pagination, temp PNG export
  - `ShareKit/Action`: Photos save and `UIActivityViewController` share handoff
  - `ShareKit/Analytics`: local event logging abstraction
- Current phase wires iPhone `TrackDetailView` and `AlbumDetailView` into ShareKit.
- Do not move share data assembly into page-local SwiftUI bodies; keep it in builder/render/action layers.
- Bridge requests attach `X-SonicLens-Terminal` so the server can omit internal debug fields from app responses.

## Detail Page Responsibilities
- `TrackDetailViewModel` owns:
  - 歌词加载
  - 曲目音眸读取与生成
  - 模型选择状态
- `AlbumDetailViewModel` owns:
  - 专辑详情与曲目列表加载
  - MusicBrainz 候选整理
  - 专辑音眸读取与生成
  - 模型选择状态
- `AlbumDetailView` now keeps a dual-tab structure:
  - `信息`: 保留原专辑 hero、曲目列表、候选版本整理
  - `音眸`: 展示专辑总评、按 `GetAlbumInsightSchema()` 固定顺序渲染的 section、背景信息、时代语境与补充元数据

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
