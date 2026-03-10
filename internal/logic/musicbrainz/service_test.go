package musicbrainz

import (
	"context"
	"testing"
)

func TestInitializeAlbums(t *testing.T) {
	// 这是一个集成测试，需要数据库连接
	// 在 sonic-lens 中，通常在 TestMain 或测试函数开头初始化 DB
	// 这里我们假设 DB 已经初始化，或者手动初始化一个内存 SQLite
	
	ctx := context.Background()
	// 如果需要手动初始化 DB，可以调用 model.InitDB(":memory:", nil)
	// 但通常项目中已经有现成的方法处理测试 DB

	err := InitializeAlbums(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize albums: %v", err)
	}
	t.Log("Successfully initialized albums")
}
