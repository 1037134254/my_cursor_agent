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

// StreamChatHandler 将连接升级为 WebSocket，接收一条 JSON 消息 {"msg":"..."}，以流式增量返回模型输出。
func StreamChatHandler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

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
	sessionID, err := agent.ChatStream(ctx, in.SessionID, in.Msg, func(delta string) error {
		hasDelta = true
		return write(wsOutMsg{Type: "delta", Content: delta})
	})
	_ = write(wsOutMsg{Type: "meta", SessionID: sessionID})
	if err != nil {
		// 流式超时时若已有部分输出，优雅结束，避免前端只看到报错。
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
