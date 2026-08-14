package testutil

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestNewMemoryDB(t *testing.T) {
	db := NewMemoryDB(
		t,
		`CREATE TABLE sample_item (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL
		);`,
	)

	type SampleItem struct {
		ID    int64  `gorm:"primaryKey"`
		Title string `gorm:"column:title"`
	}

	item := &SampleItem{Title: "Dark Side"}
	require.NoError(t, db.Table("sample_item").Create(item).Error)
	assert.Equal(t, int64(1), item.ID)

	var retrieved SampleItem
	require.NoError(t, db.Table("sample_item").First(&retrieved, item.ID).Error)
	assert.Equal(t, "Dark Side", retrieved.Title)
}

func TestNewMockDB(t *testing.T) {
	db, mock := NewMockDB(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `album` WHERE `id` = ? ORDER BY `album`.`id` LIMIT ?")).
		WithArgs(1, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "artist"}).
				AddRow(1, "Kid A", "Radiohead"),
		)

	type MockAlbum struct {
		ID     int64
		Name   string
		Artist string
	}

	var row MockAlbum
	require.NoError(t, db.Table("album").Where("`id` = ?", 1).First(&row).Error)
	assert.Equal(t, "Kid A", row.Name)
	assert.Equal(t, "Radiohead", row.Artist)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetupTestGlobalMySQL(t *testing.T) {
	db, _ := NewMockDB(t)
	SetupTestGlobalMySQL(t, db)

	assert.Same(t, db, model.GetDB())
}
