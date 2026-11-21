import XCTest

final class DashboardMacUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testLaunchDashboardAndSwitchTabs() throws {
        let app = XCUIApplication()
        app.launchArguments = ["-ui-mock", "-ui-tab", "home", "-ui-appearance", "light"]
        app.launch()

        let window = app.windows.firstMatch
        XCTAssertTrue(window.waitForExistence(timeout: 10), "macOS 主窗口未出现")

        let trendSection = app.descendants(matching: .any).matching(identifier: "home.trend.section").firstMatch
        XCTAssertTrue(trendSection.waitForExistence(timeout: 8), "聆听趋势卡片未渲染")
        attachScreenshot(of: window, name: "mac_home_light")

        let albumsTab = app.descendants(matching: .any).matching(identifier: "sidebar.tab.albums").firstMatch
        XCTAssertTrue(albumsTab.waitForExistence(timeout: 5), "侧边栏专辑入口未找到")
        albumsTab.click()

        let albumsGrid = app.descendants(matching: .any).matching(identifier: "albums.grid").firstMatch
        XCTAssertTrue(albumsGrid.waitForExistence(timeout: 6), "专辑网格未加载")
        attachScreenshot(of: window, name: "mac_albums_light")

        let tracksTab = app.descendants(matching: .any).matching(identifier: "sidebar.tab.tracks").firstMatch
        XCTAssertTrue(tracksTab.waitForExistence(timeout: 5), "侧边栏曲目入口未找到")
        tracksTab.click()

        let tracksList = app.descendants(matching: .any).matching(identifier: "tracks.list").firstMatch
        XCTAssertTrue(tracksList.waitForExistence(timeout: 6), "曲目列表未加载")
        attachScreenshot(of: window, name: "mac_tracks_light")
    }

    private func attachScreenshot(of element: XCUIElement, name: String) {
        let attachment = XCTAttachment(screenshot: element.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
