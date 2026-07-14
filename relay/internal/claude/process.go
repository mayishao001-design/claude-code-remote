package claude

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/creack/pty"
	"github.com/mys/relay/internal/config"
)

// ClaudeProcess 管理单个 Claude Code 子进程
type ClaudeProcess struct {
	project  *config.Project
	cmd      *exec.Cmd
	ptmx     *os.File
	writer   io.WriteCloser
	stdin    io.Writer

	mu        sync.Mutex
	running   bool
	startedAt time.Time

	onOutput func(text string)
	onDone   func()
	onError  func(err error)

	doneCh chan struct{}
}

// NewClaudeProcess 创建新的 Claude 进程管理器
func NewClaudeProcess(project *config.Project) *ClaudeProcess {
	return &ClaudeProcess{
		project: project,
		doneCh:  make(chan struct{}),
	}
}

// Start 启动 claude CLI 子进程
func (cp *ClaudeProcess) Start() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.running {
		return nil
	}

	// 查找 claude 可执行文件
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return &ClaudeNotFoundError{Err: err}
	}

	cp.cmd = exec.Command(claudePath)
	cp.cmd.Dir = cp.project.Path

	// 启动 PTY
	ptmx, err := pty.Start(cp.cmd)
	if err != nil {
		return &PtyError{Err: err}
	}
	cp.ptmx = ptmx
	cp.stdin = ptmx

	cp.running = true
	cp.startedAt = time.Now()

	// 读取 stdout
	go cp.readOutput()

	// 等待进程退出
	go cp.waitExit()

	return nil
}

// Write 写入消息到 Claude stdin
func (cp *ClaudeProcess) Write(text string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.running {
		return ErrProcessNotRunning
	}

	_, err := cp.stdin.Write([]byte(text + "\n"))
	return err
}

// Interrupt 中断当前生成（SIGINT）
func (cp *ClaudeProcess) Interrupt() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.running || cp.cmd == nil || cp.cmd.Process == nil {
		return ErrProcessNotRunning
	}

	// 先尝试 SIGINT
	return cp.cmd.Process.Signal(os.Interrupt)
}

// Stop 强制停止进程
func (cp *ClaudeProcess) Stop() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.running {
		return nil
	}

	cp.running = false

	if cp.ptmx != nil {
		cp.ptmx.Close()
	}
	if cp.cmd != nil && cp.cmd.Process != nil {
		cp.cmd.Process.Kill()
	}

	cp.onDone = nil
	cp.onError = nil
	cp.onOutput = nil

	return nil
}

// IsRunning 检查进程是否存活
func (cp *ClaudeProcess) IsRunning() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.running
}

// SetCallbacks 设置输出回调
func (cp *ClaudeProcess) SetCallbacks(onOutput func(string), onDone func(), onError func(error)) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onOutput = onOutput
	cp.onDone = onDone
	cp.onError = onError
}

// WaitDone 等待进程结束
func (cp *ClaudeProcess) WaitDone() {
	<-cp.doneCh
}

// readOutput 持续读取 PTY 输出
func (cp *ClaudeProcess) readOutput() {
	reader := bufio.NewReader(cp.ptmx)
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				cp.handleError(err)
			}
			return
		}

		if n == 0 {
			continue
		}

		text := string(buf[:n])
		// 剥离 ANSI 转义码
		clean := stripansi.Strip(text)

		cp.mu.Lock()
		cb := cp.onOutput
		cp.mu.Unlock()

		if cb != nil && clean != "" {
			cb(clean)
		}
	}
}

// waitExit 等待子进程退出
func (cp *ClaudeProcess) waitExit() {
	err := cp.cmd.Wait()
	cp.mu.Lock()
	cp.running = false
	cbDone := cp.onDone
	cbErr := cp.onError
	cp.mu.Unlock()

	close(cp.doneCh)

	if err != nil {
		if cbErr != nil {
			cbErr(err)
		}
	} else {
		if cbDone != nil {
			cbDone()
		}
	}
}

func (cp *ClaudeProcess) handleError(err error) {
	cp.mu.Lock()
	cb := cp.onError
	cp.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}

// ClaudeProcessWatchDog 看门狗：监控子进程健康
type ClaudeProcessWatchDog struct {
	processes map[string]*ClaudeProcess
	mu        sync.RWMutex
	interval  time.Duration
}

func NewWatchDog(interval time.Duration) *ClaudeProcessWatchDog {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &ClaudeProcessWatchDog{
		processes: make(map[string]*ClaudeProcess),
		interval:  interval,
	}
}

func (wd *ClaudeProcessWatchDog) Register(id string, p *ClaudeProcess) {
	wd.mu.Lock()
	wd.processes[id] = p
	wd.mu.Unlock()
}

func (wd *ClaudeProcessWatchDog) Unregister(id string) {
	wd.mu.Lock()
	delete(wd.processes, id)
	wd.mu.Unlock()
}

func (wd *ClaudeProcessWatchDog) Start() {
	go func() {
		for {
			time.Sleep(wd.interval)
			wd.checkAll()
		}
	}()
}

func (wd *ClaudeProcessWatchDog) checkAll() {
	wd.mu.RLock()
	defer wd.mu.RUnlock()

	for id, p := range wd.processes {
		if !p.IsRunning() {
			log.Printf("[WATCHDOG] 进程 %s 已停止", id)
		}
	}
}

// Errors
type ClaudeNotFoundError struct {
	Err error
}

func (e *ClaudeNotFoundError) Error() string {
	return "未找到 claude 命令: " + e.Err.Error()
}

type PtyError struct {
	Err error
}

func (e *PtyError) Error() string {
	return "PTY 创建失败: " + e.Err.Error()
}

var ErrProcessNotRunning = &ProcessNotRunningError{}

type ProcessNotRunningError struct{}

func (e *ProcessNotRunningError) Error() string {
	return "Claude 进程未运行"
}
