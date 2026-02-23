#!/bin/zsh

launchctl stop  com.vincent.sonic-lens.job
launchctl remove  com.vincent.sonic-lens.job
rm -f ~/Library/LaunchAgents/com.vincent.sonic-lens.job.plist
rm -f ./shell/bin/sonic-lens
rm -f ./shell/bin/nowplaying-cli-mac
sudo rm -f /opt/local/bin/nowplaying-cli-mac



