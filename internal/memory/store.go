package memory

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Message struct {
	Role    string
	Content string
}

var (
	mu        sync.RWMutex
	sessions  = map[string][]Message{}
	seqNumber uint64
)

func EnsureSessionID(sessionID string) string {
	if sessionID != "" {
		return sessionID
	}
	n := atomic.AddUint64(&seqNumber, 1)
	return fmt.Sprintf("s-%d-%d", time.Now().UnixMilli(), n)
}

func Append(sessionID string, msg Message) {
	mu.Lock()
	defer mu.Unlock()
	sessions[sessionID] = append(sessions[sessionID], msg)
}

func GetRecent(sessionID string, max int) []Message {
	mu.RLock()
	defer mu.RUnlock()
	src := sessions[sessionID]
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
