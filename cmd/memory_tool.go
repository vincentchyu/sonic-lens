package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vincentchyu/sonic-lens/memory"
)

// NewMemoryToolCommand returns a new memory tool command
func NewMemoryToolCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory-tool",
		Short: "A tool to manage memory index and feature manifests",
	}

	// Add command to create a new feature manifest
	createCmd := &cobra.Command{
		Use:   "create [feature name] [summary]",
		Short: "Create a new feature manifest and add it to memory index",
		Args:  cobra.ExactArgs(2),
		RunE:  createFeature,
	}

	createCmd.Flags().StringP(
		"content", "c", "# {feature_name} 特性清单\n\n## 特性概述\n\n## 功能要点\n\n## 实现细节\n\n## 扩展性考虑",
		"Template content for the feature manifest",
	)

	cmd.AddCommand(createCmd)
	return cmd
}

func createFeature(cmd *cobra.Command, args []string) error {
	featureName := args[0]
	summary := args[1]

	// Get template content
	content, err := cmd.Flags().GetString("content")
	if err != nil {
		return err
	}

	// Replace placeholder with actual feature name
	content = fmt.Sprintf(content, featureName)

	// Create feature manifest
	filePath, err := memory.CreateFeatureManifest(featureName, content)
	if err != nil {
		return fmt.Errorf("failed to create feature manifest: %w", err)
	}

	// Add entry to memory index
	if err := memory.AddEntry(summary, filePath, featureName+" 特性清单"); err != nil {
		return fmt.Errorf("failed to add entry to memory index: %w", err)
	}

	fmt.Printf("Successfully created feature manifest: %s\n", filePath)
	fmt.Printf("Successfully added entry to memory index\n")

	return nil
}
