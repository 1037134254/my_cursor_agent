package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const stateTTL = 10 * time.Minute

type stateRec struct {
	exp time.Time
}

var (
	stateMu sync.Mutex
	states  = map[string]stateRec{}
)

// NewState 生成 CSRF state 并登记。
func NewState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := hex.EncodeToString(b)
	stateMu.Lock()
	states[s] = stateRec{exp: time.Now().Add(stateTTL)}
	stateMu.Unlock()
	return s, nil
}

// ConsumeState 校验并消费 state（一次性）。
func ConsumeState(s string) bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	rec, ok := states[s]
	if !ok || time.Now().After(rec.exp) {
		delete(states, s)
		return false
	}
	delete(states, s)
	return true
}
