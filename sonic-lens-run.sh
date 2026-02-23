#!/bin/zsh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)/shell/script"

case "$1" in
  init)
    sh "$SCRIPT_DIR/build_sonic-lens_launchctl.sh"
    ;;
  start)
    sh "$SCRIPT_DIR/start_sonic-lens.sh"
    ;;
  stop)
    sh "$SCRIPT_DIR/stop_sonic-lens.sh"
    ;;
  *)
    echo "用法: $0 {init|start|stop}"
    exit 1
    ;;
esac