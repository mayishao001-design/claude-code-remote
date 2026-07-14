package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mys/relay/internal/claude"
	"github.com/mys/relay/internal/relay"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --- Wire Protocol ---

// ClientMessage 手机→Relay
type ClientMessage struct {
	Type      string `json:"type"`       // send_message | start_session | interrupt | ping
	SessionID string `json:"session_id,omitempty"`
	Text      string `json:"text,omitempty"`
	Project   string `json:"project,omitempty"`
}

// ServerMessage Relay→手机
type ServerMessage struct {
	Type      string                 `json:"type"` // stream_chunk | stream_end | stream_error | session_updated | pong
	SessionID string                 `json:"session_id,omitempty"`
	Text      string                 `json:"text,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Session   *claude.SessionSummary `json:"session,omitempty"`
}

// WSCallback WebSocket 特定连接的回调
type WSCallback struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	hub    *WSHub
	userID string
}

// WSHub 管理所有活跃的 WebSocket 连接
type WSHub struct {
	clients map[*WSCallback]bool
	mu      sync.RWMutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*WSCallback]bool),
	}
}

func (h *WSHub) Register(cb *WSCallback) {
	h.mu.Lock()
	h.clients[cb] = true
	h.mu.Unlock()
}

func (h *WSHub) Unregister(cb *WSCallback) {
	h.mu.Lock()
	delete(h.clients, cb)
	h.mu.Unlock()
}

func (h *WSHub) Broadcast(msg ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] 序列化失败: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.Lock()
		err := client.conn.WriteMessage(websocket.TextMessage, data)
		client.mu.Unlock()
		if err != nil {
			log.Printf("[WS] 发送失败: %v", err)
			go h.Unregister(client)
		}
	}
}

// EventCallback 实现 relay.EventCallback 接口
func (cb *WSCallback) OnStreamChunk(sessionID, text string) {
	cb.send(ServerMessage{
		Type:      "stream_chunk",
		SessionID: sessionID,
		Text:      text,
	})
}

func (cb *WSCallback) OnStreamEnd(sessionID string) {
	cb.send(ServerMessage{
		Type:      "stream_end",
		SessionID: sessionID,
	})
}

func (cb *WSCallback) OnStreamError(sessionID string, err error) {
	cb.send(ServerMessage{
		Type:      "stream_error",
		SessionID: sessionID,
		Error:     err.Error(),
	})
}

func (cb *WSCallback) OnSessionUpdated(summary claude.SessionSummary) {
	cb.send(ServerMessage{
		Type:    "session_updated",
		Session: &summary,
	})
}

func (cb *WSCallback) send(msg ServerMessage) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err := cb.conn.WriteJSON(msg); err != nil {
		log.Printf("[WS] 发送失败: %v", err)
	}
}

// WebSocketHandler WebSocket 升级和消息循环
func WebSocketHandler(relayCore *relay.Relay) gin.HandlerFunc {
	hub := NewWSHub()

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] 升级失败: %v", err)
			return
		}

		cb := &WSCallback{
			conn: conn,
			hub:  hub,
		}

		hub.Register(cb)
		relayCore.RegisterCallback(cb)

		log.Printf("[WS] 新连接建立")
		defer func() {
			hub.Unregister(cb)
			relayCore.RemoveCallback(cb)
			conn.Close()
			log.Printf("[WS] 连接关闭")
		}()

		// 消息接收循环
		for {
			var msg ClientMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("[WS] 读取错误: %v", err)
				}
				return
			}

			switch msg.Type {
			case "send_message":
				if msg.SessionID == "" {
					cb.send(ServerMessage{Type: "stream_error", Error: "缺少 session_id"})
					continue
				}
				go func() {
					if err := relayCore.SendMessage(msg.SessionID, msg.Text); err != nil {
						cb.send(ServerMessage{
							Type: "stream_error", SessionID: msg.SessionID, Error: err.Error(),
						})
					}
				}()

			case "start_session":
				go func() {
					sessionID, err := relayCore.StartSession(msg.Project, msg.Text)
					if err != nil {
						cb.send(ServerMessage{Type: "stream_error", Error: err.Error()})
						return
					}
					// 通知手机端 session 已创建
					session, _ := relayCore.GetSession(sessionID)
					if session != nil {
						s := sessionsummary(sessionID, session)
						cb.send(ServerMessage{
							Type:    "session_updated",
							Session: &s,
						})
					}
				}()

			case "interrupt":
				if msg.SessionID == "" {
					cb.send(ServerMessage{Type: "stream_error", Error: "缺少 session_id"})
					continue
				}
				if err := relayCore.InterruptSession(msg.SessionID); err != nil {
					cb.send(ServerMessage{
						Type: "stream_error", SessionID: msg.SessionID, Error: err.Error(),
					})
				}

			case "ping":
				cb.send(ServerMessage{Type: "pong"})

			default:
				cb.send(ServerMessage{Type: "stream_error", Error: "未知消息类型: " + msg.Type})
			}
		}
	}
}

// 辅助：从 ClaudeSession 提取 SessionSummary
func sessionsummary(id string, session *claude.ClaudeSession) claude.SessionSummary {
	title := session.Title
	if title == "" {
		title = "新会话"
	}
	count := len(session.Messages)

	return claude.SessionSummary{
		ID:           id,
		Title:        title,
		Project:      session.Project,
		MessageCount: count,
	}
}
