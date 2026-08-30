import SwiftUI
import Foundation

enum AppRoute: Hashable {
    case albumDetail(albumID: Int64)
    case trackDetail(track: Track)
    case insightDetail(insight: Insight)
    case insightSummary(summary: InsightSummary)
}

struct NavigationSnapshot: Equatable {
    var tab: SidebarDestination
    var path: [AppRoute]
}

@MainActor
final class NavigationCoordinator: ObservableObject {
    @Published var selectedTab: SidebarDestination = .home
    @Published var path: [AppRoute] = [] {
        didSet {
            guard !isNavigatingHistory else { return }
            if path.count < previousPathCount {
                let poppedPath = Array(previousPath.suffix(previousPathCount - path.count))
                if let lastPopped = poppedPath.last {
                    var restoredPath = path
                    restoredPath.append(lastPopped)
                    forwardHistory.append(NavigationSnapshot(tab: selectedTab, path: restoredPath))
                }
            } else if path.count > previousPathCount {
                let lastSnapshot = NavigationSnapshot(tab: selectedTab, path: previousPath)
                if backHistory.last != lastSnapshot {
                    backHistory.append(lastSnapshot)
                }
                forwardHistory.removeAll()
            }
            previousPath = path
            previousPathCount = path.count
            updateCapabilities()
        }
    }

    @Published private(set) var canGoBack: Bool = false
    @Published private(set) var canGoForward: Bool = false

    private var backHistory: [NavigationSnapshot] = []
    private var forwardHistory: [NavigationSnapshot] = []
    private var isNavigatingHistory = false
    private var previousPath: [AppRoute] = []
    private var previousPathCount: Int = 0

    init() {
        updateCapabilities()
    }

    var currentSnapshot: NavigationSnapshot {
        NavigationSnapshot(tab: selectedTab, path: path)
    }

    private func updateCapabilities() {
        canGoBack = !backHistory.isEmpty || !path.isEmpty
        canGoForward = !forwardHistory.isEmpty
    }

    func selectTab(_ newTab: SidebarDestination) {
        guard newTab != selectedTab || !path.isEmpty else { return }
        if !isNavigatingHistory {
            let snap = currentSnapshot
            if backHistory.last != snap {
                backHistory.append(snap)
            }
            forwardHistory.removeAll()
        }
        selectedTab = newTab
        isNavigatingHistory = true
        path.removeAll()
        previousPath.removeAll()
        previousPathCount = 0
        isNavigatingHistory = false
        updateCapabilities()
    }

    func push(_ route: AppRoute) {
        if !isNavigatingHistory {
            let snap = currentSnapshot
            if backHistory.last != snap {
                backHistory.append(snap)
            }
            forwardHistory.removeAll()
        }
        path.append(route)
        previousPath = path
        previousPathCount = path.count
        updateCapabilities()
    }

    func goBack() {
        guard canGoBack else { return }
        isNavigatingHistory = true
        let current = currentSnapshot

        if !path.isEmpty {
            let popped = path.removeLast()
            var forwardTarget = path
            forwardTarget.append(popped)
            forwardHistory.append(NavigationSnapshot(tab: selectedTab, path: forwardTarget))
        } else if let previous = backHistory.popLast() {
            forwardHistory.append(current)
            selectedTab = previous.tab
            path = previous.path
        }
        previousPath = path
        previousPathCount = path.count
        isNavigatingHistory = false
        updateCapabilities()
    }

    func goForward() {
        guard canGoForward else { return }
        guard let next = forwardHistory.popLast() else { return }
        isNavigatingHistory = true
        let current = currentSnapshot
        if backHistory.last != current {
            backHistory.append(current)
        }
        selectedTab = next.tab
        path = next.path
        previousPath = path
        previousPathCount = path.count
        isNavigatingHistory = false
        updateCapabilities()
    }

    func popToRoot() {
        guard !path.isEmpty else { return }
        isNavigatingHistory = true
        path.removeAll()
        previousPath.removeAll()
        previousPathCount = 0
        isNavigatingHistory = false
        updateCapabilities()
    }
}
