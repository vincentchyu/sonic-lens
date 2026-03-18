package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// NewReplayTrackPlayRecordsCommand 返回播放流水补归因命令。
func NewReplayTrackPlayRecordsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "replay-track-play-records",
		Short: "Replay unapplied track play records by default",
		RunE:  replayTrackPlayRecords,
	}

	command.Flags().Bool("apply", false, "Apply replay instead of dry-run")
	command.Flags().Int("limit", 20, "Maximum play records to process")
	command.Flags().String("source", "", "Only process play records for the source")
	command.Flags().Int64Slice("id", nil, "Only process specific play record ids")
	command.Flags().String("played-from", "", "Only process play records on or after RFC3339 time")
	command.Flags().String("played-to", "", "Only process play records on or before RFC3339 time")
	command.Flags().Bool("only-unapplied", false, "Only process play records not yet applied to library")
	command.Flags().Bool("only-unresolved", false, "Include play records with pending/unresolved resolution status")

	return command
}

func replayTrackPlayRecords(cmd *cobra.Command, args []string) error {
	if err := model.InitDB(config.ConfigObj.Database.Path, nil); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	apply, _ := cmd.Flags().GetBool("apply")
	limit, _ := cmd.Flags().GetInt("limit")
	source, _ := cmd.Flags().GetString("source")
	recordIDs, _ := cmd.Flags().GetInt64Slice("id")
	playedFromRaw, _ := cmd.Flags().GetString("played-from")
	playedToRaw, _ := cmd.Flags().GetString("played-to")
	onlyUnapplied, _ := cmd.Flags().GetBool("only-unapplied")
	onlyUnresolved, _ := cmd.Flags().GetBool("only-unresolved")

	playedFrom, err := parseReplayTimeFlag(playedFromRaw)
	if err != nil {
		return fmt.Errorf("invalid --played-from: %w", err)
	}
	playedTo, err := parseReplayTimeFlag(playedToRaw)
	if err != nil {
		return fmt.Errorf("invalid --played-to: %w", err)
	}

	report, err := model.ReplayTrackPlayRecords(
		model.ReplayTrackPlayRecordsParams{
			Ctx:            context.Background(),
			Limit:          limit,
			Source:         source,
			RecordIDs:      recordIDs,
			PlayedFrom:     playedFrom,
			PlayedTo:       playedTo,
			DryRun:         !apply,
			OnlyUnapplied:  onlyUnapplied,
			OnlyUnresolved: onlyUnresolved,
		},
	)
	if err != nil {
		return fmt.Errorf("replay track play records failed: %w", err)
	}

	if len(report.Results) == 0 {
		fmt.Println("No replayable track play records found")
		return nil
	}

	mode := "dry-run"
	if apply {
		mode = "applied"
	}
	fmt.Printf("Track play record replay %s for %d records\n", mode, len(report.Results))
	for _, result := range report.Results {
		if apply {
			fmt.Printf(
				"id=%d source=%s artist=%s album=%s track=%s before_status=%s before_applied=%t after_status=%s after_applied=%t resolved_track_id=%d\n",
				result.ID,
				result.Source,
				result.Artist,
				result.Album,
				result.Track,
				result.BeforeStatus,
				result.BeforeApplied,
				result.AfterStatus,
				result.AfterApplied,
				result.ResolvedTrackID,
			)
			continue
		}
		fmt.Printf(
			"id=%d source=%s artist=%s album=%s track=%s before_status=%s before_applied=%t\n",
			result.ID,
			result.Source,
			result.Artist,
			result.Album,
			result.Track,
			result.BeforeStatus,
			result.BeforeApplied,
		)
	}

	return nil
}

func parseReplayTimeFlag(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}
