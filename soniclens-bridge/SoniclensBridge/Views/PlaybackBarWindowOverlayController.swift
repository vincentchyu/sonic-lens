#if os(macOS)
import SwiftUI
import AppKit

@MainActor
final class PlaybackBarWindowOverlayController {
    private weak var window: NSWindow?
    private weak var hostView: NSView?
    private var overlayView: PlaybackBarWindowOverlayView?
    private let model = PlaybackBarContentModel()
    private var lastAppliedState: PlaybackBarOverlayRenderState?

    func attach(to window: NSWindow?) {
        guard self.window !== window else { return }
        detach()

        guard let window, let contentView = window.contentView else { return }
        let hostView = contentView.superview ?? contentView
        self.window = window
        self.hostView = hostView

        let overlayView = PlaybackBarWindowOverlayView(model: model)
        overlayView.translatesAutoresizingMaskIntoConstraints = false
        hostView.addSubview(overlayView, positioned: .above, relativeTo: contentView)

        NSLayoutConstraint.activate([
            overlayView.leadingAnchor.constraint(equalTo: hostView.leadingAnchor, constant: 16),
            overlayView.trailingAnchor.constraint(equalTo: hostView.trailingAnchor, constant: -16),
            overlayView.bottomAnchor.constraint(equalTo: hostView.bottomAnchor, constant: -10)
        ])

        self.overlayView = overlayView
    }

    func update(
        nowPlaying: NowPlaying?,
        performanceModeEnabled: Bool,
        isVisible: Bool,
        style: PlaybackBarStyle,
        onActivate: @escaping () -> Void
    ) {
        let state = PlaybackBarOverlayRenderState(
            nowPlaying: nowPlaying,
            performanceModeEnabled: performanceModeEnabled,
            isVisible: isVisible,
            style: style
        )
        guard lastAppliedState != state else {
            overlayView?.updateInteraction(onActivate: onActivate)
            return
        }
        lastAppliedState = state
        Task { @MainActor [weak self] in
            guard let self else { return }
            self.model.update(
                nowPlaying: nowPlaying,
                performanceModeEnabled: performanceModeEnabled
            )
        }
        overlayView?.update(
            style: style,
            performanceModeEnabled: performanceModeEnabled,
            isVisible: isVisible,
            onActivate: onActivate
        )
    }

    func detach() {
        model.stop()
        lastAppliedState = nil
        overlayView?.removeFromSuperview()
        overlayView = nil
        hostView = nil
        window = nil
    }
}

struct PlaybackBarWindowOverlayBridge: NSViewRepresentable {
    let controller: PlaybackBarWindowOverlayController
    let nowPlaying: NowPlaying?
    let performanceModeEnabled: Bool
    let isVisible: Bool
    let style: PlaybackBarStyle
    let onActivate: () -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(controller: controller)
    }

    func makeNSView(context: Context) -> NSView {
        let view = NSView(frame: .zero)
        view.isHidden = true
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        context.coordinator.controller = controller
        context.coordinator.controller?.attach(to: nsView.window)
        context.coordinator.controller?.update(
            nowPlaying: nowPlaying,
            performanceModeEnabled: performanceModeEnabled,
            isVisible: isVisible,
            style: style,
            onActivate: onActivate
        )
    }

    static func dismantleNSView(_ nsView: NSView, coordinator: Coordinator) {
        coordinator.controller?.detach()
    }

    final class Coordinator {
        var controller: PlaybackBarWindowOverlayController?

        init(controller: PlaybackBarWindowOverlayController) {
            self.controller = controller
        }
    }
}

private final class PlaybackBarWindowOverlayView: NSView {
    private let model: PlaybackBarContentModel
    private let activationRelay = PlaybackBarActivationRelay()
    private let glassView = NSVisualEffectView()
    private let fallbackView = NSView()
    private let hostingView: NSHostingView<PlaybackBarWindowOverlayRootView>
    private var heightConstraint: NSLayoutConstraint?
    private var currentStyle: PlaybackBarStyle = .regular

