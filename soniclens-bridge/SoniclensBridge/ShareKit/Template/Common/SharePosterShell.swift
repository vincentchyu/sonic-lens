import SwiftUI

struct SharePosterShell<Content: View>: View {
    let header: ShareHeaderPayload
    let footer: ShareFooterPayload
    let renderedImage: PlatformShareImage?
    let continuationLabel: String?
    let pageText: String?
    let showsFooter: Bool
    let topCaptionText: String?
    let content: Content

    init(
        header: ShareHeaderPayload,
        footer: ShareFooterPayload,
        renderedImage: PlatformShareImage? = nil,
        continuationLabel: String? = nil,
        pageText: String? = nil,
        showsFooter: Bool = true,
        topCaptionText: String? = "SonicLens Bridge · iPhone",
        @ViewBuilder content: () -> Content
    ) {
        self.header = header
        self.footer = footer
        self.renderedImage = renderedImage
        self.continuationLabel = continuationLabel
        self.pageText = pageText
        self.showsFooter = showsFooter
        self.topCaptionText = topCaptionText
        self.content = content()
    }

    var body: some View {
        SharePosterBackground {
            VStack(alignment: .leading, spacing: 18) {
                VStack(alignment: .leading, spacing: 5) {
                    if let topCaptionText {
                        Text(topCaptionText)
                            .font(.system(size: 8, weight: .medium))
                            .foregroundStyle(Color.white.opacity(0.72))
                            .frame(maxWidth: .infinity, alignment: .center)
                            .padding(.bottom,15)
                    }

                    SharePosterHeader(
                        header: header,
                        continuationLabel: continuationLabel,
                        renderedImage: renderedImage
                    )
                }

                SharePosterCard {
                    content
                }

                if showsFooter {
                    SharePosterFooter(footer: footer, pageText: pageText)
                }
            }
            .padding(20)
        }
    }
}
