package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLibraryUpdateBatcherFlushesLatestVersionOnce(t *testing.T) {
	var versions []int64
	batcher := newLibraryUpdateBatcher(
		time.Minute, func(ctx context.Context, version int64) {
			versions = append(versions, version)
		},
	)

	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "track",
		EntityID:   101,
		Operation:  "upsert",
		Version:    11,
	})
	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "track",
		EntityID:   101,
		Operation:  "upsert",
		Version:    12,
	})
	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "album",
		EntityID:   202,
		Operation:  "delete",
		Version:    15,
	})

	batcher.flush(context.Background())

	require.Equal(t, []int64{15}, versions)
	require.Empty(t, batcher.pending)
	require.Nil(t, batcher.timer)
}

func TestLibraryUpdateBatcherDeduplicatesByEntity(t *testing.T) {
	batcher := newLibraryUpdateBatcher(time.Minute, func(ctx context.Context, version int64) {})

	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "track",
		EntityID:   3022,
		Operation:  "upsert",
		Version:    21,
	})
	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "track",
		EntityID:   3022,
		Operation:  "delete",
		Version:    22,
	})
	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "track",
		EntityID:   3023,
		Operation:  "upsert",
		Version:    23,
	})

	require.Len(t, batcher.pending, 2)
	require.Equal(t, "delete", batcher.pending[batcher.pendingKey("track", 3022)].Operation)
	require.Equal(t, int64(22), batcher.pending[batcher.pendingKey("track", 3022)].Version)
	require.Equal(t, "upsert", batcher.pending[batcher.pendingKey("track", 3023)].Operation)

	if batcher.timer != nil {
		batcher.timer.Stop()
	}
}

func TestLibraryUpdateBatcherSkipsInvalidEvent(t *testing.T) {
	var versions []int64
	batcher := newLibraryUpdateBatcher(
		time.Minute, func(ctx context.Context, version int64) {
			versions = append(versions, version)
		},
	)

	batcher.enqueue(context.Background(), libraryUpdateEvent{
		EntityType: "track",
		EntityID:   0,
		Operation:  "upsert",
		Version:    1,
	})
	batcher.flush(context.Background())

	require.Empty(t, versions)
	require.Empty(t, batcher.pending)
	require.Nil(t, batcher.timer)
}
