package scrobbler

import (
	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
)

// NetEasePlayerController 网易云音乐播放器控制器。
type NetEasePlayerController struct {
	MediaControlPlayerController
}

func NewNetEasePlayerController() *NetEasePlayerController {
	return &NetEasePlayerController{
		MediaControlPlayerController: MediaControlPlayerController{
			source:   common.PlayerNetEase,
			bundleID: exec.MRMediaNowPlaying163,
		},
	}
}
