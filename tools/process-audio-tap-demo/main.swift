import Foundation
import CoreAudio

/// Process Audio Tap 格式感知监听器
/// 
/// 基于 macOS 14.2+ 引入的公开 CoreAudio Tap API：
/// - CATapDescription
/// - AudioHardwareCreateProcessTap
/// - kAudioTapPropertyFormat
final class ProcessAudioTapDetector {
    private let targetBundleID: String
    private var targetObjectID: AudioObjectID?
    private var tapObjectID: AudioObjectID?
    
    private var formatListenerBlock: AudioObjectPropertyListenerBlock?
    private var runningListenerBlock: AudioObjectPropertyListenerBlock?
    private var processListListenerBlock: AudioObjectPropertyListenerBlock?
    
    private let queue = DispatchQueue(label: "com.soniclens.audiotap.detector", qos: .userInteractive)
    
    init(targetBundleID: String = "com.apple.Music") {
        self.targetBundleID = targetBundleID
    }
    
    func start() {
        print("=================================================================")
        print("  🎵 Process Audio Tap 探测器已启动 (macOS 14.2+ Sonoma/Sequoia)")
        print("  🎯 目标进程 Bundle ID: \(targetBundleID)")
        print("  ⚡ 机制: CoreAudio HAL 进程级 Tap 零侵入格式捕获")
        print("=================================================================\n")
        
        // 1. 注册全局进程列表监听（应对 Music.app 退出与重启）
        registerProcessListListener()
        
        // 2. 初始检测并挂载
        queue.async {
            self.findAndAttach()
        }
    }
    
    func stop() {
        queue.sync {
            detachTap()
            unregisterProcessListListener()
            print("\n👋 探测器已安全退出，CoreAudio 资源已释放。")
        }
    }
    
    // MARK: - Process Discovery & Attachment
    
    private func findAndAttach() {
        guard let objID = findProcessObjectID(forBundleID: targetBundleID) else {
            print("⏳ 尚未检测到 [\(targetBundleID)] 运行或其尚未建立 CoreAudio 连接，等待进程上线...")
            return
        }
        
        if targetObjectID == objID && tapObjectID != nil {
            return // 已经挂载当前实例
        }
        
        detachTap()
        targetObjectID = objID
        
        print("✅ 检测到目标进程 CoreAudio 实例: AudioObjectID = \(objID)")
        attachTap(forProcessID: objID)
    }
    
