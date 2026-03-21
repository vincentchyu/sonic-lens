import SwiftUI

struct InsightModelPickerContent: View {
    let subjectLabel: String
    @Binding var selectedAIPlatform: String
    @Binding var selectedAIModel: String
    let availableAIPlatforms: [AIPlatformOption]
    let availableAIModels: [AIModelOption]
    let isConfirmDisabled: Bool
    let onCancel: () -> Void
    let onConfirm: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("请选择本次\(subjectLabel)音眸分析的平台和模型")
                .font(.headline)

            if availableAIPlatforms.isEmpty {
                Text("当前服务器暂无可选平台")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            } else {
                Picker("平台", selection: $selectedAIPlatform) {
                    ForEach(availableAIPlatforms) { platform in
                        Text(platform.displayName).tag(platform.id)
                    }
                }
                .pickerStyle(.menu)

#if os(iOS)
                if availableAIModels.isEmpty {
                    InsightPickerValueRow(
                        title: "模型",
                        value: "当前平台暂无可选模型",
                        showsChevron: false,
                        isDisabled: true
                    )
                } else {
                    NavigationLink {
                        InsightModelSelectionListView(
                            title: "选择音眸模型",
                            models: availableAIModels,
                            selection: $selectedAIModel
                        )
                    } label: {
                        InsightPickerValueRow(
                            title: "模型",
                            value: selectedModelDisplayName,
                            showsChevron: true,
                            isDisabled: false
                        )
                    }
                    .buttonStyle(.plain)
                }
#else
                Picker("模型", selection: $selectedAIModel) {
                    ForEach(availableAIModels) { model in
                        Text(model.displayName).tag(model.id)
                    }
                }
                .pickerStyle(.menu)
#endif

                Text("该选择会按当前服务器记忆，下次默认预选。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            HStack {
                Button("取消") {
                    onCancel()
                }
                .buttonStyle(.bordered)

                Spacer()

                Button("开始生成") {
                    onConfirm()
                }
                .buttonStyle(.borderedProminent)
                .disabled(isConfirmDisabled)
            }
            .padding(.top, 4)
        }
    }

    private var selectedModelDisplayName: String {
        if let selected = availableAIModels.first(where: { $0.id == selectedAIModel }) {
            return selected.displayName
        }
        let trimmed = selectedAIModel.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? "请选择模型" : trimmed
    }
}

private struct InsightPickerValueRow: View {
    let title: String
    let value: String
    let showsChevron: Bool
    let isDisabled: Bool

    var body: some View {
        HStack(spacing: 10) {
            Text(title)
                .foregroundStyle(.primary)

            Spacer(minLength: 12)

            Text(value)
                .lineLimit(1)
                .foregroundStyle(isDisabled ? .secondary : .secondary)

            if showsChevron {
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 14)
                .fill(Color.white.opacity(isDisabled ? 0.08 : 0.12))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .stroke(Color.white.opacity(0.18), lineWidth: 1)
        )
    }
}

struct InsightModelSelectionListView: View {
    let title: String
    let models: [AIModelOption]
    @Binding var selection: String
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 10) {
                    if models.isEmpty {
                        Text("当前平台暂无可选模型")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.vertical, 12)
                    } else {
                        ForEach(models) { model in
                            Button {
                                selection = model.id
                                dismiss()
                            } label: {
                                MenuSelectionRow(title: model.displayName, isSelected: selection == model.id)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 12)
                                    .glassCard(cornerRadius: 14, isSimplified: true)
                            }
                            .buttonStyle(.plain)
                            .id(model.id)
                        }
                    }
                }
                .padding(16)
            }
            .navigationTitle(title)
#if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
#endif
            .onAppear {
                scrollToSelected(proxy)
            }
            .onChange(of: selection) { _, _ in
                scrollToSelected(proxy)
            }
        }
    }

    private func scrollToSelected(_ proxy: ScrollViewProxy) {
        guard let targetID = models.contains(where: { $0.id == selection }) ? selection : models.first?.id else {
            return
        }
        proxy.scrollTo(targetID, anchor: .center)
    }
}
