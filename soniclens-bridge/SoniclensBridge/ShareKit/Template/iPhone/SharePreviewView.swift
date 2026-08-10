#if os(iOS)
import SwiftUI

private struct SharePreviewNotice: Identifiable {
    enum Style {
        case success
        case error
    }

    let id = UUID()
    let message: String
    let style: Style
}

struct SharePreviewView: View {
    let payload: SharePayload
    var analytics: ShareAnalyticsReporting = ShareAnalytics.shared

    @Environment(\.dismiss) private var dismiss
    @State private var isRendering = false
    @State private var notice: SharePreviewNotice?
    @State private var shareItems: [Any] = []
    @State private var saveOptionsPresented = false

    private let renderer = ShareRenderer.shared
    private let photoSaveCoordinator = PhotoSaveCoordinator()

    var body: some View {
        NavigationStack {
            ZStack(alignment: .bottom) {
                ScrollView {
                    SharePosterViewFactory.makeView(payload: payload)
                        .frame(width: LongPosterPaginator.posterWidth)
                        .clipShape(RoundedRectangle(cornerRadius: 32, style: .continuous))
                        .padding(.horizontal, 12)
                        .padding(.top, 16)
                        .padding(.bottom, 120)
                }
                .background(SonicTheme.background.ignoresSafeArea())

                actionBar
            }
            .navigationTitle("分享预览")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("关闭") {
                        dismiss()
                    }
                }
            }
            .overlay(alignment: .top) {
                if let notice {
                    Text(notice.message)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 12)
                        .background(
                            (notice.style == .success ? Color.green : Color.red)
                                .opacity(0.92),
                            in: Capsule()
                        )
                        .padding(.top, 10)
                        .transition(.move(edge: .top).combined(with: .opacity))
                }
            }
            .overlay {
                if isRendering {
                    LoadingOverlay()
                }
            }
            .sheet(isPresented: shareSheetPresented) {
                ShareSheetPresenter(items: shareItems)
            }
            .confirmationDialog("保存图片方式", isPresented: $saveOptionsPresented, titleVisibility: .visible) {
                Button("保存为长图") {
                    saveToPhotos(mode: .singleLongImage, label: "save_single")
                }
                Button("分页保存") {
                    saveToPhotos(mode: .pagedImages, label: "save_paged")
                }
                Button("取消", role: .cancel) {}
            } message: {
                Text("长图会尽量保持与预览一致；分页保存会拆成多张图片。")
            }
            .task {
                analytics.log(
                    event: .previewOpened,
                    scene: payload.scene,
                    metadata: ["track": payload.header.trackName]
                )
            }
        }
    }

    private var actionBar: some View {
        HStack(spacing: 12) {
            Button(action: presentSaveOptions) {
                Label("保存图片", systemImage: "square.and.arrow.down")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
            .tint(SonicTheme.primary)
            .disabled(isRendering)

            Button(action: shareExternally) {
                Label("系统分享", systemImage: "square.and.arrow.up")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .tint(SonicTheme.primary)
            .disabled(isRendering)
        }
        .padding(16)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 24, style: .continuous))
        .padding(.horizontal, 16)
        .padding(.bottom, 16)
    }

    private var shareSheetPresented: Binding<Bool> {
        Binding(
            get: { !shareItems.isEmpty },
            set: { newValue in
                if !newValue {
                    shareItems = []
                }
            }
        )
    }

    private func presentSaveOptions() {
        saveOptionsPresented = true
    }

    private func saveToPhotos(mode: ShareRenderer.RenderMode, label: String) {
        Task {
            await withRenderTask(label: label, mode: mode) { urls in
                try await photoSaveCoordinator.saveImageFiles(at: urls)
                analytics.log(event: .saveSucceeded, scene: payload.scene, metadata: ["pages": "\(urls.count)"])
                showNotice("已保存到照片", style: .success)
            } onError: { error in
                analytics.log(
                    event: .saveFailed,
                    scene: payload.scene,
                    metadata: ["reason": error.localizedDescription]
                )
                showNotice(error.localizedDescription, style: .error)
            }
        }
    }

    private func shareExternally() {
        Task {
            await withRenderTask(label: "share", mode: .singleLongImage) { urls in
                analytics.log(event: .shareTriggered, scene: payload.scene, metadata: ["pages": "\(urls.count)"])
                shareItems = ShareActivityItemFactory.makeItems(fileURLs: urls)
            } onError: { error in
                showNotice(error.localizedDescription, style: .error)
            }
        }
    }

    private func withRenderTask(
        label: String,
        mode: ShareRenderer.RenderMode,
        operation: @escaping ([URL]) async throws -> Void,
        onError: @escaping (Error) -> Void
    ) async {
        isRendering = true
        defer { isRendering = false }

        do {
            let result = try await renderer.render(payload: payload, mode: mode)
            analytics.log(
                event: .renderSucceeded,
                scene: payload.scene,
                metadata: ["pages": "\(result.pageCount)", "entry": label]
            )
            try await operation(result.fileURLs)
        } catch {
            analytics.log(
                event: .renderFailed,
                scene: payload.scene,
                metadata: ["reason": error.localizedDescription, "entry": label]
            )
            onError(error)
        }
    }

    private func showNotice(_ message: String, style: SharePreviewNotice.Style) {
        withAnimation(.easeInOut(duration: 0.2)) {
            notice = SharePreviewNotice(message: message, style: style)
        }

        Task {
            try? await Task.sleep(nanoseconds: 2_200_000_000)
            await MainActor.run {
                withAnimation(.easeInOut(duration: 0.2)) {
                    notice = nil
                }
            }
        }
    }
}
#else
import SwiftUI
import AppKit

