package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MemoryIndexEntry represents an entry in the memory index
type MemoryIndexEntry struct {
	Date        string `json:"date"`
	FeatureName string `json:"feature_name"`
	Summary     string `json:"summary"`
	Link        string `json:"link"`
}

// AddEntry adds a new entry to the memory index
func AddEntry(summary, link, featureName string) error {
	// Create directory for today's date
	dateStr := time.Now().Format("2006-01-02")
	dirPath := filepath.Join("memory", dateStr)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Move the feature manifest to the date directory if it exists in model/
	if filepath.Dir(link) == "model" {
		oldPath := link
		newPath := filepath.Join(dirPath, filepath.Base(link))
		if err := os.Rename(oldPath, newPath); err != nil {
			// If rename fails, try to copy and then remove
			data, err := os.ReadFile(oldPath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			if err := os.WriteFile(newPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			os.Remove(oldPath)
		}
	}

	// Update the memory index file
	return updateMemoryIndex(dateStr, summary, link, featureName)
}

// CreateFeatureManifest creates a new feature manifest file
func CreateFeatureManifest(featureName, content string) (string, error) {
	// Create directory for today's date
	dateStr := time.Now().Format("2006-01-02")
	dirPath := filepath.Join("memory", dateStr)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create feature manifest file
	fileName := fmt.Sprintf("%s_feature_manifest.md", featureName)
	filePath := filepath.Join(dirPath, fileName)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to create feature manifest: %w", err)
	}

	return filePath, nil
}

// updateMemoryIndex updates the main memory index file with a new entry
func updateMemoryIndex(dateStr, summary, link, featureName string) error {
	indexPath := "memory_index.md"

	// Read existing content
	content, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read memory index: %w", err)
	}

	// Prepare new entry
	newEntry := fmt.Sprintf(
		"\n## %s\n- **特性摘要**: %s\n- **链接**: [%s](%s)\n",
		dateStr, summary, featureName, link,
	)

	// If file is empty, add header
	if len(content) == 0 {
		newEntry = "# 内存索引" + newEntry
	}

	// Append new entry
	newContent := string(content) + newEntry

	// Write back to file
	return os.WriteFile(indexPath, []byte(newContent), 0644)
}
