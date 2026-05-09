package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"my_cursor/internal/agent"
	"my_cursor/internal/auth"
	"my_cursor/internal/tenant"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsClientMsg struct {
	Msg       string `json:"msg"`
	SessionID string `json:"session_id"`
}

type wsOutMsg struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Message   string `json:"message,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

func wsPrincipal(c *gin.Context) (auth.Principal, bool) {
	if !auth.Enabled() {
		return auth.DevBootstrapPrincipal(), true
	}
	q := strings.TrimSpace(c.Query("access_token"))
	if q != "" {
		p, err := auth.ParseAccessJWT(q)
		if err == nil {
			return p, Can(p, PermChat)
		}
		return auth.Principal{}, false
	}
	if auth.AllowAnonymousDebug() {
		p := auth.Principal{
			UserID:         "anonymous",
			TenantID:       tenant.DefaultID,
			Roles:          []string{auth.RoleAnon},
			AnonymousDebug: true,
		}
		return p, Can(p, PermChat)
	}
	return auth.Principal{}, false
}

// StreamChatHandler WebSocket：握手前完成鉴权；URL 可带 ?access_token=（浏览器无法自定义 WS Header）。
func StreamChatHandler(c *gin.Context) {
	pr, ok := wsPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或无权限"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	_, payload, err := conn.ReadMessage()
	if err != nil {
		return
	}

	var in wsClientMsg
	if err := json.Unmarshal(payload, &in); err != nil {
		_ = conn.WriteJSON(wsOutMsg{Type: "error", Message: "无效 JSON"})
		return
	}
	if strings.TrimSpace(in.Msg) == "" {
		_ = conn.WriteJSON(wsOutMsg{Type: "error", Message: "msg 不能为空"})
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	streamTimeoutSeconds := envIntOrDefault("LLM_STREAM_TIMEOUT_SECONDS", 600)
	ctx, timeoutCancel := context.WithTimeout(ctx, time.Duration(streamTimeoutSeconds)*time.Second)
	defer timeoutCancel()

	conn.SetCloseHandler(func(code int, text string) error {
		cancel()
		return nil
	})

	var mu sync.Mutex
	write := func(msg wsOutMsg) error {
		mu.Lock()
		defer mu.Unlock()
		return conn.WriteJSON(msg)
	}
	_ = write(wsOutMsg{Type: "start", Message: "stream started"})

	hasDelta := false
	sessionID, err := agent.ChatStream(ctx, in.SessionID, in.Msg, pr.TenantID, func(delta string) error {
		hasDelta = true
		return write(wsOutMsg{Type: "delta", Content: delta})
	})
	_ = write(wsOutMsg{Type: "meta", SessionID: sessionID})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && hasDelta {
			_ = write(wsOutMsg{Type: "done", Message: "stream timeout reached, partial output returned", SessionID: sessionID})
			return
		}
		_ = write(wsOutMsg{Type: "error", Message: err.Error(), SessionID: sessionID})
		return
	}
	_ = write(wsOutMsg{Type: "done", SessionID: sessionID})
}

func envIntOrDefault(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}
