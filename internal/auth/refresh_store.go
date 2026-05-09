package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type refreshRec struct {
	user UserRecord
	exp  time.Time
}

var (
	refreshMu   sync.Mutex
	refreshByID = map[string]refreshRec{}
)

// NewRefreshToken 生成 refresh 令牌并登记。
func NewRefreshToken(user UserRecord) (token string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	refreshMu.Lock()
	defer refreshMu.Unlock()
	refreshByID[token] = refreshRec{
		user: user,
		exp:  time.Now().Add(refreshTTL()),
	}
	return token, nil
}

// ConsumeRefreshToken 校验并删除 refresh（一次性），返回用户信息。
func ConsumeRefreshToken(token string) (UserRecord, bool) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	rec, ok := refreshByID[token]
	if !ok || time.Now().After(rec.exp) {
		if ok {
			delete(refreshByID, token)
		}
		return UserRecord{}, false
	}
	delete(refreshByID, token)
	return rec.user, true
}

// RevokeRefreshToken 登出。
func RevokeRefreshToken(token string) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	delete(refreshByID, token)
}
