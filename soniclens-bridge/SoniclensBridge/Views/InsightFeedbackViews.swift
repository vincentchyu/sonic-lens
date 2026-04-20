import SwiftUI

struct InsightFeedbackSection: View {
    let summary: InsightFeedbackSummary?
    let history: [InsightFeedbackRecord]
    let isLoading: Bool
    let isSubmitting: Bool
    let errorMessage: String?
    let isCompact: Bool
    let onHelpful: () -> Void
    let onSubmitIssue: (InsightIssueDraft) -> Void

    @State private var isComposerPresented = false
    @State private var isShowingAllHistory = false
    @State private var draft = InsightIssueDraft()

    private var statusText: String {
        summary?.displayStatus ?? "未评价"
    }

    private var negativeHistory: [InsightFeedbackRecord] {
        history.filter(\.isNegative)
    }

    private var displayedHistory: [InsightFeedbackRecord] {
        let source = negativeHistory
        guard !isShowingAllHistory else { return source }
        return Array(source.prefix(3))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: isCompact ? 12 : 16) {
            DetailSectionCard(title: "我的反馈", compact: isCompact) {
                VStack(alignment: .leading, spacing: 12) {
                    HStack(alignment: .firstTextBaseline, spacing: 10) {
                        Text(statusText)
                            .font(isCompact ? .subheadline.weight(.semibold) : .headline.weight(.semibold))
                            .foregroundStyle(statusText == "待修正" ? Color.orange : SonicTheme.textPrimary)
                        Text("赞 \(summary?.likeCount ?? 0) / 踩 \(summary?.dislikeCount ?? 0)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    if let summary,
                       let latestNegativeFeedback = summary.latestNegativeFeedback,
                       !(summary.topReasonCodes ?? []).isEmpty || !latestNegativeFeedback.comment.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            if let topReasons = summary.topReasonCodes, !topReasons.isEmpty {
                                Text("最近集中问题：\(topReasons.joined(separator: "、"))")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            if !latestNegativeFeedback.comment.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                                Text(latestNegativeFeedback.comment)
                                    .font(isCompact ? .caption : .callout)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(3)
                            }
                        }
                    }

                    HStack(spacing: 10) {
                        Button(action: onHelpful) {
                            Label("有帮助", systemImage: summary?.latestFeedback?.score == 1 ? "hand.thumbsup.fill" : "hand.thumbsup")
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(isSubmitting)

                        Button {
                            draft = InsightIssueDraft()
                            isComposerPresented = true
                        } label: {
                            Label("有问题", systemImage: summary?.latestFeedback?.score == -1 ? "hand.thumbsdown.fill" : "hand.thumbsdown")
                        }
                        .buttonStyle(.bordered)
                        .disabled(isSubmitting)
                    }

                    if isLoading || isSubmitting {
                        ProgressView()
                            .controlSize(.small)
                    }

                    if let errorMessage, !errorMessage.isEmpty {
                        Text(errorMessage)
                            .font(.caption)
                            .foregroundStyle(Color.red)
                    }
                }
            }

            DetailSectionCard(title: "历史反馈", compact: isCompact) {
                VStack(alignment: .leading, spacing: 10) {
                    if displayedHistory.isEmpty {
                        Text((summary?.likeCount ?? 0) > 0 ? "当前还没有待修正记录，你之前留下的主要是认可反馈。" : "你还没有为这条音眸留下反馈。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(displayedHistory) { feedback in
                            InsightFeedbackHistoryRow(feedback: feedback, compact: isCompact)
                        }
                    }

                    if negativeHistory.count > 3 {
                        Button(isShowingAllHistory ? "收起较早记录" : "展开更早记录") {
                            withAnimation(.easeInOut(duration: 0.18)) {
                                isShowingAllHistory.toggle()
                            }
                        }
                        .buttonStyle(.plain)
                        .font(.caption.weight(.medium))
                    }

                    if (summary?.likeCount ?? 0) > 0 {
                        Text("已累计记录 \(summary?.likeCount ?? 0) 次“有帮助”反馈。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .sheet(isPresented: $isComposerPresented) {
            NavigationStack {
                InsightIssueComposerView(
                    draft: $draft,
                    isSubmitting: isSubmitting,
                    onCancel: { isComposerPresented = false },
                    onSubmit: {
                        isComposerPresented = false
                        onSubmitIssue(draft)
                    }
                )
            }
            #if os(iOS)
            .presentationDetents([.medium, .large])
            .presentationDragIndicator(.visible)
            #endif
        }
    }
}

private struct InsightFeedbackHistoryRow: View {
    let feedback: InsightFeedbackRecord
    let compact: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Text(feedback.isNegative ? "待修正" : "已认可")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(feedback.isNegative ? Color.orange : SonicTheme.primary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background((feedback.isNegative ? Color.orange : SonicTheme.primary).opacity(0.12), in: Capsule())

                if let createdAt = feedback.createdAt, !createdAt.isEmpty {
                    Text(createdAt)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if let reasons = feedback.reasonCodes, !reasons.isEmpty {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: compact ? 72 : 96), spacing: 8)], alignment: .leading, spacing: 8) {
                    ForEach(reasons, id: \.self) { reason in
                        Text(reason)
                            .font(.caption2.weight(.medium))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 5)
                            .background(Color.primary.opacity(0.06), in: Capsule())
                    }
                }
            }

            if !feedback.comment.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                Text(feedback.comment)
                    .font(compact ? .caption : .callout)
                    .foregroundStyle(.primary)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }

            if let sectionKey = feedback.sectionKey, !sectionKey.isEmpty {
                Text("分区：\(sectionKey)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(compact ? 12 : 14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

private struct InsightIssueComposerView: View {
    @Binding var draft: InsightIssueDraft
    let isSubmitting: Bool
    let onCancel: () -> Void
    let onSubmit: () -> Void

    private var canSubmit: Bool {
        !draft.selectedReasons.isEmpty || !draft.normalizedComment.isEmpty
    }

    var body: some View {
        Form {
            Section("问题标签") {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 110), spacing: 10)], alignment: .leading, spacing: 10) {
                    ForEach(InsightFeedbackReason.allCases) { reason in
                        Button {
                            if draft.selectedReasons.contains(reason) {
                                draft.selectedReasons.remove(reason)
                            } else {
                                draft.selectedReasons.insert(reason)
                            }
                        } label: {
                            Text(reason.rawValue)
                                .font(.subheadline.weight(.medium))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(draft.selectedReasons.contains(reason) ? SonicTheme.primary : Color.secondary.opacity(0.35))
                    }
                }
            }

            Section("补充备注") {
                TextField("例如：总评太空泛，和这首歌真正的叙事重点不够贴合。", text: $draft.comment, axis: .vertical)
                    .lineLimit(4...8)
            }

            Section("可选分区") {
                TextField("例如：summary、background_info", text: $draft.sectionKey)
            }
        }
        .navigationTitle("记录问题")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("取消", action: onCancel)
            }
            ToolbarItem(placement: .confirmationAction) {
                Button("提交", action: onSubmit)
                    .disabled(!canSubmit || isSubmitting)
            }
        }
    }
}
