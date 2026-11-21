#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BRIDGE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BRIDGE_DIR/.." && pwd)"
PROJECT_PATH="$BRIDGE_DIR/SoniclensBridge.xcodeproj"
SNAPSHOT_MAC_DIR="$REPO_ROOT/.derivedData/ui_snapshots/macos"
SNAPSHOT_IOS_DIR="$REPO_ROOT/.derivedData/ui_snapshots/ios"

usage() {
    cat <<'EOF'
SonicLens Bridge UI 自检与自动化验证脚本

用法:
  verify_ui.sh snapshots
  verify_ui.sh mac-ui
  verify_ui.sh phone-tests
  verify_ui.sh ios-ui
  verify_ui.sh ios-sim [--tab home|albums|tracks|unreported|sonicLens] [--appearance light|dark] [--fixture dense|empty|longText] [--udid <UUID>]
EOF
}

resolve_iphone_sim_udid() {
    local explicit_udid="${1:-}"
    if [[ -n "$explicit_udid" ]]; then
        echo "$explicit_udid"
        return 0
    fi

    xcrun simctl list devices available -j | python3 -c '
import json, sys
data = json.load(sys.stdin)
devices = []
for runtime, items in data.get("devices", {}).items():
    if "iOS" not in runtime:
        continue
    for item in items:
        if not item.get("isAvailable", False):
            continue
        name = item.get("name", "")
        if "iPhone" not in name:
            continue
        devices.append((
            0 if item.get("state") == "Booted" else 1,
            0 if "Pro" in name else 1,
            name,
            item["udid"]
        ))
if not devices:
    sys.exit(1)
devices.sort(key=lambda x: (x[0], x[1], x[2]), reverse=False)
# 优先 Booted；若均未启动，取靠后的高版本 Pro/标准机型
booted = [d for d in devices if d[0] == 0]
chosen = booted[0] if booted else sorted(devices, key=lambda x: x[2], reverse=True)[0]
print(chosen[3])
'
}

