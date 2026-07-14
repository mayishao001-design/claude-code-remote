package relay

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mys/relay/internal/claude"
	"github.com/mys/relay/internal/config"
)

// SessionState 手机端可见的会话运行时状态
type SessionState struct {
	SessionID string `json:"session_id"`
	Project   string `json:"project"`
	Running   bool   `json:"running"`
}

// EventCallback 事件回调
type EventCallback interface {
	OnStreamChunk(sessionID, text string)
	OnStreamEnd(sessionID string)
	OnStreamError(sessionID string, err error)
	OnSessionUpdated(summary claude.SessionSummary)
}

// Relay 核心调度器
type Relay struct {
	cfg          *config.Config
	sessionMgr   *claude.SessionManager
	streamBufMgr *claude.StreamBufferManager
	processes    map[string]*claude.ClaudeProcess
	watcher      *claude.ClaudeProcessWatchDog
	callbacks    []EventCallback
	mu           sync.RWMutex
}

func New(cfg *config.Config) (*Relay, error) {
	homeDir, err := getClaudeConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取 Claude 配置目录失败: %w", err)
	}

	r := &Relay{
		cfg:          cfg,
		sessionMgr:   claude.NewSessionManager(homeDir),
		streamBufMgr: claude.NewStreamBufferManager(),
		processes:    make(map[string]*claude.ClaudeProcess),
		callbacks:    []EventCallback{},
	}

	r.watcher = claude.NewWatchDog(30 * time.Second)
	r.watcher.Start()
	return r, nil
}

func (r *Relay) RegisterCallback(cb EventCallback) {
	r.mu.Lock()
	r.callbacks = append(r.callbacks, cb)
	r.mu.Unlock()
}

func (r *Relay) RemoveCallback(cb EventCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.callbacks {
		if c == cb {
			r.callbacks = append(r.callbacks[:i], r.callbacks[i+1:]...)
			return
		}
	}
}

// ListSessions 列出所有会话
func (r *Relay) ListSessions(archived *bool, project string) ([]claude.SessionSummary, error) {
	sessions, err := r.sessionMgr.ListSessions(archived, project)
	if err != nil {
		return nil, err
	}

	for i := range sessions {
		if proj := r.cfg.ProjectByPath(sessions[i].Project); proj != nil {
			sessions[i].Project = proj.Name
			sessions[i].ProjectPath = proj.Path
		}
	}

	return sessions, nil
}

func (r *Relay) GetSession(id string) (*claude.ClaudeSession, error) {
	return r.sessionMgr.GetSession(id)
}

func (r *Relay) DeleteSession(id string) error {
	r.mu.Lock()
	if p, ok := r.processes[id]; ok {
		p.Stop()
		delete(r.processes, id)
	}
	r.mu.Unlock()
	r.streamBufMgr.Delete(id)
	return r.sessionMgr.DeleteSession(id)
}

func (r *Relay) GetSessionState(id string) *SessionState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state := &SessionState{SessionID: id}
	if p, ok := r.processes[id]; ok {
		state.Running = p.IsRunning()
	}
	if session, err := r.sessionMgr.GetSession(id); err == nil {
		state.Project = session.Project
	}
	return state
}

func (r *Relay) ActiveSessions() []SessionState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var states []SessionState
	for id, p := range r.processes {
		states = append(states, SessionState{
			SessionID: id,
			Running:   p.IsRunning(),
		})
	}
	return states
}

// SendMessage 向指定会话发送消息
func (r *Relay) SendMessage(sessionID, text string) error {
	r.mu.RLock()
	p, ok := r.processes[sessionID]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("会话 %s 没有运行中的进程", sessionID)
	}

	// 记录用户消息到 session 文件
	if session, err := r.sessionMgr.GetSession(sessionID); err == nil {
		session.Messages = append(session.Messages, claude.SessionMessage{
			Role:    "user",
			Content: text,
		})
		r.sessionMgr.SaveSession(session)
	}

	return p.Write(text)
}

