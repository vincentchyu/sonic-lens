package scrobbler

import (
	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
)

// Foobar2000PlayerController foobar2000 播放器控制器。
type Foobar2000PlayerController struct {
	MediaControlPlayerController
}

func NewFoobar2000PlayerController() *Foobar2000PlayerController {
	return &Foobar2000PlayerController{
		MediaControlPlayerController: MediaControlPlayerController{
			source:   common.PlayerFoobar2000,
			bundleID: exec.MRMediaNowPlayingAppFoobar2000,
		},
	}
}
