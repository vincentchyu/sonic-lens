package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/lastfm"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// NewSyncRecordsCommand returns a new sync records command
func NewSyncRecordsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "sync-records",
		Short: "Sync unscrobbled records to Last.fm",
		RunE:  syncRecords,
	}

	command.Flags().IntP("limit", "l", 10, "Number of records to sync")

	return command
}

func syncRecords(cmd *cobra.Command, args []string) error {
	// Initialize database
	if err := model.InitDB(config.ConfigObj.Database.Path, nil); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	limit, _ := cmd.Flags().GetInt("limit")

	// GetC2E unscrobbled records
	records, err := model.GetUnscrobbledRecords(context.Background(), limit)
	if err != nil {
		return fmt.Errorf("failed to get unscrobbled records: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("No unscrobbled records found")
		return nil
	}

	fmt.Printf("Found %d unscrobbled records, syncing...\n", len(records))

	// Sync records to Last.fm
	for _, record := range records {
		req := &lastfm.PushTrackScrobbleReq{
			Artist:             record.Artist,
			AlbumArtist:        record.AlbumArtist,
			Track:              record.Track,
			Album:              record.Album,
			Duration:           record.Duration,
			Timestamp:          record.PlayTime.Unix(),
			MusicBrainzTrackID: record.MusicBrainzID,
			TrackNumber:        int64(record.TrackNumber),
		}

		_, err := lastfm.PushTrackScrobble(context.Background(), req)
		if err != nil {
			fmt.Printf("Failed to scrobble track %s: %v\n", record.Track, err)
			continue
		}

		// Update scrobbled status in database
		if err := model.UpdateScrobbledStatus(context.Background(), record.ID, true); err != nil {
			fmt.Printf("Failed to update scrobbled status for track %s: %v\n", record.Track, err)
		} else {
			fmt.Printf("Successfully scrobbled track: %s\n", record.Track)
		}

		// Add a small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
