import SwiftUI
import XCTest
@testable import SoniclensBridgeMac

@MainActor
final class TrendHeatmapSnapshotTests: XCTestCase {
    func testTrendFixtureAndHeightContract() {
        let trend = UIVerificationFixtures.trendData(days: 90, kind: .dense)
        XCTAssertEqual(trend.points.count, 90, "90 天趋势窗口必须严格返回 90 条每日数据")
        XCTAssertEqual(trend.hourlyData.count, 90, "90 天小时分布必须严格返回 90 条数据")
        XCTAssertEqual(trend.points.first?.date, "2026-08-23")
        XCTAssertEqual(trend.points.last?.date, "2026-11-20")
        XCTAssertFalse(trend.points.contains(where: { $0.date == "2026-11-21" }), "不应包含未来日期 2026-11-21")

        let macHeight = TrendHeatmapView.preferredHeight(
            fallbackHeight: 276,
            layout: .fixedWidthScrollable(cellWidth: 8),
            axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
        )
        XCTAssertEqual(macHeight, 316, accuracy: 0.01)

        let padHeight = TrendHeatmapView.preferredHeight(
            fallbackHeight: 276,
            layout: .fixedWidthScrollable(cellWidth: 7),
            axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
        )
        XCTAssertEqual(padHeight, 292, accuracy: 0.01)
    }

    func testRenderTrendHeatmapSnapshots() throws {
        let trend90 = UIVerificationFixtures.trendData(days: 90, kind: .dense)
        let trend30 = UIVerificationFixtures.trendData(days: 30, kind: .dense)
        let emptyTrend = UIVerificationFixtures.trendData(days: 90, kind: .empty)

        let macLightURL = try SnapshotRenderer.render(
            snapshotContainer {
                DashboardTrendSection(
                    points: trend90.points,
                    hourlyData: trend90.hourlyData,
                    subtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                    heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                    axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
                )
            },
            named: "trend_macos_light.png",
            size: CGSize(width: 880, height: 470),
            appearance: .light
        )
        assertValidPNG(at: macLightURL)

        let macDarkURL = try SnapshotRenderer.render(
            snapshotContainer {
                DashboardTrendSection(
                    points: trend90.points,
                    hourlyData: trend90.hourlyData,
                    subtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                    heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                    axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
                )
            },
            named: "trend_macos_dark.png",
            size: CGSize(width: 880, height: 470),
            appearance: .dark
        )
        assertValidPNG(at: macDarkURL)

        // 峰值日 2026-10-24（按日期降序索引为 27），05:00（hourIndex = 18）
        let macHoverLightURL = try SnapshotRenderer.render(
            snapshotContainer {
                DashboardTrendSection(
                    points: trend90.points,
                    hourlyData: trend90.hourlyData,
                    subtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                    heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                    axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42),
                    previewHoveredCell: (dayIndex: 27, hourIndex: 18)
                )
            },
            named: "trend_macos_hover_light.png",
            size: CGSize(width: 880, height: 470),
            appearance: .light
        )
        assertValidPNG(at: macHoverLightURL)

        // 最左上角边界测试：dayIndex = 0（11/20），hourIndex = 0（23:00），验证向下翻转且左边缘不裁切
        let macHoverEdgeDarkURL = try SnapshotRenderer.render(
            snapshotContainer {
                DashboardTrendSection(
                    points: trend90.points,
                    hourlyData: trend90.hourlyData,
                    subtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                    heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                    axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42),
                    previewHoveredCell: (dayIndex: 0, hourIndex: 0)
                )
            },
            named: "trend_macos_hover_edge_dark.png",
            size: CGSize(width: 880, height: 470),
            appearance: .dark
        )
        assertValidPNG(at: macHoverEdgeDarkURL)

        let phoneLightURL = try SnapshotRenderer.render(
            snapshotContainer(padding: 12) {
                DashboardTrendSection(
                    points: trend30.points,
                    hourlyData: trend30.hourlyData,
                    title: "聆听趋势",
                    subtitle: "聚焦最近 30 天，保留更清晰的日期列和时段密度。",
                    heatmapHeight: 264,
                    heatmapLayout: .fitted(minCellWidth: 6),
                    axisLabelStyle: .dayStride(step: 4, rotationDegrees: 35)
                )
            },
            named: "trend_phone_light.png",
            size: CGSize(width: 360, height: 440),
            appearance: .light
        )
        assertValidPNG(at: phoneLightURL)

        let emptyDarkURL = try SnapshotRenderer.render(
            snapshotContainer {
                DashboardTrendSection(
                    points: emptyTrend.points,
                    hourlyData: emptyTrend.hourlyData,
                    subtitle: "空数据边界渲染自检。",
                    heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                    axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
                )
            },
            named: "trend_empty_dark.png",
            size: CGSize(width: 880, height: 470),
            appearance: .dark
        )
        assertValidPNG(at: emptyDarkURL)

        print("UI_SNAPSHOT_OUTPUT_DIR=\(SnapshotRenderer.outputDirectory().path)")
    }

    @ViewBuilder
    private func snapshotContainer<Content: View>(
        padding: CGFloat = 20,
        @ViewBuilder content: () -> Content
    ) -> some View {
        ZStack(alignment: .topLeading) {
            SonicTheme.background
            content()
                .padding(padding)
        }
    }

    private func assertValidPNG(at url: URL, file: StaticString = #filePath, line: UInt = #line) {
        XCTAssertTrue(FileManager.default.fileExists(atPath: url.path), "快照文件不存在: \(url.path)", file: file, line: line)
        let data = try? Data(contentsOf: url)
        XCTAssertGreaterThan(data?.count ?? 0, 2048, "快照文件大小异常偏小: \(url.path)", file: file, line: line)
        print("UI_SNAPSHOT_FILE=\(url.path)")
    }
}
