package musicbrainz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uploadedlobster.com/musicbrainzws2"
)

func TestClient(t *testing.T) {
	/*
		todo : Context
		  "AcoustidId": "7a256462-a2f5-43ac-9d68-85ae4de33a51",
		  "MusicbrainzTrackid": "71d2adef-b536-4306-a4f8-655d5c840123",
		  "MusicbrainzReleasetrackid": "d31cf540-9e64-4958-9eab-f71dc3925aed",
		  "MusicbrainzArtistid": "4cd1ce8c-469e-4ff6-a987-59819b975a85",
		  "MusicbrainzAlbumartistid": "4cd1ce8c-469e-4ff6-a987-59819b975a85",
		  "MusicbrainzAlbumid": "46f71e9e-1516-44cb-b840-54e4160ee2a6",
		  "MusicbrainzReleasegroupid": "860dc2de-b1d5-4fc2-b872-f7072316cac1",
		todo：
		构建专辑数据
		先track Group by album 创建专辑表（字段：id,专辑名、等），将track中现有的专辑歌曲创建track_album表(track\track_number\album\album_id) 目前track存在track_number，基本上来源与audir数据按照track_number创建


		调用musicbrainzws2能力精准补全专辑数据
		识别专辑 可能需要用专辑名称 搜索到多个数据（分别保存多个json数据到release_mb表（MBID，json）） 前端展示由用户选择对应的专辑确定后 将数据保留到专辑关联album_release_mb表（专辑名，MBID） 用外键关联
		如果选择了某个mb表并创建了关联表，标识已经拥有准确的专辑信息和MBID

		调用musicbrainzws2深度维护
		通过LookupRelease查询准确的专辑信息保存当前数据，并同时检查track_album表是否存在有效数据，没有既保存歌曲数据、存在检查track_number是否正确并修正，同时将LookupRelease获取的json结果更新到release_mb指定的MBID中

		新增album、track_album、release_mb、album_release_mb，分析当前文件、当前项目的上下文设计合适的库表结构，
			建表语句在internal/model/sql/dml
			数据语句在internal/model/sql/ddl

	*/
	ctx := context.Background()
	InitClient()
	defer Close(ctx)
	client := GetClient()
	fmt.Println(client.UserAgent())
	// 获取艺术家信息
	artist, err := client.LookupArtist(
		ctx, "4cd1ce8c-469e-4ff6-a987-59819b975a85", musicbrainzws2.IncludesFilter{
			Includes: []string{
				"releases",
				"works",
				"recordings",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("artist")
	fmt.Println(artist)
	a := musicbrainzws2.Artist{}
	a = artist
	fmt.Println(a.ID) // 所有的Artist、Recording、Work、都有的id现在系统中缺少这个ID的记录
	/*releaseGroup, err := client.LookupReleaseGroup(
		ctx, "860dc2de-b1d5-4fc2-b872-f7072316cac1", musicbrainzws2.IncludesFilter{},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("releaseGroup")
	fmt.Println(releaseGroup)*/
	time.Sleep(1 * time.Second)
	// 获取专辑信息
	release, err := client.LookupRelease(
		ctx, "46f71e9e-1516-44cb-b840-54e4160ee2a6", musicbrainzws2.IncludesFilter{
			Includes: []string{
				"collections",
				"labels",
				"recordings",
				"media",
				"release-groups",
				"genres",
				"label-info",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("release")
	fmt.Println(release)
	time.Sleep(1 * time.Second)
	browseReleases, err := client.BrowseReleases(
		ctx, musicbrainzws2.ReleaseFilter{
			// AreaMBID:         "",
			ArtistMBID: "4cd1ce8c-469e-4ff6-a987-59819b975a85",
			// CollectionMBID:   "",
			// LabelMBID:        "",
			TrackMBID: "71d2adef-b536-4306-a4f8-655d5c840123",
			// TrackArtistMBID:  "",
			// RecordingMBID:    "",
			// ReleaseGroupMBID: "",
			// Status:           "",
			// Type:             "",
			// Includes:         nil,
		}, musicbrainzws2.Paginator{
			Offset: 0,
			Limit:  10,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("release")
	fmt.Println(
		browseReleases,
	)

	time.Sleep(1 * time.Second)
	// 搜索专辑数据
	searchReleases, err := client.SearchReleases(
		ctx, musicbrainzws2.SearchFilter{
			Query: "万能青年旅店",
			Includes: []string{
				"collections",
				"labels",
				"recordings",
				"media",
				"release-groups",
			},
			Dismax: false,
		}, musicbrainzws2.Paginator{
			Offset: 0,
			Limit:  10,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("searchReleases")
	fmt.Println(
		searchReleases,
	)
}
