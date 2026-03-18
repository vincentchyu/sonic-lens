package model

import (
	"context"

	"gorm.io/gorm"
)

// InTx 在 model 层统一开启事务，避免上层直接持有裸 GORM 细节。
func InTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return GetDB().WithContext(ctx).Transaction(fn)
}
