package musicbrainz

import (
	"context"
	"fmt"
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
