#if os(iOS)
import Foundation
import Photos

enum PhotoSaveError: LocalizedError {
    case denied

    var errorDescription: String? {
        switch self {
        case .denied:
            return "没有照片写入权限"
        }
    }
}

@MainActor
final class PhotoSaveCoordinator {
    func saveImageFiles(at urls: [URL]) async throws {
        let authorized = await requestIfNeeded()
        guard authorized else {
            throw PhotoSaveError.denied
        }

        try await PHPhotoLibrary.shared().performChanges {
            for url in urls {
                let request = PHAssetCreationRequest.forAsset()
                request.addResource(with: .photo, fileURL: url, options: nil)
            }
        }
    }

    private func requestIfNeeded() async -> Bool {
        let current = PHPhotoLibrary.authorizationStatus(for: .addOnly)
        switch current {
        case .authorized, .limited:
            return true
        case .notDetermined:
            let status = await PHPhotoLibrary.requestAuthorization(for: .addOnly)
            return status == .authorized || status == .limited
        default:
            return false
        }
    }
}
#else
import Foundation

@MainActor
final class PhotoSaveCoordinator {}
#endif
