package claude

import (
	"sync"
)

// StreamBuffer 累积流式输出片段，结束时拼成完整消息
type StreamBuffer struct {
	mu        sync.Mutex
	sessionID string
	chunks    []string
}

// NewStreamBuffer 创建流缓冲区
func NewStreamBuffer(sessionID string) *StreamBuffer {
	return &StreamBuffer{
		sessionID: sessionID,
		chunks:    []string{},
	}
}

// Append 追加一块文本
func (sb *StreamBuffer) Append(text string) {
	sb.mu.Lock()
	sb.chunks = append(sb.chunks, text)
	sb.mu.Unlock()
}

// FullText 返回完整累积文本
func (sb *StreamBuffer) FullText() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	var result string
	for _, chunk := range sb.chunks {
		result += chunk
	}
	return result
}

// Reset 清空缓冲区
func (sb *StreamBuffer) Reset() {
	sb.mu.Lock()
	sb.chunks = sb.chunks[:0]
	sb.mu.Unlock()
}

// StreamBufferManager 管理所有活跃的流缓冲区
type StreamBufferManager struct {
	mu      sync.Mutex
	buffers map[string]*StreamBuffer // sessionId → buffer
}

func NewStreamBufferManager() *StreamBufferManager {
	return &StreamBufferManager{
		buffers: make(map[string]*StreamBuffer),
	}
}

func (mgr *StreamBufferManager) GetOrCreate(sessionID string) *StreamBuffer {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if buf, ok := mgr.buffers[sessionID]; ok {
		return buf
	}

	buf := NewStreamBuffer(sessionID)
	mgr.buffers[sessionID] = buf
	return buf
}

func (mgr *StreamBufferManager) Get(sessionID string) *StreamBuffer {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	return mgr.buffers[sessionID]
}

func (mgr *StreamBufferManager) Delete(sessionID string) {
	mgr.mu.Lock()
	delete(mgr.buffers, sessionID)
	mgr.mu.Unlock()
}