// StartSession 启动新会话
func (r *Relay) StartSession(projectName, initialPrompt string) (string, error) {
	var project *config.Project
	for i := range r.cfg.Projects {
		if r.cfg.Projects[i].Name == projectName {
			project = &r.cfg.Projects[i]
			break
		}
	}
	if project == nil {
		return "", fmt.Errorf("未找到项目: %s", projectName)
	}

	// 校验项目路径
	if err := project.Validate(); err != nil {
		return "", fmt.Errorf("项目不可用: %w", err)
	}

	session := r.sessionMgr.NewSession(projectName)
	sessionID := session.ID

	if initialPrompt != "" {
		session.Messages = append(session.Messages, claude.SessionMessage{
			Role:    "user",
			Content: initialPrompt,
		})
	}
	r.sessionMgr.SaveSession(session)

	proc := claude.NewClaudeProcess(project)

	// 初始化流缓冲区
	buf := r.streamBufMgr.GetOrCreate(sessionID)

	proc.SetCallbacks(
		func(text string) {
			buf.Append(text)
			r.broadcastStreamChunk(sessionID, text)
		},
		func() {
			r.handleProcessDone(sessionID)
			r.broadcastStreamEnd(sessionID)
		},
		func(err error) {
			r.broadcastStreamError(sessionID, err)
		},
	)

	if err := proc.Start(); err != nil {
		return "", fmt.Errorf("启动 Claude 进程失败: %w", err)
	}

	r.mu.Lock()
	r.processes[sessionID] = proc
	r.mu.Unlock()

	if initialPrompt != "" {
		if err := proc.Write(initialPrompt); err != nil {
			return sessionID, err
		}
	}

	r.broadcastSessionUpdated(sessionID)
	return sessionID, nil
}

// InterruptSession 中断会话
func (r *Relay) InterruptSession(sessionID string) error {
	r.mu.RLock()
	p, ok := r.processes[sessionID]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("会话 %s 没有运行中的进程", sessionID)
	}

	log.Printf("[INTERRUPT] 中断会话 %s", sessionID)
	return p.Interrupt()
}

// handleProcessDone 进程结束后保存 assistant 回复
func (r *Relay) handleProcessDone(sessionID string) {
	r.mu.Lock()
	delete(r.processes, sessionID)
	r.mu.Unlock()

	session, err := r.sessionMgr.GetSession(sessionID)
	if err != nil {
		return
	}

	// 从缓冲区获取 assistant 完整回复
	buf := r.streamBufMgr.Get(sessionID)
	if buf != nil {
		fullText := buf.FullText()
		if fullText != "" {
			session.Messages = append(session.Messages, claude.SessionMessage{
				Role:    "assistant",
				Content: fullText,
			})
		}
		buf.Reset()
	}

	r.sessionMgr.SaveSession(session)
}

func (r *Relay) broadcastStreamChunk(sessionID, text string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cb := range r.callbacks {
		cb.OnStreamChunk(sessionID, text)
	}
}

func (r *Relay) broadcastStreamEnd(sessionID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cb := range r.callbacks {
		cb.OnStreamEnd(sessionID)
	}
}

func (r *Relay) broadcastStreamError(sessionID string, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cb := range r.callbacks {
		cb.OnStreamError(sessionID, err)
	}
}

func (r *Relay) broadcastSessionUpdated(sessionID string) {
	session, err := r.sessionMgr.GetSession(sessionID)
	if err != nil {
		return
	}

	// 构造摘要
	summary := claude.SessionSummary{
		ID:           session.ID,
		Title:        session.Title,
		Project:      session.Project,
		MessageCount: len(session.Messages),
		Archived:     session.Archived,
	}
	if !session.UpdatedAt.IsZero() {
		summary.LastMessageAt = session.UpdatedAt
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cb := range r.callbacks {
		cb.OnSessionUpdated(summary)
	}
}

func (r *Relay) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Printf("正在关闭 %d 个活跃进程...", len(r.processes))
	for id, p := range r.processes {
		log.Printf("  关闭会话 %s", id)
		p.Stop()
	}
}

func (r *Relay) ListProjects() []config.Project {
	return r.cfg.Projects
}

func getClaudeConfigDir() (string, error) {
	return config.GetHomeDir()
}
