package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Project 预配项目
type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Config 总配置
type Config struct {
	configDir string
	authFile  string
	projectsFile string

	AuthToken  string    `json:"auth_token"`
	Projects   []Project `json:"projects"`
}

var defaultConfigDir string

func init() {
	home, err := os.UserHomeDir()
	if err == nil {
		defaultConfigDir = filepath.Join(home, ".claude-remote")
	}
}

// Load 加载配置，首次启动自动生成
func Load(configDir string) (*Config, error) {
	if configDir == "" {
		configDir = defaultConfigDir
	}
	if configDir == "" {
		return nil, fmt.Errorf("无法确定配置目录，请用 -config 指定")
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	cfg := &Config{
		configDir:    configDir,
		authFile:     filepath.Join(configDir, "auth.json"),
		projectsFile: filepath.Join(configDir, "projects.json"),
	}

	// 加载 auth
	if err := cfg.loadAuth(); err != nil {
		return nil, err
	}

	// 加载 projects
	if err := cfg.loadProjects(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) loadAuth() error {
	if _, err := os.Stat(c.authFile); os.IsNotExist(err) {
		// 首次：生成随机 token
		token := randomHex(32)
		c.AuthToken = token
		return c.saveAuth()
	}

	data, err := os.ReadFile(c.authFile)
	if err != nil {
		return fmt.Errorf("读取 auth 失败: %w", err)
	}

	var auth struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return fmt.Errorf("解析 auth 失败: %w", err)
	}
	c.AuthToken = auth.Token
	return nil
}

func (c *Config) saveAuth() error {
	data, err := json.MarshalIndent(map[string]string{
		"token": c.AuthToken,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.authFile, data, 0600)
}

func (c *Config) loadProjects() error {
	if _, err := os.Stat(c.projectsFile); os.IsNotExist(err) {
		c.Projects = []Project{}
		return c.saveProjects()
	}

	data, err := os.ReadFile(c.projectsFile)
	if err != nil {
		return fmt.Errorf("读取 projects 失败: %w", err)
	}

	if err := json.Unmarshal(data, &c.Projects); err != nil {
		return fmt.Errorf("解析 projects 失败: %w", err)
	}
	return nil
}

func (c *Config) saveProjects() error {
	data, err := json.MarshalIndent(c.Projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.projectsFile, data, 0600)
}

func (c *Config) IsFirstRun() bool {
	authPath := filepath.Join(c.configDir, "auth.json")
	_, err := os.Stat(authPath)
	return os.IsNotExist(err)
}

func (c *Config) ProjectByPath(path string) *Project {
	for i := range c.Projects {
		if c.Projects[i].Path == path {
			return &c.Projects[i]
		}
	}
	return nil
}

func (c *Config) GetProjectsFile() string { return c.projectsFile }
func (c *Config) GetConfigDir() string    { return c.configDir }

// ValidateProject 验证项目路径是否存在
func (p *Project) Validate() error {
	info, err := os.Stat(p.Path)
	if err != nil {
		return fmt.Errorf("项目路径不可访问: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("项目路径不是目录: %s", p.Path)
	}
	return nil
}

// GetHomeDir 返回用户主目录
func GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("生成随机 token 失败: %v", err)
	}
	return hex.EncodeToString(bytes)
}
