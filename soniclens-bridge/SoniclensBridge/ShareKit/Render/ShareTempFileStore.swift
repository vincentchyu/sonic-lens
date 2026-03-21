import Foundation

final class ShareTempFileStore {
    private let fileManager = FileManager.default

    func writeImageData(_ data: Data, suggestedFilename: String, pageIndex: Int?, fileExtension: String) throws -> URL {
        let directory = fileManager.temporaryDirectory.appendingPathComponent("SonicLensShare", isDirectory: true)
        if !fileManager.fileExists(atPath: directory.path) {
            try fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
        }

        let safeName = suggestedFilename.replacingOccurrences(of: "/", with: "-")
        let filename: String
        if let pageIndex {
            filename = "\(safeName)-\(pageIndex + 1).\(fileExtension)"
        } else {
            filename = "\(safeName).\(fileExtension)"
        }

        let url = directory.appendingPathComponent(UUID().uuidString + "-" + filename)
        try data.write(to: url, options: .atomic)
        return url
    }
}
