package audirvana

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/common"
)

func TestParseAudirvanaDuration(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int64
		wantErr bool
	}{
		{name: "integer", raw: "284", want: 284},
		{name: "float", raw: "297.21600000000001", want: 297},
		{name: "float with comma", raw: "297,6", want: 298},
		{name: "empty", raw: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAudirvanaDuration(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAudirvanaDuration(%q) expected error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAudirvanaDuration(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("parseAudirvanaDuration(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResolveRunningAppAndIsRunning(t *testing.T) {
	origTellFn := tellFn
	t.Cleanup(func() {
		tellFn = origTellFn
		ClearRunningAppCache()
	})

	tests := []struct {
		name        string
		tellResult  string
		tellErr     error
		wantApp     string
		wantRunning bool
	}{
		{
			name:        "Audirvana Studio 正在运行",
			tellResult:  common.AppAudirvanaStudio,
			wantApp:     common.AppAudirvanaStudio,
			wantRunning: true,
		},
		{
			name:        "Audirvana Origin 正在运行",
			tellResult:  common.AppAudirvanaOrigin,
			wantApp:     common.AppAudirvanaOrigin,
			wantRunning: true,
		},
		{
			name:        "均未运行",
			tellResult:  "",
			wantApp:     "",
			wantRunning: false,
		},
		{
			name:        "System Events 异常报错",
			tellErr:     errors.New("execution error: System Events not responding"),
			wantApp:     "",
			wantRunning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClearRunningAppCache()
			tellFn = func(ctx context.Context, application string, command string) (string, error) {
				assert.Equal(t, common.AppSystemEvents, application)
				assert.Contains(t, command, "listApplicationProcessNames")
				return tt.tellResult, tt.tellErr
			}

			app := ResolveRunningApp(t.Context())
			assert.Equal(t, tt.wantApp, app)
			assert.Equal(t, tt.wantRunning, IsRunning(t.Context()))
		})
	}
}

func TestGetState(t *testing.T) {
	origTellFn := tellFn
	t.Cleanup(func() {
		tellFn = origTellFn
		ClearRunningAppCache()
	})

	t.Run("Audirvana Studio 播放态获取成功", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			if application == common.AppSystemEvents {
				return common.AppAudirvanaStudio, nil
			}
			assert.Equal(t, common.AppAudirvanaStudio, application)
			assert.Contains(t, command, "get player state")
			return "Playing", nil
		}

		state, err := GetState(t.Context())
		require.NoError(t, err)
		assert.Equal(t, common.PlayerState(common.PlayerStatePlaying), state)
	})

	t.Run("Audirvana Origin 暂停态获取成功", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			if application == common.AppSystemEvents {
				return common.AppAudirvanaOrigin, nil
			}
			assert.Equal(t, common.AppAudirvanaOrigin, application)
			assert.Contains(t, command, "get player state")
			return "Paused", nil
		}

		state, err := GetState(t.Context())
		require.NoError(t, err)
		assert.Equal(t, common.PlayerState(common.PlayerStatePaused), state)
	})

	t.Run("未运行时获取状态返回错误", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			if application == common.AppSystemEvents {
				return "", nil
			}
			return "", errors.New("should not be called")
		}

		state, err := GetState(t.Context())
		require.Error(t, err)
		assert.Equal(t, common.PlayerState(common.PlayerStateDefault), state)
		assert.Contains(t, err.Error(), "not running")
	})
}

func TestGetNowPlayingTrackInfo(t *testing.T) {
	origTellFn := tellFn
	t.Cleanup(func() {
		tellFn = origTellFn
		ClearRunningAppCache()
	})

	t.Run("Audirvana Studio 正在播放曲目成功解析", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			if application == common.AppSystemEvents {
				return common.AppAudirvanaStudio, nil
			}
			assert.Equal(t, common.AppAudirvanaStudio, application)
			assert.Contains(t, command, "playing track title")
			// 返回 TrackInfo pipe 分隔字符串: Title|Album|Artist|Duration|Position|Url
			return "春风|成长小说|海朋森|284|12.5|file:///Users/vincent/Music/01.m4a", nil
		}

		info := GetNowPlayingTrackInfo(t.Context())
		require.NotNil(t, info)
		assert.Equal(t, "春风", info.Title)
		assert.Equal(t, "成长小说", info.Album)
		assert.Equal(t, "海朋森", info.Artist)
		assert.Equal(t, int64(284), info.Duration)
		assert.Equal(t, 12.5, info.Position)
		assert.Equal(t, "file:///Users/vincent/Music/01.m4a", info.Url)
		assert.Equal(t, common.AppAudirvanaStudio, info.AppName)
	})

	t.Run("Audirvana Origin 正在播放曲目成功解析", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			if application == common.AppSystemEvents {
				return common.AppAudirvanaOrigin, nil
			}
			assert.Equal(t, common.AppAudirvanaOrigin, application)
			assert.Contains(t, command, "playing track title")
			return "No Surprises|OK Computer|Radiohead|229.0|55.0|file:///Users/vincent/Music/Radiohead/01.flac", nil
		}

		info := GetNowPlayingTrackInfo(t.Context())
		require.NotNil(t, info)
		assert.Equal(t, "No Surprises", info.Title)
		assert.Equal(t, "OK Computer", info.Album)
		assert.Equal(t, "Radiohead", info.Artist)
		assert.Equal(t, int64(229), info.Duration)
		assert.Equal(t, 55.0, info.Position)
		assert.Equal(t, "file:///Users/vincent/Music/Radiohead/01.flac", info.Url)
		assert.Equal(t, common.AppAudirvanaOrigin, info.AppName)
	})

	t.Run("未运行时返回 nil", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			return "", nil
		}

		info := GetNowPlayingTrackInfo(t.Context())
		assert.Nil(t, info)
	})

	t.Run("AppleScript 报错时清空缓存并返回 nil", func(t *testing.T) {
		ClearRunningAppCache()
		tellFn = func(ctx context.Context, application string, command string) (string, error) {
			if application == common.AppSystemEvents {
				return common.AppAudirvanaStudio, nil
			}
			return "", errors.New("app unexpectedly closed")
		}

		info := GetNowPlayingTrackInfo(t.Context())
		assert.Nil(t, info)
		assert.Nil(t, runningApp.Load())
	})
}
