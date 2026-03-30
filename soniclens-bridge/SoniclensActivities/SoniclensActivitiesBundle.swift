#if os(iOS)
import SwiftUI
import WidgetKit

@main
struct SoniclensActivitiesBundle: WidgetBundle {
    var body: some Widget {
        InsightJobLiveActivityWidget()
    }
}
#endif
