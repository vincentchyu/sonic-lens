#!/bin/bash
set -euo pipefail

VERSION="v1.0.0"
APP_NAME="Soniclens Bridge"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ROOT_DIR="$(cd "$PROJECT_DIR/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
BUILD_DIR="$ROOT_DIR/.derivedData/macOS_release"

echo "=========================================="
echo "🚀 开始打包 $APP_NAME $VERSION for macOS..."
echo "=========================================="

mkdir -p "$DIST_DIR"
rm -rf "$BUILD_DIR"

echo "1. 正在使用 xcodebuild 编译 Release 模式..."
xcodebuild \
  -project "$PROJECT_DIR/SoniclensBridge.xcodeproj" \
  -scheme SoniclensBridgeMac \
  -configuration Release \
  -destination 'generic/platform=macOS' \
  -derivedDataPath "$BUILD_DIR" \
  CODE_SIGN_IDENTITY="" \
  CODE_SIGNING_REQUIRED=NO \
  build -quiet

BUILT_APP="$BUILD_DIR/Build/Products/Release/SoniclensBridgeMac.app"

if [ ! -d "$BUILT_APP" ]; then
  echo "❌ 编译失败，未找到构建产物: $BUILT_APP"
  exit 1
fi

echo "2. 正在提取并重命名 App 产物..."
TARGET_APP="$DIST_DIR/$APP_NAME.app"
rm -rf "$TARGET_APP"
cp -R "$BUILT_APP" "$TARGET_APP"

if [ "${1:-}" = "--app-only" ]; then
  echo "=========================================="
  echo "🎉 App 构建完成（跳过 DMG/ZIP 制作）: $TARGET_APP"
  echo "=========================================="
  exit 0
fi

echo "3. 正在制作 .dmg 安装镜像..."
DMG_NAME="SoniclensBridge-macOS-$VERSION.dmg"
DMG_PATH="$DIST_DIR/$DMG_NAME"
rm -f "$DMG_PATH"

# 创建临时打包目录，放入 App 并创建 Applications 快捷方式链接
TMP_DMG_DIR="$DIST_DIR/_tmp_dmg"
rm -rf "$TMP_DMG_DIR"
mkdir -p "$TMP_DMG_DIR"
cp -R "$TARGET_APP" "$TMP_DMG_DIR/"
ln -s /Applications "$TMP_DMG_DIR/Applications"

hdiutil create \
  -volname "$APP_NAME" \
  -srcfolder "$TMP_DMG_DIR" \
  -ov \
  -format UDZO \
  "$DMG_PATH" -quiet

rm -rf "$TMP_DMG_DIR"

echo "4. 正在制作便携式 .zip 压缩包..."
ZIP_NAME="SoniclensBridge-macOS-$VERSION.zip"
ZIP_PATH="$DIST_DIR/$ZIP_NAME"
rm -f "$ZIP_PATH"

cd "$DIST_DIR"
ditto -c -k --sequesterRsrc --keepParent "$APP_NAME.app" "$ZIP_NAME"

echo "5. 正在计算产物校验和 (SHA-256)..."
shasum -a 256 "$DMG_PATH" "$ZIP_PATH" > "$DIST_DIR/checksums.txt"

echo "=========================================="
echo "🎉 打包完成！发布产物位于: $DIST_DIR"
echo "  - DMG: $DMG_PATH"
echo "  - ZIP: $ZIP_PATH"
echo "  - Checksums: $DIST_DIR/checksums.txt"
echo "=========================================="
