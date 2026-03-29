package d1sync

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestD1ClientSyncSingleFlight(t *testing.T) {
	client := &D1Client{}

	if !client.tryBeginSync() {
		t.Fatal("expected first sync attempt to acquire the lock")
	}
	if client.tryBeginSync() {
		t.Fatal("expected second sync attempt to be rejected while running")
	}

	client.endSync()
	if !client.tryBeginSync() {
		t.Fatal("expected sync lock to be reusable after release")
	}
	client.endSync()
}

func TestSeedSyncMetadataBootstrap(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	client := &D1Client{db: db}

	tableNames := []string{"tracks", "track_play_records", "genres", "dashboard_stats"}
	for _, tableName := range tableNames {
		mock.ExpectExec(regexp.QuoteMeta("INSERT OR IGNORE INTO sync_metadata")).
			WithArgs(tableName, sqlmock.AnyArg(), 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	require.NoError(t, client.seedSyncMetadataBootstrap(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}
