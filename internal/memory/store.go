package memory

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Message struct {
	Role    string
	Content string
}

var (
	mu       sync.RWMutex
	sessions = map[string][]Message{}
	seqNum   uint64
)

func sessionKey(tenantID, sessionID string) string {
	t := strings.TrimSpace(tenantID)
	if t == "" {
		t = "default"
	}
	return t + "\x1e" + sessionID
}

// EnsureSessionID 在同一租户下生成或沿用会话 id。
func EnsureSessionID(tenantID, sessionID string) string {
	if strings.TrimSpace(sessionID) != "" {
		return sessionID
	}
	t := strings.TrimSpace(tenantID)
	if t == "" {
		t = "default"
	}
	n := atomic.AddUint64(&seqNum, 1)
	return fmt.Sprintf("s-%s-%d-%d", t, time.Now().UnixMilli(), n)
}

// Append 追加一条消息（按租户 + 会话隔离）。
func Append(tenantID, sessionID string, msg Message) {
	sk := sessionKey(tenantID, sessionID)
	mu.Lock()
	defer mu.Unlock()
	sessions[sk] = append(sessions[sk], msg)
}

// GetRecent 获取会话近期消息。
func GetRecent(tenantID, sessionID string, max int) []Message {
	sk := sessionKey(tenantID, sessionID)
	mu.RLock()
	defer mu.RUnlock()
	src := sessions[sk]
	if len(src) == 0 {
		return nil
	}
	if max <= 0 || len(src) <= max {
		out := make([]Message, len(src))
		copy(out, src)
		return out
	}
	out := make([]Message, max)
	copy(out, src[len(src)-max:])
	return out
}
