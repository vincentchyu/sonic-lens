import Foundation

final class BonjourDiscovery: NSObject, ObservableObject {
    @Published var candidates: [ServerCandidate] = []
    @Published var isScanning = false
    @Published var lastErrorMessage: String?
    @Published var lastScanFinishedAt: Date?

    private let serviceType = "_soniclens._tcp."
    private let serviceDomain = "local."
    private let scanTimeout: TimeInterval = 4

    private var browser: NetServiceBrowser?
    private var services: [NetService] = []
    private var scanTimeoutWorkItem: DispatchWorkItem?

    func start() {
        stop()
        isScanning = true
        lastErrorMessage = nil
        lastScanFinishedAt = nil

        let browser = NetServiceBrowser()
        browser.delegate = self
        browser.searchForServices(ofType: serviceType, inDomain: serviceDomain)
        self.browser = browser

        let workItem = DispatchWorkItem { [weak self] in
            guard let self else { return }
            self.isScanning = false
            self.lastScanFinishedAt = Date()
            if self.candidates.isEmpty && self.lastErrorMessage == nil {
                self.lastErrorMessage = "未发现局域网服务器，确认客户端与服务端在同一网络并且 Bonjour 广播已启用"
            }
        }
        scanTimeoutWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + scanTimeout, execute: workItem)
    }

    func stop() {
        scanTimeoutWorkItem?.cancel()
        scanTimeoutWorkItem = nil
        browser?.stop()
        browser = nil
        services.forEach { $0.stop() }
        services.removeAll()
        candidates = []
        isScanning = false
    }
}

extension BonjourDiscovery: NetServiceBrowserDelegate {
    func netServiceBrowser(_ browser: NetServiceBrowser, didFind service: NetService, moreComing: Bool) {
        service.delegate = self
        services.append(service)
        service.resolve(withTimeout: 3)
        if !moreComing {
            publishCandidates()
        }
    }

    func netServiceBrowser(_ browser: NetServiceBrowser, didRemove service: NetService, moreComing: Bool) {
        services.removeAll { $0 == service }
        if !moreComing {
            publishCandidates()
        }
    }

    func netServiceBrowser(_ browser: NetServiceBrowser, didNotSearch errorDict: [String : NSNumber]) {
        lastErrorMessage = "局域网扫描失败，请检查本机“本地网络”权限或 Bonjour 是否可用"
        isScanning = false
        lastScanFinishedAt = Date()
    }

    func netServiceBrowserDidStopSearch(_ browser: NetServiceBrowser) {
        isScanning = false
        lastScanFinishedAt = Date()
    }
}

extension BonjourDiscovery: NetServiceDelegate {
    func netServiceDidResolveAddress(_ sender: NetService) {
        publishCandidates()
    }

    func netService(_ sender: NetService, didNotResolve errorDict: [String : NSNumber]) {
        publishCandidates()
    }

    private func publishCandidates() {
        let resolved = services.compactMap { service -> ServerCandidate? in
            guard let host = service.hostName, service.port > 0 else {
                return nil
            }
            return ServerCandidate(name: service.name, host: host, port: service.port)
        }
        candidates = resolved.sorted { lhs, rhs in
            if lhs.name == rhs.name {
                return lhs.host < rhs.host
            }
            return lhs.name < rhs.name
        }
        if !resolved.isEmpty {
            lastErrorMessage = nil
            isScanning = false
            lastScanFinishedAt = Date()
            scanTimeoutWorkItem?.cancel()
            scanTimeoutWorkItem = nil
        }
    }
}
