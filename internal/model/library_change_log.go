package model

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/core/telemetry"
	"github.com/vincentchyu/sonic-lens/core/websocket"
)

const (
	LibraryEntityAlbum = "album"
	LibraryEntityTrack = "track"

	LibraryOpUpsert = "upsert"
	LibraryOpDelete = "delete"
)

// LibraryChangeLog 记录资料库同步所需的增删改事件
type LibraryChangeLog struct {
	ID         int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	EntityType string    `gorm:"column:entity_type;type:varchar(32);not null;index:idx_library_change_entity" json:"entity_type"`
	EntityID   int64     `gorm:"column:entity_id;type:bigint;not null;index:idx_library_change_entity" json:"entity_id"`
	Operation  string    `gorm:"column:operation;type:varchar(16);not null" json:"operation"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (LibraryChangeLog) TableName() string {
	return "library_change_log"
}

func appendLibraryChangeTx(tx *gorm.DB, entityType string, entityID int64, operation string) error {
	if tx == nil || entityID <= 0 {
		return nil
	}

	entry := &LibraryChangeLog{
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  operation,
	}
	if err := tx.Session(&gorm.Session{NewDB: true, SkipHooks: true}).Create(entry).Error; err != nil {
		return err
	}

	asyncCtx := context.Background()
	if tx.Statement != nil && tx.Statement.Context != nil {
		asyncCtx = tx.Statement.Context
	}
	telemetry.GoSafeDetached(
		asyncCtx, "model.library_change_log.broadcast_update", func(goCtx context.Context) {
			websocket.BroadcastLibraryUpdate(goCtx, entityType, entityID, operation, entry.ID)
		},
	)
	return nil
}

// AppendLibraryChange 追加一条资料库变更记录，供非事务场景复用。
func AppendLibraryChange(ctx context.Context, entityType string, entityID int64, operation string) error {
	return appendLibraryChangeTx(GetDB().WithContext(ctx), entityType, entityID, operation)
}

// AppendAlbumLibraryUpsert 追加专辑索引 upsert 变更，驱动 Bridge 端专辑列表增量刷新。
func AppendAlbumLibraryUpsert(ctx context.Context, albumID int64) error {
	return AppendLibraryChange(ctx, LibraryEntityAlbum, albumID, LibraryOpUpsert)
}

// GetLibraryChangesSince 获取指定版本之后的资料库变更
func GetLibraryChangesSince(ctx context.Context, sinceVersion int64) ([]*LibraryChangeLog, error) {
	var rows []*LibraryChangeLog
	query := GetDB().WithContext(ctx).Model(&LibraryChangeLog{})
	if sinceVersion > 0 {
		query = query.Where("id > ?", sinceVersion)
	}
	err := query.Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetLatestLibraryChangeVersion 返回当前最新资料库变更版本
func GetLatestLibraryChangeVersion(ctx context.Context) (int64, error) {
	var version int64
	err := GetDB().WithContext(ctx).Model(&LibraryChangeLog{}).Select("COALESCE(MAX(id), 0)").Scan(&version).Error
	if err != nil {
		return 0, err
	}
	return version, nil
}
