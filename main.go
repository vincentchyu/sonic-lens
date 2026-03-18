package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vincentchyu/sonic-lens/api"
	"github.com/vincentchyu/sonic-lens/cmd"
	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/bonjour"
	"github.com/vincentchyu/sonic-lens/core/log"
	"github.com/vincentchyu/sonic-lens/core/musicbrainz"
	"github.com/vincentchyu/sonic-lens/core/redis"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/internal/cache"
	"github.com/vincentchyu/sonic-lens/internal/model"
	"github.com/vincentchyu/sonic-lens/internal/scrobbler"
	d1sync "github.com/vincentchyu/sonic-lens/internal/sync"
)

var (
	configFile = new(string)
	isMobile   = new(bool)
)

func main() {
	rootCmd := NewCommand("sonic-lens", "", "")
	// command.SetHelpTemplate("使用-c 设置配置文件路径\n使用-m 设置true/false")
	rootCmd.Version = "1.0.0"
	rootCmd.Args = cobra.NoArgs
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		config.InitConfig(*configFile)
	}
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error { return initServer() }

	flags := rootCmd.Flags()
	flags.SortFlags = false
	flags.StringVarP(configFile, "config", "c", "config/config.yaml", "config file")
	flags.BoolVarP(isMobile, "mobile", "m", false, "it a mobile")

	// Add sync-records subcommand
	rootCmd.AddCommand(newSyncRecordsCommand())

	// Add memory-tool subcommand
	rootCmd.AddCommand(newMemoryToolCommand())

	// Add cleanup-duplicate-albums subcommand
	rootCmd.AddCommand(newCleanupDuplicateAlbumsCommand())

	// Add replay-track-play-records subcommand
	rootCmd.AddCommand(newReplayTrackPlayRecordsCommand())

	cobra.CheckErr(rootCmd.Execute())
}

func newSyncRecordsCommand() *cobra.Command {
	return cmd.NewSyncRecordsCommand()
}

func newMemoryToolCommand() *cobra.Command {
	return cmd.NewMemoryToolCommand()
}

func newCleanupDuplicateAlbumsCommand() *cobra.Command {
	return cmd.NewCleanupDuplicateAlbumsCommand()
}

func newReplayTrackPlayRecordsCommand() *cobra.Command {
	return cmd.NewReplayTrackPlayRecordsCommand()
}

func initServer() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	c := make(chan struct{})
	dbLogger, redisLogger := log.LogInit(config.ConfigObj.Log.Path, config.ConfigObj.Log.Level, c)

	// Initialize telemetry
	if err := telemetry.Init(config.ConfigObj.Telemetry); err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	// Initialize Redis
	redis.InitRedis(config.ConfigObj.Redis, redisLogger)
	// Initialize Musicbrainz
	musicbrainz.InitClient()
	// Initialize database
	if err := model.InitDB(config.ConfigObj.Database.Path, dbLogger); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize genre cache with refresh timer
	cancelFuncCacheInitializeGenreCache := cache.InitializeGenreCache(ctx)

	// Start D1 sync scheduler
	// go d1sync.StartD1SyncScheduler(ctx)
	// Start dashboard stat scheduler
	go d1sync.StartDashboardStatScheduler(ctx)
	go d1sync.StartTrackPlayReplayScheduler(ctx)

	// Start scrobblerRun goroutine
	go api.StartHTTPServer(ctx, config.ConfigObj.Telemetry.Name)

	// 启动 Bonjour 广播用于局域网发现
	stopBonjour := bonjour.Start(ctx, config.ConfigObj.Bonjour, config.ConfigObj.HTTP.Port)

	// Start HTTP server in a separate goroutine
	_ = scrobblerRun(c)

	<-ctx.Done()

	fmt.Println("system exiting")
	defer func() {
		ctx := context.Background()
		stopBonjour()
		cancelFuncCacheInitializeGenreCache()
		musicbrainz.Close(ctx)
		err := telemetry.Shutdown(ctx)
		if err != nil {
			panic(err)
		}
		close(c)
	}()
	return nil
}

func scrobblerRun(c <-chan struct{}) error {
	scrobbler.Init(
		context.Background(),
		config.ConfigObj.Lastfm.ApiKey,
		config.ConfigObj.Lastfm.SharedSecret,
		config.ConfigObj.Lastfm.UserLoginToken,
		*isMobile,
		config.ConfigObj.Lastfm.UserUsername,
		config.ConfigObj.Lastfm.UserPassword,
		config.ConfigObj.Scrobblers,
		c,
	)

	// musixmatch.InitMxmClient(config.ConfigObj.Musixmatch.ApiKey)
	return nil
}
