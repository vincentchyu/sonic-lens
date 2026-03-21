import SwiftUI

struct ErrorBanner: View {
    let message: String

    var body: some View {
        HStack {
            Image(systemName: "exclamationmark.triangle")
            Text(message)
                .font(.subheadline)
            Spacer()
        }
        .padding(10)
        .background(Color.red.opacity(0.1))
        .foregroundColor(.red)
        .cornerRadius(10)
    }
}

struct LoadingOverlay: View {
    var body: some View {
        ZStack {
            Color.black.opacity(0.08)
                .ignoresSafeArea()
            ProgressView("加载中...")
                .padding(16)
                .background(Color.white)
                .cornerRadius(12)
                .shadow(color: Color.black.opacity(0.1), radius: 8, x: 0, y: 4)
        }
    }
}

struct ContentHeader: View {
    let title: String

    var body: some View {
        HStack {
            Text(title)
                .font(.title3)
                .fontWeight(.semibold)
            Spacer()
        }
    }
}

struct SectionHeader: View {
    let title: String

    var body: some View {
        Text(title)
            .font(.title3)
            .fontWeight(.semibold)
    }
}
