// Relay Daemon — 手机遥控 Claude Code 的中转服务
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mys/relay/internal/api"
	"github.com/mys/relay/internal/config"
	"github.com/mys/relay/internal/relay"
	"log"
)

func main() {
	port := flag.Int("port", 9943, "监听端口")
	configDir := flag.String("config", "", "配置目录（默认 ~/.claude-remote）")
	flag.Parse()

	cfg, err := config.Load(*configDir)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 首次启动显示 token
	if cfg.IsFirstRun() {
		fmt.Println("┌──────────────────────────────────────────────┐")
		fmt.Println("│  Claude Code Remote Relay — 首次启动          │")
		fmt.Println("├──────────────────────────────────────────────┤")
		fmt.Printf("│  Token: %-42s│\n", cfg.AuthToken)
		fmt.Println("│  请在手机端输入此 Token                       │")
		fmt.Println("│  编辑 projects.json 添加项目路径             │")
		fmt.Println("└──────────────────────────────────────────────┘")
	}

	relayCore, err := relay.New(cfg)
	if err != nil {
		log.Fatalf("初始化 Relay 失败: %v", err)
	}

	router := api.NewRouter(cfg, relayCore)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("Relay Daemon 启动于 %s", addr)
		log.Printf("Tailscale IP 查看: tailscale ip -4")
		log.Printf("手机端连接: http://<TAILSCALE_IP>:%d", *port)
		if err := router.Run(addr); err != nil {
			log.Fatalf("HTTP 服务失败: %v", err)
		}
	}()

	<-quit
	log.Println("正在关闭...")
	relayCore.Shutdown()
}
