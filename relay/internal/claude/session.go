package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const sessionDir = ".claude/sessions"

// ClaudeSession Claude Code 的完整 session 文件结构
type ClaudeSession struct {
	ID        string          `json:"id"`
	Title     string          `json:"title,omitempty"`
	Project   string          `json:"project,omitempty"`
	Messages  []SessionMessage `json:"messages,omitempty"`
	CreatedAt time.Time       `json:"created_at,omitempty"`
	UpdatedAt time.Time       `json:"updated_at,omitempty"`
	Archived  bool            `json:"archived,omitempty"`
}

// SessionMessage 一条对话消息
type SessionMessage struct {
	ID        string    `json:"id,omitempty"`
	Role      string    `json:"role"` // "user" | "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// SessionSummary 手机端显示的会话摘要
type SessionSummary struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Project       string    `json:"project"`
	ProjectPath   string    `json:"project_path,omitempty"`
	MessageCount  int       `json:"message_count"`
	LastMessageAt time.Time `json:"last_message_at"`
	Archived      bool      `json:"archived"`
}

// SessionManager 会话管理器
type SessionManager struct {
	sessionsDir string
}

func NewSessionManager(claudeConfigDir string) *SessionManager {
	return &SessionManager{
		sessionsDir: filepath.Join(claudeConfigDir, sessionDir),
	}
}

// ListSessions 列出所有会话摘要
// archived: true=仅归档, false=仅活跃, 忽略=全部
func (sm *SessionManager) ListSessions(archived *bool, project string) ([]SessionSummary, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionSummary{}, nil
		}
		return nil, fmt.Errorf("读取 session 目录失败: %w", err)
	}

	var sessions []SessionSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		session, err := sm.readSessionFile(filepath.Join(sm.sessionsDir, entry.Name()))
		if err != nil {
			continue // 跳过损坏文件
		}

		if archived != nil && session.Archived != *archived {
			continue
		}
		if project != "" && session.Project != project {
			continue
		}

		sessions = append(sessions, session)
	}

	// 按最后消息时间倒序
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastMessageAt.After(sessions[j].LastMessageAt)
	})

	return sessions, nil
}

// GetSession 获取完整会话（含消息历史）
func (sm *SessionManager) GetSession(id string) (*ClaudeSession, error) {
	path := filepath.Join(sm.sessionsDir, id+".json")
	return sm.readSession(path)
}

// GetSessionByFile 通过文件名获取
func (sm *SessionManager) GetSessionByFile(filename string) (*ClaudeSession, error) {
	path := filepath.Join(sm.sessionsDir, filename)
	return sm.readSession(path)
}

func (sm *SessionManager) readSession(path string) (*ClaudeSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 session 文件失败: %w", err)
	}

	var session ClaudeSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("解析 session 文件失败: %w", err)
	}

	// 保证有 ID
	if session.ID == "" {
		session.ID = strings.TrimSuffix(filepath.Base(path), ".json")
	}

	return &session, nil
}

func (sm *SessionManager) readSessionFile(path string) (SessionSummary, error) {
	session, err := sm.readSession(path)
	if err != nil {
		return SessionSummary{}, err
	}

	lastTime := session.UpdatedAt
	count := len(session.Messages)

	// 从消息记录中获取更准确的时间
	if count > 0 {
		last := session.Messages[count-1]
		if !last.CreatedAt.IsZero() {
			lastTime = last.CreatedAt
		}
	}

	title := session.Title
	if title == "" {
		title = sm.generateTitle(session)
	}

	return SessionSummary{
		ID:            session.ID,
		Title:         title,
		Project:       session.Project,
		MessageCount:  count,
		LastMessageAt: lastTime,
		Archived:      session.Archived,
	}, nil
}

func (sm *SessionManager) generateTitle(s *ClaudeSession) string {
	if len(s.Messages) == 0 {
		return "新会话"
	}
	// 取用户第一条消息的前 40 字作为标题
	for _, msg := range s.Messages {
		if msg.Role == "user" {
			text := strings.TrimSpace(msg.Content)
			if len(text) > 40 {
				text = text[:40] + "..."
			}
			if text == "" {
				return "未命名会话"
			}
			return text
		}
	}
	return "未命名会话"
}

// DeleteSession 删除会话文件
func (sm *SessionManager) DeleteSession(id string) error {
	path := filepath.Join(sm.sessionsDir, id+".json")
	if err := os.Remove(path); os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}

// ResolveSessionPath 解析 session id 到完整文件路径
func (sm *SessionManager) ResolveSessionPath(id string) string {
	return filepath.Join(sm.sessionsDir, id+".json")
}

// NewSession 创建新 session 文件（用于 start_session）
func (sm *SessionManager) NewSession(projectName string) *ClaudeSession {
	return &ClaudeSession{
		ID:        uuid.New().String(),
		Project:   projectName,
		Messages:  []SessionMessage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// SaveSession 保存 session 到磁盘
func (sm *SessionManager) SaveSession(s *ClaudeSession) error {
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(sm.sessionsDir, s.ID+".json")
	return os.WriteFile(path, data, 0600)
}
