package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mys/relay/internal/relay"
)

// HealthHandler 健康检查
func HealthHandler(r *relay.Relay) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"sessions": len(r.ActiveSessions()),
		})
	}
}

// ListProjectsHandler 项目列表
func ListProjectsHandler(r *relay.Relay) gin.HandlerFunc {
	return func(c *gin.Context) {
		projects := r.ListProjects()
		c.JSON(http.StatusOK, gin.H{
			"projects": projects,
		})
	}
}

// ListSessionsHandler 会话列表
// Query params:
//   archived: "true"=仅归档, "false"=仅活跃, 默认全返回
//   project: 按项目名过滤
func ListSessionsHandler(r *relay.Relay) gin.HandlerFunc {
	return func(c *gin.Context) {
		var archived *bool
		if val := c.Query("archived"); val != "" {
			b, err := strconv.ParseBool(val)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid archived"})
				return
			}
			archived = &b
		}

		project := c.Query("project")

		sessions, err := r.ListSessions(archived, project)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sessions": sessions,
		})
	}
}

// GetSessionHandler 会话详情
func GetSessionHandler(r *relay.Relay) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		session, err := r.GetSession(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"session": session,
		})
	}
}

// InterruptHandler 中断会话
func InterruptHandler(r *relay.Relay) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := r.InterruptSession(id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// DeleteSessionHandler 删除会话
func DeleteSessionHandler(r *relay.Relay) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := r.DeleteSession(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
