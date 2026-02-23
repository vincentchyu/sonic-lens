package musixmatch

import (
	"testing"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

func init() {

}

func init() {
	c := make(chan struct{})
	config.InitConfig("../config/config_bak.yaml")
	_, _ = log.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, c)
	InitMxmClient(config.ConfigObj.Musixmatch.ApiKey)
}
func TestGetMatcherLyrics(t *testing.T) {
	err := GetMatcherLyrics("Omnipotent Youth Society", "秦皇岛")
	if err != nil {
		return
	}
}
func TestSearchArtist(t *testing.T) {
	SearchArtist("Omnipotent Youth Society")
	SearchArtist("万能青年旅店")
}
