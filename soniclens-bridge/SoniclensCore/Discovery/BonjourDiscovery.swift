import Foundation
import OSLog

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
    private var publishCandidatesWorkItem: DispatchWorkItem?
    private var lastPublishedSignature: String = ""
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "BonjourDiscovery")

    func start() {
        stop()
        isScanning = true
        lastErrorMessage = nil
        lastScanFinishedAt = nil
        logger.info("开始局域网服务器扫描")

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
            self.logger.info("局域网扫描超时结束，候选数量 \(self.candidates.count)")
        }
        scanTimeoutWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + scanTimeout, execute: workItem)
    }

    func stop() {
        logger.debug("停止局域网扫描")
        scanTimeoutWorkItem?.cancel()
        scanTimeoutWorkItem = nil
        publishCandidatesWorkItem?.cancel()
        publishCandidatesWorkItem = nil
        browser?.stop()
        browser = nil
        services.forEach { $0.stop() }
        services.removeAll()
        candidates = []
        isScanning = false
        lastPublishedSignature = ""
    }
}

extension BonjourDiscovery: NetServiceBrowserDelegate {
    func netServiceBrowser(_ browser: NetServiceBrowser, didFind service: NetService, moreComing: Bool) {
        service.delegate = self
        services.append(service)
        logger.debug("发现 Bonjour 服务 \(service.name, privacy: .public)，等待解析")
        service.resolve(withTimeout: 3)
        if !moreComing {
            schedulePublishCandidates()
        }
    }

    func netServiceBrowser(_ browser: NetServiceBrowser, didRemove service: NetService, moreComing: Bool) {
        services.removeAll { $0 == service }
        logger.debug("移除 Bonjour 服务 \(service.name, privacy: .public)")
        if !moreComing {
            schedulePublishCandidates()
        }
    }

    func netServiceBrowser(_ browser: NetServiceBrowser, didNotSearch errorDict: [String : NSNumber]) {
        lastErrorMessage = "局域网扫描失败，请检查本机“本地网络”权限或 Bonjour 是否可用"
        isScanning = false
        lastScanFinishedAt = Date()
        logger.error("局域网扫描失败，错误字典 \(String(describing: errorDict), privacy: .public)")
    }

    func netServiceBrowserDidStopSearch(_ browser: NetServiceBrowser) {
        isScanning = false
        lastScanFinishedAt = Date()
        logger.debug("局域网扫描已停止")
    }
}

extension BonjourDiscovery: NetServiceDelegate {
    func netServiceDidResolveAddress(_ sender: NetService) {
        logger.debug("Bonjour 服务解析成功 \(sender.name, privacy: .public)")
        schedulePublishCandidates()
    }

    func netService(_ sender: NetService, didNotResolve errorDict: [String : NSNumber]) {
        logger.debug("Bonjour 服务解析失败 \(sender.name, privacy: .public)，准备刷新候选列表")
        schedulePublishCandidates()
    }

    private func schedulePublishCandidates() {
        publishCandidatesWorkItem?.cancel()

        let workItem = DispatchWorkItem { [weak self] in
            self?.publishCandidates()
        }
        publishCandidatesWorkItem = workItem
        logger.debug("安排刷新候选服务器列表")
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.12, execute: workItem)
    }

    private func publishCandidates() {
        let startedAt = CFAbsoluteTimeGetCurrent()
        let resolved = services.compactMap { service -> ServerCandidate? in
            guard let host = service.hostName, service.port > 0 else {
                return nil
            }
            return ServerCandidate(name: service.name, host: host, port: service.port)
        }.sorted { lhs, rhs in
            if lhs.name == rhs.name {
                return lhs.host < rhs.host
            }
            return lhs.name < rhs.name
        }

        let signature = resolved.map { "\($0.name)|\($0.host)|\($0.port)" }.joined(separator: "||")
        guard signature != lastPublishedSignature else {
            let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
            logger.debug("候选服务器列表未变化，跳过发布，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
            return
        }
        lastPublishedSignature = signature
        candidates = resolved
        let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
        logger.info("发布候选服务器列表，数量 \(resolved.count, privacy: .public)，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
        if !resolved.isEmpty {
            lastErrorMessage = nil
            isScanning = false
            lastScanFinishedAt = Date()
            scanTimeoutWorkItem?.cancel()
            scanTimeoutWorkItem = nil
        }
    }
}