private struct PreparedPreviewData {
    let plan: ShareContinuousPaginationPlan
    let allNodes: [ShareFlowItemNode]
    let footerPayload: ShareFooterPayload
}

struct SharePreviewView: View {
    let payload: SharePayload
    @Environment(\.dismiss) private var dismiss
    @State private var isRendering = false
    @State private var noticeMessage: String?
    @State private var noticeStyle: Color = .green

    @State private var preparedData: PreparedPreviewData?

    private let renderer = ShareRenderer.shared

    init(payload: SharePayload) {
        let t0 = CFAbsoluteTimeGetCurrent()
        print("[ShareTiming] 4. SharePreviewView.init 触发, scene: \(payload.scene)")
        self.payload = payload
        let tPlanStart = CFAbsoluteTimeGetCurrent()
        let plan = SharePayloadPaginator.makePlan(payload: payload, targetPageHeight: 1080)
        let allNodes = SharePayloadPaginator.extractFlowNodes(from: payload)
        let footerPayload = Self.extractFooter(for: payload)
        let tPlanEnd = CFAbsoluteTimeGetCurrent()
        print("[ShareTiming] 5. SharePayloadPaginator 分页计划构建完成，耗时: \(String(format: "%.2f", (tPlanEnd - tPlanStart) * 1000)) ms, init 总耗时: \(String(format: "%.2f", (tPlanEnd - t0) * 1000)) ms")
        _preparedData = State(initialValue: PreparedPreviewData(plan: plan, allNodes: allNodes, footerPayload: footerPayload))
    }

