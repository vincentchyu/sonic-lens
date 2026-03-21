import SwiftUI
import CoreImage
import CoreImage.CIFilterBuiltins

#if os(macOS)
import AppKit
typealias PlatformColor = NSColor
typealias PlatformImage = NSImage
#else
import UIKit
typealias PlatformColor = UIColor
typealias PlatformImage = UIImage
#endif

extension Color {
    init(platformColor: PlatformColor) {
        #if os(macOS)
        self.init(nsColor: platformColor)
        #else
        self.init(uiColor: platformColor)
        #endif
    }
}

extension PlatformColor {
    static func sonicRGBA(_ red: CGFloat, _ green: CGFloat, _ blue: CGFloat, _ alpha: CGFloat = 1) -> PlatformColor {
        #if os(macOS)
        return PlatformColor(calibratedRed: red, green: green, blue: blue, alpha: alpha)
        #else
        return PlatformColor(red: red, green: green, blue: blue, alpha: alpha)
        #endif
    }

    static func sonicWhite(_ white: CGFloat, alpha: CGFloat = 1) -> PlatformColor {
        #if os(macOS)
        return PlatformColor(calibratedWhite: white, alpha: alpha)
        #else
        return PlatformColor(white: white, alpha: alpha)
        #endif
    }

    static func sonicDynamic(light: PlatformColor, dark: PlatformColor) -> PlatformColor {
        #if os(macOS)
        return PlatformColor(name: nil) { appearance in
            appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? dark : light
        }
        #else
        return PlatformColor { traits in
            traits.userInterfaceStyle == .dark ? dark : light
        }
        #endif
    }

    var sonicSRGB: PlatformColor {
        #if os(macOS)
        return usingColorSpace(.sRGB) ?? self
        #else
        return self
        #endif
    }

    func mixed(with color: PlatformColor, amount: CGFloat) -> PlatformColor {
        let lhs = sonicSRGB
        let rhs = color.sonicSRGB
        let ratio = min(max(amount, 0), 1)
        guard
            let lhsComponents = lhs.sonicRGBAComponents,
            let rhsComponents = rhs.sonicRGBAComponents
        else {
            return self
        }

        return .sonicRGBA(
            lhsComponents.red + (rhsComponents.red - lhsComponents.red) * ratio,
            lhsComponents.green + (rhsComponents.green - lhsComponents.green) * ratio,
            lhsComponents.blue + (rhsComponents.blue - lhsComponents.blue) * ratio,
            lhsComponents.alpha + (rhsComponents.alpha - lhsComponents.alpha) * ratio
        )
    }

    func adjustedBrightness(_ factor: CGFloat) -> PlatformColor {
        var hue: CGFloat = 0
        var saturation: CGFloat = 0
        var brightness: CGFloat = 0
        var alpha: CGFloat = 0
        #if os(macOS)
        sonicSRGB.getHue(&hue, saturation: &saturation, brightness: &brightness, alpha: &alpha)
        #else
        guard sonicSRGB.getHue(&hue, saturation: &saturation, brightness: &brightness, alpha: &alpha) else {
            return self
        }
        #endif
        return PlatformColor(
            hue: hue,
            saturation: saturation,
            brightness: min(max(brightness * factor, 0), 1),
            alpha: alpha
        )
    }

    func adjustedSaturation(_ factor: CGFloat) -> PlatformColor {
        var hue: CGFloat = 0
        var saturation: CGFloat = 0
        var brightness: CGFloat = 0
        var alpha: CGFloat = 0
        #if os(macOS)
        sonicSRGB.getHue(&hue, saturation: &saturation, brightness: &brightness, alpha: &alpha)
        #else
        guard sonicSRGB.getHue(&hue, saturation: &saturation, brightness: &brightness, alpha: &alpha) else {
            return self
        }
        #endif
        return PlatformColor(
            hue: hue,
            saturation: min(max(saturation * factor, 0), 1),
            brightness: brightness,
            alpha: alpha
        )
    }

    private var sonicRGBAComponents: (red: CGFloat, green: CGFloat, blue: CGFloat, alpha: CGFloat)? {
        #if os(macOS)
        let color = sonicSRGB
        return (color.redComponent, color.greenComponent, color.blueComponent, color.alphaComponent)
        #else
        var red: CGFloat = 0
        var green: CGFloat = 0
        var blue: CGFloat = 0
        var alpha: CGFloat = 0
        guard sonicSRGB.getRed(&red, green: &green, blue: &blue, alpha: &alpha) else { return nil }
        return (red, green, blue, alpha)
        #endif
    }
}

extension PlatformImage {
    func averageSRGBColor() -> PlatformColor? {
        let inputImage: CIImage?
        #if os(macOS)
        if let cgImage = cgImage(forProposedRect: nil, context: nil, hints: nil) {
            inputImage = CIImage(cgImage: cgImage)
        } else if let tiffData = tiffRepresentation {
            inputImage = CIImage(data: tiffData)
        } else {
            inputImage = nil
        }
        #else
        if let cgImage = cgImage {
            inputImage = CIImage(cgImage: cgImage)
        } else {
            inputImage = nil
        }
        #endif

        guard let inputImage else { return nil }
        let extent = inputImage.extent
        guard !extent.isEmpty else { return nil }

        let filter = CIFilter.areaAverage()
        filter.inputImage = inputImage
        filter.extent = extent

        guard let outputImage = filter.outputImage else { return nil }
        let context = CIContext(options: [.workingColorSpace: NSNull()])

        var bitmap = [UInt8](repeating: 0, count: 4)
        context.render(
            outputImage,
            toBitmap: &bitmap,
            rowBytes: 4,
            bounds: CGRect(x: 0, y: 0, width: 1, height: 1),
            format: .RGBA8,
            colorSpace: nil
        )

        return .sonicRGBA(
            CGFloat(bitmap[0]) / 255,
            CGFloat(bitmap[1]) / 255,
            CGFloat(bitmap[2]) / 255,
            1
        )
    }
}
