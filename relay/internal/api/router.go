package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mys/relay/internal/config"
	"github.com/mys/relay/internal/relay"
)

// NewRouter 创建路由
func NewRouter(cfg *config.Config, relayCore *relay.Relay) *gin.Engine {
	r := gin.Default()

	// 全局中间件
	r.Use(corsMiddleware())
	r.Use(authMiddleware(cfg.AuthToken))

	// REST API
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", HealthHandler(relayCore))
		v1.GET("/projects", ListProjectsHandler(relayCore))
		v1.GET("/sessions", ListSessionsHandler(relayCore))
		v1.GET("/sessions/:id", GetSessionHandler(relayCore))
		v1.POST("/sessions/:id/interrupt", InterruptHandler(relayCore))
		v1.DELETE("/sessions/:id", DeleteSessionHandler(relayCore))

		// WebSocket
		v1.GET("/ws", WebSocketHandler(relayCore))
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func authMiddleware(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// WebSocket 升级需要在 query 中带 token
		if c.Request.URL.Path == "/api/v1/ws" {
			if c.Query("token") != token {
				c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
				return
			}
			c.Next()
			return
		}

		// REST 用 Authorization header
		auth := c.GetHeader("Authorization")
		if auth != "Bearer "+token {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
