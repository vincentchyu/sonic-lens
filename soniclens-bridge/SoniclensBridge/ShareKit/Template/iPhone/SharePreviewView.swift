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

struct SharePreviewView: View {
    let payload: SharePayload

    var body: some View {
        EmptyView()
    }
}
#endif