    init(model: PlaybackBarContentModel) {
        self.model = model
        hostingView = NSHostingView(
            rootView: PlaybackBarWindowOverlayRootView(
                model: model,
                style: .regular,
                activationRelay: activationRelay
            )
        )
        super.init(frame: .zero)
        commonInit()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func update(
        style: PlaybackBarStyle,
        performanceModeEnabled: Bool,
        isVisible: Bool,
        onActivate: @escaping () -> Void
    ) {
        activationRelay.onActivate = onActivate
        if currentStyle != style {
            currentStyle = style
            hostingView.rootView = PlaybackBarWindowOverlayRootView(
                model: model,
                style: style,
                activationRelay: activationRelay
            )
        }
        isHidden = !isVisible
        heightConstraint?.constant = style == .compact
        ? PlaybackBarView.compactHeight
        : PlaybackBarView.regularHeight

        glassView.isHidden = performanceModeEnabled
        fallbackView.isHidden = !performanceModeEnabled
        applySurfaceChrome(performanceModeEnabled: performanceModeEnabled)
        updateCornerRadius(for: style)
    }

    func updateInteraction(onActivate: @escaping () -> Void) {
        activationRelay.onActivate = onActivate
    }

    private func commonInit() {
        wantsLayer = true
        layer?.shadowColor = NSColor.black.withAlphaComponent(0.08).cgColor
        layer?.shadowOpacity = 1
        layer?.shadowRadius = 10
        layer?.shadowOffset = CGSize(width: 0, height: 4)

        glassView.translatesAutoresizingMaskIntoConstraints = false
        glassView.blendingMode = .withinWindow
        glassView.material = .hudWindow
        glassView.state = .active
        glassView.isEmphasized = false
        glassView.wantsLayer = true
        glassView.layer?.masksToBounds = true

        fallbackView.translatesAutoresizingMaskIntoConstraints = false
        fallbackView.wantsLayer = true
        fallbackView.layer?.masksToBounds = true
        fallbackView.isHidden = true

        hostingView.translatesAutoresizingMaskIntoConstraints = false
        hostingView.wantsLayer = true
        hostingView.layer?.backgroundColor = NSColor.clear.cgColor

        addSubview(glassView)
        addSubview(fallbackView)
        addSubview(hostingView)

        heightConstraint = heightAnchor.constraint(equalToConstant: PlaybackBarView.regularHeight)
        heightConstraint?.isActive = true

        NSLayoutConstraint.activate([
            glassView.leadingAnchor.constraint(equalTo: leadingAnchor),
            glassView.trailingAnchor.constraint(equalTo: trailingAnchor),
            glassView.topAnchor.constraint(equalTo: topAnchor),
            glassView.bottomAnchor.constraint(equalTo: bottomAnchor),

            fallbackView.leadingAnchor.constraint(equalTo: leadingAnchor),
            fallbackView.trailingAnchor.constraint(equalTo: trailingAnchor),
            fallbackView.topAnchor.constraint(equalTo: topAnchor),
            fallbackView.bottomAnchor.constraint(equalTo: bottomAnchor),

            hostingView.leadingAnchor.constraint(equalTo: leadingAnchor),
            hostingView.trailingAnchor.constraint(equalTo: trailingAnchor),
            hostingView.topAnchor.constraint(equalTo: topAnchor),
            hostingView.bottomAnchor.constraint(equalTo: bottomAnchor)
        ])

        updateCornerRadius(for: .regular)
        applySurfaceChrome(performanceModeEnabled: false)
    }

    private func applySurfaceChrome(performanceModeEnabled: Bool) {
        let borderWidth: CGFloat = performanceModeEnabled ? 0.5 : 0.75
        let borderColor = surfaceBorderColor(
            performanceModeEnabled: performanceModeEnabled,
            isDarkAppearance: isDarkAppearance
        ).cgColor
        let fillColor = surfaceFillColor(
            performanceModeEnabled: performanceModeEnabled,
            isDarkAppearance: isDarkAppearance
        ).cgColor

        glassView.layer?.borderWidth = borderWidth
        glassView.layer?.borderColor = borderColor

        fallbackView.layer?.backgroundColor = fillColor
        fallbackView.layer?.borderWidth = borderWidth
        fallbackView.layer?.borderColor = borderColor
    }

    private func surfaceBorderColor(
        performanceModeEnabled: Bool,
        isDarkAppearance: Bool
    ) -> PlatformColor {
        if isDarkAppearance {
            return PlatformColor.black.withAlphaComponent(performanceModeEnabled ? 0.16 : 0.24)
        }

        return PlatformColor
            .sonicWhite(1.0, alpha: performanceModeEnabled ? 0.42 : 0.66)
    }

    private func surfaceFillColor(
        performanceModeEnabled: Bool,
        isDarkAppearance: Bool
    ) -> PlatformColor {
        if isDarkAppearance {
            return PlatformColor
                .sonicWhite(0.1, alpha: performanceModeEnabled ? 0.78 : 0.84)
        }

        return PlatformColor
            .sonicWhite(1.0, alpha: performanceModeEnabled ? 0.88 : 0.92)
    }

    private var isDarkAppearance: Bool {
        window?.effectiveAppearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
    }

    private func updateCornerRadius(for style: PlaybackBarStyle) {
        let cornerRadius = style == .compact ? 16.0 : 18.0
        glassView.layer?.cornerRadius = cornerRadius
        fallbackView.layer?.cornerRadius = cornerRadius
    }
}

private struct PlaybackBarOverlayRenderState: Equatable {
    let nowPlayingToken: PlaybackBarOverlayNowPlayingToken
    let performanceModeEnabled: Bool
    let isVisible: Bool
    let style: PlaybackBarStyle

