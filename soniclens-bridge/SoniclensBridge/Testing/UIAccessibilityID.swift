import Foundation

/// 统一的三端与 XCUITest 稳定无障碍标识（Release / Debug 共用）
enum UIAccessibilityID {
    static func sidebarTab(_ rawValue: String) -> String {
        "sidebar.tab.\(rawValue)"
    }

    static let toolbarTitleMenu = "toolbar.titleMenu"
    static let toolbarPerformanceMode = "toolbar.performanceMode"
    static let toolbarDisconnect = "toolbar.disconnect"
    static let toolbarSearchField = "toolbar.searchField"

    static let homeScrollView = "home.scrollView"
    static let homeHero = "home.hero"
    static let homeTrendSection = "home.trend.section"
    static let homeTrendHeatmap = "home.trend.heatmap"
    static let homeTrendCanvas = "home.trend.canvas"
    static let homeTrendTooltip = "home.trend.tooltip"
    static let homeTrendScrollHint = "home.trend.scrollHint"
    static let homeProfileCard = "home.profile.card"

    static let albumsScrollView = "albums.scrollView"
    static let albumsGrid = "albums.grid"

    static let tracksScrollView = "tracks.scrollView"
    static let tracksList = "tracks.list"
}
