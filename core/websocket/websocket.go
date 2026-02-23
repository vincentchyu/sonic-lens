package websocket

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/log"
)

// WebSocket连接池
var (
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex = sync.RWMutex{}
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

type WsTrackInfo struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Data   struct {
		Title      string `json:"title"`
		Album      string `json:"album"`
		Artist     string `json:"artist"`
		AppleMusic bool   `json:"apple_music"`
		LastFM     bool   `json:"lastfm"`
		Duration   int64  `json:"duration"` // 歌曲时长，单位秒
		Position   int64  `json:"position"` // 歌曲当前播放位置，单位秒
	} `json:"data"`
}

// 向所有连接的客户端广播消息
func BroadcastMessage(ctx context.Context, message *WsTrackInfo) {
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