cmd_snapshots() {
    mkdir -p "$SNAPSHOT_MAC_DIR"
    rm -f "$SNAPSHOT_MAC_DIR"/*.png
    echo "📸 生成 macOS SwiftUI 离屏快照至: $SNAPSHOT_MAC_DIR"

    SONICLENS_UI_SNAPSHOT_DIR="$SNAPSHOT_MAC_DIR" \
    xcodebuild test \
        -project "$PROJECT_PATH" \
        -scheme SoniclensBridgeMac \
        -configuration Debug \
        -destination 'platform=macOS' \
        -derivedDataPath "$REPO_ROOT/.derivedData/macOS_test" \
        -only-testing:SoniclensBridgeMacTests/TrendHeatmapSnapshotTests \
        CODE_SIGN_IDENTITY="-" \
        -quiet

    echo "✅ 快照生成完成，产出文件:"
    ls -lh "$SNAPSHOT_MAC_DIR"/*.png
}

cmd_mac_ui() {
    local result_bundle="$REPO_ROOT/.derivedData/ui_snapshots/mac_ui.xcresult"
    mkdir -p "$(dirname "$result_bundle")"
    rm -rf "$result_bundle"

    echo "🧪 运行 macOS XCUITest (SoniclensBridgeMacUITests)..."
    xcodebuild test \
        -project "$PROJECT_PATH" \
        -scheme SoniclensBridgeMac \
        -configuration Debug \
        -destination 'platform=macOS' \
        -derivedDataPath "$REPO_ROOT/.derivedData/macOS_test" \
        -only-testing:SoniclensBridgeMacUITests \
        -resultBundlePath "$result_bundle" \
        CODE_SIGN_IDENTITY="-" \
        -quiet
    echo "✅ macOS UI Test 通过，结果包: $result_bundle"
}

cmd_phone_tests() {
    local sim_udid
    sim_udid="$(resolve_iphone_sim_udid)" || {
        echo "❌ 未找到可用的 iPhone Simulator，请先在 Xcode 中安装 iOS Simulator Runtime。" >&2
        exit 1
    }
    echo "🧪 在 iPhone Simulator ($sim_udid) 运行 SoniclensBridgePhoneTests..."
    xcodebuild test \
        -project "$PROJECT_PATH" \
        -scheme SoniclensBridgePhone \
        -configuration Debug \
        -destination "id=$sim_udid" \
        -derivedDataPath "$REPO_ROOT/.derivedData/iOS_sim" \
        -only-testing:SoniclensBridgePhoneTests \
        -quiet
    echo "✅ SoniclensBridgePhoneTests 全部通过"
}

cmd_ios_ui() {
    local sim_udid
    sim_udid="$(resolve_iphone_sim_udid)" || {
        echo "❌ 未找到可用的 iPhone Simulator，请先在 Xcode 中安装 iOS Simulator Runtime。" >&2
        exit 1
    }
    local result_bundle="$REPO_ROOT/.derivedData/ui_snapshots/ios_ui.xcresult"
    mkdir -p "$(dirname "$result_bundle")"
    rm -rf "$result_bundle"

    echo "🧪 在 iPhone Simulator ($sim_udid) 运行 SoniclensBridgePhoneUITests..."
    xcodebuild test \
        -project "$PROJECT_PATH" \
        -scheme SoniclensBridgePhone \
        -configuration Debug \
        -destination "id=$sim_udid" \
        -derivedDataPath "$REPO_ROOT/.derivedData/iOS_sim" \
        -only-testing:SoniclensBridgePhoneUITests \
        -resultBundlePath "$result_bundle" \
        -quiet
    echo "✅ iPhone UI Test 通过，结果包: $result_bundle"
}

cmd_ios_sim() {
    local tab="home"
    local appearance="dark"
    local fixture="dense"
    local explicit_udid=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tab)
                tab="${2:-home}"
                shift 2
                ;;
            --appearance)
                appearance="${2:-dark}"
                shift 2
                ;;
            --fixture)
                fixture="${2:-dense}"
                shift 2
                ;;
            --udid)
                explicit_udid="${2:-}"
                shift 2
                ;;
            *)
                echo "未知参数: $1" >&2
                usage
                exit 1
                ;;
        esac
    done

    local sim_udid
    sim_udid="$(resolve_iphone_sim_udid "$explicit_udid")" || {
        echo "❌ 未找到可用的 iPhone Simulator，请先在 Xcode 中安装 iOS Simulator Runtime。" >&2
        exit 1
    }

    echo "📱 目标模拟器 UDID: $sim_udid"
    xcrun simctl boot "$sim_udid" >/dev/null 2>&1 || true
    xcrun simctl bootstatus "$sim_udid" -b

    echo "🔨 编译 SoniclensBridgePhone (Simulator Debug)..."
    xcodebuild \
        -project "$PROJECT_PATH" \
        -scheme SoniclensBridgePhone \
        -configuration Debug \
        -destination "id=$sim_udid" \
        -derivedDataPath "$REPO_ROOT/.derivedData/iOS_sim" \
        build \
        -quiet

    local app_path="$REPO_ROOT/.derivedData/iOS_sim/Build/Products/Debug-iphonesimulator/SoniclensBridgePhone.app"
    if [[ ! -d "$app_path" ]]; then
        echo "❌ 未找到编译产物: $app_path" >&2
        exit 1
    fi

    echo "📲 安装并启动 Mock 模式 (-ui-tab $tab -ui-appearance $appearance -ui-fixture $fixture)..."
    xcrun simctl terminate "$sim_udid" com.vincentchyu.soniclens-bridge.phone >/dev/null 2>&1 || true
    xcrun simctl install "$sim_udid" "$app_path"
    xcrun simctl launch "$sim_udid" com.vincentchyu.soniclens-bridge.phone \
        -ui-mock -ui-tab "$tab" -ui-appearance "$appearance" -ui-fixture "$fixture" >/dev/null

    sleep 2.5

    mkdir -p "$SNAPSHOT_IOS_DIR"
    local out_png="$SNAPSHOT_IOS_DIR/ios_${tab}_${appearance}_${fixture}.png"
    xcrun simctl io "$sim_udid" screenshot "$out_png" >/dev/null
    echo "✅ iPhone 模拟器截图已保存: $out_png"
}

SUBCOMMAND="${1:-}"
if [[ -z "$SUBCOMMAND" ]]; then
    usage
    exit 1
fi
shift

case "$SUBCOMMAND" in
    snapshots)
        cmd_snapshots "$@"
        ;;
    mac-ui)
        cmd_mac_ui "$@"
        ;;
    phone-tests)
        cmd_phone_tests "$@"
        ;;
    ios-ui)
        cmd_ios_ui "$@"
        ;;
    ios-sim)
        cmd_ios_sim "$@"
        ;;
    *)
        usage
        exit 1
        ;;
esac
