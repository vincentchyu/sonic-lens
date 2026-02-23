package musixmatch

import (
	"context"
	"fmt"
	"log"
	"net/http"

	mxm "github.com/milindmadhukar/go-musixmatch"
	"github.com/milindmadhukar/go-musixmatch/params"
)

var (
	client    = http.DefaultClient
	mxmClient *mxm.Client
)

func InitMxmClient(apiKey string) {
	mxmClient = mxm.New(apiKey, client)
}

// HasClient 用于判断 Musixmatch Client 是否已初始化
func HasClient() bool {
	return mxmClient != nil
}

func SearchArtist(artist string) {
	artists, err := mxmClient.SearchArtist(context.Background(), params.QueryArtist(artist))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(artists)
}

// GetLyrics 获取指定艺术家与曲目的歌词文本
// 这里简单封装 matcher.lyrics.get 接口，返回 lyrics_body 字段
func GetLyrics(ctx context.Context, artist, track string) (string, error) {
	if mxmClient == nil {
		return "", fmt.Errorf("musixmatch client 未初始化")
	}
	lyrics, err := mxmClient.GetMatcherLyrics(
		ctx,
		params.QueryTrack(track),
		params.QueryArtist(artist),
	)
	if err != nil {
		return "", err
	}
	return lyrics.Body, nil
}

func GetMatcherLyrics(artist, track string) error {
	lyrics, err := mxmClient.GetMatcherLyrics(
		context.Background(), params.QueryTrack(track),
		params.QueryArtist(artist),
	)
	if err != nil {
		return err
	}
	fmt.Println(lyrics)
	return nil
}