    private static func extractFooter(for payload: SharePayload) -> ShareFooterPayload {
        switch payload {
        case let .insight(p): return p.footer
        case let .lyrics(p): return p.footer
        case let .info(p): return p.footer
        case let .albumInfo(p): return p.footer
        case let .albumInsight(p): return p.footer
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            // 顶部 Header 状态栏
            HStack {
                Text("SonicLens 大屏海报导出预览 (16:9 Ultra HD)")
                    .font(.system(size: 14, weight: .bold, design: .rounded))
                    .foregroundStyle(.white)

                Spacer()

                Button("关闭") {
                    dismiss()
                }
                .keyboardShortcut(.escape, modifiers: [])
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 14)
            .background(Color.black.opacity(0.4))

            // 中间海报全图自适应等比缩放预览 (16:9 Fit Canvas)
            ZStack {
                Color.black.opacity(0.85).ignoresSafeArea()
                    .onTapGesture { dismiss() } // Click outside to dismiss

                GeometryReader { geo in
                    let availableWidth = geo.size.width - 60
                    let availableHeight = geo.size.height - 60
                    let scale = min(availableWidth / 1920, availableHeight / 1080, 1.0)

                    if let prepared = preparedData {
                        ScrollView(.horizontal, showsIndicators: true) {
                            HStack(spacing: 24) {
                                ForEach(prepared.plan.slices, id: \.id) { slice in
                                    let sliceNodes: [ShareFlowItemNode] = {
                                        if slice.startIndex <= slice.endIndex && slice.endIndex < prepared.allNodes.count {
                                            return Array(prepared.allNodes[slice.startIndex...slice.endIndex])
                                        }
                                        return []
                                    }()

                                    MacSharePaginatedPosterView(
                                        header: payload.header,
                                        footer: prepared.footerPayload,
                                        slice: slice,
                                        scene: payload.scene,
                                        nodes: sliceNodes,
                                        targetPageHeight: 1080,
                                        renderedImage: nil
                                    )
                                    .scaleEffect(scale)
                                    .frame(width: 1920 * scale, height: 1080 * scale)
                                    .shadow(color: Color.black.opacity(0.6), radius: 24, x: 0, y: 12)
                                }
                            }
                            .padding(.horizontal, 30)
                            .frame(minWidth: geo.size.width, minHeight: geo.size.height, alignment: .center)
                        }
                    }
                }
            }

            // 底部控制动作条
            HStack(spacing: 16) {
                if let noticeMessage {
                    Text(noticeMessage)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 14)
                        .padding(.vertical, 8)
                        .background(noticeStyle.opacity(0.9), in: Capsule())
                }

                Spacer()

                Button(action: copyToClipboard) {
                    Label("复制图片 (Cmd+C)", systemImage: "doc.on.doc")
                }
                .keyboardShortcut("c", modifiers: [.command])

                Button(action: saveImageFile) {
                    Label("保存为 PNG 文件", systemImage: "square.and.arrow.down")
                }

                Button(action: sharePicker) {
                    Label("系统分享", systemImage: "square.and.arrow.up")
                }
            }
            .padding(16)
            .background(Color(nsColor: .windowBackgroundColor))
        }
        .frame(minWidth: 1000, minHeight: 680)
        .onAppear {
            print("[ShareTiming] 6. SharePreviewView.onAppear (macOS) 视图挂载完成！scene: \(payload.scene)")
        }
    }

    private func copyToClipboard() {
        Task {
            isRendering = true
            defer { isRendering = false }
            do {
                let result = try await renderer.render(payload: payload)
                if MacShareActionHelper.copyImagesToPasteboard(fileURLs: result.fileURLs) {
                    showNotice("已复制 \(result.fileURLs.count) 张图片到剪贴板！", color: .green)
                }
            } catch {
                showNotice("复制失败: \(error.localizedDescription)", color: .red)
            }
        }
    }

    private func saveImageFile() {
        Task {
            isRendering = true
            defer { isRendering = false }
            do {
                let result = try await renderer.render(payload: payload)
                guard !result.fileURLs.isEmpty else { return }

                let panel = NSSavePanel()
                panel.allowedContentTypes = [.png]
                panel.nameFieldStringValue = "\(payload.filename).png"
                panel.canCreateDirectories = true
                panel.begin { response in
                    if response == .OK, let targetURL = panel.url {
                        if result.fileURLs.count == 1 {
                            try? FileManager.default.copyItem(at: result.fileURLs[0], to: targetURL)
                        } else {
                            let baseName = targetURL.deletingPathExtension().path
                            let ext = targetURL.pathExtension
                            for (index, url) in result.fileURLs.enumerated() {
                                let target = URL(fileURLWithPath: "\(baseName)-\(index + 1).\(ext)")
                                try? FileManager.default.copyItem(at: url, to: target)
                            }
                        }
                        showNotice("保存成功 (\(result.fileURLs.count) 页)！", color: .green)
                    }
                }
            } catch {
                showNotice("导出失败: \(error.localizedDescription)", color: .red)
            }
        }
    }

    private func sharePicker() {
        Task {
            isRendering = true
            defer { isRendering = false }
            do {
                let result = try await renderer.render(payload: payload)
                guard !result.fileURLs.isEmpty else { return }
                if let window = NSApp.keyWindow, let contentView = window.contentView {
                    MacShareActionHelper.showSharingPicker(fileURLs: result.fileURLs, relativeTo: contentView.bounds, of: contentView)
                }
            } catch {
                showNotice("分享失败", color: .red)
            }
        }
    }

    private func showNotice(_ message: String, color: Color) {
        noticeMessage = message
        noticeStyle = color
        Task {
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            await MainActor.run { noticeMessage = nil }
        }
    }

    private func footer(for payload: SharePayload) -> ShareFooterPayload {
        switch payload {
        case let .insight(p): return p.footer
        case let .lyrics(p): return p.footer
        case let .info(p): return p.footer
        case let .albumInfo(p): return p.footer
        case let .albumInsight(p): return p.footer
        }
    }
}
#endif
