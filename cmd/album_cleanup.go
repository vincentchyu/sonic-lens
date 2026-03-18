package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

// NewCleanupDuplicateAlbumsCommand 返回重复专辑清洗命令。
func NewCleanupDuplicateAlbumsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cleanup-duplicate-albums",
		Short: "Merge duplicate albums grouped by artist and album name",
		RunE:  cleanupDuplicateAlbums,
	}

	command.Flags().Bool("apply", false, "Apply destructive cleanup instead of dry-run")
	command.Flags().Int("limit", 0, "Maximum duplicate groups to process")
	command.Flags().String("artist", "", "Only process duplicate groups for the artist")
	command.Flags().String("album", "", "Only process duplicate groups for the album name")
	command.Flags().Bool("continue-on-error", false, "Skip conflicted groups and continue cleanup")

	return command
}

func cleanupDuplicateAlbums(cmd *cobra.Command, args []string) error {
	if err := model.InitDB(config.ConfigObj.Database.Path, nil); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	apply, _ := cmd.Flags().GetBool("apply")
	limit, _ := cmd.Flags().GetInt("limit")
	artist, _ := cmd.Flags().GetString("artist")
	name, _ := cmd.Flags().GetString("album")
	continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

	report, err := model.CleanupDuplicateAlbums(
		model.CleanupDuplicateAlbumsParams{
			Ctx:             context.Background(),
			Artist:          artist,
			Name:            name,
			Limit:           limit,
			DryRun:          !apply,
			ContinueOnError: continueOnError,
		},
	)
	if err != nil {
		return fmt.Errorf("cleanup duplicate albums failed: %w", err)
	}

	if len(report.Groups) == 0 {
		fmt.Println("No duplicate album groups found")
		return nil
	}

	mode := "dry-run"
	if apply {
		mode = "applied"
	}
	fmt.Printf("Duplicate album cleanup %s for %d groups\n", mode, len(report.Groups))
	for _, group := range report.Groups {
		fmt.Printf(
			"artist=%s album=%s canonical_album_id=%d merged_album_ids=%v\n",
			group.Artist, group.Name, group.CanonicalAlbumID, group.MergedAlbumIDs,
		)
	}
	if len(report.Skipped) > 0 {
		fmt.Printf("Skipped %d groups due to conflicts\n", len(report.Skipped))
		for _, skipped := range report.Skipped {
			fmt.Printf("skipped artist=%s album=%s reason=%s\n", skipped.Artist, skipped.Name, skipped.Reason)
		}
	}
	return nil
}
