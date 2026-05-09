package api

import (
	"my_cursor/internal/agent"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatRequest struct {
	Msg       string `json:"msg"`
	SessionID string `json:"session_id"`
}

func ChatHandler(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	tid := PrincipalFrom(c).TenantID
	resp, sessionID, err := agent.Chat(req.SessionID, req.Msg, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "success",
		"session_id": sessionID,
		"data":       resp,
	})
}