    init(
        nowPlaying: NowPlaying?,
        performanceModeEnabled: Bool,
        isVisible: Bool,
        style: PlaybackBarStyle
    ) {
        nowPlayingToken = PlaybackBarOverlayNowPlayingToken(nowPlaying: nowPlaying)
        self.performanceModeEnabled = performanceModeEnabled
        self.isVisible = isVisible
        self.style = style
    }
}

private struct PlaybackBarOverlayNowPlayingToken: Equatable {
    let artist: String
    let album: String
    let track: String
    let duration: Int
    let artwork: String

    init(nowPlaying: NowPlaying?) {
        artist = nowPlaying?.artist ?? ""
        album = nowPlaying?.album ?? ""
        track = nowPlaying?.track ?? ""
        duration = nowPlaying?.duration ?? 0
        artwork = nowPlaying?.artwork ?? ""
    }
}

private final class PlaybackBarActivationRelay {
    var onActivate: (() -> Void)?

    func activate() {
        onActivate?()
    }
}

private struct PlaybackBarWindowOverlayRootView: View {
    @ObservedObject var model: PlaybackBarContentModel
    let style: PlaybackBarStyle
    let activationRelay: PlaybackBarActivationRelay

    var body: some View {
        ZStack {
            Color.clear
            PlaybackBarContentView(
                model: model,
                style: style,
                onActivate: activationRelay.activate
            )
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

@MainActor
final class NowPlayingWindowOverlayController {
    private weak var window: NSWindow?
    private weak var hostView: NSView?
    private var overlayView: NowPlayingWindowOverlayView?
    private var lastRenderState: NowPlayingWindowOverlayRenderState?

    func attach(to window: NSWindow?) {
        guard self.window !== window else { return }
        detach()

        guard let window, let contentView = window.contentView else { return }
        let hostView = contentView.superview ?? contentView
        self.window = window
        self.hostView = hostView

        let overlayView = NowPlayingWindowOverlayView()
        overlayView.translatesAutoresizingMaskIntoConstraints = false
        hostView.addSubview(overlayView, positioned: .above, relativeTo: nil)

        NSLayoutConstraint.activate([
            overlayView.leadingAnchor.constraint(equalTo: hostView.leadingAnchor),
            overlayView.trailingAnchor.constraint(equalTo: hostView.trailingAnchor),
            overlayView.topAnchor.constraint(equalTo: hostView.topAnchor),
            overlayView.bottomAnchor.constraint(equalTo: hostView.bottomAnchor)
        ])

        self.overlayView = overlayView
    }

    func update(
        nowPlaying: NowPlaying?,
        appStore: AppStore,
        playbackStore: PlaybackStore,
        isVisible: Bool,
        onClose: @escaping () -> Void
    ) {
        let state = NowPlayingWindowOverlayRenderState(
            nowPlaying: nowPlaying,
            isVisible: isVisible
        )

        guard lastRenderState != state else {
            overlayView?.updateInteraction(onClose: onClose)
            return
        }

        lastRenderState = state
        overlayView?.update(
            nowPlaying: nowPlaying,
            appStore: appStore,
            playbackStore: playbackStore,
            isVisible: isVisible,
            onClose: onClose
        )
    }

    func detach() {
        lastRenderState = nil
        overlayView?.removeFromSuperview()
        overlayView = nil
        hostView = nil
        window = nil
    }
}

struct NowPlayingWindowOverlayBridge: NSViewRepresentable {
    let controller: NowPlayingWindowOverlayController
    let nowPlaying: NowPlaying?
    let appStore: AppStore
    let playbackStore: PlaybackStore
    let isVisible: Bool
    let onClose: () -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(controller: controller)
    }

    func makeNSView(context: Context) -> NSView {
        let view = NSView(frame: .zero)
        view.isHidden = true
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        context.coordinator.controller = controller
        context.coordinator.controller?.attach(to: nsView.window)
        context.coordinator.controller?.update(
            nowPlaying: nowPlaying,
            appStore: appStore,
            playbackStore: playbackStore,
            isVisible: isVisible,
            onClose: onClose
        )
    }

    static func dismantleNSView(_ nsView: NSView, coordinator: Coordinator) {
        coordinator.controller?.detach()
    }

    final class Coordinator {
        var controller: NowPlayingWindowOverlayController?

        init(controller: NowPlayingWindowOverlayController) {
            self.controller = controller
        }
    }
}

private final class NowPlayingWindowOverlayView: NSView {
    private let closeRelay = NowPlayingOverlayCloseRelay()
    private let hostingView = NSHostingView(rootView: AnyView(EmptyView()))

    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        commonInit()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func update(
        nowPlaying: NowPlaying?,
        appStore: AppStore,
        playbackStore: PlaybackStore,
        isVisible: Bool,
        onClose: @escaping () -> Void
    ) {
        closeRelay.onClose = onClose

        guard isVisible, let nowPlaying else {
            setVisible(false)
            return
        }

        hostingView.rootView = AnyView(
            NowPlayingWindowOverlayRootView(
                nowPlaying: nowPlaying,
                closeRelay: closeRelay
            )
            .environmentObject(appStore)
            .environment(playbackStore)
            .environment(appStore.favoriteActionStore)
        )
        setVisible(true)
    }

    func updateInteraction(onClose: @escaping () -> Void) {
        closeRelay.onClose = onClose
    }

    private func commonInit() {
        wantsLayer = true
        layer?.backgroundColor = NSColor.clear.cgColor
        hostingView.translatesAutoresizingMaskIntoConstraints = false
        addSubview(hostingView)

        NSLayoutConstraint.activate([
            hostingView.leadingAnchor.constraint(equalTo: leadingAnchor),
            hostingView.trailingAnchor.constraint(equalTo: trailingAnchor),
            hostingView.topAnchor.constraint(equalTo: topAnchor),
            hostingView.bottomAnchor.constraint(equalTo: bottomAnchor)
        ])

        alphaValue = 0
        isHidden = true
    }

    private func setVisible(_ visible: Bool) {
        let alreadyVisible = !isHidden && alphaValue >= 0.999
        let alreadyHidden = isHidden && alphaValue <= 0.001

        if (visible && alreadyVisible) || (!visible && alreadyHidden) {
            return
        }

        if visible {
            isHidden = false
        }

        NSAnimationContext.runAnimationGroup { context in
            context.duration = 0.18
            context.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
            animator().alphaValue = visible ? 1 : 0
        } completionHandler: { [weak self] in
            guard let self else { return }
            if !visible {
                self.isHidden = true
                self.hostingView.rootView = AnyView(EmptyView())
            }
        }
    }
}

private struct NowPlayingWindowOverlayRenderState: Equatable {
    let nowPlayingToken: NowPlayingWindowOverlayToken
    let isVisible: Bool

    init(nowPlaying: NowPlaying?, isVisible: Bool) {
        self.nowPlayingToken = NowPlayingWindowOverlayToken(nowPlaying: nowPlaying)
        self.isVisible = isVisible
    }
}

private struct NowPlayingWindowOverlayToken: Equatable {
    let artist: String
    let album: String
    let track: String
    let artwork: String

    init(nowPlaying: NowPlaying?) {
        artist = nowPlaying?.artist ?? ""
        album = nowPlaying?.album ?? ""
        track = nowPlaying?.track ?? ""
        artwork = nowPlaying?.artwork ?? ""
    }
}

private final class NowPlayingOverlayCloseRelay {
    var onClose: (() -> Void)?

    func close() {
        onClose?()
    }
}

private struct NowPlayingWindowOverlayRootView: View {
    let nowPlaying: NowPlaying
    let closeRelay: NowPlayingOverlayCloseRelay

    var body: some View {
        NowPlayingView(nowPlaying: nowPlaying) {
            closeRelay.close()
        }
        .ignoresSafeArea()
    }
}
#endif
