import XCTest

final class PhoneDashboardUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testPhoneLaunchDashboardAndSwitchTabs() throws {
        let app = XCUIApplication()
        app.launchArguments = ["-ui-mock", "-ui-tab", "home", "-ui-appearance", "dark"]
        app.launch()

        let trendSection = app.descendants(matching: .any).matching(identifier: "home.trend.section").firstMatch
        XCTAssertTrue(trendSection.waitForExistence(timeout: 10), "iPhone 聆听趋势卡片未渲染")
        attachScreenshot(name: "phone_home_dark")

        let albumsTab = app.tabBars.buttons["专辑"]
        if albumsTab.waitForExistence(timeout: 4) {
            albumsTab.tap()
            let albumsGrid = app.descendants(matching: .any).matching(identifier: "albums.grid").firstMatch
            XCTAssertTrue(albumsGrid.waitForExistence(timeout: 6), "iPhone 专辑网格未加载")
            attachScreenshot(name: "phone_albums_dark")
        }

        let tracksTab = app.tabBars.buttons["曲目"]
        if tracksTab.waitForExistence(timeout: 4) {
            tracksTab.tap()
            let tracksList = app.descendants(matching: .any).matching(identifier: "tracks.list").firstMatch
            XCTAssertTrue(tracksList.waitForExistence(timeout: 6), "iPhone 曲目列表未加载")
            attachScreenshot(name: "phone_tracks_dark")
        }
    }

    private func attachScreenshot(name: String) {
        let attachment = XCTAttachment(screenshot: XCUIScreen.main.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
