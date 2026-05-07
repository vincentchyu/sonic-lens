package scrobbler

import (
	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
)

// RoonPlayerController Roon播放器控制器。
type RoonPlayerController struct {
	MediaControlPlayerController
}

func NewRoonPlayerController() *RoonPlayerController {
	return &RoonPlayerController{
		MediaControlPlayerController: MediaControlPlayerController{
			source:   common.PlayerRoon,
			bundleID: exec.MRMediaNowPlayingAppRoon,
		},
	}
}
