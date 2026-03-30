package model

import "context"

func ensureInsightJobSchema(ctx context.Context) error {
	return GetDB().WithContext(ctx).AutoMigrate(&InsightJob{})
}
