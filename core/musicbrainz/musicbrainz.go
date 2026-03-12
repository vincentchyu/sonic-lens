package musicbrainz

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uploadedlobster.com/musicbrainzws2"
)

var musicbrainzClient *musicbrainzws2.Client
var one sync.Once

func InitClient() {
	one.Do(
		func() {
			musicbrainzClient = musicbrainzws2.NewClient(
				musicbrainzws2.AppInfo{
					Name:    "sonic-lens",
					Version: "1.0",
					URL:     "https://blog-vincent.chyu.org/web/sonic-lens/",
				},
			)
		},
	)
}

func GetClient() *musicbrainzws2.Client {
	return musicbrainzClient
}

// Close
func Close(ctx context.Context) {
	if musicbrainzClient != nil {
		err := musicbrainzClient.Close()
		if err != nil {
			fmt.Println("error closing musicbrainz client", err)
		}
	}
}

// TrackTitleWithFeat 检查 Track 的 ArtistCredit 是否包含 feat 合作者，
// 如果包含，将其格式化为 Title(feat.ArtistName) 的形式，否则返回原 Title。
// Hikky Burr (feat. Bill Cosby)
func TrackTitleWithFeat(track musicbrainzws2.Track) string {
	title := track.Title
	if len(track.ArtistCredit) <= 1 {
		return title
	}

	for i := 0; i < len(track.ArtistCredit)-1; i++ {
		joinPhrase := strings.ToLower(strings.TrimSpace(track.ArtistCredit[i].JoinPhrase))
		if strings.Contains(joinPhrase, "feat") || strings.Contains(joinPhrase, "ft") {
			// 找到 feat，提取下一个艺术家作为合作者
			featArtist := track.ArtistCredit[i+1].Name
			title = fmt.Sprintf("%s(feat.%s)", title, featArtist)
		}
	}

	return title
}
