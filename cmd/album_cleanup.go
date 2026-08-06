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

// cleanupDuplicateAlbums 标记过期
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

// NewCleanupReleaseTypeSuffixesCommand 返回发行类型后缀清洗命令。
// 该命令扫描专辑名中含 " - EP"/" - Single"/" - LP" 的历史记录，
// 剥离后缀并写入独立的 release_type 列；若已存在同名干净专辑则自动合并。
// ./sonic-lens -c ./config/config_dev.yaml  cleanup-release-type-suffixes --apply --limit 1
func NewCleanupReleaseTypeSuffixesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cleanup-release-type-suffixes",
		Short: "剥离历史专辑名中的 Apple Music 发行类型后缀（EP/Single/LP），写入独立 release_type 列",
		Long: `扫描所有专辑名中含 " - EP"、" - Single"、" - LP" 等连字符后缀的记录，
将其剥离为干净主标题并存入 release_type 列。
若数据库中已存在相同作者和名称的干净专辑，则将曲目关联和 MB 关联迁移过去后删除含后缀记录。

默认以 dry-run 模式运行，使用 --apply 标志执行真实写入。`,
		RunE: cleanupReleaseTypeSuffixes,
	}

	command.Flags().Bool("apply", false, "执行真实写入而非 dry-run 预览")
	command.Flags().Int("limit", 0, "最多处理的专辑条数（0 表示无限制）")

	return command
}

func cleanupReleaseTypeSuffixes(cmd *cobra.Command, args []string) error {
	if err := model.InitDB(config.ConfigObj.Database.Path, nil); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}

	apply, _ := cmd.Flags().GetBool("apply")
	limit, _ := cmd.Flags().GetInt("limit")

	report, err := model.CleanupReleaseTypeSuffixes(
		model.CleanupReleaseTypeSuffixesParams{
			Ctx:    context.Background(),
			Limit:  limit,
			DryRun: !apply,
		},
	)
	if err != nil {
		return fmt.Errorf("清洗发行类型后缀失败: %w", err)
	}

	if len(report.Items) == 0 && len(report.Skipped) == 0 {
		fmt.Println("未找到含发行类型后缀的专辑记录")
		return nil
	}

	mode := "dry-run"
	if apply {
		mode = "已应用"
	}
	fmt.Printf("发行类型后缀清洗 [%s]，共处理 %d 条\n", mode, len(report.Items))
	for _, item := range report.Items {
		if item.MergedIntoID > 0 {
			fmt.Printf(
				"  合并: album_id=%d [%s] -> album_id=%d [%s] (release_type=%s)\n",
				item.AlbumID, item.OldName, item.MergedIntoID, item.NewName, item.ReleaseType,
			)
		} else {
			fmt.Printf(
				"  重命名: album_id=%d [%s] -> [%s] (release_type=%s)\n",
				item.AlbumID, item.OldName, item.NewName, item.ReleaseType,
			)
		}
	}
	if len(report.Skipped) > 0 {
		fmt.Printf("跳过（出错）%d 条\n", len(report.Skipped))
		for _, item := range report.Skipped {
			fmt.Printf("  跳过: album_id=%d [%s]\n", item.AlbumID, item.OldName)
		}
	}
	return nil
}