    private func attachTap(forProcessID processID: AudioObjectID) {
        // 1. 创建 CATapDescription
        // initStereoMixdownOfProcesses 将目标进程输出混音到立体声 Tap 流
        let tapDesc = CATapDescription(stereoMixdownOfProcesses: [processID])
        tapDesc.name = "SonicLens_ProcessTap_\(targetBundleID)"
        tapDesc.muteBehavior = CATapMuteBehavior.unmuted // 保持原声播放，不静音目标应用
        
        var createdTapID: AudioObjectID = 0
        let status = AudioHardwareCreateProcessTap(tapDesc, &createdTapID)
        guard status == noErr, createdTapID != 0 else {
            print("❌ AudioHardwareCreateProcessTap 失败，错误码: \(status)")
            return
        }
        
        self.tapObjectID = createdTapID
        print("🎉 成功建立 Process Tap: TapID = \(createdTapID)")
        
        // 2. 立即读取一次当前格式
        readAndPrintFormat(tapID: createdTapID, reason: "初始挂载")
        readAndPrintRunningState(processID: processID)
        
        // 3. 注册 kAudioTapPropertyFormat 格式变更监听器
        var formatAddr = AudioObjectPropertyAddress(
            mSelector: kAudioTapPropertyFormat,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        
        let formatBlock: AudioObjectPropertyListenerBlock = { [weak self] _, _ in
            guard let self = self, let tap = self.tapObjectID else { return }
            self.readAndPrintFormat(tapID: tap, reason: "流格式动态变更 (SampleRate/BitDepth)")
        }
        self.formatListenerBlock = formatBlock
        
        let formatStatus = AudioObjectAddPropertyListenerBlock(
            createdTapID,
            &formatAddr,
            queue,
            formatBlock
        )
        if formatStatus != noErr {
            print("⚠️ 注册 Format Listener 失败: \(formatStatus)")
        }
        
        // 4. 注册 kAudioProcessPropertyIsRunningOutput 输出状态监听器（感知播放/暂停）
        var runAddr = AudioObjectPropertyAddress(
            mSelector: kAudioProcessPropertyIsRunningOutput,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        
        let runBlock: AudioObjectPropertyListenerBlock = { [weak self] _, _ in
            guard let self = self, let procID = self.targetObjectID else { return }
            self.readAndPrintRunningState(processID: procID)
        }
        self.runningListenerBlock = runBlock
        
        let runStatus = AudioObjectAddPropertyListenerBlock(
            processID,
            &runAddr,
            queue,
            runBlock
        )
        if runStatus != noErr {
            print("⚠️ 注册 IsRunningOutput Listener 失败: \(runStatus)")
        }
    }
    
    private func detachTap() {
        if let tap = tapObjectID {
            if let block = formatListenerBlock {
                var formatAddr = AudioObjectPropertyAddress(
                    mSelector: kAudioTapPropertyFormat,
                    mScope: kAudioObjectPropertyScopeGlobal,
                    mElement: kAudioObjectPropertyElementMain
                )
                AudioObjectRemovePropertyListenerBlock(tap, &formatAddr, queue, block)
                formatListenerBlock = nil
            }
            
            let status = AudioHardwareDestroyProcessTap(tap)
            if status == noErr {
                print("🧹 Process Tap [\(tap)] 已安全注销")
            }
            tapObjectID = nil
        }
        
        if let procID = targetObjectID, let block = runningListenerBlock {
            var runAddr = AudioObjectPropertyAddress(
                mSelector: kAudioProcessPropertyIsRunningOutput,
                mScope: kAudioObjectPropertyScopeGlobal,
                mElement: kAudioObjectPropertyElementMain
            )
            AudioObjectRemovePropertyListenerBlock(procID, &runAddr, queue, block)
            runningListenerBlock = nil
        }
        
        targetObjectID = nil
    }
    
    // MARK: - CoreAudio Property Readers
    
    private func readAndPrintFormat(tapID: AudioObjectID, reason: String) {
        var formatAddr = AudioObjectPropertyAddress(
            mSelector: kAudioTapPropertyFormat,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        
        var asbd = AudioStreamBasicDescription()
        var size = UInt32(MemoryLayout<AudioStreamBasicDescription>.size)
        let status = AudioObjectGetPropertyData(tapID, &formatAddr, 0, nil, &size, &asbd)
        
        guard status == noErr else {
            print("⚠️ 读取 Tap 格式失败: \(status)")
            return
        }
        
        let timestamp = ISO8601DateFormatter().string(from: Date())
        let sampleRateKHz = asbd.mSampleRate / 1000.0
        let formatDesc = describeASBD(asbd)
        
        print("\n-------------------------------------------------------------")
        print("⏱ [\(timestamp)] 事件: \(reason)")
        print("  🎛 采样率:    \(asbd.mSampleRate) Hz (\(String(format: "%.1f", sampleRateKHz)) kHz)")
        print("  💎 量化精度:  \(asbd.mBitsPerChannel)-bit")
        print("  🔊 通道数:    \(asbd.mChannelsPerFrame) 声道")
        print("  📦 帧格式:    \(asbd.mBytesPerFrame) 字节/帧, \(asbd.mBytesPerPacket) 字节/包")
        print("  🏷 数据类型:  \(formatDesc)")
        print("-------------------------------------------------------------")
    }
    
    private func readAndPrintRunningState(processID: AudioObjectID) {
        var runAddr = AudioObjectPropertyAddress(
            mSelector: kAudioProcessPropertyIsRunningOutput,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        var isRunning: UInt32 = 0
        var size = UInt32(MemoryLayout<UInt32>.size)
        let status = AudioObjectGetPropertyData(processID, &runAddr, 0, nil, &size, &isRunning)
        
        if status == noErr {
            let stateStr = (isRunning == 1) ? "🟢 正在输出音频 (PLAYING)" : "⏸ 静止/暂停 (IDLE/PAUSED)"
            print("  ▶ 播放器 IO 状态: \(stateStr)")
        }
    }
    
    // MARK: - Helper Methods
    
    private func describeASBD(_ asbd: AudioStreamBasicDescription) -> String {
        var parts: [String] = []
        if asbd.mFormatID == kAudioFormatLinearPCM {
            parts.append("Linear PCM")
        } else {
            parts.append("FormatID: \(asbd.mFormatID)")
        }
        
        let flags = asbd.mFormatFlags
        if (flags & kAudioFormatFlagIsFloat) != 0 {
            parts.append("Float")
        } else if (flags & kAudioFormatFlagIsSignedInteger) != 0 {
            parts.append("SignedInt")
        }
        
        if (flags & kAudioFormatFlagIsPacked) != 0 {
            parts.append("Packed")
        }
        if (flags & kAudioFormatFlagIsNonInterleaved) != 0 {
            parts.append("NonInterleaved")
        } else {
            parts.append("Interleaved")
        }
        
        return parts.joined(separator: ", ")
    }
    
    private func findProcessObjectID(forBundleID bundleID: String) -> AudioObjectID? {
        var address = AudioObjectPropertyAddress(
            mSelector: kAudioHardwarePropertyProcessObjectList,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        
        var dataSize: UInt32 = 0
        guard AudioObjectGetPropertyDataSize(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &dataSize) == noErr else {
            return nil
        }
        
        let count = Int(dataSize) / MemoryLayout<AudioObjectID>.size
        var ids = [AudioObjectID](repeating: 0, count: count)
        guard AudioObjectGetPropertyData(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &dataSize, &ids) == noErr else {
            return nil
        }
        
        for objID in ids {
            var bundleAddr = AudioObjectPropertyAddress(
                mSelector: kAudioProcessPropertyBundleID,
                mScope: kAudioObjectPropertyScopeGlobal,
                mElement: kAudioObjectPropertyElementMain
            )
            var cfStr: Unmanaged<CFString>?
            var strSize = UInt32(MemoryLayout<Unmanaged<CFString>?>.size)
            if AudioObjectGetPropertyData(objID, &bundleAddr, 0, nil, &strSize, &cfStr) == noErr,
               let unmanaged = cfStr {
                let currentBundle = unmanaged.takeRetainedValue() as String
                if currentBundle == bundleID {
                    return objID
                }
            }
        }
        return nil
    }
    
    private func registerProcessListListener() {
        var procListAddr = AudioObjectPropertyAddress(
            mSelector: kAudioHardwarePropertyProcessObjectList,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        
        let block: AudioObjectPropertyListenerBlock = { [weak self] _, _ in
            guard let self = self else { return }
            self.queue.async {
                self.findAndAttach()
            }
        }
        self.processListListenerBlock = block
        
        AudioObjectAddPropertyListenerBlock(
            AudioObjectID(kAudioObjectSystemObject),
            &procListAddr,
            queue,
            block
        )
    }
    
    private func unregisterProcessListListener() {
        if let block = processListListenerBlock {
            var procListAddr = AudioObjectPropertyAddress(
                mSelector: kAudioHardwarePropertyProcessObjectList,
                mScope: kAudioObjectPropertyScopeGlobal,
                mElement: kAudioObjectPropertyElementMain
            )
            AudioObjectRemovePropertyListenerBlock(
                AudioObjectID(kAudioObjectSystemObject),
                &procListAddr,
                queue,
                block
            )
            processListListenerBlock = nil
        }
    }
}

// MARK: - Entry Point & Signal Handling

let bundleID = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "com.apple.Music"
let detector = ProcessAudioTapDetector(targetBundleID: bundleID)

// 拦截 SIGINT / SIGTERM，确保退出前清理 CoreAudio Tap
signal(SIGINT) { _ in
    print("\n[Ctrl+C] 正在清理退出...")
    detector.stop()
    exit(0)
}
signal(SIGTERM) { _ in
    detector.stop()
    exit(0)
}

detector.start()

// 进入主运行循环
RunLoop.main.run()
