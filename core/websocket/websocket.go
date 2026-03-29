package websocket

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// WebSocket连接池
var (
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex = sync.RWMutex{}

	libraryUpdateBatch = newLibraryUpdateBatcher(2*time.Minute, broadcastLibraryUpdateNow)
)

// WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// HandleWebSocketMessages  处理WebSocket消息
func HandleWebSocketMessages(conn *websocket.Conn) {
	defer func() {
		// 从连接池中移除连接
		RemoveClient(conn)
	}()

	for {
		// 读取消息
		_, _, err := conn.ReadMessage()
		if err != nil {
			// 连接已关闭
			break
		}
	}
}

type WsInfo struct {
	Type   string      `json:"type"`
	Source string      `json:"source"`
	Data   WsTrackData `json:"data"`
}

type WsLibraryUpdate struct {
	Type    string `json:"type"`
	Version int64  `json:"version"`
}

type libraryUpdateEvent struct {
	EntityType string
	EntityID   int64
	Operation  string
	Version    int64
}

type libraryUpdateBatcher struct {
	mu            sync.Mutex
	flushInterval time.Duration
	pending       map[string]libraryUpdateEvent
	timer         *time.Timer
	emit          func(ctx context.Context, version int64)
}

type WsTrackData struct {
	Title           string                    `json:"title"`
	Album           string                    `json:"album"`
	Artist          string                    `json:"artist"`
	AppleMusic      bool                      `json:"apple_music"`
	LastFM          bool                      `json:"lastfm"`
	AppleMusicState common.TrackFavoriteState `json:"apple_music_state"`
	LastFMState     common.TrackFavoriteState `json:"lastfm_state"`
	FavoriteState   common.TrackFavoriteState `json:"favorite_state"`
	Duration        int64                     `json:"duration"`      // 歌曲时长，单位秒
	Position        int64                     `json:"position"`      // 歌曲当前播放位置，单位秒
	PositionMs      int64                     `json:"position_ms"`   // 歌曲当前播放位置，单位毫秒
	TrackNumber     int8                      `json:"track_number"`  // 曲目号
	DiscNumber      int8                      `json:"disc_number"`   // 盘号
	CoverArtURL     string                    `json:"cover_art_url"` // 专辑封面访问地址
	CoverArtMime    string                    `json:"cover_art_mime"`

	Confidence        common.TrackMetadataConfidence `json:"confidence"`
	PlayerInfoHandler common.PlayerInfoHandler       `json:"-"`
}

// 向所有连接的客户端广播消息
func BroadcastMessage(ctx context.Context, message *WsInfo) {
	broadcastJSON(ctx, message)
}

func BroadcastLibraryUpdate(ctx context.Context, entityType string, entityID int64, operation string, version int64) {
	libraryUpdateBatch.enqueue(
		ctx, libraryUpdateEvent{
			EntityType: entityType,
			EntityID:   entityID,
			Operation:  operation,
			Version:    version,
		},
	)
}

func broadcastLibraryUpdateNow(ctx context.Context, version int64) {
	broadcastJSON(
		ctx, &WsLibraryUpdate{
			Type:    "library_updated",
			Version: version,
		},
	)
}

func newLibraryUpdateBatcher(
	flushInterval time.Duration, emit func(ctx context.Context, version int64),
) *libraryUpdateBatcher {
	return &libraryUpdateBatcher{
		flushInterval: flushInterval,
		pending:       make(map[string]libraryUpdateEvent),
		emit:          emit,
	}
}

func (b *libraryUpdateBatcher) enqueue(ctx context.Context, event libraryUpdateEvent) {
	if b == nil || event.EntityID <= 0 || event.Version <= 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.pending[b.pendingKey(event.EntityType, event.EntityID)] = event
	if b.timer != nil {
		return
	}

	b.timer = time.AfterFunc(
		b.flushInterval, func() {
			b.flush(context.Background())
		},
	)

	if log.Logger != nil {
		log.Info(
			ctx,
			"Schedule library update batch flush",
			zap.Duration("flush_interval", b.flushInterval),
			zap.Int("pending_count", len(b.pending)),
		)
	}
}

func (b *libraryUpdateBatcher) flush(ctx context.Context) {
	if b == nil {
		return
	}

	b.mu.Lock()
	if len(b.pending) == 0 {
		b.timer = nil
		b.mu.Unlock()
		return
	}

	maxVersion := int64(0)
	upsertCount := 0
	deleteCount := 0
	for _, event := range b.pending {
		if event.Version > maxVersion {
			maxVersion = event.Version
		}
		if event.Operation == "delete" {
			deleteCount++
		} else {
			upsertCount++
		}
	}

	pendingCount := len(b.pending)
	b.pending = make(map[string]libraryUpdateEvent)
	b.timer = nil
	b.mu.Unlock()

	if log.Logger != nil {
		log.Info(
			ctx,
			"Flush library update batch",
			zap.Int("pending_count", pendingCount),
			zap.Int("upsert_count", upsertCount),
			zap.Int("delete_count", deleteCount),
			zap.Int64("version", maxVersion),
		)
	}

	b.emit(ctx, maxVersion)
}

func (b *libraryUpdateBatcher) pendingKey(entityType string, entityID int64) string {
	return fmt.Sprintf("%s:%d", entityType, entityID)
}

func broadcastJSON(ctx context.Context, message interface{}) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	// 将消息序列化为JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Error(ctx, "Failed to marshal message", zap.Error(err))
		return
	}

	// 向所有客户端发送消息
	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Error(ctx, "Failed to send message to client", zap.Error(err))
		}
	}
}

// UpgradeConnection 升级HTTP连接到WebSocket连接
func UpgradeConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

// AddClient 添加客户端到连接池
func AddClient(conn *websocket.Conn) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	clients[conn] = true
}

// RemoveClient 从连接池中移除客户端
func RemoveClient(conn *websocket.Conn) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	delete(clients, conn)
	err := conn.Close()
	if err != nil {
		return
	}
}
