package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/vincentchyu/sonic-lens/internal/logic/genre"
	"github.com/vincentchyu/sonic-lens/internal/model"
)

func TestGenreCache_Logic(t *testing.T) {
	// 1. 准备 Mock 数据
	mockGenres := []*model.Genre{
		{Name: "Pop", NameZh: "流行"},
		{Name: "Rock", NameZh: "摇滚"},
		{Name: "Jazz", NameZh: "爵士"},
		{Name: "Hip-Hop", NameZh: "嘻哈"},
		{Name: "Electronic", NameZh: "电子"},
		{Name: "Classical", NameZh: "古典"},
		{Name: "Alternative", NameZh: "另类"},
		{Name: "R&B", NameZh: "节奏蓝调"},
		{Name: "Indie Pop", NameZh: "独立流行"},
		{Name: "Synth-pop", NameZh: "合成器流行"},
		{Name: "Progressive Rock", NameZh: "前卫摇滚"},
	}

	// 2. 初始化 Mock Service
	mockService := new(genre.MockGenreService)
	mockService.On("GetGenreCount", mock.Anything).Return(int64(len(mockGenres)), nil)
	mockService.On("GetAllGenres", mock.Anything, len(mockGenres), 0).Return(mockGenres, nil)

	// 3. 创建 Cache 实例并注入 Mock Service
	gc := NewGenreCache()
	gc.genreService = mockService

	ctx := context.Background()

	t.Run(
		"RefreshFromDB", func(t *testing.T) {
			err := gc.RefreshFromDB(ctx)
			assert.NoError(t, err)

			// 验证 C2E
			eng, ok := gc.GetC2E("流行")
			assert.True(t, ok)
			assert.Equal(t, "Pop", eng)

			eng, ok = gc.GetC2E("前卫摇滚")
			assert.True(t, ok)
			assert.Equal(t, "Progressive Rock", eng)

			// 验证 E2C
			chi, ok := gc.GetE2C("Jazz")
			assert.True(t, ok)
			assert.Equal(t, "爵士", chi)

			chi, ok = gc.GetE2C("Hip-Hop")
			assert.True(t, ok)
			assert.Equal(t, "嘻哈", chi)
		},
	)

	t.Run(
		"SetAll", func(t *testing.T) {
			c2e := map[string]string{"金属": "Metal"}
			e2c := map[string]string{"Metal": "金属"}
			gc.SetAll(c2e, e2c)

			eng, ok := gc.GetC2E("金属")
			assert.True(t, ok)
			assert.Equal(t, "Metal", eng)

			chi, ok := gc.GetE2C("Metal")
			assert.True(t, ok)
			assert.Equal(t, "金属", chi)

			// 验证旧数据已被清除（SetAll 会重新创建 map）
			_, ok = gc.GetC2E("流行")
			assert.False(t, ok)

			assert.WithinDuration(t, time.Now(), gc.GetLastUpdate(), 1*time.Second)
		},
	)

	t.Run(
		"GetAll", func(t *testing.T) {
			c2e := map[string]string{"A": "1", "B": "2"}
			gc.SetAll(c2e, map[string]string{})

			all := gc.GetAll()
			assert.Equal(t, 2, len(all))
			assert.Equal(t, "1", all["A"])
			assert.Equal(t, "2", all["B"])
		},
	)
}

func TestGetEnglishGenre_Integration(t *testing.T) {
	// 备份并重置全局 Cache，注入 Mock
	oldCache := globalGenreCache
	defer func() { globalGenreCache = oldCache }()

	mockGenres := []*model.Genre{
		{Name: "Pop", NameZh: "流行"},
		{Name: "Rock", NameZh: "摇滚"},
		{Name: "K-Pop", NameZh: "韩语流行"},
		{Name: "Progressive Rock", NameZh: "前卫摇滚"},
	}

	mockService := new(genre.MockGenreService)
	mockService.On("GetGenreCount", mock.Anything).Return(int64(len(mockGenres)), nil)
	mockService.On("GetAllGenres", mock.Anything, len(mockGenres), 0).Return(mockGenres, nil)

	globalGenreCache = NewGenreCache()
	globalGenreCache.genreService = mockService
	_ = globalGenreCache.RefreshFromDB(context.Background())

	tests := []struct {
		input    string
		expected string
	}{
		{"流行", "Pop"},
		{"流行乐", "Pop"}, // NormalizeChineseGenre 应该去掉 "乐"
		{"摇滚", "Rock"},
		{"前卫摇滚", "Progressive Rock"},
		{"Pop", "Pop"},     // 英文命中了 E2C 缓存
		{"Metal", "Metal"}, // 没命中缓存，返回原始值
		{"Unknown", "Unknown"},
		{"韩语流行乐", "K-Pop"}, // NormalizeChineseGenre 去掉 "乐" -> 韩语流行 -> 命中
	}

	for _, tt := range tests {
		t.Run(
			tt.input, func(t *testing.T) {
				actual := GetEnglishGenre(tt.input)
				assert.Equal(t, tt.expected, actual)
			},
		)
	}

	// 测试小写/规范流派动态缓存解析
	canonical, ok := globalGenreCache.ResolveCanonicalGenre("progressive rock")
	assert.True(t, ok)
	assert.Equal(t, "Progressive Rock", canonical)

	// 测试从 model 调用的 NormalizeGenre 挂载解耦
	assert.Equal(t, "Progressive Rock", model.NormalizeGenre(nil, "progressive rock"))
	assert.Equal(t, "Pop", model.NormalizeGenre(nil, "pop"))
}
