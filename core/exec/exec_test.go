package exec

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-audio/wav"

	"github.com/vincentchyu/sonic-lens/core/log"
)

func init() {
	_, _ = log.LogInit("./.logs", "info", make(<-chan struct{}))
}

func TestExecExiftoolHandl(t *testing.T) {
	ctx := context.Background()
	info, err := BuildExiftoolHandle(
		ctx,
		"/Users/vincent/Music/本地音乐/CD/寸铁/近人可讀/01 若你心年輕.flac",
	)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(info)
	fmt.Println(info.GetMusicBrainzTrackId())
	fmt.Println(info.GetTrackNumber())
	fmt.Println(info.GetArtists())
	fmt.Println(info.GetArtist())
	fmt.Println(info.GetUniqueID())
	fmt.Println(info.GetTitle())
	fmt.Println(info.GetSource())
	fmt.Println(info.GetComposer())
	fmt.Println(info.GetGenre())
	fmt.Println(info.GetDuration())
	fmt.Println(info.GetReleaseDate())
	fmt.Println("==============================")
	info, err = BuildExiftoolHandle(
		ctx, "/Users/vincent/Documents/多媒体/音乐/CD/Chinese Football/Chinese Football/02 守门员.m4a",
	)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(info)
	fmt.Println(info.GetMusicBrainzTrackId())
	fmt.Println(info.GetTrackNumber())
	fmt.Println(info.GetArtists())
	fmt.Println(info.GetArtist())
	fmt.Println(info.GetUniqueID())
	fmt.Println(info.GetTitle())
	fmt.Println(info.GetSource())
	fmt.Println(info.GetComposer())
	fmt.Println(info.GetGenre())
	fmt.Println(info.GetDuration())
	fmt.Println(info.GetReleaseDate())
	fmt.Println("==============================")
	info, err = BuildExiftoolHandle(ctx, "/Users/vincent/Documents/多媒体/音乐/CD/李志/梵高先生/05 广场.wav")
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(info)
	fmt.Println(info.GetMusicBrainzTrackId())
	fmt.Println(info.GetTrackNumber())
	fmt.Println(info.GetArtists())
	fmt.Println(info.GetArtist())
	fmt.Println(info.GetUniqueID())
	fmt.Println(info.GetTitle())
	fmt.Println(info.GetSource())
	fmt.Println(info.GetComposer())
	fmt.Println(info.GetGenre())
	fmt.Println(info.GetDuration())
	fmt.Println(info.GetReleaseDate())
	info, err = BuildExiftoolHandle(nil, "/Users/vincent/Documents/多媒体/音乐/CD/万能青年旅店/张洲/01 张洲.wav")
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(info)
	fmt.Println(info.GetMusicBrainzTrackId())
	fmt.Println(info.GetTrackNumber())
	fmt.Println(info.GetArtist())
	fmt.Println(info.GetArtists())
	fmt.Println(info.GetBundleID())

}

func TestName(t *testing.T) {
	output, err := runCommand(nil, "nowplaying-cli", "get", "title", "album", "artist")
	if err != nil {
		t.Fatal(err)
	}
	split := strings.Split(output, "\n")
	fmt.Println(split)
}

func TestWavInfoHandle(t *testing.T) {
	ok, file, err := IsValidPath(nil, "file:///Users/vincent/Documents/多媒体/音乐/CD/万能青年旅店/张洲/01 张洲.wav")
	if err != nil {
		t.Fatal(err)
		return
	}
	if ok {
		in, err := os.Open(file)
		defer func(in *os.File) {
			err := in.Close()
			if err != nil {

			}
		}(in)
		if err != nil {
			t.Fatal(err)
		}
		mwav := wav.NewDecoder(in)
		/*buf, err := d.FullPCMBuffer()
		if err != nil {
			t.Fatal(err)
		}*/
		mwav.ReadInfo()
		fmt.Println(mwav)
	}

	handle, err := BuildWavInfoHandle("file:///Users/vincent/Documents/多媒体/音乐/CD/李志/我爱南京/2-05 思念观世音.wav")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(handle.GetTitle())
	fmt.Println(handle.GetArtists())
	fmt.Println(handle.GetArtist())
	fmt.Println(handle.GetTrackNumber())
	fmt.Println(handle.GetMusicBrainzTrackId())
	handle, err = BuildWavInfoHandle("file:///Users/vincent/Documents/多媒体/音乐/CD/万能青年旅店/张洲/01 张洲.wav")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(handle.GetTitle())
	fmt.Println(handle.GetArtists())
	fmt.Println(handle.GetArtist())
	fmt.Println(handle.GetTrackNumber())
	fmt.Println(handle.GetMusicBrainzTrackId())
}
func TestGetMRMediaNowPlaying(t *testing.T) {
	nowPlaying, err := GetMRMediaNowPlayingCli(nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(nowPlaying)
}

func TestEnv(t *testing.T) {
	getenv := os.Getenv("PATH")
	fmt.Println(getenv)
	err := os.Setenv("PATH", getenv+":./shell/bin")
	if err != nil {
		return
	}
	getenv = os.Getenv("PATH")
	fmt.Println(getenv)
}
